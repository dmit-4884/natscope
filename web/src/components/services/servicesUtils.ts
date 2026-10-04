import { schemaTypeId } from '@/api/proto'
import type { RequestDraft } from '@/stores/requestDraftStore'
import type { MicroEndpoint, MicroService } from '@/api/discovery'

export interface ServiceTotals {
  requests: number
  errors: number
  averageNs: number
}

export function serviceTotals(service: MicroService): ServiceTotals | null {
  const withStats = service.endpoints.filter((e) => e.stats)
  if (withStats.length === 0) return null
  let requests = 0
  let errors = 0
  let processingNs = 0
  for (const e of withStats) {
    requests += e.stats!.num_requests
    errors += e.stats!.num_errors
    processingNs += e.stats!.processing_time_ns
  }
  return { requests, errors, averageNs: requests > 0 ? processingNs / requests : 0 }
}

export function filterServices(services: MicroService[], query: string): MicroService[] {
  const q = query.trim().toLowerCase()
  if (!q) return services
  return services.filter(
    (s) =>
      s.name.toLowerCase().includes(q) ||
      s.description.toLowerCase().includes(q) ||
      s.endpoints.some((e) => e.subject.toLowerCase().includes(q) || e.name.toLowerCase().includes(q)),
  )
}

export function callDraftPatch(
  endpoint: MicroEndpoint,
  draft: Pick<RequestDraft, 'subject' | 'replyTypes' | 'requestTypes'>,
): Partial<RequestDraft> {
  const method = endpoint.proto_method
  const patch: Partial<RequestDraft> = { subject: endpoint.subject }
  if (draft.subject !== endpoint.subject || method) patch.payload = method ? '{}' : ''
  if (method) {
    patch.requestTypes = { ...draft.requestTypes, [endpoint.subject]: { messageType: method.input_type, sourceId: method.source_id } }
    patch.replyTypes = { ...draft.replyTypes, [endpoint.subject]: schemaTypeId(method.source_id, method.output_type) }
  }
  return patch
}
