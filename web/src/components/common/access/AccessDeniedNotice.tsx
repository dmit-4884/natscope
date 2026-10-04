import { Alert, LockClosedIcon } from '@/components/ui'
import { describePermission, type AccessCheck } from '@/shared/domain/access'

export function AccessDeniedNotice({ check, title = 'No permission' }: { check: AccessCheck; title?: string }) {
  return (
    <Alert variant="warning" title={title} icon={<LockClosedIcon className="w-5 h-5" />} data-testid="access-denied-notice">
      This NATS user may not <code>{describePermission(check)}</code>. Ask the operator of this NATS account to grant it.
    </Alert>
  )
}
