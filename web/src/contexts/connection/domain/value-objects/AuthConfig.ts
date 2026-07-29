import { ValueObject, Result, DomainError } from '@/shared'
import { type AuthMethod, isAuthMethod } from './AuthMethod'

export interface AuthConfigProps {
  method: AuthMethod
  username?: string
  password?: string
  token?: string
  nkeySeed?: string
  credentials?: string
  jwt?: string
}

export class AuthConfig extends ValueObject<AuthConfigProps> {
  private constructor(props: AuthConfigProps) {
    super(props)
  }

  get method(): AuthMethod {
    return this.props.method
  }
  get username(): string | undefined {
    return this.props.username
  }
  get password(): string | undefined {
    return this.props.password
  }
  get token(): string | undefined {
    return this.props.token
  }
  get nkeySeed(): string | undefined {
    return this.props.nkeySeed
  }
  get credentials(): string | undefined {
    return this.props.credentials
  }
  get jwt(): string | undefined {
    return this.props.jwt
  }

  toObject(): AuthConfigProps {
    return { ...this.props }
  }

  isNone(): boolean {
    return this.props.method === 'none'
  }

  /** Short, human-readable summary for list views. */
  summary(): string {
    switch (this.props.method) {
      case 'none':
        return 'No auth'
      case 'userpass':
        return this.props.username ? `User: ${this.props.username}` : 'User / password'

      case 'token':
        return 'Token'
      case 'nkey':
        return 'NKey'
      case 'credentials':
        return 'Credentials'
    }
  }

  /** Factory for the common "no auth" case. */
  static none(): AuthConfig {
    return new AuthConfig({ method: 'none' })
  }

  static create(input: AuthConfigProps): Result<AuthConfig, DomainError> {
    if (!isAuthMethod(input.method)) {
      return Result.err(DomainError.validation('Unknown auth method', 'method'))
    }
    switch (input.method) {
      case 'userpass':
        if (!input.username || input.username.trim() === '') {
          return Result.err(DomainError.validation('Username is required', 'username'))
        }
        break
      case 'token':
        if (!input.token || input.token.trim() === '') {
          return Result.err(DomainError.validation('Token is required', 'token'))
        }
        break
      case 'nkey':
        if (!input.nkeySeed || input.nkeySeed.trim() === '') {
          return Result.err(DomainError.validation('NKey seed is required', 'nkeySeed'))
        }
        break
      case 'credentials':
        if (!input.credentials || input.credentials.trim() === '') {
          return Result.err(DomainError.validation('Credentials content is required', 'credentials'))
        }
        break
      case 'none':
        break
    }
    return Result.ok(new AuthConfig({ ...input }))
  }

  /** Factory for adapters — builds without validation for trusted input. */
  static fromTrusted(input: AuthConfigProps): AuthConfig {
    return new AuthConfig({ ...input })
  }
}
