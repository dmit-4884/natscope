import { Entity } from '@/shared'
import type { AuthMethod, SavedConnection } from '@/api/connections'
import { NatsUrl } from '../value-objects/NatsUrl'

/** @deprecated use AuthMethod from '@/contexts/connection' */
export type AuthMethodType = AuthMethod

interface ConnectionProps {
  id: string
  name: string
  description: string | null
  urls: NatsUrl[]
  authMethod: AuthMethodType
  hasTls: boolean
  hasAdvancedConfig: boolean
  createdAt: Date
  updatedAt: Date | null
}

/** A saved NATS connection configuration. */
export class Connection extends Entity<ConnectionProps> {
  private constructor(props: ConnectionProps) {
    super(props)
  }

  static fromApi(data: SavedConnection): Connection {
    return new Connection({
      id: data.id,
      name: data.name,
      description: data.description ?? null,
      urls: data.urls.map((u) => NatsUrl.fromTrusted(u)),
      authMethod: data.auth?.method ?? 'none',
      hasTls: !!data.tls,
      hasAdvancedConfig: !!data.connection || !!data.reconnect || !!data.ping,
      createdAt: new Date(data.createdAt),
      updatedAt: data.updatedAt ? new Date(data.updatedAt) : null,
    })
  }

  get name(): string {
    return this.props.name
  }

  get description(): string | null {
    return this.props.description
  }

  get urls(): NatsUrl[] {
    return this.props.urls
  }

  get primaryUrl(): NatsUrl | undefined {
    return this.props.urls[0]
  }

  get authMethod(): AuthMethodType {
    return this.props.authMethod
  }

  get hasTls(): boolean {
    return this.props.hasTls
  }

  get hasAdvancedConfig(): boolean {
    return this.props.hasAdvancedConfig
  }

  get createdAt(): Date {
    return this.props.createdAt
  }

  get isSecure(): boolean {
    return this.props.hasTls || this.props.urls.some((u) => u.isSecure())
  }

  get serverCount(): number {
    return this.props.urls.length
  }
}

