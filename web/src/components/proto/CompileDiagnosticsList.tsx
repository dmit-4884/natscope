import type { CompileDiagnostic } from '@/api/protoSources'
import { plural } from '@/utils/plural'

interface Props {
  diagnostics: CompileDiagnostic[]
  compact?: boolean
}

function locationOf(d: CompileDiagnostic): string {
  if (!d.file) return ''
  if (d.line > 0) {
    return `${d.file}:${d.line}${d.column > 0 ? `:${d.column}` : ''}`
  }
  return d.file
}

export function CompileDiagnosticsList({ diagnostics, compact }: Props) {
  if (diagnostics.length === 0) return null

  const errors = diagnostics.filter((d) => d.severity === 'error')
  const warnings = diagnostics.filter((d) => d.severity === 'warning')
  const notes = diagnostics.filter((d) => d.severity === 'info')

  const textSize = compact ? 'text-xs' : 'text-sm'
  const titleSize = compact ? 'text-sm' : 'text-base'

  return (
    <div className="space-y-2">
      {errors.length > 0 && (
        <div
          className={`rounded border border-red-200 bg-status-error-bg ${compact ? 'p-2.5' : 'p-3'}`}
          data-testid="compile-diagnostics-errors"
        >
          <div className={`font-semibold text-red-800 ${titleSize}`}>
            {plural(errors.length, 'compile error')}
          </div>
          <ul className="mt-1.5 space-y-1.5">
            {errors.map((d, i) => (
              <li key={i} className={`text-red-700 ${textSize}`}>
                <div className="flex flex-wrap items-baseline gap-x-2">
                  {locationOf(d) && (
                    <code className={`font-mono ${textSize} text-red-900`}>{locationOf(d)}</code>
                  )}
                  <span>{d.message}</span>
                </div>
                {d.hint && <div className={`mt-1 ${textSize} text-status-error-text italic`}>↳ {d.hint}</div>}
                {d.missingImport && (
                  <div className={`mt-1 ${textSize} text-status-error-text`}>
                    Missing import: <code className="font-mono">{d.missingImport}</code>
                  </div>
                )}
              </li>
            ))}
          </ul>
        </div>
      )}
      {warnings.length > 0 && (
        <div
          className={`rounded border border-amber-200 bg-status-warning-bg ${compact ? 'p-2.5' : 'p-3'}`}
          data-testid="compile-diagnostics-warnings"
        >
          <div className={`font-semibold text-amber-800 ${titleSize}`}>
            {plural(warnings.length, 'warning')}
          </div>
          <ul className="mt-1.5 space-y-1.5">
            {warnings.map((d, i) => (
              <li key={i} className={`text-amber-700 ${textSize}`}>
                <div className="flex flex-wrap items-baseline gap-x-2">
                  {locationOf(d) && (
                    <code className={`font-mono ${textSize} text-amber-900`}>{locationOf(d)}</code>
                  )}
                  <span>{d.message}</span>
                </div>
                {d.hint && <div className={`mt-1 ${textSize} text-status-warning-text italic`}>↳ {d.hint}</div>}
              </li>
            ))}
          </ul>
        </div>
      )}
      {notes.length > 0 && (
        <div
          className={`rounded border border-border bg-surface-secondary ${compact ? 'p-2.5' : 'p-3'}`}
          data-testid="compile-diagnostics-notes"
        >
          <div className={`font-semibold text-content-secondary ${titleSize}`}>
            {plural(notes.length, 'note')}
          </div>
          <ul className="mt-1.5 space-y-1.5">
            {notes.map((d, i) => (
              <li key={i} className={`text-content-secondary ${textSize}`}>
                <div className="flex flex-wrap items-baseline gap-x-2">
                  {locationOf(d) && (
                    <code className={`font-mono ${textSize} text-gray-700`}>{locationOf(d)}</code>
                  )}
                  <span>{d.message}</span>
                </div>
                {d.hint && <div className={`mt-1 ${textSize} text-content-tertiary italic`}>↳ {d.hint}</div>}
              </li>
            ))}
          </ul>
        </div>
      )}
    </div>
  )
}
