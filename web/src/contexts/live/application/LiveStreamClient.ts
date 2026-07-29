import { Code, ConnectError } from '@connectrpc/connect'
import type { LiveEvent } from '@/gen/services/grpc/nats/v1/live/nats_live_service_pb'
import { liveClient } from '@/api/grpc/clients'
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

/** Max messages retained in the paused buffer before oldest batches are dropped. */
const MAX_PAUSED_MESSAGES = 5000

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

  private currentSubscription: string | null = null

  public onConnected?: (payload: WSConnectedPayload) => void
  public onSubscribed?: (payload: WSSubscribedPayload) => void
  public onBatch?: (payload: WSBatchPayload) => void
  public onStats?: (payload: WSStatsPayload) => void
  public onError?: (payload: WSErrorPayload) => void
  public onDisconnect?: () => void
  public onReconnecting?: (attempt: number, maxRetries: number) => void
  public onProtoReload?: (payload: WSProtoReloadPayload) => void

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
    this.cancelStream()
    this.currentSubscription = null
    this.connected = false
    this.onDisconnect?.()
  }

  subscribe(stream: string): void {
    if (this.currentSubscription === stream) {
      logger.debug(`Already subscribed to stream: ${stream}`)
      return
    }
    this.clearReconnectTimeout()
    logger.debug(`Subscribing to stream: ${stream}`)
    this.currentSubscription = stream
    this.reconnectAttempts = 0
    this.cancelStream()
    this.startStream(stream)
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

  private async startStream(streamName: string): Promise<void> {
    this.abortController = new AbortController()

    try {
      const stream = liveClient.subscribe(
        {
          connectionId: this.connectionId,
          subscriptions: [{ subject: '>', streamName }],
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

      if (!this.intentionalClose && this.currentSubscription === streamName) {
        this.scheduleReconnect(streamName)
      }
    } catch (err: unknown) {
      const connectErr = ConnectError.from(err)
      const errorMessage = connectErr.rawMessage || 'Stream error'

      if (this.intentionalClose || connectErr.code === Code.Canceled || connectErr.code === Code.Aborted) {
        return
      }

      logger.error('gRPC stream error:', err)
      this.onError?.({
        message: errorMessage,
        code: 'stream_error',
      })

      // Errors that won't be fixed by reconnecting: authz and missing resources.
      const isFatal =
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

      if (!this.intentionalClose && this.currentSubscription === streamName) {
        this.scheduleReconnect(streamName)
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
        this.onError?.({ message: err.message, code: err.code || undefined })
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
    }
  }

  private scheduleReconnect(streamName: string): void {
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
      this.startStream(streamName)
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
    this.paused = true
  }

  /**
   * Bound the paused buffer so a high-throughput stream can't grow memory
   * without limit while the view is paused. The live view only keeps the most
   * recent messages anyway, so dropping the oldest buffered batches is safe.
   */
  private trimPausedBatches(): void {
    let total = 0
    for (const b of this.pausedBatches) total += b.count
    while (total > MAX_PAUSED_MESSAGES && this.pausedBatches.length > 1) {
      const dropped = this.pausedBatches.shift()
      total -= dropped?.count ?? 0
    }
  }

  resume(): void {
    this.paused = false
    const buffered = this.pausedBatches
    this.pausedBatches = []
    for (const batch of buffered) {
      this.onBatch?.(batch)
    }
  }
}
