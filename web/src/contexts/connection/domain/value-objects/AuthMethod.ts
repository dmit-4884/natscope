/** Auth method for NATS connections; string union avoids the proto enum. */
export type AuthMethod =
  | 'none'
  | 'userpass'
  | 'token'
  | 'nkey'
  | 'credentials'

const AUTH_METHODS: readonly AuthMethod[] = [
  'none',
  'userpass',
  'token',
  'nkey',
  'credentials',
]

export function isAuthMethod(value: unknown): value is AuthMethod {
  return typeof value === 'string' && (AUTH_METHODS as readonly string[]).includes(value)
}
