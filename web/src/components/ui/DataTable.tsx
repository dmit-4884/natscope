import { type KeyboardEvent, type ReactNode } from 'react'
import { cn } from '@/utils/cn'
import { QueryErrorState } from './QueryErrorState'
import { SkeletonRows } from './Skeleton'

export interface DataTableColumn<T> {
  key: string
  header: ReactNode
  width?: string
  align?: 'left' | 'right'
  render: (item: T) => ReactNode
}

export interface DataTableProps<T> {
  columns: DataTableColumn<T>[]
  items: T[]
  rowKey: (item: T) => string
  onRowClick?: (item: T) => void
  rowActions?: (item: T) => ReactNode
  isLoading?: boolean
  loadingRows?: number
  emptyState?: ReactNode
  error?: unknown
  onRetry?: () => void
  className?: string
  selectedKey?: string
  rowClassName?: (item: T) => string
}

export function DataTable<T>({
  columns,
  items,
  rowKey,
  onRowClick,
  rowActions,
  isLoading = false,
  loadingRows = 5,
  emptyState,
  error,
  onRetry,
  className,
  selectedKey,
  rowClassName,
}: DataTableProps<T>) {
  if (isLoading) {
    return <SkeletonRows count={loadingRows} rowClassName="h-9" className="p-2" />
  }

  if (error) {
    return <QueryErrorState error={error} onRetry={onRetry} />
  }

  if (items.length === 0) {
    return <>{emptyState ?? null}</>
  }

  const hasActions = Boolean(rowActions)

  return (
    <table className={cn('w-full text-sm', className)}>
      <colgroup>
        {columns.map((column) => (
          <col key={column.key} className={column.width} />
        ))}
        {hasActions && <col />}
      </colgroup>
      <thead className="bg-surface-secondary text-xs uppercase tracking-wide text-content-tertiary sticky top-0">
        <tr>
          {columns.map((column) => (
            <th
              key={column.key}
              className={cn('px-4 py-2 font-medium', column.align === 'right' ? 'text-right' : 'text-left')}
            >
              {column.header}
            </th>
          ))}
          {hasActions && <th className="px-4 py-2 text-right font-medium">Actions</th>}
        </tr>
      </thead>
      <tbody className="bg-surface-primary divide-y divide-gray-100">
        {items.map((item) => {
          const key = rowKey(item)
          const clickable = Boolean(onRowClick)
          const selected = selectedKey !== undefined && key === selectedKey

          const handleKeyDown = (event: KeyboardEvent<HTMLTableRowElement>) => {
            if (!clickable) return
            if (event.key === 'Enter' || event.key === ' ') {
              event.preventDefault()
              onRowClick?.(item)
            }
          }

          return (
            <tr
              key={key}
              className={cn(
                'hover:bg-surface-secondary',
                clickable && 'cursor-pointer',
                selected && 'bg-accent-light',
                rowClassName?.(item),
              )}
              role={clickable ? 'button' : undefined}
              tabIndex={clickable ? 0 : undefined}
              aria-selected={selectedKey !== undefined ? selected : undefined}
              onClick={clickable ? () => onRowClick?.(item) : undefined}
              onKeyDown={clickable ? handleKeyDown : undefined}
            >
              {columns.map((column) => (
                <td
                  key={column.key}
                  className={cn('px-4 py-2.5', column.align === 'right' ? 'text-right' : 'text-left')}
                >
                  {column.render(item)}
                </td>
              ))}
              {hasActions && (
                <td
                  className="px-4 py-2.5 text-right whitespace-nowrap"
                  onClick={(event) => event.stopPropagation()}
                  onKeyDown={(event) => event.stopPropagation()}
                >
                  <div className="inline-flex items-center gap-1">{rowActions?.(item)}</div>
                </td>
              )}
            </tr>
          )
        })}
      </tbody>
    </table>
  )
}

DataTable.displayName = 'DataTable'
