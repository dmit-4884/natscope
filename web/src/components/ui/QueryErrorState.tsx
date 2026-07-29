import { cn } from '@/utils/cn'
import { Button } from './Button'
import { WarningIcon } from './icons'

export interface QueryErrorStateProps {
  error: unknown
  onRetry?: () => void
  title?: string
  className?: string
}

/**
 * Inline error placeholder for hook-returned errors.
 *
 * Pairs with `useQuery({ error })` — the common case where you want
 * to show a retry UI without throwing to an ErrorBoundary.
 *
 * @example
 * const { data, error, refetch } = useStreams()
 * if (error) return <QueryErrorState error={error} onRetry={refetch} />
 */
export function QueryErrorState({
  error,
  onRetry,
  title = 'Failed to load',
  className,
}: QueryErrorStateProps) {
  const message =
    error instanceof Error ? error.message : typeof error === 'string' ? error : 'Unknown error'

  return (
    <div
      role="alert"
      className={cn('flex flex-col items-center justify-center py-10 text-center px-4', className)}
    >
      <WarningIcon className="w-10 h-10 text-red-400 mb-3" />
      <h3 className="text-sm font-semibold text-content-primary mb-1">{title}</h3>
      <p className="text-xs text-content-secondary max-w-sm mb-4 whitespace-pre-wrap">{message}</p>
      {onRetry && (
        <Button variant="secondary" size="sm" onClick={onRetry}>
          Retry
        </Button>
      )}
    </div>
  )
}
