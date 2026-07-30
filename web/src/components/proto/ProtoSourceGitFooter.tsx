import { Badge, Dropdown, Spinner } from '@/components/ui'
import ErrorAlert from '@/components/ui/ErrorAlert'
import type { ProtoSelection, CompileDiagnostic } from '@/api/protoSources'
import { getErrorMessage } from '@/api/errors'
import { CompileDiagnosticsList } from './CompileDiagnosticsList'
import { TagIcon, CheckIcon } from './protoCardIcons'

interface Props {
  selection?: ProtoSelection
  showTagSelector: boolean
  onToggleTagSelector: () => void
  onRemoveSelection: () => void
  tags: string[]
  isLoadingTags: boolean
  isSelecting: boolean
  selectError?: Error | null
  onSelectTag: (tag: string) => void
  diagnostics?: CompileDiagnostic[]
}

export function ProtoSourceGitFooter({
  selection,
  showTagSelector,
  onToggleTagSelector,
  onRemoveSelection,
  tags,
  isLoadingTags,
  isSelecting,
  selectError,
  onSelectTag,
  diagnostics,
}: Props) {
  return (
    <>
      {selection ? (
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <TagIcon className="w-3.5 h-3.5 text-content-muted" />
            <span className="text-xs text-content-tertiary">Version</span>
            <div className="flex items-center gap-1.5">
              <Badge variant="primary" size="sm">
                {selection.tag}
              </Badge>
              <CheckIcon className="w-3.5 h-3.5 text-green-500" />
            </div>
          </div>
          <div className="flex items-center gap-3">
            <button
              onClick={onToggleTagSelector}
              className="text-xs font-medium text-accent hover:text-blue-800 transition-colors"
            >
              Change
            </button>
            <button
              onClick={onRemoveSelection}
              className="text-xs font-medium text-content-tertiary hover:text-status-error-text transition-colors"
            >
              Remove
            </button>
          </div>
        </div>
      ) : (
        <button
          onClick={onToggleTagSelector}
          className="flex items-center gap-1.5 text-xs font-medium text-accent hover:text-blue-800 transition-colors"
        >
          <TagIcon className="w-3.5 h-3.5" />
          Select version to load proto files
        </button>
      )}

      {showTagSelector && (
        <div className="mt-3">
          {isSelecting ? (
            <div className="flex items-center gap-2 text-xs text-content-secondary py-2 px-3 bg-accent-light rounded-md">
              <Spinner size="sm" />
              <span>Fetching and loading proto files...</span>
            </div>
          ) : (
            <Dropdown
              value=""
              disabled={tags.length === 0}
              loading={isLoadingTags}
              onChange={onSelectTag}
              placeholder={
                isLoadingTags
                  ? 'Loading available versions...'
                  : tags.length === 0
                    ? 'No versions found'
                    : 'Select a version...'
              }
              options={tags.map((tag) => ({ value: tag, label: tag }))}
            />
          )}
          {selectError && <ErrorAlert compact message={getErrorMessage(selectError)} className="mt-2" />}
        </div>
      )}
      {diagnostics && diagnostics.length > 0 && (
        <div className="mt-3">
          <CompileDiagnosticsList diagnostics={diagnostics} compact />
        </div>
      )}
    </>
  )
}
