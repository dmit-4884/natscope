import { useState } from 'react'
import { Button } from '@/components/ui'
import ErrorAlert from '@/components/ui/ErrorAlert'
import type { CompileDiagnostic } from '@/api/protoSources'
import { plural } from '@/utils/plural'
import { CompileDiagnosticsList } from './CompileDiagnosticsList'

interface Props {
  files: string[]
  includeDirs: string[]
  onCompile: () => Promise<{ valid: boolean; messageTypes: number; fileDescriptors: number; diagnostics: CompileDiagnostic[] } | undefined>
  isCompiling: boolean
  compileError?: Error | null
}

/**
 * Footer for files-type sources. One action — Compile now — which both runs
 * the compiler and persists the resulting descriptor. The previous Validate
 * button was a dry-run of the same compile (same parser, same diagnostics)
 * but didn't save anything; effectively it duplicated Compile now without the
 * "actually take effect" half. Removed for clarity.
 */
export function ProtoSourceFilesFooter({
  files,
  includeDirs,
  onCompile,
  isCompiling,
  compileError,
}: Props) {
  const [diagnostics, setDiagnostics] = useState<CompileDiagnostic[] | null>(null)
  const [result, setResult] = useState<{ messageTypes: number; fileDescriptors: number } | null>(null)

  const handleCompile = async () => {
    setDiagnostics(null)
    setResult(null)
    const r = await onCompile()
    if (!r) return
    setDiagnostics(r.diagnostics)
    if (r.valid) setResult({ messageTypes: r.messageTypes, fileDescriptors: r.fileDescriptors })
  }

  return (
    <div className="space-y-2">
      <div className="flex items-center justify-between">
        <div className="text-xs text-content-tertiary">
          {plural(files.length, 'file')}
          {includeDirs.length > 0 && (
            <>
              {' · '}
              {plural(includeDirs.length, 'include dir')}
            </>
          )}
        </div>
        <Button size="sm" variant="secondary" onClick={handleCompile} loading={isCompiling}>
          Compile now
        </Button>
      </div>

      {compileError && <ErrorAlert compact message={compileError.message} />}
      {result && !diagnostics?.some((d) => d.severity === 'error') && (
        <div className="text-xs text-green-700 p-2 bg-status-success-bg rounded">
          Compiled: {result.fileDescriptors} file descriptors, {result.messageTypes} message types
        </div>
      )}
      {diagnostics && diagnostics.length > 0 && <CompileDiagnosticsList diagnostics={diagnostics} compact />}
    </div>
  )
}
