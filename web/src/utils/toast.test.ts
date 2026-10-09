import { afterEach, describe, expect, it, vi } from 'vitest'
import { toast as sonnerToast } from 'sonner'
import { toast } from './toast'

vi.mock('sonner', () => {
  let next = 0
  const show = vi.fn(() => ++next)
  return {
    toast: { success: show, error: show, warning: show, info: show, loading: show, dismiss: vi.fn() },
  }
})

describe('toast.dismissShownBefore', () => {
  afterEach(() => {
    vi.useRealTimers()
  })

  it('keeps the toasts shown after the cutoff', () => {
    vi.useFakeTimers()
    vi.setSystemTime(1_000)
    const old = toast.error('old')
    vi.setSystemTime(5_000)
    toast.success('fresh')

    toast.dismissShownBefore(4_000)

    expect(vi.mocked(sonnerToast.dismiss).mock.calls).toEqual([[old]])
  })
})
