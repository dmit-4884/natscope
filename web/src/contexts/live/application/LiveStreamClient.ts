import { Code, ConnectError } from '@connectrpc/connect'
import type { LiveEvent, LiveSubscription } from '@/gen/services/grpc/nats/v1/live/nats_live_service_pb'
import { toAccessCheck } from '@/api/access'
import { liveClient } from '@/api/grpc/clients'
import type { AccessCheck } from '@/shared/domain/access'
import { logger } from '@/utils/logger'
import {
  toLiveMessagePayload,
  type LiveMessagePayload,
} from '../adapters/toLiveMessagePayload'

export type WSMessagePayload = LiveMessagePayload

export interface WSBatchPayload {
  messages: WSMessagePayload[]
  count: number
}

export interface WSStatsPayload {
  messages_received: number
  messages_dropped: number
  msg_per_second: number
}

export interface WSErrorPayload {
  message: string
  code?: string
  access?: AccessCheck
}

export interface SubjectSessionLimits {
  maxPayloadBytes?: number
  maxDisplayRate?: number
  exclude?: string[]
}

type LiveTarget = { stream: string } | { subjects: string[]; limits?: SubjectSessionLimits }

function targetKey(target: LiveTarget): string {
  if ('stream' in target) return `stream:${target.stream}`
  const { maxPayloadBytes = '', maxDisplayRate = '', exclude = [] } = target.limits ?? {}
  return `subjects:${target.subjects.join('\n')}|${maxPayloadBytes}|${maxDisplayRate}|${exclude.join('\n')}`
}

function targetSubscriptions(target: LiveTarget): Pick<LiveSubscription, 'subject' | 'streamName'>[] {
  return 'stream' in target
    ? [{ subject: '>', streamName: target.stream }]
    : target.subjects.map((subject) => ({ subject }))
}

export interface WSSubscribedPayload {
  stream: string
  status: string
}

export interface WSConnectedPayload {
  status: string
  stream_id: string
}

export interface WSProtoReloadPayload {
  path: string
  messages_count: number
}

interface ReconnectConfig {
  maxRetries: number
  baseDelay: number
  maxDelay: number
}

const DEFAULT_RECONNECT: ReconnectConfig = {
  maxRetries: 10,
  baseDelay: 1000,
  maxDelay: 30000,
}

/** Max messages, and characters of payload, retained in the paused buffer before oldest batches are dropped. */
const MAX_PAUSED_MESSAGES = 5000
const MAX_PAUSED_CHARS = 32 << 20
const PAUSED_REPORT_MS = 1000

function batchChars(batch: WSBatchPayload): number {
  let chars = 0
  for (const m of batch.messages) chars += m.data_base64.length + (m.decoded !== undefined ? m.data_size : 0)
  return chars
}

/**
 * gRPC server-streaming wrapper for live NATS subscriptions.
 * Keeps the legacy LiveWebSocket alias for unchanged call-sites.
 */
export class LiveStreamClient {
  private abortController: AbortController | null = null
  private connectionId: string
  private reconnectAttempts = 0
  private reconnectConfig: ReconnectConfig
  private reconnectTimeout: ReturnType<typeof setTimeout> | null = null
  private intentionalClose = false
  private connected = false
  private paused = false
  private pausedBatches: WSBatchPayload[] = []
  private pausedArrivals = 0
  private arrivalsReportedAt = 0
  private arrivalsTimer: ReturnType<typeof setTimeout> | null = null

  private currentSubscription: string | null = null

  public onConnected?: (payload: WSConnectedPayload) => void
  public onSubscribed?: (payload: WSSubscribedPayload) => void
  public onBatch?: (payload: WSBatchPayload) => void
  public onStats?: (payload: WSStatsPayload) => void
  public onError?: (payload: WSErrorPayload) => void
  public onDisconnect?: () => void
  public onReconnecting?: (attempt: number, maxRetries: number) => void
  public onProtoReload?: (payload: WSProtoReloadPayload) => void
  public onBuffered?: (count: number) => void
  public onLink?: (connected: boolean) => void

  constructor(connectionId: string, _token?: string, config?: Partial<ReconnectConfig>) {
    this.connectionId = connectionId
    this.reconnectConfig = { ...DEFAULT_RECONNECT, ...config }
  }

  connect(): void {
    if (this.connected) return
    this.intentionalClose = false
    this.connected = true
    this.onConnected?.({
      status: 'connected',
      stream_id: this.connectionId,
    })
  }

  disconnect(): void {
    this.intentionalClose = true
    this.clearArrivalsTimer()
    this.cancelStream()
    this.currentSubscription = null
    this.connected = false
    this.onDisconnect?.()
  }

  subscribe(stream: string): void {
    this.subscribeTarget({ stream })
  }

  subscribeSubjects(subjects: string[], limits?: SubjectSessionLimits): void {
    this.subscribeTarget({ subjects, limits })
  }

  private subscribeTarget(target: LiveTarget): void {
    if (!this.connected) return
    const key = targetKey(target)
    if (this.currentSubscription === key) {
      logger.debug(`Already subscribed: ${key}`)
      return
    }
    this.clearReconnectTimeout()
    logger.debug(`Subscribing: ${key}`)
    this.currentSubscription = key
    this.reconnectAttempts = 0
    this.cancelStream()
    this.startStream(target)
  }

  unsubscribe(): void {
    this.clearReconnectTimeout()
    if (!this.currentSubscription) return
    logger.debug('Unsubscribing from current stream')
    this.currentSubscription = null
    this.cancelStream()
  }

  private clearReconnectTimeout(): void {
    if (this.reconnectTimeout) {
      clearTimeout(this.reconnectTimeout)
      this.reconnectTimeout = null
    }
  }

  private cancelStream(): void {
    this.clearReconnectTimeout()
    if (this.abortController) {
      this.abortController.abort()
      this.abortController = null
    }
  }

  private async startStream(target: LiveTarget): Promise<void> {
    this.abortController = new AbortController()
    const key = targetKey(target)
    const streamName = 'stream' in target ? target.stream : ''

    try {
      const limits = 'subjects' in target ? target.limits : undefined
      const stream = liveClient.subscribe(
        {
          connectionId: this.connectionId,
          subscriptions: targetSubscriptions(target),
          ...(limits?.maxPayloadBytes !== undefined && { maxPayloadBytes: limits.maxPayloadBytes }),
          ...(limits?.maxDisplayRate !== undefined && { maxDisplayRate: limits.maxDisplayRate }),
          ...(limits?.exclude !== undefined && { excludeSubjects: limits.exclude }),
        },
        { signal: this.abortController.signal },
      )

      let established = false
      for await (const event of stream) {
        if (!established) {
          established = true
          this.reconnectAttempts = 0
          this.onSubscribed?.({ stream: streamName, status: 'subscribed' })
        }
        this.handleEvent(event, streamName)
      }

      if (!this.intentionalClose && this.currentSubscription === key) {
        this.scheduleReconnect(target)
      }
    } catch (err: unknown) {
      const connectErr = ConnectError.from(err)
      const errorMessage = connectErr.rawMessage || 'Stream error'

      if (this.intentionalClose || connectErr.code === Code.Canceled) {
        return
      }

      logger.error('gRPC stream error:', err)
      this.onError?.({
        message: errorMessage,
        code: 'stream_error',
      })

      // Errors that won't be fixed by reconnecting: authz, missing resources and rejected requests.
      const isFatal =
        connectErr.code === Code.InvalidArgument ||
        connectErr.code === Code.PermissionDenied ||
        connectErr.code === Code.Unauthenticated ||
        connectErr.code === Code.NotFound ||
        connectErr.code === Code.FailedPrecondition
      if (isFatal) {
        logger.warn('Fatal error, not reconnecting:', errorMessage)
        this.currentSubscription = null
        this.onDisconnect?.()
        return
      }

      if (!this.intentionalClose && this.currentSubscription === key) {
        this.scheduleReconnect(target)
      }
    }
  }

  private handleEvent(event: LiveEvent, streamName: string): void {
    switch (event.event.case) {
      case 'batch': {
        const messages = event.event.value.messages.map((m) =>
          toLiveMessagePayload(m, streamName),
        )
        const payload: WSBatchPayload = { messages, count: messages.length }
        if (this.paused) {
          this.pausedBatches.push(payload)
          this.trimPausedBatches()
          this.pausedArrivals += payload.count
          this.reportArrivals()
        } else {
          this.onBatch?.(payload)
        }
        break
      }
      case 'stats': {
        const stats = event.event.value
        this.onStats?.({
          messages_received: Number(stats.totalMessages),
          messages_dropped: Number(stats.messagesDropped),
          msg_per_second: Number(stats.messagesPerSecond),
        })
        break
      }
      case 'error': {
        const err = event.event.value
        this.onError?.({ message: err.message, code: err.code || undefined, access: toAccessCheck(err.access) })
        break
      }
      case 'protoReload': {
        const reload = event.event.value
        this.onProtoReload?.({
          path: '',
          messages_count: reload.messagesCount,
        })
        break
      }
      case 'connection':
        this.onLink?.(event.event.value.connected)
        break
    }
  }

  private scheduleReconnect(target: LiveTarget): void {
    if (this.intentionalClose) return
    if (this.reconnectAttempts >= this.reconnectConfig.maxRetries) {
      logger.warn('Max reconnection attempts reached')
      this.currentSubscription = null
      this.onError?.({ message: 'Connection failed after max retries', code: 'max_retries' })
      this.onDisconnect?.()
      return
    }

    const delay = Math.min(
      this.reconnectConfig.baseDelay * Math.pow(2, this.reconnectAttempts) + Math.random() * 1000,
      this.reconnectConfig.maxDelay,
    )

    this.reconnectAttempts++
    this.onReconnecting?.(this.reconnectAttempts, this.reconnectConfig.maxRetries)

    logger.debug(`Reconnecting in ${Math.round(delay)}ms (attempt ${this.reconnectAttempts}/${this.reconnectConfig.maxRetries})`)

    this.reconnectTimeout = setTimeout(() => {
      this.startStream(target)
    }, delay)
  }

  getCurrentSubscription(): string | null {
    return this.currentSubscription
  }

  isPaused(): boolean {
    return this.paused
  }

  isConnected(): boolean {
    return this.connected
  }

  pause(): void {
    if (!this.paused) this.pausedArrivals = 0
    this.paused = true
  }

  private reportArrivals(): void {
    const wait = this.arrivalsReportedAt + PAUSED_REPORT_MS - Date.now()
    if (wait <= 0) {
      this.arrivalsReportedAt = Date.now()
      this.onBuffered?.(this.pausedArrivals)
      return
    }
    this.arrivalsTimer ??= setTimeout(() => {
      this.arrivalsTimer = null
      this.arrivalsReportedAt = Date.now()
      if (this.paused) this.onBuffered?.(this.pausedArrivals)
    }, wait)
  }

  private clearArrivalsTimer(): void {
    if (this.arrivalsTimer) clearTimeout(this.arrivalsTimer)
    this.arrivalsTimer = null
  }

  /**
   * Bound the paused buffer so a high-throughput stream can't grow memory
   * without limit while the view is paused. The live view only keeps the most
   * recent messages anyway, so dropping the oldest buffered batches is safe.
   */
  private trimPausedBatches(): void {
    let total = 0
    let chars = 0
    for (const b of this.pausedBatches) {
      total += b.count
      chars += batchChars(b)
    }
    while ((total > MAX_PAUSED_MESSAGES || chars > MAX_PAUSED_CHARS) && this.pausedBatches.length > 1) {
      const dropped = this.pausedBatches.shift()
      if (!dropped) break
      total -= dropped.count
      chars -= batchChars(dropped)
    }
  }

  discardPaused(): void {
    this.pausedBatches = []
    this.pausedArrivals = 0
    this.clearArrivalsTimer()
    this.onBuffered?.(0)
  }

  resume(): void {
    this.paused = false
    const buffered = this.pausedBatches
    this.pausedBatches = []
    this.pausedArrivals = 0
    this.clearArrivalsTimer()
    this.onBuffered?.(0)
    for (const batch of buffered) {
      this.onBatch?.(batch)
    }
  }
}
