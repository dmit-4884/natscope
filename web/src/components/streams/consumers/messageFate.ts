import { matchSubject } from '@/shared/domain/subjectMatch'
import type { ConsumerInfo } from '@/types/nats'
import { getFilterSubjectsArray } from './consumerUtils'

export type FateState =
  | 'done'
  | 'done_or_skipped'
  | 'skipped'
  | 'delivered'
  | 'delivered_or_skipped'
  | 'awaiting_ack'
  | 'awaiting_or_skipped'
  | 'not_delivered'
  | 'not_delivered_or_skipped'

export interface MessageFate {
  consumer: ConsumerInfo
  state: FateState
  label: string
  detail: string
}

interface FateMessage {
  sequence: number
  subject: string
  timestamp: number
}

function takesSubject(consumer: ConsumerInfo, subject: string): boolean {
  const filters = getFilterSubjectsArray(consumer)
  return filters.length === 0 || filters.some((filter) => matchSubject(subject, filter))
}

function startedAfter(consumer: ConsumerInfo, message: FateMessage): 'yes' | 'maybe' | 'no' {
  const config = consumer.config
  switch (config?.deliver_policy) {
    case 'by_start_sequence':
      return (config.opt_start_seq ?? 0) > message.sequence ? 'yes' : 'no'
    case 'by_start_time': {
      const start = config.opt_start_time ? Date.parse(config.opt_start_time) : NaN
      return !Number.isNaN(start) && start > message.timestamp ? 'yes' : 'no'
    }
    case 'new':
      return consumer.created != null && consumer.created > message.timestamp ? 'yes' : 'no'
    case 'last':
    case 'last_per_subject':
      return consumer.created != null && consumer.created > message.timestamp ? 'maybe' : 'no'
    default:
      return 'no'
  }
}

function doneDetail(consumer: ConsumerInfo): string {
  const maxDeliver = consumer.config?.max_deliver ?? 0
  return maxDeliver > 0
    ? `Acknowledged, terminated by a client, or dropped after ${maxDeliver} delivery attempts.`
    : 'Acknowledged or terminated by a client.'
}

function fateOf(consumer: ConsumerInfo, message: FateMessage): Omit<MessageFate, 'consumer'> {
  const delivered = consumer.delivered?.stream_seq ?? 0
  const ackFloor = consumer.ack_floor?.stream_seq ?? 0
  const ackPolicy = consumer.config?.ack_policy ?? 'explicit'
  const before = startedAfter(consumer, message)

  if (before === 'yes') {
    return { state: 'skipped', label: 'Skipped', detail: 'The consumer starts after this message, so it never gets it.' }
  }
  if (message.sequence > delivered) {
    const inLine = consumer.paused ? 'Waiting in line; the consumer is paused.' : 'Waiting in line for this consumer.'
    return before === 'maybe'
      ? {
          state: 'not_delivered_or_skipped',
          label: 'Not delivered yet or skipped',
          detail: `${inLine} Or skipped: the consumer started from the last message.`,
        }
      : { state: 'not_delivered', label: 'Not delivered yet', detail: inLine }
  }
  if (ackPolicy === 'none') {
    return before === 'maybe'
      ? {
          state: 'delivered_or_skipped',
          label: 'Delivered or skipped',
          detail: 'Delivered, or skipped: the consumer started from the last message. This consumer does not use acks.',
        }
      : { state: 'delivered', label: 'Delivered', detail: 'Delivered. This consumer does not use acks.' }
  }
  if (message.sequence <= ackFloor || consumer.num_ack_pending === 0) {
    return before === 'maybe'
      ? {
          state: 'done_or_skipped',
          label: 'Done or skipped',
          detail: `${doneDetail(consumer)} Or skipped: the consumer started from the last message.`,
        }
      : { state: 'done', label: 'Done', detail: doneDetail(consumer) }
  }
  if (before === 'maybe') {
    return {
      state: 'awaiting_or_skipped',
      label: 'Waiting for ack or skipped',
      detail: 'Delivered and not acknowledged yet, or skipped: the consumer started from the last message.',
    }
  }
  return {
    state: 'awaiting_ack',
    label: 'Waiting for ack',
    detail:
      ackPolicy === 'all'
        ? 'Delivered, not acknowledged yet.'
        : 'Delivered, not acknowledged yet, unless a client acked it out of order.',
  }
}

export function messageFates(consumers: ConsumerInfo[], message: FateMessage): { fates: MessageFate[]; unrelated: number } {
  const fates: MessageFate[] = []
  let unrelated = 0
  for (const consumer of consumers) {
    if (!takesSubject(consumer, message.subject)) {
      unrelated++
      continue
    }
    fates.push({ consumer, ...fateOf(consumer, message) })
  }
  return { fates, unrelated }
}
