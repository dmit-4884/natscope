import { describe, it, expect, beforeEach, vi } from 'vitest'
import type { MessageInitShape } from '@bufbuild/protobuf'
import type { ConsumerInfo as ProtoConsumerInfo } from '@/gen/types/nats/nats_stream_pb'
import type {
  CreateConsumerRequestSchema,
  UpdateConsumerRequestSchema,
} from '@/gen/services/grpc/nats/v1/management/nats_management_service_pb'
import { createConsumer, updateConsumer } from './management'

const mocks = vi.hoisted(() => ({
  createConsumer: vi.fn(),
  updateConsumer: vi.fn(),
}))

vi.mock('./grpc/clients', () => ({ managementClient: mocks }))

type CreateInit = MessageInitShape<typeof CreateConsumerRequestSchema>
type UpdateInit = MessageInitShape<typeof UpdateConsumerRequestSchema>

const consumerResponse = {
  consumer: { name: 'orders-worker', stream: 'ORDERS' } as unknown as ProtoConsumerInfo,
}

beforeEach(() => {
  mocks.createConsumer.mockReset().mockResolvedValue(consumerResponse)
  mocks.updateConsumer.mockReset().mockResolvedValue(consumerResponse)
})

describe('createConsumer', () => {
  it('sends the ephemeral flag when the caller asks for a non-durable consumer', async () => {
    await createConsumer('conn-1', 'ORDERS', { name: 'orders-worker', ephemeral: true })
    expect((mocks.createConsumer.mock.calls[0][0] as CreateInit).ephemeral).toBe(true)
  })

  it('defaults ephemeral to false so consumers stay durable', async () => {
    await createConsumer('conn-1', 'ORDERS', { name: 'orders-worker' })
    expect((mocks.createConsumer.mock.calls[0][0] as CreateInit).ephemeral).toBe(false)
  })

  it('still sends the filters supplied at creation time', async () => {
    await createConsumer('conn-1', 'ORDERS', {
      name: 'orders-worker',
      filter_subjects: ['orders.created', 'orders.paid'],
    })
    expect((mocks.createConsumer.mock.calls[0][0] as CreateInit).filterSubjects)
      .toEqual(['orders.created', 'orders.paid'])
  })
})

describe('updateConsumer', () => {
  it('omits filterSubject when the caller does not provide one', async () => {
    await updateConsumer('conn-1', 'ORDERS', 'orders-worker', { description: 'same filters' })
    const request = mocks.updateConsumer.mock.calls[0][0] as UpdateInit
    expect(request.filterSubject).toBeUndefined()
    expect(request.filterSubjects).toEqual([])
  })

  it('sends a provided filterSubject so the server replaces the filter', async () => {
    await updateConsumer('conn-1', 'ORDERS', 'orders-worker', { filter_subject: 'orders.paid' })
    expect((mocks.updateConsumer.mock.calls[0][0] as UpdateInit).filterSubject).toBe('orders.paid')
  })

  it('sends an empty filterSubject so the server clears the filter', async () => {
    await updateConsumer('conn-1', 'ORDERS', 'orders-worker', { filter_subject: '' })
    expect((mocks.updateConsumer.mock.calls[0][0] as UpdateInit).filterSubject).toBe('')
  })

  it('sends filterSubjects so the server replaces the whole list', async () => {
    await updateConsumer('conn-1', 'ORDERS', 'orders-worker', {
      filter_subjects: ['orders.created', 'orders.paid'],
    })
    const request = mocks.updateConsumer.mock.calls[0][0] as UpdateInit
    expect(request.filterSubjects).toEqual(['orders.created', 'orders.paid'])
    expect(request.filterSubject).toBeUndefined()
  })

  it('treats an empty filterSubjects list as "keep the existing filters"', async () => {
    await updateConsumer('conn-1', 'ORDERS', 'orders-worker', { filter_subjects: [] })
    expect((mocks.updateConsumer.mock.calls[0][0] as UpdateInit).filterSubjects).toEqual([])
  })
})
