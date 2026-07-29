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

/** Domain AuthConfig → proto. Returns undefined for 'none' (empty on wire). */
export function toApiAuthConfig(config: AuthConfig | undefined | null): ProtoAuthConfig | undefined {
  if (!config || config.isNone()) return undefined
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
