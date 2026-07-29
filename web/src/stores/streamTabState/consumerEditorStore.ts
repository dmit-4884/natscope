import type { ConsumerCreateRequest } from '@/types/management'
import { createStreamScopedStore, type StreamScope } from './index'

export type EditorMode = 'form' | 'json'

/**
 * Persisted Consumers-tab state for one stream (selection, search, form draft).
 * Excluded on purpose: selectedConsumer (derive from live list), originalConfig
 * (recomputed at edit-start), modal flags (ephemeral, shouldn't survive nav).
 */
interface ConsumerEditorEntry {
  selectedName: string | null
  searchQuery: string
  isCreating: boolean
  isEditing: boolean
  editorMode: EditorMode
  /** Draft body — only present when creating or editing. */
  formDraft: ConsumerCreateRequest | null
}

const DEFAULTS: ConsumerEditorEntry = {
  selectedName: null,
  searchQuery: '',
  isCreating: false,
  isEditing: false,
  editorMode: 'form',
  formDraft: null,
}

const consumerEditorStore = createStreamScopedStore<ConsumerEditorEntry>({
  name: 'consumer-editor',
  defaults: DEFAULTS,
  version: 1,
})

/** React hook returning `[entry, partialPatcher]` for the given scope. */
export function useConsumerEditorEntry(scope: StreamScope) {
  return consumerEditorStore.useEntry(scope)
}

/** Drop every entry (e.g. on logout). */
export function clearAllConsumerEditor(): void {
  consumerEditorStore.clearAll()
}
