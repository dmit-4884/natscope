import type { ValidationResult } from '@/api/publish'

interface Props {
  result: ValidationResult | null
}

export function ValidationResultDisplay({ result }: Props) {
  if (!result) return null

  if (!result.valid) {
    return (
      <div className="mt-3 p-3 bg-status-error-bg border border-red-200 rounded-lg">
        <div className="flex items-start gap-2">
          <svg className="w-5 h-5 text-red-500 flex-shrink-0 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
            />
          </svg>
          <div className="flex-1">
            <p className="text-sm font-medium text-red-800">Validation Failed</p>
            {result.error && <p className="text-sm text-status-error-text mt-1">{result.error}</p>}
            {result.violations && result.violations.length > 0 && (
              <ul className="mt-2 space-y-1">
                {result.violations.map((v, i) => (
                  <li key={i} className="text-sm text-red-700">
                    <code className="bg-status-error-light px-1 rounded text-xs">{v.field_path || 'root'}</code>
                    <span className="ml-2">{v.message}</span>
                  </li>
                ))}
              </ul>
            )}
          </div>
        </div>
      </div>
    )
  }

  return (
    <div className="mt-3 p-3 bg-status-success-bg border border-green-200 rounded-lg">
      <div className="flex items-center gap-2">
        <svg className="w-5 h-5 text-green-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
        </svg>
        <p className="text-sm font-medium text-green-800">Message is valid</p>
      </div>
    </div>
  )
}
