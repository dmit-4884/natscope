import { Entity } from '@/shared'
import { StreamName } from '../value-objects/StreamName'
import { StreamConfig } from '../value-objects/StreamConfig'

interface StreamStateInfo {
  messages: number
  bytes: number
  firstSeq: number
  lastSeq: number
  firstTs: Date | null
  lastTs: Date | null
  consumerCount: number
}

interface StreamProps {
  id: string
  name: StreamName
  description: string
  subjects: string[]
  config: StreamConfig
  state: StreamStateInfo
  created: Date
}

/** NATS JetStream stream with config and state. */
export class Stream extends Entity<StreamProps> {
  private constructor(props: StreamProps) {
    super(props)
  }

  static fromApi(data: {
    name: string
    description?: string
    subjects: string[]
    messages: number
    bytes: number
    consumer_count: number
    created: number // Unix ms
    config: {
      retention: string
      max_msgs: number
      max_bytes: number
      max_age: number
      max_consumers?: number
      max_msgs_per_subject?: number
      max_msg_size?: number
      storage?: string
      discard?: string
      num_replicas?: number
      duplicate_window?: number
      sealed?: boolean
      deny_delete?: boolean
      deny_purge?: boolean
    }
    state?: {
      messages: number
      bytes: number
      first_seq: number
      last_seq: number
      first_ts: number // Unix ms
      last_ts: number // Unix ms
      consumer_count?: number
    }
  }): Stream {
    const name = StreamName.fromTrusted(data.name)
    const config = StreamConfig.fromApi(data.config)

    const state: StreamStateInfo = data.state
      ? {
          messages: data.state.messages,
          bytes: data.state.bytes,
          firstSeq: data.state.first_seq,
          lastSeq: data.state.last_seq,
          firstTs: data.state.first_ts ? new Date(data.state.first_ts) : null,
          lastTs: data.state.last_ts ? new Date(data.state.last_ts) : null,
          consumerCount: data.state.consumer_count ?? data.consumer_count,
        }
      : {
          messages: data.messages,
          bytes: data.bytes,
          firstSeq: 0,
          lastSeq: 0,
          firstTs: null,
          lastTs: null,
          consumerCount: data.consumer_count,
        }

    return new Stream({
      id: data.name,
      name,
      description: data.description ?? '',
      subjects: data.subjects,
      config,
      state,
      created: new Date(data.created),
    })
  }

  get name(): StreamName {
    return this.props.name
  }

  get description(): string {
    return this.props.description
  }

  get subjects(): ReadonlyArray<string> {
    return this.props.subjects
  }

  get config(): StreamConfig {
    return this.props.config
  }

  get messageCount(): number {
    return this.props.state.messages
  }

  get bytes(): number {
    return this.props.state.bytes
  }

  get consumerCount(): number {
    return this.props.state.consumerCount
  }

  get firstSequence(): number {
    return this.props.state.firstSeq
  }

  get lastSequence(): number {
    return this.props.state.lastSeq
  }

  get created(): Date {
    return this.props.created
  }
}
