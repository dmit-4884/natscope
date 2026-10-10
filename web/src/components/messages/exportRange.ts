import type { Message } from '@/types/nats'

export interface RangePage {
  messages: Message[]
  has_more: boolean
  next_seq: number
}

export interface WalkRangeResult {
  /** Number of messages handed to `onPage`. */
  count: number
  /** True when the stream was paged to its end (has_more became false). */
  reachedEnd: boolean
  /** True when the caller's AbortSignal fired, between pages or inside a page fetch. */
  aborted: boolean
  /** True when the walk stopped at the limit with more still available. */
  truncated: boolean
}

export interface WalkRangeOptions {
  startSeq: number
  /** Hard ceiling on the number of messages handed over. */
  limit: number
  signal?: AbortSignal
  /** Receives each page, already clipped to the limit, so callers can serialize it and let go of it. */
  onPage: (messages: Message[]) => void
  /** Called after each page with the running total handed over so far. */
  onProgress?: (count: number) => void
}

/**
 * Page forward from startSeq until end/limit/abort, handing every page over as
 * it arrives. Pure over injected `fetchPage` for testability.
 * `truncated`: stopped at the limit with more remaining — distinct from clean
 * `reachedEnd`/`aborted`. An abort that rejects an in-flight fetch ends the
 * walk as `aborted`, with the earlier pages already delivered.
 */
export async function walkRange(
  fetchPage: (startSeq: number, signal?: AbortSignal) => Promise<RangePage>,
  opts: WalkRangeOptions,
): Promise<WalkRangeResult> {
  const { startSeq, limit, signal, onPage, onProgress } = opts
  let count = 0
  let seq = startSeq
  let reachedEnd = false
  let aborted = false
  let overshoot = false

  for (;;) {
    if (signal?.aborted) {
      aborted = true
      break
    }
    let page: RangePage
    try {
      page = await fetchPage(seq, signal)
    } catch (error) {
      if (signal?.aborted) {
        aborted = true
        break
      }
      throw error
    }
    const room = limit - count
    const taken = page.messages.length > room ? page.messages.slice(0, room) : page.messages
    if (taken.length < page.messages.length) overshoot = true
    if (taken.length > 0) onPage(taken)
    count += taken.length
    onProgress?.(count)

    if (!page.has_more || page.next_seq <= 0) {
      reachedEnd = true
      break
    }
    if (count >= limit) {
      break
    }
    // m19: guard against infinite loop when has_more=true but no forward
    // progress (cursor stalls / empty page).
    if (page.next_seq <= seq || page.messages.length === 0) {
      reachedEnd = true
      break
    }
    seq = page.next_seq
  }

  return { count, reachedEnd, aborted, truncated: overshoot || (!reachedEnd && !aborted) }
}
