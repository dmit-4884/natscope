import type { ConsumerInfo } from '@/types/nats'
import type { ConsumerCreateRequest } from '@/types/management'

export const defaultConsumerConfig: ConsumerCreateRequest = {
  name: '',
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
    description: consumer.config?.description || '',
    deliver_policy: (consumer.config?.deliver_policy?.toLowerCase() || 'all') as ConsumerCreateRequest['deliver_policy'],
    opt_start_seq: consumer.config?.opt_start_seq,
    opt_start_time: consumer.config?.opt_start_time,
    ack_policy: (consumer.config?.ack_policy?.toLowerCase() || 'explicit') as 'none' | 'all' | 'explicit',
    ack_wait: consumer.config?.ack_wait,
    max_deliver: consumer.config?.max_deliver,
    backoff: consumer.config?.backoff,
    filter_subject: consumer.config?.filter_subject,
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

export function getFilterSubjectsArray(consumer: ConsumerInfo): string[] {
  if (consumer.config?.filter_subjects && consumer.config.filter_subjects.length > 0) {
    return consumer.config.filter_subjects
  }
  if (consumer.config?.filter_subject) {
    return [consumer.config.filter_subject]
  }
  return []
}
