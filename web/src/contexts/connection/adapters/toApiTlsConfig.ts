import { create } from '@bufbuild/protobuf'
import { TlsConfigSchema, type TlsConfig as ProtoTlsConfig } from '@/gen/types/nats/nats_connection_pb'

/** Plain TLS input (ConnectionForm shape); avoids importing the proto class. */
export interface TlsConfigInput {
  caCert: string
  clientCert: string
  clientKey: string
  skipVerify: boolean
  tlsFirst: boolean
}

/**
 * Domain TLS settings → proto TlsConfig.
 *  - explicit: true — always emit an object (even empty) so the backend
 *    treats it as a replace; empty collapses to nil. Propagates "Remove TLS".
 *  - explicit: false — undefined when nothing set; used by Test on unsaved.
 */
export function toApiTlsConfig(
  tls: TlsConfigInput,
  opts: { explicit: boolean },
): ProtoTlsConfig | undefined {
  const hasContent =
    tls.caCert.trim() !== '' ||
    tls.clientCert.trim() !== '' ||
    tls.clientKey.trim() !== '' ||
    tls.skipVerify ||
    tls.tlsFirst
  if (!hasContent && !opts.explicit) return undefined
  return create(TlsConfigSchema, {
    caCert: tls.caCert.trim(),
    clientCert: tls.clientCert.trim(),
    clientKey: tls.clientKey.trim(),
    skipVerify: tls.skipVerify,
    tlsFirst: tls.tlsFirst,
  })
}
