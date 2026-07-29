import { safeGetItem, safeSetItem, safeRemoveItem } from '@/utils/safeStorage'
import { createStreamScopedStore, type StreamScope } from './index'

const NEW_PERSIST_KEY = 'natscope:stream-scope:publish-draft'

export interface HeaderDraft {
  key: string
  value: string
}

/** Per-pattern draft (subject pattern is the inner key). */
export interface PatternDraft {
  wildcards: string[]
  messageJson: string
  /** Optional, omitted entirely when no headers were ever added. */
  headers?: HeaderDraft[]
}

/**
 * Per-stream publish state: lastPattern (selected subject pattern) + per-pattern
 * drafts. Split so switching patterns doesn't wipe another pattern's draft.
 */
export interface PublishDraftEntry {
  lastPattern: string
  drafts: Record<string, PatternDraft>
}

const DEFAULTS: PublishDraftEntry = { lastPattern: '', drafts: {} }
const EMPTY_DRAFT: PatternDraft = { wildcards: [], messageJson: '{}' }

const LEGACY_STATES_KEY = 'nats_publish_states'
const LEGACY_LAST_PATTERN_KEY = 'nats_publish_last_pattern'

/**
 * One-shot adoption of two legacy localStorage formats:
 *  - nats_publish_last_pattern: { "<url>:<stream>" -> pattern }; lifts directly
 *    into a PublishDraftEntry with empty drafts.
 *  - nats_publish_states: { entries: { "<url>:<pattern>" -> body } }; keyed by
 *    pattern not stream, so each draft attaches to every scope on that
 *    connection (collisions vanishingly rare; last-saved wins).
 * Runs synchronously at module load before Zustand reads its persist key.
 * Idempotent: skips if the new key exists.
 */
function migrateLegacyPublishStateOnce(): void {
  if (typeof localStorage === 'undefined') return
  if (safeGetItem(NEW_PERSIST_KEY)) return // already migrated/initialized

  const out: Record<string, PublishDraftEntry> = {}

  // Last-pattern map is the migration spine — the only legacy entry keyed
  // by (connectionUrl, streamName).
  const connectionUrlByScope = new Map<string, string>()
  try {
    const raw = safeGetItem(LEGACY_LAST_PATTERN_KEY)
    if (raw) {
      const parsed = JSON.parse(raw) as Record<string, string>
      if (parsed && typeof parsed === 'object') {
        for (const [scopeKey, lastPattern] of Object.entries(parsed)) {
          if (typeof lastPattern !== 'string') continue
          out[scopeKey] = { lastPattern, drafts: {} }
          // Capture connectionUrl prefix to cross-reference draft bodies.
          const lastColon = scopeKey.lastIndexOf(':')
          if (lastColon > 0) {
            connectionUrlByScope.set(scopeKey, scopeKey.slice(0, lastColon))
          }
        }
      }
    }
  } catch {
    /* ignore */
  }

  // Pattern-keyed drafts: distribute by matching connectionUrl prefix.
  try {
    const raw = safeGetItem(LEGACY_STATES_KEY)
    if (raw) {
      const parsed = JSON.parse(raw)
      const draftEntries: Record<string, { wildcards?: string[]; messageJson?: string }> | undefined =
        parsed?.entries ?? (parsed && typeof parsed === 'object' && !('timestamps' in parsed) ? parsed : undefined)
      if (draftEntries && typeof draftEntries === 'object') {
        for (const [combined, body] of Object.entries(draftEntries)) {
          if (!body || typeof body !== 'object') continue
          const lastColon = combined.lastIndexOf(':')
          if (lastColon < 0) continue
          const connUrl = combined.slice(0, lastColon)
          const pattern = combined.slice(lastColon + 1)
          if (!pattern) continue
          // Stash the draft on every scope (stream) for this connection.
          for (const [scopeKey, knownConn] of connectionUrlByScope) {
            if (knownConn !== connUrl) continue
            const target = out[scopeKey] ?? { lastPattern: '', drafts: {} }
            target.drafts[pattern] = {
              wildcards: Array.isArray(body.wildcards) ? body.wildcards.map(String) : [],
              messageJson: typeof body.messageJson === 'string' ? body.messageJson : '{}',
            }
            out[scopeKey] = target
          }
        }
      }
    }
  } catch {
    /* ignore */
  }

  // Drop legacy keys so we don't re-run on every reload.
  try {
    safeRemoveItem(LEGACY_STATES_KEY)
    safeRemoveItem(LEGACY_LAST_PATTERN_KEY)
  } catch {
    /* ignore */
  }

  if (Object.keys(out).length === 0) return

  // Seed the new key in Zustand persist's on-disk shape so the store reads it:
  // { state: { entries: { key -> { data, updatedAt } } }, version }.
  const now = Date.now()
  const entries: Record<string, { data: PublishDraftEntry; updatedAt: number }> = {}
  for (const [k, v] of Object.entries(out)) {
    entries[k] = { data: v, updatedAt: now }
  }
  try {
    safeSetItem(NEW_PERSIST_KEY, JSON.stringify({ state: { entries }, version: 1 }))
  } catch {
    /* ignore */
  }
}

migrateLegacyPublishStateOnce()

/** Backing store. Use the helpers below for ergonomic access. */
const publishDraftStore = createStreamScopedStore<PublishDraftEntry>({
  name: 'publish-draft',
  defaults: DEFAULTS,
  version: 1,
})

// ---- Convenience API ---------------------------------------------------

/** React hook returning the current entry + a partial patcher. */
export function usePublishDraftEntry(scope: StreamScope) {
  return publishDraftStore.useEntry(scope)
}

/** Imperative read. */
export function getPublishDraftEntry(scope: StreamScope): PublishDraftEntry {
  return publishDraftStore.get(scope)
}

/** Pull a single pattern's draft out of the entry, or defaults. */
export function getPatternDraft(entry: PublishDraftEntry, pattern: string): PatternDraft {
  return entry.drafts[pattern] ?? EMPTY_DRAFT
}

/** Update which pattern is "selected" for this stream. */
export function setLastPattern(scope: StreamScope, pattern: string): void {
  publishDraftStore.set(scope, { lastPattern: pattern })
}

/** Update one pattern's draft. Always rewrites the full drafts map. */
export function setPatternDraft(scope: StreamScope, pattern: string, patch: Partial<PatternDraft>): void {
  const current = publishDraftStore.get(scope)
  const prev = current.drafts[pattern] ?? EMPTY_DRAFT
  const next = { ...prev, ...patch }
  publishDraftStore.set(scope, {
    drafts: { ...current.drafts, [pattern]: next },
  })
}

/** Remove a single pattern's draft (used when subject is no longer valid). */
export function clearPatternDraft(scope: StreamScope, pattern: string): void {
  const current = publishDraftStore.get(scope)
  if (!(pattern in current.drafts)) return
  const nextDrafts = { ...current.drafts }
  delete nextDrafts[pattern]
  publishDraftStore.set(scope, { drafts: nextDrafts })
}

/** Drop every entry (e.g. on logout). */
export function clearAllPublishDrafts(): void {
  publishDraftStore.clearAll()
}
