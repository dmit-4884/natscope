import { Button, Toggle } from '@/components/ui'
import ErrorAlert from '@/components/ui/ErrorAlert'
import type { CompileDiagnostic } from '@/api/protoSources'
import { getErrorMessage } from '@/api/errors'
import { plural } from '@/utils/plural'
import { CompileDiagnosticsList } from './CompileDiagnosticsList'
import { FolderIcon } from './protoCardIcons'

interface Props {
  localPath?: string
  enabled: boolean
  watcherEnabled: boolean
  onCompile: () => void
  onToggleWatcher: (enabled: boolean) => void
  isCompiling: boolean
  isWatcherPending: boolean
  compileError?: Error | null
  compileResult?: { fileDescriptors: number; messageTypes: number; valid?: boolean } | null
  diagnostics?: CompileDiagnostic[]
  detectedRoots?: string[]
  rootsOrigin?: string
}

export function ProtoSourceLocalFooter({
  localPath,
  enabled,
  watcherEnabled,
  onCompile,
  onToggleWatcher,
  isCompiling,
  isWatcherPending,
  compileError,
  compileResult,
  diagnostics,
  detectedRoots,
  rootsOrigin,
}: Props) {
  return (
    <div className="space-y-2">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <FolderIcon className="w-3.5 h-3.5 text-content-muted" />
          <span className="text-xs text-content-tertiary font-mono truncate max-w-[160px]" title={localPath}>
            {localPath}
          </span>
        </div>
        <Button size="sm" variant="secondary" onClick={onCompile} loading={isCompiling}>
          Compile now
        </Button>
      </div>
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <span className="text-xs text-content-tertiary">File Watcher:</span>
          <Toggle
            checked={watcherEnabled}
            onChange={onToggleWatcher}
            disabled={isWatcherPending}
            size="xs"
            label="File watcher"
          />
          {watcherEnabled && enabled ? (
            <span className="flex items-center gap-1 text-xs text-status-success-text">
              <span aria-hidden="true" className="relative flex h-2 w-2">
                <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-green-400 opacity-75" />
                <span className="relative inline-flex rounded-full h-2 w-2 bg-green-500" />
              </span>
              Watching
            </span>
          ) : (
            <span className="text-xs text-content-muted">Paused</span>
          )}
        </div>
      </div>
      {compileError && <ErrorAlert compact message={getErrorMessage(compileError)} />}
      {compileResult?.valid && !diagnostics?.some((d) => d.severity === 'error') && (
        <div className="text-xs text-green-700 p-2 bg-status-success-bg rounded">
          Compiled: {plural(compileResult.fileDescriptors, 'file descriptor')}, {plural(compileResult.messageTypes, 'message type')}
        </div>
      )}
      {detectedRoots && detectedRoots.length > 0 && (
        <div className="text-xs text-content-tertiary">
          Roots ({rootsOrigin || 'inferred'}): <span className="font-mono">{detectedRoots.join(', ')}</span>
        </div>
      )}
      {diagnostics && diagnostics.length > 0 && <CompileDiagnosticsList diagnostics={diagnostics} compact />}
    </div>
  )
}
