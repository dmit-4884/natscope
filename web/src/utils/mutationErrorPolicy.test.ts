import { describe, it, expect } from 'vitest'
import type { Mutation } from '@tanstack/react-query'
import { shouldToastMutationError } from './mutationErrorPolicy'

function fakeMutation(options: { onError?: () => void; silent?: boolean }): Mutation<unknown, unknown, unknown, unknown> {
  return {
    options: { onError: options.onError },
    meta: options.silent === undefined ? undefined : { silent: options.silent },
  } as unknown as Mutation<unknown, unknown, unknown, unknown>
}

describe('shouldToastMutationError', () => {
  it('toasts when the mutation has neither a hook-level onError nor a silent meta flag', () => {
    expect(shouldToastMutationError(fakeMutation({}))).toBe(true)
  })

  it('does not toast when the mutation defines its own onError', () => {
    expect(shouldToastMutationError(fakeMutation({ onError: () => {} }))).toBe(false)
  })

  it('does not toast when the mutation opts out via meta.silent', () => {
    expect(shouldToastMutationError(fakeMutation({ silent: true }))).toBe(false)
  })

  it('toasts when meta.silent is explicitly false', () => {
    expect(shouldToastMutationError(fakeMutation({ silent: false }))).toBe(true)
  })
})
