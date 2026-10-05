import { useState, useEffect, useMemo } from 'react'
import { useNavigate, useOutletContext, useSearchParams } from 'react-router-dom'
import type { ConsumerInfo, Message } from '@/types/nats'
import {
  useConsumers,
  useCreateConsumer,
  useUpdateConsumer,
  useDeleteConsumer,
  usePauseConsumer,
  useResumeConsumer,
  useResetConsumer,
  useUnpinConsumer,
  useStreamDetail,
} from '@/contexts/streams'
import { Button, PlusIcon, UsersIcon } from '@/components/ui'
import { useConnectionPolicy, useServerCapabilities } from '@/contexts/connection'
import { useConfirmation } from '@/contexts/settings'
import { useConsumerEditorEntry } from '@/stores/streamTabState/consumerEditorStore'
import { useMessagesViewEntry } from '@/stores/streamTabState/messagesViewStore'
import { toSelectedHistoryMessage } from '../messages/unified/selectedMessage'
import ConfigDiffModal from '../common/ConfigDiffModal'
import type { StreamViewOutletContext } from './StreamView'
import { ConsumerList } from './consumers/ConsumerList'
import { ConsumerEditor } from './consumers/ConsumerEditor'
import { ConsumerView } from './consumers/ConsumerView'
import { ConsumerConfirmDialog, type ConsumerConfirmAction } from './consumers/ConsumerConfirmDialog'
import { consumerToConfig, defaultConsumerConfig, parseResetSequence, toConsumerUpdateRequest } from './consumers/consumerUtils'
import { consumerIssues } from './consumers/consumerHealth'
import { ConsumerPosition } from './consumers/ConsumerPosition'

export default function StreamConsumersTab() {
  const { scope, connectionId, streamName } = useOutletContext<StreamViewOutletContext>()

  // Per-stream UI state — persisted across navigations via consumerEditorStore.
  const [editorState, setEditorState] = useConsumerEditorEntry(scope)
  const { selectedName, searchQuery, isCreating, isEditing, editorMode, formDraft } = editorState

  // Ephemeral modals stay local — they should never survive a route change.
  const [showDiffModal, setShowDiffModal] = useState(false)
  const [confirmAction, setConfirmAction] = useState<ConsumerConfirmAction | null>(null)

  const {
    data: consumers = [],
    isLoading,
    error: consumersError,
    refetch,
    isFetching,
    dataUpdatedAt,
  } = useConsumers(connectionId, streamName)
  const { data: streamDetail } = useStreamDetail(streamName, connectionId)
  const issues = useMemo(
    () => Object.fromEntries(consumers.map((c) => [c.name, consumerIssues(c, streamDetail, dataUpdatedAt)])),
    [consumers, streamDetail, dataUpdatedAt],
  )

  const navigate = useNavigate()
  const [searchParams, setSearchParams] = useSearchParams()
  const requestedConsumer = searchParams.get('consumer')
  useEffect(() => {
    if (!requestedConsumer) return
    setEditorState({ selectedName: requestedConsumer, isCreating: false, isEditing: false, editorMode: 'form', formDraft: null })
    setSearchParams(
      (params) => {
        params.delete('consumer')
        return params
      },
      { replace: true },
    )
  }, [requestedConsumer, setEditorState, setSearchParams])

  const [, setMessagesView] = useMessagesViewEntry(scope)
  const openMessage = (message: Message) => {
    setMessagesView({ selectedMessage: toSelectedHistoryMessage(message) })
    navigate(`/streams/${encodeURIComponent(streamName)}/messages`)
  }

  const { readOnly } = useConnectionPolicy()
  const createConsumer = useCreateConsumer(connectionId, streamName)
  const updateConsumer = useUpdateConsumer(connectionId, streamName)
  const deleteConsumer = useDeleteConsumer(connectionId, streamName)
  const pauseConsumer = usePauseConsumer(connectionId, streamName)
  const resumeConsumer = useResumeConsumer(connectionId, streamName)
  const resetConsumer = useResetConsumer(connectionId, streamName)
  const unpinConsumer = useUnpinConsumer(connectionId, streamName)

  const { unsupportedReason } = useServerCapabilities(connectionId)
  const pauseUnsupportedReason = unsupportedReason('consumerPause')
  const resetUnsupportedReason = unsupportedReason('consumerReset')
  const unpinUnsupportedReason = unsupportedReason('priorityGroups')
  const prioritizedUnsupportedReason = unsupportedReason('priorityPrioritized')

  const deleteConfirmation = useConfirmation('deleteConsumer')

  // Delete consumer + clear selection; shared by confirm-dialog and
  const performDeleteConsumer = async (consumer: ConsumerInfo): Promise<boolean> => {
    try {
      await deleteConsumer.mutateAsync(consumer.name)
      if (selectedName === consumer.name) {
        setEditorState({ selectedName: null, isEditing: false, formDraft: null })
      }
      return true
    } catch {
      // onError already toasted; swallow to avoid unhandled rejection.
      return false
    }
  }

  const requestDeleteConsumer = (consumer: ConsumerInfo) => {
    if (!deleteConfirmation.enabled) {
      void performDeleteConsumer(consumer)
      return
    }
    setConfirmAction({ type: 'delete', consumer })
  }

  // Resolve live consumer from persisted name; null if deleted server-side
  // (selection reset below).
  const selectedConsumer = useMemo<ConsumerInfo | null>(() => {
    if (!selectedName) return null
    return consumers.find((c) => c.name === selectedName) ?? null
  }, [consumers, selectedName])

  // Drop a stale selection if the consumer no longer exists upstream.
  useEffect(() => {
    if (selectedName && !isLoading && consumers.length > 0 && !selectedConsumer) {
      setEditorState({ selectedName: null, isEditing: false, formDraft: null })
    }
  }, [selectedName, isLoading, consumers.length, selectedConsumer, setEditorState])

  // "Original" config derived from live consumer; not persisted, recomputed to
  // stay in sync with backend.
  const originalConfig = useMemo(() => {
    if (!selectedConsumer || !isEditing) return null
    return consumerToConfig(selectedConsumer)
  }, [selectedConsumer, isEditing])

  // Seed formDraft on entering create/edit if empty; preserve existing drafts
  // (e.g. returned mid-edit).
  useEffect(() => {
    if (isCreating && !formDraft) {
      setEditorState({ formDraft: defaultConsumerConfig })
    } else if (selectedConsumer && isEditing && !formDraft) {
      setEditorState({ formDraft: consumerToConfig(selectedConsumer) })
    }
  }, [isCreating, isEditing, selectedConsumer, formDraft, setEditorState])

  const formValue = formDraft ?? defaultConsumerConfig
  const setFormValue = (next: typeof formValue) => setEditorState({ formDraft: next })

  const hasChanges = useMemo(() => {
    if (!formDraft || !originalConfig) return false
    return JSON.stringify(formDraft) !== JSON.stringify(originalConfig)
  }, [formDraft, originalConfig])

  const handleCreate = async () => {
    try {
      await createConsumer.mutateAsync(formValue)
      setEditorState({ isCreating: false, formDraft: null })
    } catch {
      /* toasted by the mutation hook; keep the editor open */
    }
  }

  const handleUpdate = async () => {
    if (!selectedConsumer || !originalConfig) return
    try {
      await updateConsumer.mutateAsync({
        name: selectedConsumer.name,
        config: toConsumerUpdateRequest(originalConfig, formValue),
      })
      setShowDiffModal(false)
      setEditorState({ isEditing: false, formDraft: null })
    } catch {
      /* toasted by the mutation hook; keep the editor open */
    }
  }

  const handleConfirmAction = async () => {
    if (!confirmAction) return
    if (confirmAction.type === 'delete') {
      // Persist the opt-out only after a successful delete, so a failed op
      // never silently disables prompts.
      const didDisable = confirmAction.dontAskAgain ?? false
      const succeeded = await performDeleteConsumer(confirmAction.consumer)
      if (didDisable && succeeded) {
        deleteConfirmation.disable()
      }
    } else if (confirmAction.type === 'pause') {
      const pauseUntil = new Date()
      pauseUntil.setMinutes(pauseUntil.getMinutes() + (confirmAction.pauseMinutes ?? 5))
      try {
        await pauseConsumer.mutateAsync({
          name: confirmAction.consumer.name,
          pauseUntil,
        })
      } catch {
        /* toasted by the mutation hook */
      }
    } else if (confirmAction.type === 'reset') {
      try {
        await resetConsumer.mutateAsync({
          name: confirmAction.consumer.name,
          sequence: parseResetSequence(confirmAction.resetSequence) ?? undefined,
        })
      } catch {
        /* toasted by the mutation hook */
      }
    }
    setConfirmAction(null)
  }

  const handleResume = async (consumer: ConsumerInfo) => {
    try {
      await resumeConsumer.mutateAsync(consumer.name)
    } catch {
      /* toasted by the mutation hook */
    }
  }

  return (
    <div className="flex-1 flex min-h-0 h-full">
      <ConsumerList
        consumers={consumers}
        selectedName={selectedName}
        searchQuery={searchQuery}
        onSearchChange={(q) => setEditorState({ searchQuery: q })}
        isLoading={isLoading}
        isFetching={isFetching}
        error={consumersError}
        onRefetch={() => refetch()}
        onSelect={(consumer) => {
          setEditorState({
            selectedName: consumer.name,
            isCreating: false,
            isEditing: false,
            editorMode: 'form',
            formDraft: null,
          })
        }}
        onCreate={readOnly ? undefined : () => {
          setEditorState({ isCreating: true, selectedName: null, formDraft: null })
        }}
        issues={issues}
      />

      <div className="flex-1 flex flex-col min-h-0 min-w-0">
        {isCreating && !readOnly ? (
          <ConsumerEditor
            title="Create New Consumer"
            subtitle={`Create a consumer for stream "${streamName}"`}
            mode={editorMode}
            onModeChange={(m) => setEditorState({ editorMode: m })}
            value={formValue}
            onChange={setFormValue}
            isEditMode={false}
            isSaving={createConsumer.isPending}
            onCancel={() => setEditorState({ isCreating: false, formDraft: null })}
            onCreate={handleCreate}
            priorityUnsupportedReason={unpinUnsupportedReason}
            prioritizedUnsupportedReason={prioritizedUnsupportedReason}
          />
        ) : selectedConsumer ? (
          isEditing && !readOnly ? (
            <ConsumerEditor
              title={selectedConsumer.name}
              subtitle="Edit consumer configuration"
              mode={editorMode}
              onModeChange={(m) => setEditorState({ editorMode: m })}
              value={formValue}
              onChange={setFormValue}
              originalValue={originalConfig}
              isEditMode={true}
              hasChanges={hasChanges}
              isSaving={updateConsumer.isPending}
              onCancel={() => {
                setEditorState({ isEditing: false, editorMode: 'form', formDraft: null })
              }}
              onShowDiff={() => setShowDiffModal(true)}
              priorityUnsupportedReason={unpinUnsupportedReason}
              prioritizedUnsupportedReason={prioritizedUnsupportedReason}
            />
          ) : (
            <ConsumerView
              consumer={selectedConsumer}
              streamName={streamName}
              isFetching={isFetching}
              onRefetch={() => refetch()}
              onEdit={() => setEditorState({ isEditing: true })}
              onPause={() =>
                setConfirmAction({
                  type: 'pause',
                  consumer: selectedConsumer,
                  pauseMinutes: 5,
                })
              }
              onResume={() => handleResume(selectedConsumer)}
              onReset={() => setConfirmAction({ type: 'reset', consumer: selectedConsumer })}
              onUnpin={(group) => unpinConsumer.mutate({ name: selectedConsumer.name, group })}
              onDelete={() => requestDeleteConsumer(selectedConsumer)}
              readOnly={readOnly}
              isResuming={resumeConsumer.isPending}
              isPausing={pauseConsumer.isPending}
              isResetting={resetConsumer.isPending}
              isUnpinning={unpinConsumer.isPending}
              pauseUnsupportedReason={pauseUnsupportedReason}
              resetUnsupportedReason={resetUnsupportedReason}
              unpinUnsupportedReason={unpinUnsupportedReason}
              issues={issues[selectedConsumer.name]}
              position={
                <ConsumerPosition
                  connectionId={connectionId}
                  streamName={streamName}
                  consumer={selectedConsumer}
                  firstSeq={streamDetail?.state?.first_seq}
                  onOpenMessage={openMessage}
                />
              }
            />
          )
        ) : (
          <div className="flex-1 flex items-center justify-center text-content-tertiary">
            <div className="text-center">
              <UsersIcon className="mx-auto h-12 w-12 text-content-muted mb-4" />
              <p className="text-sm mb-4">Select a consumer to edit or create a new one</p>
              <Button
                onClick={() => {
                  setEditorState({ isCreating: true, selectedName: null, formDraft: null })
                }}
              >
                <PlusIcon className="w-4 h-4 mr-2" />
                Create New Consumer
              </Button>
            </div>
          </div>
        )}
      </div>

      {confirmAction && (
        <ConsumerConfirmDialog
          action={confirmAction}
          onChange={setConfirmAction}
          onCancel={() => setConfirmAction(null)}
          onConfirm={handleConfirmAction}
        />
      )}

      {showDiffModal && originalConfig && selectedConsumer && (
        <ConfigDiffModal
          isOpen={showDiffModal}
          onClose={() => setShowDiffModal(false)}
          onConfirm={handleUpdate}
          title="Confirm Consumer Configuration Changes"
          description={`Review the changes to the consumer "${selectedConsumer.name}" before applying them.`}
          originalConfig={originalConfig}
          newConfig={formValue}
          isLoading={updateConsumer.isPending}
        />
      )}
    </div>
  )
}
