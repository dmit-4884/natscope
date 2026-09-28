import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { renderHook } from '@testing-library/react'
import { ConnectError, Code } from '@connectrpc/connect'
import { create, toBinary } from '@bufbuild/protobuf'
import { ErrorInfoSchema } from '@/gen/google/rpc/error_details_pb'
import { useConnectionValidation } from './useConnectionValidation'
import { useConnectionHealth } from './useConnectionHealth'

vi.mock('./useConnectionHealth', () => ({
  useConnectionHealth: vi.fn(),
}))

const useConnectionHealthMock = vi.mocked(useConnectionHealth)

function reasonError(code: Code, message: string, reason: string): ConnectError {
  const err = new ConnectError(message, code)
  err.details = [{ type: ErrorInfoSchema.typeName, value: toBinary(ErrorInfoSchema, create(ErrorInfoSchema, { reason })) }]
  return err
}

function mockHealth(queryError: Error | null) {
  useConnectionHealthMock.mockReturnValue({
    status: 'disconnected',
    rtt: undefined,
    error: undefined,
    queryError,
    serverVersion: undefined,
    refetch: vi.fn(),
  })
}

describe('useConnectionValidation', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-01-01T00:00:00Z'))
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('does not disconnect on a brief unavailable blip', () => {
    const onInvalid = vi.fn()
    mockHealth(new ConnectError('unavailable', Code.Unavailable))
    renderHook(() => useConnectionValidation('conn-1', onInvalid))

    expect(onInvalid).not.toHaveBeenCalled()
  })

  it('disconnects once an unavailable outage is sustained past the threshold', () => {
    const onInvalid = vi.fn()
    mockHealth(new ConnectError('unavailable', Code.Unavailable))
    const { rerender } = renderHook(() => useConnectionValidation('conn-1', onInvalid))
    expect(onInvalid).not.toHaveBeenCalled()

    vi.setSystemTime(new Date('2026-01-01T00:01:01Z'))
    mockHealth(new ConnectError('unavailable', Code.Unavailable))
    rerender()

    expect(onInvalid).toHaveBeenCalledTimes(1)
  })

  it('recovers the sustained-failure window once the connection is healthy again', () => {
    const onInvalid = vi.fn()
    mockHealth(new ConnectError('unavailable', Code.Unavailable))
    const { rerender } = renderHook(() => useConnectionValidation('conn-1', onInvalid))

    vi.setSystemTime(new Date('2026-01-01T00:00:30Z'))
    mockHealth(null)
    rerender()

    vi.setSystemTime(new Date('2026-01-01T00:01:05Z'))
    mockHealth(new ConnectError('unavailable', Code.Unavailable))
    rerender()

    expect(onInvalid).not.toHaveBeenCalled()
  })

  it('disconnects immediately when the connection is gone', () => {
    const onInvalid = vi.fn()
    mockHealth(reasonError(Code.NotFound, 'not found', 'CONNECTION_NOT_FOUND'))
    renderHook(() => useConnectionValidation('conn-1', onInvalid))

    expect(onInvalid).toHaveBeenCalledTimes(1)
  })
})
