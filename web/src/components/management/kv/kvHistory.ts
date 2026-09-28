export const KV_HISTORY_MIN = 1
export const KV_HISTORY_MAX = 64

export function isKVHistoryValid(history: number | undefined): boolean {
  if (history === undefined) return true
  return history >= KV_HISTORY_MIN && history <= KV_HISTORY_MAX
}
