import type { StreamCreateRequest } from '@/types/management'
import type { EditorMode } from './consumerEditorStore'
import { createStreamScopedStore, type StreamScope } from './index'

/**
 * Persisted Stream Config-tab state (survives navigation). originalConfig is
 * excluded — derived live each render to avoid diffing against stale state.
 */
interface ConfigEditorEntry {
  isEditing: boolean
  editorMode: EditorMode
  formDraft: StreamCreateRequest | null
}

const DEFAULTS: ConfigEditorEntry = {
  isEditing: false,
  editorMode: 'form',
  formDraft: null,
}

const configEditorStore = createStreamScopedStore<ConfigEditorEntry>({
  name: 'config-editor',
  defaults: DEFAULTS,
  version: 1,
})

export function useConfigEditorEntry(scope: StreamScope) {
  return configEditorStore.useEntry(scope)
}

/** Drop every entry (e.g. on logout). */
export function clearAllConfigEditor(): void {
  configEditorStore.clearAll()
}
