import { useState, useEffect, useMemo } from 'react'
import { useOutletContext, useNavigate } from 'react-router-dom'
import { getErrorMessage } from '@/api/errors'
import { Alert, Spinner } from '@/components/ui'
import { useStreamDetail, useUpdateStream, useDeleteStream, usePurgeStream, useSealStream } from '@/contexts/streams'
import { useConfigEditorEntry } from '@/stores/streamTabState/configEditorStore'
import type { StreamPurgeRequest } from '@/types/management'
import { stableJson } from '@/utils/stableJson'
import ConfigDiffModal from '../common/ConfigDiffModal'
import type { StreamViewOutletContext } from './StreamView'
import StreamNotFoundState from './StreamNotFoundState'
import { isStreamNotFound } from './streamErrors'
import { StreamConfigEditor } from './config/StreamConfigEditor'
import { StreamConfigView } from './config/StreamConfigView'
import { StreamConfirmDialog, type StreamConfirmType } from './config/StreamConfirmDialog'
import { streamToConfig } from './config/streamConfigUtils'
import { buildStreamUpdatePayload, getIgnoredUpdateKeys } from './config/streamFieldDefinitions'

export default function StreamConfigTab() {
  const { scope, connectionId, streamName } = useOutletContext<StreamViewOutletContext>()
  const navigate = useNavigate()

  // Long-lived per-stream UI state (survives navigation).
  const [editorState, setEditorState] = useConfigEditorEntry(scope)
  const { isEditing, editorMode, formDraft } = editorState

  // Ephemeral modals stay local.
  const [showDiffModal, setShowDiffModal] = useState(false)
  const [confirmAction, setConfirmAction] = useState<StreamConfirmType | null>(null)

  const { data: streamDetail, isLoading, error, refetch, isFetching } = useStreamDetail(streamName, connectionId)

  const updateStream = useUpdateStream(connectionId)
  const deleteStream = useDeleteStream(connectionId)
  const purgeStream = usePurgeStream(connectionId)
  const sealStream = useSealStream(connectionId)

  // Original config derived from live server data, recomputed each render to
  // avoid drift if backend changes mid-edit.
  const originalConfig = useMemo(() => {
    if (!streamDetail || !isEditing) return null
    return streamToConfig(streamDetail)
  }, [streamDetail, isEditing])

  // Seed the form draft on entering edit mode if no draft exists yet.
  useEffect(() => {
    if (isEditing && streamDetail && !formDraft) {
      setEditorState({ formDraft: streamToConfig(streamDetail) })
    }
  }, [isEditing, streamDetail, formDraft, setEditorState])

  const formValue = formDraft

  const handleCancelEdit = () => {
    setEditorState({ isEditing: false, editorMode: 'form', formDraft: null })
    setShowDiffModal(false)
  }

  const pendingUpdate = useMemo(() => {
    if (!formValue || !originalConfig) return null
    const original = buildStreamUpdatePayload(originalConfig)
    const next = buildStreamUpdatePayload(formValue)
    return {
      original,
      next,
      ignoredKeys: getIgnoredUpdateKeys(originalConfig, formValue),
      hasEffectiveChanges: stableJson(original) !== stableJson(next),
    }
  }, [formValue, originalConfig])

  const hasChanges = !!pendingUpdate && (pendingUpdate.hasEffectiveChanges || pendingUpdate.ignoredKeys.length > 0)

  const handleUpdate = async () => {
    if (!pendingUpdate) return
    await updateStream.mutateAsync({
      name: streamName,
      config: pendingUpdate.next,
    })
    setShowDiffModal(false)
    handleCancelEdit()
    refetch()
  }

  const handleConfirmAction = async (purgeOptions?: StreamPurgeRequest) => {
    if (!confirmAction) return
    if (confirmAction === 'delete') {
      await deleteStream.mutateAsync(streamName)
      navigate(`/streams`)
    } else if (confirmAction === 'purge') {
      await purgeStream.mutateAsync({ name: streamName, options: purgeOptions })
    } else if (confirmAction === 'seal') {
      await sealStream.mutateAsync(streamName)
    }
    setConfirmAction(null)
    refetch()
  }

  if (isStreamNotFound(error)) {
    return (
      <div className="flex-1 flex items-center justify-center">
        <StreamNotFoundState streamName={streamName} />
      </div>
    )
  }
  if (isLoading) {
    return (
      <div className="flex-1 flex items-center justify-center">
        <Spinner size="lg" />
      </div>
    )
  }
  if (error) {
    return (
      <div className="flex-1 flex items-center justify-center">
        <Alert variant="error">Failed to load stream: {getErrorMessage(error)}</Alert>
      </div>
    )
  }
  if (!streamDetail) return null

  // JSON-edit view fills the viewport (flex lets children grow); form/read-only
  // use a scrollable overflow-auto container.
  const isJsonEdit = isEditing && editorMode === 'json'
  const innerClass = isJsonEdit
    ? 'flex-1 flex flex-col min-h-0 overflow-hidden'
    : 'flex-1 overflow-auto'

  return (
    <div className="flex-1 flex flex-col min-h-0 overflow-hidden">
      <div className={innerClass}>
        {isEditing && formValue ? (
          <StreamConfigEditor
            editorMode={editorMode}
            onModeChange={(m) => setEditorState({ editorMode: m })}
            value={formValue}
            onChange={(v) => setEditorState({ formDraft: v })}
            originalValue={originalConfig}
            isSaving={updateStream.isPending}
            hasChanges={hasChanges}
            onCancel={handleCancelEdit}
            onSave={() => setShowDiffModal(true)}
          />
        ) : (
          <StreamConfigView
            streamDetail={streamDetail}
            isFetching={isFetching}
            onRefetch={() => refetch()}
            onStartEdit={() => setEditorState({ isEditing: true })}
            onPurge={() => setConfirmAction('purge')}
            onSeal={() => setConfirmAction('seal')}
            onDelete={() => setConfirmAction('delete')}
          />
        )}
      </div>

      {confirmAction && (
        <StreamConfirmDialog
          type={confirmAction}
          streamName={streamName}
          onCancel={() => setConfirmAction(null)}
          onConfirm={handleConfirmAction}
        />
      )}

      {showDiffModal && pendingUpdate && (
        <ConfigDiffModal
          isOpen={showDiffModal}
          onClose={() => setShowDiffModal(false)}
          onConfirm={handleUpdate}
          title="Confirm Stream Configuration Changes"
          description={`Review the changes to the stream "${streamName}" before applying them.`}
          originalConfig={pendingUpdate.original}
          newConfig={pendingUpdate.next}
          notice={
            pendingUpdate.ignoredKeys.length > 0 ? (
              <Alert variant="warning" title="Some edits cannot be applied">
                <span className="font-mono">{pendingUpdate.ignoredKeys.join(', ')}</span> cannot be changed on an
                existing stream, so {pendingUpdate.ignoredKeys.length > 1 ? 'those edits are' : 'that edit is'} dropped
                on save. Recreate the stream to apply {pendingUpdate.ignoredKeys.length > 1 ? 'them' : 'it'}.
              </Alert>
            ) : undefined
          }
          isLoading={updateStream.isPending}
        />
      )}
    </div>
  )
}
