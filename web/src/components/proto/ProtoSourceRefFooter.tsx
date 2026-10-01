import { useState } from 'react'
import { Badge, Button, Input, SearchableSelect, Spinner } from '@/components/ui'
import ErrorAlert from '@/components/ui/ErrorAlert'
import type { CompileOutcome, ProtoRef, SchemaRevision } from '@/api/protoSources'
import { getErrorMessage } from '@/api/errors'
import { plural } from '@/utils/plural'
import { CompileDiagnosticsList } from './CompileDiagnosticsList'
import { TagIcon } from './protoCardIcons'

const TEXT = {
  git: {
    pick: 'Pick a tag, branch or commit to compile',
    select: 'Tag or branch',
    none: 'No tags or branches',
    placeholder: 'Select a tag or branch…',
    commit: 'Commit SHA',
    commitPlaceholder: 'or a 40-character commit SHA',
    busy: 'Fetching and compiling…',
  },
  bsr: {
    pick: 'Pick a label or commit to load',
    select: 'Label',
    none: 'No labels',
    placeholder: 'Select a label…',
    commit: 'Commit ID',
    commitPlaceholder: 'or a 32-character commit ID',
    busy: 'Downloading the schema…',
  },
}

interface Props {
  registry: keyof typeof TEXT
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

export function ProtoSourceRefFooter({
  registry,
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
  const text = TEXT[registry]

  return (
    <div className="space-y-2">
      {selectedRef ? (
        <div className="flex items-center justify-between gap-3">
          <div className="flex items-center gap-2 min-w-0">
            <TagIcon className="w-3.5 h-3.5 text-content-muted shrink-0" />
            <Badge variant="primary" size="sm" data-testid="ref-badge">
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
            <Button size="sm" variant="secondary" onClick={onRefresh} loading={isBusy} data-testid="ref-refresh">
              Refresh
            </Button>
            <Button size="sm" variant="ghost" onClick={onTogglePicker} data-testid="ref-change">
              Change
            </Button>
          </div>
        </div>
      ) : (
        <button
          type="button"
          onClick={onTogglePicker}
          className="flex items-center gap-1.5 text-xs font-medium text-accent hover:text-accent-text transition-colors"
          data-testid="ref-change"
        >
          <TagIcon className="w-3.5 h-3.5" />
          {text.pick}
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
              label={text.select}
              value={selectedRef?.kind === 'commit' ? undefined : selectedRef?.name}
              loading={isLoadingRefs}
              disabled={isBusy || (!isLoadingRefs && refs.length === 0)}
              placeholder={isLoadingRefs ? 'Loading refs…' : refs.length === 0 ? text.none : text.placeholder}
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
              aria-label={text.commit}
              value={commit}
              onChange={(e) => setCommit(e.target.value)}
              placeholder={text.commitPlaceholder}
              size="sm"
              mono
              data-testid="ref-commit-input"
            />
            <Button type="submit" size="sm" variant="secondary" disabled={isBusy || !commit.trim()} data-testid="ref-commit-use">
              Use
            </Button>
          </form>
        </div>
      )}

      {isBusy && (
        <div className="flex items-center gap-2 text-xs text-content-secondary py-2 px-3 bg-accent-light rounded-md">
          <Spinner size="sm" />
          <span>{text.busy}</span>
        </div>
      )}
      {actionError && <ErrorAlert compact message={getErrorMessage(actionError)} />}
      {outcome?.valid && (
        <div className="text-xs text-status-success-text p-2 bg-status-success-bg rounded" data-testid="ref-compile-ok">
          Compiled {plural(outcome.messageTypes, 'message type')}
        </div>
      )}
      {outcome && outcome.diagnostics.length > 0 && <CompileDiagnosticsList diagnostics={outcome.diagnostics} compact />}
    </div>
  )
}
