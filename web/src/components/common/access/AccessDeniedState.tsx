import { Button, LockClosedIcon } from '@/components/ui'
import { describePermission, type AccessCheck } from '@/shared/domain/access'

interface Props {
  check: AccessCheck
  title: string
  description: string
  onRetry?: () => void
  retrying?: boolean
}

export function AccessDeniedState({ check, title, description, onRetry, retrying }: Props) {
  return (
    <div className="flex-1 flex items-center justify-center p-8" role="status" data-testid="access-denied">
      <div className="max-w-md text-center">
        <div className="mx-auto mb-4 w-11 h-11 rounded-full bg-surface-tertiary flex items-center justify-center text-content-tertiary">
          <LockClosedIcon className="w-5 h-5" />
        </div>
        <h3 className="text-base font-semibold text-content-primary">{title}</h3>
        <p className="mt-1.5 text-sm text-content-secondary">{description}</p>
        <div className="mt-4 rounded-md border border-border bg-surface-secondary px-3 py-2 text-left">
          <div className="text-2xs font-semibold uppercase tracking-wide text-content-tertiary">Missing permission</div>
          <code className="mt-0.5 block text-sm text-content-primary break-all">{describePermission(check)}</code>
        </div>
        <p className="mt-3 text-xs text-content-tertiary">
          Ask the operator of this NATS account to grant it. Everything else on this connection keeps working.
        </p>
        {onRetry && (
          <Button variant="secondary" size="sm" className="mt-4" onClick={onRetry} loading={retrying}>
            Check again
          </Button>
        )}
      </div>
    </div>
  )
}
