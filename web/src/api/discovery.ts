import type { AccessCheck } from '@/shared/domain/access'
import { durToNanos, tsToMillis } from '@/utils/timestamp'
import type {
  MicroEndpoint as ProtoMicroEndpoint,
  MicroService as ProtoMicroService,
} from '../gen/types/nats/nats_micro_pb'
import { toAccessCheck } from './access'
import { discoveryClient } from './grpc/clients'

interface MicroEndpointStats {
  num_requests: number
  num_errors: number
  last_error: string
  processing_time_ns: number
  average_processing_time_ns: number
}

interface ProtoMethodMatch {
  source_id: string
  service: string
  method: string
  input_type: string
  output_type: string
}

export interface MicroEndpoint {
  name: string
  subject: string
  queue_group: string
  metadata: Record<string, string>
  stats?: MicroEndpointStats
  proto_method?: ProtoMethodMatch
}

export interface MicroInstance {
  id: string
  version: string
  metadata: Record<string, string>
  started?: number
  endpoints: MicroEndpoint[]
  rtt_ns?: number
  info_json?: string
  stats_json?: string
}

export interface MicroService {
  name: string
  description: string
  versions: string[]
  instances: MicroInstance[]
  endpoints: MicroEndpoint[]
}

export interface MicroDiscovery {
  info_access: AccessCheck
  stats_access?: AccessCheck
  services: MicroService[]
}

function toEndpoint(e: ProtoMicroEndpoint): MicroEndpoint {
  return {
    name: e.name,
    subject: e.subject,
    queue_group: e.queueGroup,
    metadata: e.metadata,
    stats: e.stats
      ? {
          num_requests: Number(e.stats.numRequests),
          num_errors: Number(e.stats.numErrors),
          last_error: e.stats.lastError,
          processing_time_ns: durToNanos(e.stats.processingTime),
          average_processing_time_ns: durToNanos(e.stats.averageProcessingTime),
        }
      : undefined,
    proto_method: e.protoMethod
      ? {
          source_id: e.protoMethod.sourceId,
          service: e.protoMethod.service,
          method: e.protoMethod.method,
          input_type: e.protoMethod.inputType,
          output_type: e.protoMethod.outputType,
        }
      : undefined,
  }
}

function toService(s: ProtoMicroService): MicroService {
  return {
    name: s.name,
    description: s.description,
    versions: s.versions,
    instances: s.instances.map((i) => ({
      id: i.id,
      version: i.version,
      metadata: i.metadata,
      started: i.started ? tsToMillis(i.started) : undefined,
      endpoints: i.endpoints.map(toEndpoint),
      rtt_ns: i.rtt ? durToNanos(i.rtt) : undefined,
      info_json: i.infoJson || undefined,
      stats_json: i.statsJson || undefined,
    })),
    endpoints: s.endpoints.map(toEndpoint),
  }
}

export async function listServices(
  connectionId: string,
  { skipStats }: { skipStats: boolean },
  signal?: AbortSignal,
): Promise<MicroDiscovery> {
  const response = await discoveryClient.listServices({ connectionId, skipStats }, { signal })
  return {
    info_access: toAccessCheck(response.infoAccess) ?? { status: 'unknown', operation: 'publish', subject: '$SRV.INFO' },
    stats_access: toAccessCheck(response.statsAccess),
    services: response.services.map(toService),
  }
}
