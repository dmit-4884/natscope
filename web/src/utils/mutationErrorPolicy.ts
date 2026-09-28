import type { Mutation } from '@tanstack/react-query'

export function shouldToastMutationError(mutation: Mutation<unknown, unknown, unknown, unknown>): boolean {
  return !mutation.options.onError && !mutation.meta?.silent
}
