import type { MappingItem, MappingHealthInfo } from '@/contexts/mappings'
import { Button, ClipboardIcon, DataTable, EmptyState, PlusIcon, RowActionButton, WarningIcon, type DataTableColumn } from '@/components/ui'
import { MappingHealthBadge } from './MappingHealthBadge'

const MAPPINGS_ICON = <ClipboardIcon className="w-full h-full" />

function isSpecificSubject(pattern: string): boolean {
  const uuidPattern = /[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}/i
  return uuidPattern.test(pattern)
}

interface Props {
  items: MappingItem[]
  healthById?: Record<string, MappingHealthInfo>
  sourceNamesById?: Record<string, string>
  onEdit?: (item: MappingItem) => void
  onDelete: (id: string) => void
  onAdd?: () => void
  searchFilter: string
}

export function MappingTable({ items, healthById, sourceNamesById, onEdit, onDelete, onAdd, searchFilter }: Props) {
  if (items.length === 0) {
    return (
      <EmptyState
        icon={MAPPINGS_ICON}
        title={searchFilter ? 'No mappings match your search' : 'No mappings configured'}
        description={searchFilter ? undefined : 'Add your first mapping to start decoding messages by subject pattern.'}
        action={
          searchFilter || !onAdd ? undefined : (
            <Button onClick={onAdd} icon={<PlusIcon />}>
              Add mapping
            </Button>
          )
        }
      />
    )
  }

  const columns: DataTableColumn<MappingItem>[] = [
    {
      key: 'pattern',
      header: 'Subject Pattern',
      width: 'w-[28%]',
      render: (item) => (
        <div className="flex items-center gap-2 min-w-0">
          <code
            className="text-xs font-mono text-content-primary bg-surface-tertiary px-2 py-1 rounded truncate"
            title={item.pattern}
          >
            {item.pattern}
          </code>
          {isSpecificSubject(item.pattern) && (
            <span
              className="text-xs text-status-warning-text shrink-0"
              title="This looks like a specific subject, not a wildcard pattern."
            >
              <WarningIcon className="w-3.5 h-3.5" />
            </span>
          )}
        </div>
      ),
    },
    {
      key: 'messageType',
      header: 'Proto Type',
      width: 'w-[34%]',
      render: (item) => (
        <span
          className="block text-xs text-accent-text font-mono truncate"
          title={item.messageType}
        >
          {item.messageType}
        </span>
      ),
    },
    {
      key: 'source',
      header: 'Source',
      width: 'w-[14%]',
      render: (item) => {
        const sourceName = sourceNamesById?.[item.sourceId]
        const sourceDeleted = healthById?.[item.id]?.health === 'source_missing'
        return (
          <span
            className={`inline-flex items-center px-2 py-0.5 text-xs rounded border truncate max-w-full ${
              sourceName
                ? 'bg-slate-50 text-slate-700 border-slate-200'
                : sourceDeleted
                  ? 'bg-rose-50 text-rose-600 border-rose-200 italic'
                  : 'bg-slate-50 text-slate-500 border-slate-200'
            }`}
            title={`Source ID: ${item.sourceId}${sourceDeleted ? ' (deleted)' : ''}`}
          >
            {sourceName ?? (sourceDeleted ? '(deleted source)' : item.sourceId)}
          </span>
        )
      },
    },
    {
      key: 'health',
      header: 'Health',
      width: 'w-[16%]',
      render: (item) => <MappingHealthBadge info={healthById?.[item.id]} />,
    },
  ]

  return (
    <DataTable
      className="min-w-[760px] table-fixed"
      columns={columns}
      items={items}
      rowKey={(item) => item.id}
      rowClassName={(item) => (isSpecificSubject(item.pattern) ? 'bg-status-warning-bg' : '')}
      rowActions={(item) => (
        <>
          {onEdit && (
            <RowActionButton kind="edit" onClick={() => onEdit(item)} label={`Edit mapping ${item.pattern}`} />
          )}
          <RowActionButton kind="delete" onClick={() => onDelete(item.id)} label={`Delete mapping ${item.pattern}`} />
        </>
      )}
    />
  )
}
