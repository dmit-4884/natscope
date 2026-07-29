import { ValueObject, Result } from '@/shared'

/** Validation errors for NatsUrl. */
export class NatsUrlValidationError extends Error {
  constructor(
    message: string,
    public readonly code: 'EMPTY' | 'INVALID_FORMAT'
  ) {
    super(message)
    this.name = 'NatsUrlValidationError'
  }
}

interface NatsUrlProps {
  value: string
  protocol: 'nats' | 'tls' | 'ws' | 'wss'
  host: string
  port: number
}

/** NATS server URL with protocol, host, and port. */
export class NatsUrl extends ValueObject<NatsUrlProps> {
  private static readonly DEFAULT_PORT = 4222
  private static readonly URL_PATTERN = /^(nats|tls|ws|wss):\/\/(?:[^@/]*@)?([^:/@]+)(?::(\d+))?$/

  static create(value: string): Result<NatsUrl, NatsUrlValidationError> {
    const trimmed = value.trim()

    if (!trimmed) {
      return Result.err(
        new NatsUrlValidationError('URL cannot be empty', 'EMPTY')
      )
    }

    const match = NatsUrl.URL_PATTERN.exec(trimmed)
    if (!match) {
      return Result.err(
        new NatsUrlValidationError(
          'Invalid NATS URL format. Expected: nats://host:port',
          'INVALID_FORMAT'
        )
      )
    }

    const [, protocol, host, portStr] = match
    const port = portStr ? parseInt(portStr, 10) : NatsUrl.DEFAULT_PORT

    return Result.ok(
      new NatsUrl({
        value: trimmed,
        protocol: protocol as 'nats' | 'tls' | 'ws' | 'wss',
        host,
        port,
      })
    )
  }

  static fromTrusted(value: string): NatsUrl {
    const match = NatsUrl.URL_PATTERN.exec(value)
    if (match) {
      const [, protocol, host, portStr] = match
      return new NatsUrl({
        value,
        protocol: protocol as 'nats' | 'tls' | 'ws' | 'wss',
        host,
        port: portStr ? parseInt(portStr, 10) : NatsUrl.DEFAULT_PORT,
      })
    }

    return new NatsUrl({
      value,
      protocol: 'nats',
      host: 'unknown',
      port: NatsUrl.DEFAULT_PORT,
    })
  }

  get value(): string {
    return this.props.value
  }

  get protocol(): string {
    return this.props.protocol
  }

  get host(): string {
    return this.props.host
  }

  get port(): number {
    return this.props.port
  }

  isSecure(): boolean {
    return this.props.protocol === 'tls' || this.props.protocol === 'wss'
  }

  isWebSocket(): boolean {
    return this.props.protocol === 'ws' || this.props.protocol === 'wss'
  }

  displayString(): string {
    return `${this.props.host}:${this.props.port}`
  }

  toString(): string {
    return this.props.value
  }
}
