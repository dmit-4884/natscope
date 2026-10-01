import { useState } from 'react'
import { Badge, Button, Input, SearchableSelect, Spinner } from '@/components/ui'
import ErrorAlert from '@/components/ui/ErrorAlert'
import type { CompileOutcome, ProtoRef, SchemaRevision } from '@/api/protoSources'
import { getErrorMessage } from '@/api/errors'
import { plural } from '@/utils/plural'
import { CompileDiagnosticsList } from './CompileDiagnosticsList'
import { TagIcon } from './protoCardIcons'

interface Props {
  selectedRef?: ProtoRef
  activeSchema?: SchemaRevision
  showPicker: boolean
  onTogglePicker: () => void
  refs: ProtoRef[]
  isLoadingRefs: boolean
  refsError?: Error | null
  onRetryRefs?: () => void
  onSelectRef: (ref: string) => void
  onRefresh: () => void
  isBusy: boolean
  actionError?: Error | null
  outcome?: CompileOutcome | null
}

const shortRevision = (revision: string) => revision.slice(0, 7)

export function ProtoSourceGitFooter({
  selectedRef,
  activeSchema,
  showPicker,
  onTogglePicker,
  refs,
  isLoadingRefs,
  refsError,
  onRetryRefs,
  onSelectRef,
  onRefresh,
  isBusy,
  actionError,
  outcome,
}: Props) {
  const [commit, setCommit] = useState('')

  return (
    <div className="space-y-2">
      {selectedRef ? (
        <div className="flex items-center justify-between gap-3">
          <div className="flex items-center gap-2 min-w-0">
            <TagIcon className="w-3.5 h-3.5 text-content-muted shrink-0" />
            <Badge variant="primary" size="sm" data-testid="git-ref-badge">
              {selectedRef.name}
            </Badge>
            <span className="text-2xs text-content-tertiary shrink-0">{selectedRef.kind}</span>
            <code className="text-2xs text-content-tertiary" title={selectedRef.revision}>
              {shortRevision(selectedRef.revision)}
            </code>
            {activeSchema && (
              <span className="text-2xs text-content-muted truncate">· {plural(activeSchema.messageCount, 'message type')}</span>
            )}
          </div>
          <div className="flex items-center gap-2 shrink-0">
            <Button size="sm" variant="secondary" onClick={onRefresh} loading={isBusy} data-testid="git-ref-refresh">
              Refresh
            </Button>
            <Button size="sm" variant="ghost" onClick={onTogglePicker} data-testid="git-ref-change">
              Change
            </Button>
          </div>
        </div>
      ) : (
        <button
          type="button"
          onClick={onTogglePicker}
          className="flex items-center gap-1.5 text-xs font-medium text-accent hover:text-accent-text transition-colors"
          data-testid="git-ref-change"
        >
          <TagIcon className="w-3.5 h-3.5" />
          Pick a tag, branch or commit to compile
        </button>
      )}

      {showPicker && (
        <div className="space-y-2">
          {refsError ? (
            <div className="flex items-center justify-between gap-2">
              <ErrorAlert compact message={`Failed to list refs: ${getErrorMessage(refsError)}`} />
              {onRetryRefs && (
                <Button size="sm" variant="ghost" onClick={onRetryRefs}>
                  Retry
                </Button>
              )}
            </div>
          ) : (
            <SearchableSelect
              label="Tag or branch"
              value={selectedRef?.kind === 'commit' ? undefined : selectedRef?.name}
              loading={isLoadingRefs}
              disabled={isBusy || (!isLoadingRefs && refs.length === 0)}
              placeholder={isLoadingRefs ? 'Loading refs…' : refs.length === 0 ? 'No tags or branches' : 'Select a tag or branch…'}
              searchPlaceholder="Filter refs…"
              onChange={onSelectRef}
              options={refs.map((ref) => ({
                value: ref.name,
                label: `${ref.name} · ${ref.kind} · ${shortRevision(ref.revision)}`,
              }))}
            />
          )}
          <form
            className="flex items-center gap-2"
            onSubmit={(e) => {
              e.preventDefault()
              if (commit.trim()) onSelectRef(commit.trim())
            }}
          >
            <Input
              aria-label="Commit SHA"
              value={commit}
              onChange={(e) => setCommit(e.target.value)}
              placeholder="or a 40-character commit SHA"
              size="sm"
              mono
              data-testid="git-commit-input"
            />
            <Button type="submit" size="sm" variant="secondary" disabled={isBusy || !commit.trim()} data-testid="git-commit-use">
              Use
            </Button>
          </form>
        </div>
      )}

      {isBusy && (
        <div className="flex items-center gap-2 text-xs text-content-secondary py-2 px-3 bg-accent-light rounded-md">
          <Spinner size="sm" />
          <span>Fetching and compiling…</span>
        </div>
      )}
      {actionError && <ErrorAlert compact message={getErrorMessage(actionError)} />}
      {outcome?.valid && (
        <div className="text-xs text-status-success-text p-2 bg-status-success-bg rounded" data-testid="git-compile-ok">
          Compiled {plural(outcome.messageTypes, 'message type')}
        </div>
      )}
      {outcome && outcome.diagnostics.length > 0 && <CompileDiagnosticsList diagnostics={outcome.diagnostics} compact />}
    </div>
  )
}
