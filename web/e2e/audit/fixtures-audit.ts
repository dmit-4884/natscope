/**
 * Shared fixture surface for the audit suite.
 *
 * Re-exports the connected-app `test`/`expect` from ../fixtures (which seeds
 * localStorage so the SPA boots already-connected to the `local` connection
 * and exposes `env.connectionId`), and adds namespacing helpers.
 *
 * NAMESPACING (workers:1 — every spec shares one backend + one NATS):
 *   - subjects: `audit.<slice>.*`
 *   - streams:  `AUDIT_<SLICE>`
 *   - KV/object buckets: `audit-<slice>`
 *   - templates/mappings/sources: prefix `AUDIT_<SLICE>` / `audit.<slice>.`
 * Each spec creates its own state and cleans it up in afterAll via the API.
 * Never touch the legacy E2E_PUBLISH fixtures except through ../fixtures.
 */
export { test, expect, publishUrl } from '../fixtures'

/** Build the canonical namespaced identifiers for a slice. */
export function ns(slice: string) {
  const S = slice.toUpperCase().replace(/[^A-Z0-9]/g, '_')
  const s = slice.toLowerCase().replace(/[^a-z0-9]/g, '-')
  return {
    stream: `AUDIT_${S}`,
    stream2: `AUDIT_${S}_2`,
    subjectRoot: `audit.${s}`,
    subject: (leaf: string) => `audit.${s}.${leaf}`,
    wildcard: `audit.${s}.>`,
    kvBucket: `audit-${s}`,
    objBucket: `audit-${s}-obj`,
    tplPrefix: `AUDIT_${S}_`,
    mappingPrefix: `audit.${s}.`,
    sourcePrefix: `AUDIT_${S}_`,
  }
}

/** A unique-ish suffix without Math.random/Date coupling in assertions. */
let _seq = 0
export const uniq = (base = 'x') => `${base}-${Date.now()}-${_seq++}`
