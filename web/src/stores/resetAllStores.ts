// Reset session-scoped stores on logout. Preferences (density, panel sizes)
// are kept — not session data.
import { useLiveStatsStore, clearLiveMessagesStorage } from '@/contexts/live'
import { useManagementStore } from './managementStore'
import { clearAllMessagesView } from './streamTabState/messagesViewStore'
import { clearAllPublishDrafts } from './streamTabState/publishDraftStore'
import { clearAllConsumerEditor } from './streamTabState/consumerEditorStore'
import { clearAllConfigEditor } from './streamTabState/configEditorStore'

/** Reset all stores on logout. Templates live on the backend, not wiped. */
export function resetAllStores(): void {
  useManagementStore.getState().reset()
  useLiveStatsStore.getState().setStats(null)
  // Drop per-stream scoped state (message selections, publish/consumer/config drafts)
  clearAllMessagesView()
  clearAllPublishDrafts()
  clearAllConsumerEditor()
  clearAllConfigEditor()
  // Prune live message localStorage keys
  clearLiveMessagesStorage()
}
