import { cn } from '@/utils/cn'

interface SkeletonProps {
  className?: string
}

function Skeleton({ className }: SkeletonProps) {
  return <div aria-hidden="true" className={cn('animate-shimmer rounded', className)} />
}

Skeleton.displayName = 'Skeleton'

export interface SkeletonRowsProps {
  count?: number
  rowClassName?: string
  className?: string
}

export function SkeletonRows({ count = 5, rowClassName = 'h-8', className }: SkeletonRowsProps) {
  return (
    <div role="status" aria-busy="true" aria-label="Loading" className={cn('space-y-2', className)}>
      {Array.from({ length: count }, (_, i) => (
        <Skeleton key={i} className={rowClassName} />
      ))}
    </div>
  )
}

SkeletonRows.displayName = 'SkeletonRows'
