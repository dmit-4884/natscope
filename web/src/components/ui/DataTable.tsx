import { type KeyboardEvent, type MouseEvent, type ReactNode } from 'react'
import Tooltip from '@/components/common/Tooltip'
import { isPlainClick } from '@/utils/clicks'
import { cn } from '@/utils/cn'
import { ChevronDownIcon, ChevronUpIcon } from './icons'
import { QueryErrorState } from './QueryErrorState'
import { SkeletonRows } from './Skeleton'

export interface DataTableColumn<T> {
  key: string
  header: ReactNode
  hint?: string
  sortable?: boolean
  width?: string
  align?: 'left' | 'right'
  render: (item: T) => ReactNode
}

export interface DataTableSort {
  key: string
  direction: 'asc' | 'desc'
}

export interface DataTableProps<T> {
  columns: DataTableColumn<T>[]
  items: T[]
  rowKey: (item: T) => string
  sort?: DataTableSort
  onSortChange?: (key: string) => void
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

function ColumnHeader<T>({
  column,
  sort,
  onSortChange,
}: {
  column: DataTableColumn<T>
  sort?: DataTableSort
  onSortChange?: (key: string) => void
}) {
  const label = column.hint ? (
    <span className="underline decoration-dotted underline-offset-2">{column.header}</span>
  ) : (
    column.header
  )

  if (!column.sortable || !onSortChange) {
    if (!column.hint) return <>{label}</>
    return (
      <Tooltip content={column.hint}>
        <span className="cursor-help" tabIndex={0}>
          {label}
        </span>
      </Tooltip>
    )
  }

  const active = sort?.key === column.key
  const Arrow = active && sort.direction === 'asc' ? ChevronUpIcon : ChevronDownIcon
  const button = (
    <button
      type="button"
      onClick={() => onSortChange(column.key)}
      className={cn(
        'group inline-flex items-center gap-1 rounded uppercase tracking-wide font-medium hover:text-content-primary',
        'focus:outline-none focus-visible:ring-2 focus-visible:ring-border-focus',
        active && 'text-content-primary',
      )}
    >
      {label}
      <span aria-hidden="true" className={cn(active ? 'visible' : 'invisible group-hover:visible text-content-muted')}>
        <Arrow className="w-3 h-3" />
      </span>
    </button>
  )
  return column.hint ? <Tooltip content={column.hint}>{button}</Tooltip> : button
}

function ariaSort(column: { key: string; sortable?: boolean }, sort?: DataTableSort) {
  if (!column.sortable || sort?.key !== column.key) return undefined
  return sort.direction === 'asc' ? 'ascending' : 'descending'
}

export function DataTable<T>({
  columns,
  items,
  rowKey,
  sort,
  onSortChange,
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
              aria-sort={ariaSort(column, sort)}
              className={cn('px-4 py-2 font-medium', column.align === 'right' ? 'text-right' : 'text-left')}
            >
              <ColumnHeader column={column} sort={sort} onSortChange={onSortChange} />
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
