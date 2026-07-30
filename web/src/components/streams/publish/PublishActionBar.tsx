import { Spinner } from '@/components/ui'
import { isMacPlatform } from '@/utils/platform'
import { plural } from '@/utils/plural'

/**
 * Schema-validation status by the Publish button; 'none' = JSON mode or nothing
 * typed yet.
 */
export type ValidationState = 'none' | 'validating' | 'valid' | 'invalid'

interface Props {
  validationState: ValidationState
  violationCount?: number
  canPublish: boolean
  disabledReason?: string | null
  isPublishing: boolean
  onPublish: () => void
  children?: React.ReactNode
}

const DISABLED_REASON_ID = 'publish-disabled-reason'

function ValidationBadge({ state, violationCount }: { state: ValidationState; violationCount: number }) {
  if (state === 'none') return null
  if (state === 'validating') {
    return (
      <span
        className="flex items-center gap-1.5 text-xs text-content-muted"
        data-testid="validation-badge"
        data-state="validating"
      >
        <Spinner size="sm" />
        Validating…
      </span>
    )
  }
  if (state === 'valid') {
    return (
      <span
        className="flex items-center gap-1 px-2 py-1 text-xs font-medium text-green-700 bg-status-success-bg rounded-md"
        data-testid="validation-badge"
        data-state="valid"
      >
        <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
        </svg>
        Schema valid
      </span>
    )
  }
  return (
    <span
      className="flex items-center gap-1 px-2 py-1 text-xs font-medium text-red-700 bg-status-error-bg rounded-md"
      data-testid="validation-badge"
      data-state="invalid"
    >
      <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
      </svg>
      {violationCount > 0 ? plural(violationCount, 'violation') : 'Schema invalid'}
    </span>
  )
}

export function PublishActionBar({
  validationState,
  violationCount = 0,
  canPublish,
  disabledReason,
  isPublishing,
  onPublish,
  children,
}: Props) {
  const platformCmd = isMacPlatform() ? '⌘' : 'Ctrl'
  const showDisabledReason = !isPublishing && !canPublish && !!disabledReason

  return (
    <div className="flex items-center justify-between">
      <div className="flex items-center gap-2">
        <span className="text-xs text-content-muted">{platformCmd}+Enter to publish</span>
        {children}
      </div>
      <div className="flex items-center gap-3">
        {showDisabledReason && (
          <span
            id={DISABLED_REASON_ID}
            data-testid="publish-disabled-reason"
            className="text-xs text-status-warning-text"
          >
            {disabledReason}
          </span>
        )}
        <ValidationBadge state={validationState} violationCount={violationCount} />
        <button
          onClick={onPublish}
          disabled={isPublishing || !canPublish}
          aria-describedby={showDisabledReason ? DISABLED_REASON_ID : undefined}
          className="px-6 py-2.5 bg-accent text-content-inverse font-medium rounded-lg hover:bg-accent-hover transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
        >
          {isPublishing ? (
            <>
              <Spinner size="sm" />
              Publishing...
            </>
          ) : (
            <>
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 19l9 2-9-18-9 18 9-2zm0 0v-8" />
              </svg>
              Publish Message
            </>
          )}
        </button>
      </div>
    </div>
  )
}
