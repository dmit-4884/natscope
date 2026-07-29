import { useState } from 'react'
import { Button, RowActionButton, Toggle } from '@/components/ui'
import {
  useDeleteProtoSource,
  useProtoSourceTags,
  useSelectProtoVersion,
  useSetSourceEnabled,
  useSetWatcher,
  useCompileLocal,
  useCompileFiles,
} from '@/contexts/proto'
import type { ProtoSource, ProtoSourceType, ProtoSelection, CompileDiagnostic } from '@/api/protoSources'
import { ProtoSourceGitFooter } from './ProtoSourceGitFooter'
import { ProtoSourceLocalFooter } from './ProtoSourceLocalFooter'
import { ProtoSourceFilesFooter } from './ProtoSourceFilesFooter'

const TYPE_BADGE_CONFIG: Record<ProtoSourceType, { bg: string; text: string; label: string }> = {
  git: { bg: 'bg-accent-muted', text: 'text-accent-text', label: 'Git' },
  local: { bg: 'bg-status-warning-light', text: 'text-amber-700', label: 'Local' },
  files: { bg: 'bg-status-success-light', text: 'text-emerald-700', label: 'Files' },
}

interface ProtoSourceCardProps {
  source: ProtoSource
  selection?: ProtoSelection
  onEdit: (source: ProtoSource) => void
  onRemoveSelection?: (selectionId: string) => void
}

export default function ProtoSourceCard({
  source,
  selection,
  onEdit,
  onRemoveSelection,
}: ProtoSourceCardProps) {
  const [isDeleting, setIsDeleting] = useState(false)
  const [showConfirmDelete, setShowConfirmDelete] = useState(false)
  const [showTagSelector, setShowTagSelector] = useState(false)
  const [compileResult, setCompileResult] = useState<{
    messageTypes: number
    fileDescriptors: number
    valid?: boolean
    diagnostics?: CompileDiagnostic[]
  } | null>(null)

  const deleteMutation = useDeleteProtoSource()
  const selectMutation = useSelectProtoVersion()
  const enabledMutation = useSetSourceEnabled()
  const watcherMutation = useSetWatcher()
  const compileMutation = useCompileLocal()
  const compileFilesMutation = useCompileFiles()
  const { data: tags = [], isLoading: isLoadingTags } = useProtoSourceTags(
    showTagSelector && source.sourceType === 'git' ? source.id : null,
  )

  const handleDelete = async () => {
    setIsDeleting(true)
    try {
      await deleteMutation.mutateAsync({ id: source.id })
    } catch {
      /* toasted globally */
    } finally {
      setIsDeleting(false)
      setShowConfirmDelete(false)
    }
  }

  const handleSelectTag = async (tag: string) => {
    if (!tag) return
    try {
      await selectMutation.mutateAsync({ sourceId: source.id, tag })
      setShowTagSelector(false)
    } catch {
      /* toasted globally */
    }
  }

  const handleRemoveSelection = () => {
    if (selection && onRemoveSelection) {
      onRemoveSelection(selection.id)
    }
  }

  const handleToggleEnabled = async (enabled: boolean) => {
    try {
      await enabledMutation.mutateAsync({ sourceId: source.id, enabled })
    } catch {
      /* toasted globally */
    }
  }

  const handleToggleWatcher = async (enabled: boolean) => {
    try {
      await watcherMutation.mutateAsync({ sourceId: source.id, enabled })
    } catch {
      /* toasted globally */
    }
  }

  const handleCompileLocal = async () => {
    try {
      const result = await compileMutation.mutateAsync({ sourceId: source.id })
      setCompileResult(result)
    } catch {
      /* toasted globally */
    }
  }

  const handleCompileFiles = async () => {
    try {
      return await compileFilesMutation.mutateAsync({ sourceId: source.id })
    } catch {
      return undefined
    }
  }

  const typeBadge = TYPE_BADGE_CONFIG[source.sourceType]
  const isDisabled = !source.enabled
  const repoDisplayName =
    source.sourceType === 'git'
      ? source.repository.replace(/^https?:\/\//, '').replace(/\.git$/, '')
      : undefined

  return (
    <div
      className={`bg-surface-primary border rounded-lg transition-all border-border hover:border-border-strong hover:shadow-sm ${
        isDisabled ? 'opacity-60' : ''
      }`}
    >
      <div className="p-3">
        {/* Header */}
        <div className="flex items-start justify-between gap-3">
          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2 flex-wrap">
              <h4 className="text-sm font-medium truncate text-content-primary">{source.name}</h4>
              <span
                className={`inline-flex items-center px-1.5 py-0.5 rounded text-2xs font-medium ${typeBadge.bg} ${typeBadge.text}`}
              >
                {typeBadge.label}
              </span>
              {/*
                Live watcher signal next to the type pill — applies only to
                local sources. fsnotify is the live channel that turns "edit
                .proto in IDE" into "decoder picks up new schema", so its state
                belongs in the header, not just the footer's toggle.
              */}
              {source.sourceType === 'local' && source.enabled && (
                <span
                  className={`inline-flex items-center gap-1 px-1.5 py-0.5 rounded-full text-2xs font-medium ${
                    source.watcherEnabled
                      ? 'bg-status-success-bg text-emerald-700'
                      : 'bg-surface-tertiary text-content-tertiary'
                  }`}
                  title={
                    source.watcherEnabled
                      ? 'fsnotify is watching this directory — saves trigger automatic recompile'
                      : 'File watcher is off — recompile must be triggered manually'
                  }
                >
                  <span
                    aria-hidden="true"
                    className={`w-1 h-1 rounded-full ${
                      source.watcherEnabled ? 'bg-emerald-500 animate-pulse' : 'bg-gray-400'
                    }`}
                  />
                  {source.watcherEnabled ? 'watching' : 'watcher off'}
                </span>
              )}
            </div>
            {repoDisplayName && (
              <p className="text-xs text-content-muted truncate mt-0.5" title={source.repository}>
                {repoDisplayName}
              </p>
            )}
            {source.sourceType === 'local' && source.localPath && (
              <p className="text-xs text-content-muted truncate mt-0.5 font-mono" title={source.localPath}>
                {source.localPath}
              </p>
            )}
          </div>

          <div className="flex items-center gap-1.5 shrink-0">
            <Toggle
              checked={source.enabled}
              onChange={handleToggleEnabled}
              disabled={enabledMutation.isPending}
              size="xs"
              label={`Enable ${source.name}`}
            />
            <RowActionButton
              kind="edit"
              onClick={() => onEdit(source)}
              label={`Edit source ${source.name}`}
              title="Edit source"
            />
            <RowActionButton
              kind="delete"
              onClick={() => setShowConfirmDelete(true)}
              label={`Delete source ${source.name}`}
              title="Delete source"
            />
          </div>
        </div>

        {/* Type-specific footer */}
        <div className="mt-3 pt-3 border-t border-gray-100">
          {source.sourceType === 'git' && (
            <ProtoSourceGitFooter
              selection={selection}
              showTagSelector={showTagSelector}
              onToggleTagSelector={() => setShowTagSelector((v) => !v)}
              onRemoveSelection={handleRemoveSelection}
              tags={tags}
              isLoadingTags={isLoadingTags}
              isSelecting={selectMutation.isPending}
              selectError={selectMutation.isError ? (selectMutation.error as Error) : null}
              onSelectTag={handleSelectTag}
              diagnostics={source.lastCompile?.diagnostics}
            />
          )}

          {source.sourceType === 'local' && (
            <ProtoSourceLocalFooter
              localPath={source.localPath}
              enabled={source.enabled}
              watcherEnabled={source.watcherEnabled}
              onCompile={handleCompileLocal}
              onToggleWatcher={handleToggleWatcher}
              isCompiling={compileMutation.isPending}
              isWatcherPending={watcherMutation.isPending}
              compileError={compileMutation.isError ? (compileMutation.error as Error) : null}
              compileResult={compileResult}
              diagnostics={compileResult?.diagnostics ?? source.lastCompile?.diagnostics}
              detectedRoots={source.lastCompile?.roots}
              rootsOrigin={source.lastCompile?.rootsOrigin}
            />
          )}

          {source.sourceType === 'files' && (
            <ProtoSourceFilesFooter
              files={source.files}
              includeDirs={source.includeDirs}
              onCompile={handleCompileFiles}
              isCompiling={compileFilesMutation.isPending}
              compileError={compileFilesMutation.isError ? (compileFilesMutation.error as Error) : null}
            />
          )}
        </div>
      </div>

      {/* Delete confirmation */}
      {showConfirmDelete && (
        <div className="px-3 pb-3">
          <div className="flex items-center justify-between gap-3 p-3 bg-status-error-bg rounded-md border border-red-100">
            <span className="text-xs text-red-700">Delete this source?</span>
            <div className="flex items-center gap-2">
              <Button
                size="sm"
                variant="secondary"
                onClick={() => setShowConfirmDelete(false)}
                disabled={isDeleting}
              >
                Cancel
              </Button>
              <Button size="sm" variant="danger" onClick={handleDelete} loading={isDeleting}>
                Delete
              </Button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
