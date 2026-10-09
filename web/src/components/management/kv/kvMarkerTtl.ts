const KV_MARKER_TTL_MIN_NS = 1_000_000_000

export function kvMarkerTtlError(markerTtl: number | undefined, locked: boolean): string | undefined {
  if (!markerTtl) return locked ? "Per-key TTL can't be turned off once the bucket allows it" : undefined
  if (markerTtl < KV_MARKER_TTL_MIN_NS) return `At least ${KV_MARKER_TTL_MIN_NS} (1s), or 0 for off`
  return undefined
}
