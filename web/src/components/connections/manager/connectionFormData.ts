import type { AuthMethod } from '@/contexts/connection'

export type AuthMethodTab = AuthMethod

/**
 * The shape `ConnectionForm` operates on. Pure data — no React. Lives in its
 * own file so the form component file can be HMR fast-refresh friendly
 * (mixed component + constant exports break fast refresh).
 */
export interface ConnectionFormData {
  name: string
  description: string
  urls: string[]
  authMethod: AuthMethodTab
  username: string
  password: string
  token: string
  nkeySeed: string
  credentials: string
  tls: {
    caCert: string
    clientCert: string
    clientKey: string
    skipVerify: boolean
    tlsFirst: boolean
  }
  /**
   * Presence of stored secrets, from the server's read-only `has_*` flags. The
   * API never returns secret values, so on edit these drive a "leave blank to
   * keep" hint; an empty secret input on submit preserves the stored value.
   */
  secretsSet: {
    password: boolean
    token: boolean
    nkeySeed: boolean
    credentials: boolean
    clientKey: boolean
  }
}

/** No stored secrets — used by `defaultFormData` and create mode. */
export const noSecretsSet: ConnectionFormData['secretsSet'] = {
  password: false,
  token: false,
  nkeySeed: false,
  credentials: false,
  clientKey: false,
}

/** A blank TLS sub-tree — used by `defaultFormData` and edit pre-fill. */
export const emptyTls: ConnectionFormData['tls'] = {
  caCert: '',
  clientCert: '',
  clientKey: '',
  skipVerify: false,
  tlsFirst: false,
}

export const AUTH_LABELS: Record<AuthMethodTab, string> = {
  none: 'None',
  userpass: 'User/Pass',
  token: 'Token',
  nkey: 'NKey',
  credentials: 'Credentials',
}
