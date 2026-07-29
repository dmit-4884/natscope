/** Visual arrow direction in the list: 'up' = previous row, 'down' = next row. */
export type VisualDirection = 'up' | 'down'
export type ListDirection = 'backward' | 'forward'

export interface NavTarget {
  startSeq: number
  apiDirection: ListDirection
}

/**
 * Maps an arrow press to a listMessages(limit=1) request. Visual order
 * follows the list's direction: a backward list (newest first) walks to
 * older messages on "down"; a forward list (jump-to-time) inverts this.
 * Returns 'edge' when the step would need start_seq <= 0 — that must never
 * be sent, since an unset start_seq means "newest" server-side.
 */
export function resolveNavTarget(
  visual: VisualDirection,
  listDirection: ListDirection,
  sequence: number,
): NavTarget | 'edge' {
  const towardOlder = (listDirection === 'backward') === (visual === 'down')
  if (!towardOlder) {
    return { startSeq: sequence + 1, apiDirection: 'forward' }
  }
  if (sequence <= 1) return 'edge'
  return { startSeq: sequence - 1, apiDirection: 'backward' }
}
