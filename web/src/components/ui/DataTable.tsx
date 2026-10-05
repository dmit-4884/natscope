import { type KeyboardEvent, type MouseEvent, type ReactNode } from 'react'
import { isPlainClick } from '@/utils/clicks'
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
  rowHref?: (item: T) => string
  rowActions?: (item: T) => ReactNode
  isLoading?: boolean
  loadingRows?: number
  emptyState?: ReactNode
  error?: unknown
  onRetry?: () => void
  className?: string
  selectedKey?: string
  rowClassName?: (item: T) => string
  rowLabel?: (item: T) => string
}

export function DataTable<T>({
  columns,
  items,
  rowKey,
  onRowClick,
  rowHref,
  rowActions,
  isLoading = false,
  loadingRows = 5,
  emptyState,
  error,
  onRetry,
  className,
  selectedKey,
  rowClassName,
  rowLabel,
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
          const href = rowHref?.(item)
          const rowButton = clickable && href === undefined
          const selected = selectedKey !== undefined && key === selectedKey

          const handleKeyDown = (event: KeyboardEvent<HTMLTableRowElement>) => {
            if (event.key === 'Enter' || event.key === ' ') {
              event.preventDefault()
              onRowClick?.(item)
            }
          }

          const handleClick = (event: MouseEvent<HTMLTableRowElement>) => {
            if (href !== undefined) {
              if (event.target instanceof Element && event.target.closest('a, button, input, select, textarea')) return
              if (!isPlainClick(event)) {
                window.open(href, '_blank', 'noopener')
                return
              }
            }
            onRowClick?.(item)
          }

          const handleAuxClick = (event: MouseEvent<HTMLTableRowElement>) => {
            if (href === undefined || event.button !== 1) return
            if (event.target instanceof Element && event.target.closest('a')) return
            event.preventDefault()
            window.open(href, '_blank', 'noopener')
          }

          const handleLinkClick = (event: MouseEvent<HTMLAnchorElement>) => {
            if (!isPlainClick(event)) return
            event.preventDefault()
            onRowClick?.(item)
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
              role={rowButton ? 'button' : undefined}
              aria-label={rowButton ? rowLabel?.(item) : undefined}
              tabIndex={rowButton ? 0 : undefined}
              aria-selected={selectedKey !== undefined ? selected : undefined}
              onClick={clickable ? handleClick : undefined}
              onAuxClick={href !== undefined ? handleAuxClick : undefined}
              onKeyDown={rowButton ? handleKeyDown : undefined}
            >
              {columns.map((column, index) => (
                <td
                  key={column.key}
                  className={cn('px-4 py-2.5', column.align === 'right' ? 'text-right' : 'text-left')}
                >
                  {href !== undefined && index === 0 ? (
                    <a
                      href={href}
                      aria-label={rowLabel?.(item)}
                      className="block rounded focus:outline-none focus-visible:ring-2 focus-visible:ring-border-focus"
                      onClick={handleLinkClick}
                    >
                      {column.render(item)}
                    </a>
                  ) : (
                    column.render(item)
                  )}
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
