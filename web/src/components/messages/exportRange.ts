import type { Message } from '@/types/nats'

export interface RangePage {
  messages: Message[]
  has_more: boolean
  next_seq: number
}

export interface CollectRangeResult {
  messages: Message[]
  /** True when the stream was paged to its end (has_more became false). */
  reachedEnd: boolean
  /** True when the caller's AbortSignal fired mid-collection. */
  aborted: boolean
  /** True when collection stopped at the limit with more still available. */
  truncated: boolean
}

export interface CollectRangeOptions {
  startSeq: number
  /** Hard ceiling on the number of messages collected. */
  limit: number
  signal?: AbortSignal
  /** Called after each page with the running total collected so far. */
  onProgress?: (count: number) => void
}

/**
 * Page forward from startSeq until end/limit/abort. Pure over injected
 * `fetchPage` for testability.
 * `truncated`: stopped at the limit with more remaining — distinct from clean
 * `reachedEnd`/`aborted`.
 */
export async function collectRange(
  fetchPage: (startSeq: number, signal?: AbortSignal) => Promise<RangePage>,
  opts: CollectRangeOptions,
): Promise<CollectRangeResult> {
  const { startSeq, limit, signal, onProgress } = opts
  const out: Message[] = []
  let seq = startSeq
  let reachedEnd = false
  let aborted = false

  for (;;) {
    if (signal?.aborted) {
      aborted = true
      break
    }
    const page = await fetchPage(seq, signal)
    out.push(...page.messages)
    // Clamp reported count to limit so progress never overshoots.
    onProgress?.(Math.min(out.length, limit))

    if (!page.has_more || page.next_seq <= 0) {
      reachedEnd = true
      break
    }
    if (out.length >= limit) {
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

  return {
    messages: out.slice(0, limit),
    reachedEnd,
    aborted,
    truncated: out.length > limit || (!reachedEnd && !aborted),
  }
}
