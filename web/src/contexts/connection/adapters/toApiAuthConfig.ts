import { create } from '@bufbuild/protobuf'
import {
  AuthConfigSchema,
  AuthMethod as ProtoAuthMethod,
  type AuthConfig as ProtoAuthConfig,
} from '@/gen/types/nats/nats_connection_pb'
import type { AuthConfig } from '../domain/value-objects/AuthConfig'
import type { AuthMethod } from '../domain/value-objects/AuthMethod'

const DOMAIN_TO_PROTO: Record<AuthMethod, ProtoAuthMethod> = {
  none: ProtoAuthMethod.UNSPECIFIED,
  userpass: ProtoAuthMethod.USER_PASSWORD,
  token: ProtoAuthMethod.TOKEN,
  nkey: ProtoAuthMethod.NKEY,
  credentials: ProtoAuthMethod.CREDENTIALS,
}

/**
 * Domain AuthConfig → proto.
 *  - explicit: false (default) — undefined for 'none' (leave auth untouched on
 *    UpdateConnection; used by Test on unsaved).
 *  - explicit: true — always emit an object, even for 'none', so the backend
 *    treats it as a replace (AUTH_METHOD_UNSPECIFIED clears stored auth).
 *    Without this, an Edit-mode "Authentication: None" save sent no `auth`
 *    field at all, which the backend reads as "leave unchanged" — the UI
 *    reported success while the old credentials stayed in effect.
 */
export function toApiAuthConfig(
  config: AuthConfig | undefined | null,
  opts: { explicit: boolean } = { explicit: false },
): ProtoAuthConfig | undefined {
  if (!config || config.isNone()) {
    if (!opts.explicit) return undefined
    return create(AuthConfigSchema, { method: ProtoAuthMethod.UNSPECIFIED })
  }
  return create(AuthConfigSchema, {
    method: DOMAIN_TO_PROTO[config.method],
    username: config.username,
    password: config.password,
    token: config.token,
    nkeySeed: config.nkeySeed,
    credentials: config.credentials,
    jwt: config.jwt,
  })
}
