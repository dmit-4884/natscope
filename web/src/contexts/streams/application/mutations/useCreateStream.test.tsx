import { beforeEach, describe, expect, it, vi } from 'vitest'
import { Code, ConnectError } from '@connectrpc/connect'
import { create, toBinary } from '@bufbuild/protobuf'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { renderHook, act } from '@testing-library/react'
import type { ReactNode } from 'react'
import { ErrorInfoSchema } from '@/gen/google/rpc/error_details_pb'
import { streamKeys } from '../queries/streamKeys'

const mocks = vi.hoisted(() => ({ createStream: vi.fn(), toastError: vi.fn() }))

vi.mock('@/api/management', () => ({ createStream: mocks.createStream }))
vi.mock('@/utils/toast', () => ({ toast: { error: mocks.toastError, success: vi.fn() } }))

import { useCreateStream } from './useStreamMutations'

function overlapError(): ConnectError {
  const err = new ConnectError('nats: subjects overlap with an existing stream', Code.InvalidArgument)
  err.details = [
    {
      type: ErrorInfoSchema.typeName,
      value: toBinary(ErrorInfoSchema, create(ErrorInfoSchema, { reason: 'NATS_API_ERROR', metadata: { err_code: '10065' } })),
    },
  ]
  return err
}

function setup(cachedStreams: unknown) {
  const client = new QueryClient()
  if (cachedStreams) client.setQueryData(streamKeys.list('c1'), cachedStreams)
  const wrapper = ({ children }: { children: ReactNode }) => <QueryClientProvider client={client}>{children}</QueryClientProvider>
  return renderHook(() => useCreateStream('c1'), { wrapper })
}

describe('useCreateStream overlap error', () => {
  beforeEach(() => {
    mocks.createStream.mockReset()
    mocks.toastError.mockReset()
  })

  it('names the streams whose subjects overlap when the stream list is loaded', async () => {
    mocks.createStream.mockRejectedValue(overlapError())
    const { result } = setup({ streams: [{ name: 'ORDERS', subjects: ['orders.>'] }, { name: 'OTHER', subjects: ['other.x'] }] })

    await act(async () => {
      await result.current.mutateAsync({ name: 'NEW', subjects: ['orders.new'] }).catch(() => {})
    })

    const message = mocks.toastError.mock.calls[0][0] as string
    expect(message).toContain('ORDERS (orders.>)')
    expect(message).not.toContain('OTHER')
    expect(message).not.toContain('nats:')
  })

  it('falls back to the plain message when no loaded stream explains it', async () => {
    mocks.createStream.mockRejectedValue(overlapError())
    const { result } = setup(undefined)

    await act(async () => {
      await result.current.mutateAsync({ name: 'NEW', subjects: ['orders.new'] }).catch(() => {})
    })

    expect(mocks.toastError.mock.calls[0][0]).toBe('Failed to create stream: Subjects overlap with an existing stream')
  })
})
