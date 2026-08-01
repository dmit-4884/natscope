import type { ConsumerInfo } from '@/types/nats'
import type { ConsumerCreateRequest, ConsumerUpdateRequest } from '@/types/management'

export const defaultConsumerConfig: ConsumerCreateRequest = {
  name: '',
  ephemeral: false,
  deliver_policy: 'all',
  ack_policy: 'explicit',
  replay_policy: 'instant',
  max_ack_pending: 1000,
  max_waiting: 512,
}

export function consumerToConfig(consumer: ConsumerInfo): ConsumerCreateRequest {
  return {
    name: consumer.name,
    durable_name: consumer.config?.durable_name,
    ephemeral: !consumer.config?.durable_name,
    description: consumer.config?.description || '',
    deliver_policy: (consumer.config?.deliver_policy?.toLowerCase() || 'all') as ConsumerCreateRequest['deliver_policy'],
    opt_start_seq: consumer.config?.opt_start_seq,
    opt_start_time: consumer.config?.opt_start_time,
    ack_policy: (consumer.config?.ack_policy?.toLowerCase() || 'explicit') as 'none' | 'all' | 'explicit',
    ack_wait: consumer.config?.ack_wait,
    max_deliver: consumer.config?.max_deliver,
    backoff: consumer.config?.backoff,
    filter_subject: consumer.config?.filter_subject,
    filter_subjects: consumer.config?.filter_subjects,
    replay_policy: (consumer.config?.replay_policy?.toLowerCase() || 'instant') as 'instant' | 'original',
    rate_limit_bps: consumer.config?.rate_limit_bps,
    sample_freq: consumer.config?.sample_freq,
    max_waiting: consumer.config?.max_waiting,
    max_ack_pending: consumer.config?.max_ack_pending,
    headers_only: consumer.config?.headers_only,
    max_batch: consumer.config?.max_batch,
    max_expires: consumer.config?.max_expires,
    inactive_threshold: consumer.config?.inactive_threshold,
    num_replicas: consumer.config?.num_replicas,
    memory_storage: consumer.config?.mem_storage,
    flow_control: consumer.config?.flow_control,
    idle_heartbeat: consumer.config?.idle_heartbeat,
    metadata: consumer.config?.metadata,
  }
}

type ConsumerFilterUpdate = Pick<ConsumerUpdateRequest, 'filter_subject' | 'filter_subjects'>

function diffConsumerFilters(
  original: ConsumerCreateRequest,
  next: ConsumerCreateRequest,
): ConsumerFilterUpdate {
  const nextSubjects = (next.filter_subjects ?? []).filter((s) => s !== '')
  const originalSubjects = (original.filter_subjects ?? []).filter((s) => s !== '')
  const nextSubject = next.filter_subject ?? ''
  const originalSubject = original.filter_subject ?? ''

  if (nextSubjects.length > 0) {
    const unchanged =
      nextSubjects.length === originalSubjects.length &&
      nextSubjects.every((s, i) => s === originalSubjects[i])
    return unchanged ? {} : { filter_subjects: nextSubjects }
  }
  if (originalSubjects.length > 0 || nextSubject !== originalSubject) {
    return { filter_subject: nextSubject }
  }
  return {}
}

export function toConsumerUpdateRequest(
  original: ConsumerCreateRequest,
  next: ConsumerCreateRequest,
): ConsumerUpdateRequest {
  return {
    description: next.description,
    ack_wait: next.ack_wait,
    max_deliver: next.max_deliver,
    max_ack_pending: next.max_ack_pending,
    max_waiting: next.max_waiting,
    rate_limit_bps: next.rate_limit_bps,
    sample_freq: next.sample_freq,
    inactive_threshold: next.inactive_threshold,
    backoff: next.backoff,
    max_batch: next.max_batch,
    max_bytes: next.max_bytes,
    max_expires: next.max_expires,
    metadata: next.metadata,
    ...diffConsumerFilters(original, next),
  }
}

export function getFilterSubjectsArray(consumer: ConsumerInfo): string[] {
  if (consumer.config?.filter_subjects && consumer.config.filter_subjects.length > 0) {
    return consumer.config.filter_subjects
  }
  if (consumer.config?.filter_subject) {
    return [consumer.config.filter_subject]
  }
  return []
}
