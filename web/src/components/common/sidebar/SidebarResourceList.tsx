import { useEffect, useId, useMemo, useRef, useState, type DragEvent, type KeyboardEvent, type ReactNode } from 'react'
import { Link } from 'react-router-dom'
import { useSidebarLayout, useUpdateSidebarLayout, type SectionLayout, type SidebarSection } from '@/contexts/connection'
import { DragHandleIcon, OverflowMenu, RefreshIcon, SearchInput, StarIcon, StarSolidIcon } from '@/components/ui'
import { useSidebarUiStore } from '@/stores/sidebarUiStore'
import { cn } from '@/utils/cn'
import { formatCount } from '@/utils/formatters'
import Tooltip from '../Tooltip'
import { arrangeSection, dropIndex, moveItem, togglePin } from './sectionArrangement'

const INITIAL_COUNT = 12
const PAGE_SIZE = 50

type Group = 'pinned' | 'rest'

interface DragState {
  group: Group
  from: number
  over: number | null
  after: boolean
}

interface SidebarResourceListProps {
  connectionId: string
  section: SidebarSection
  names: string[]
  selectedName?: string
  hrefFor: (name: string) => string
  noun: string
  isRefreshing: boolean
  onRefresh: () => void
  renderBadge?: (name: string) => ReactNode
}

const capitalize = (s: string) => s.charAt(0).toUpperCase() + s.slice(1)

export function SidebarResourceList({
  connectionId,
  section,
  names,
  selectedName,
  hrefFor,
  noun,
  isRefreshing,
  onRefresh,
  renderBadge,
}: SidebarResourceListProps) {
  const layout = useSidebarLayout(connectionId)[section]
  const { mutate: updateLayout } = useUpdateSidebarLayout(connectionId)
  const query = useSidebarUiStore((s) => s.filters[section] ?? '')
  const storeFilter = useSidebarUiStore((s) => s.setFilter)
  const setQuery = (next: string) => storeFilter(section, next)
  const [paging, setPaging] = useState({ query: '', limit: INITIAL_COUNT })
  const [drag, setDrag] = useState<DragState | null>(null)
  const [announcement, setAnnouncement] = useState('')
  const pendingFocus = useRef<string | null>(null)
  const containerRef = useRef<HTMLDivElement>(null)
  const hintId = useId()

  const { pinned, rest } = useMemo(() => arrangeSection(names, layout, query), [names, layout, query])
  const searching = query.trim() !== ''
  const limit = paging.query === query ? paging.limit : INITIAL_COUNT
  const selectedIndex = selectedName ? rest.indexOf(selectedName) : -1
  const visibleRest = rest.slice(0, Math.max(limit, selectedIndex + 1))
  const hiddenCount = rest.length - visibleRest.length
  const plural = `${noun}s`

  useEffect(() => {
    const name = pendingFocus.current
    if (!name || !containerRef.current) return
    pendingFocus.current = null
    const links = containerRef.current.querySelectorAll<HTMLAnchorElement>('a[data-name]')
    Array.from(links).find((link) => link.dataset.name === name)?.focus()
  }, [pinned, rest])

  const save = (next: SectionLayout) => updateLayout({ [section]: next })

  const move = (group: Group, from: number, to: number) => {
    const items = group === 'pinned' ? pinned : rest
    if (searching || from === to || to < 0 || to >= items.length) return
    const moved = moveItem(items, from, to)
    save(group === 'pinned' ? { pinned: moved, order: layout.order } : { pinned: layout.pinned, order: moved })
    if (group === 'rest' && to >= visibleRest.length) setPaging({ query, limit: to + 1 })
    setAnnouncement(`${items[from]} moved to position ${to + 1} of ${items.length}`)
  }

  const handleKeyDown = (e: KeyboardEvent, group: Group, index: number, name: string) => {
    if (!e.altKey || (e.key !== 'ArrowUp' && e.key !== 'ArrowDown')) return
    e.preventDefault()
    pendingFocus.current = name
    move(group, index, index + (e.key === 'ArrowUp' ? -1 : 1))
  }

  const dragProps = (group: Group, index: number, name: string) => {
    if (searching) return {}
    return {
      draggable: true,
      onDragStart: (e: DragEvent<HTMLLIElement>) => {
        e.dataTransfer.effectAllowed = 'move'
        e.dataTransfer.setData('text/plain', name)
        setDrag({ group, from: index, over: null, after: false })
      },
      onDragOver: (e: DragEvent<HTMLLIElement>) => {
        if (!drag || drag.group !== group) return
        e.preventDefault()
        e.dataTransfer.dropEffect = 'move'
        const rect = e.currentTarget.getBoundingClientRect()
        const after = e.clientY > rect.top + rect.height / 2
        if (drag.over !== index || drag.after !== after) setDrag({ ...drag, over: index, after })
      },
      onDrop: (e: DragEvent<HTMLLIElement>) => {
        if (!drag || drag.group !== group || drag.over === null) return
        e.preventDefault()
        move(group, drag.from, dropIndex(drag.from, drag.over, drag.after))
        setDrag(null)
      },
      onDragEnd: () => setDrag(null),
    }
  }

  const renderRow = (name: string, group: Group, index: number) => {
    const isPinned = group === 'pinned'
    const isSelected = name === selectedName
    const isDragged = drag?.group === group && drag.from === index
    const marker =
      drag && drag.group === group && drag.over === index && drag.from !== index ? (drag.after ? 'after' : 'before') : null

    return (
      <li
        key={name}
        {...dragProps(group, index, name)}
        className={cn(
          'group relative flex items-center border-l-2 transition-colors',
          isSelected ? 'bg-accent-light border-l-blue-500' : 'border-l-transparent hover:bg-surface-secondary',
          isDragged && 'opacity-50',
        )}
        data-testid="sidebar-item"
      >
        {marker && (
          <span
            aria-hidden="true"
            className={cn(
              'pointer-events-none absolute inset-x-2 h-0.5 rounded-full bg-accent',
              marker === 'before' ? '-top-px' : '-bottom-px',
            )}
          />
        )}
        {!searching && (
          <span className="absolute left-0.5 text-content-muted opacity-0 group-hover:opacity-100 cursor-grab">
            <DragHandleIcon className="w-3 h-3" />
          </span>
        )}
        <Link
          to={hrefFor(name)}
          draggable={false}
          data-name={name}
          title={name}
          aria-current={isSelected ? 'page' : undefined}
          aria-describedby={searching ? undefined : hintId}
          onKeyDown={(e) => handleKeyDown(e, group, index, name)}
          className={cn(
            'flex-1 min-w-0 flex items-center gap-2 pl-3 py-2 text-sm',
            isSelected ? 'font-medium text-content-primary' : 'text-gray-700',
          )}
        >
          <span className="truncate">{name}</span>
          {renderBadge?.(name)}
        </Link>
        <Tooltip content={isPinned ? 'Unpin' : 'Pin to top'}>
          <button
            type="button"
            onClick={() => save(togglePin(layout, name))}
            aria-pressed={isPinned}
            aria-label={isPinned ? `Unpin ${name}` : `Pin ${name}`}
            className={cn(
              'mr-1.5 p-1 rounded hover:bg-surface-tertiary focus:opacity-100 transition-opacity',
              isPinned ? 'text-amber-500' : 'text-content-muted opacity-0 group-hover:opacity-100',
            )}
          >
            {isPinned ? <StarSolidIcon className="w-3.5 h-3.5" /> : <StarIcon className="w-3.5 h-3.5" />}
          </button>
        </Tooltip>
      </li>
    )
  }

  return (
    <div ref={containerRef}>
      <div className="flex items-center gap-1 px-2 py-1.5 border-b border-border bg-surface-secondary">
        <SearchInput
          value={query}
          onChange={setQuery}
          placeholder={`Filter ${plural}`}
          size="sm"
          debounce={0}
          resultsCount={searching ? pinned.length + rest.length : undefined}
          className="flex-1 min-w-0"
        />
        {layout.order.length > 0 && !searching && (
          <OverflowMenu
            label={`${capitalize(noun)} list options`}
            items={[{ label: 'Reset to A–Z order', onSelect: () => save({ pinned: layout.pinned, order: [] }) }]}
          />
        )}
        <Tooltip content={`Refresh ${plural}`}>
          <button
            type="button"
            onClick={onRefresh}
            disabled={isRefreshing}
            aria-label={`Refresh ${plural}`}
            className="p-1.5 rounded text-content-muted hover:text-content-secondary hover:bg-surface-tertiary transition-colors disabled:opacity-50"
          >
            <RefreshIcon className={cn('w-3.5 h-3.5', isRefreshing && 'animate-spin')} />
          </button>
        </Tooltip>
      </div>

      <span id={hintId} className="sr-only">
        Drag, or press Alt with the up or down arrow, to reorder
      </span>

      {pinned.length > 0 && (
        <>
          <p className="px-3 pt-2 pb-1 text-2xs font-semibold uppercase tracking-wide text-content-muted">Pinned</p>
          <ul aria-label={`Pinned ${plural}`}>{pinned.map((name, i) => renderRow(name, 'pinned', i))}</ul>
          {rest.length > 0 && <div className="mx-3 my-1 border-t border-border" />}
        </>
      )}

      <ul aria-label={capitalize(plural)}>{visibleRest.map((name, i) => renderRow(name, 'rest', i))}</ul>

      {hiddenCount > 0 && (
        <button
          type="button"
          onClick={() => setPaging({ query, limit: visibleRest.length + PAGE_SIZE })}
          className="w-full px-3 py-2 text-left text-xs text-accent hover:bg-surface-secondary"
          data-testid="sidebar-show-more"
        >
          Show {Math.min(PAGE_SIZE, hiddenCount)} more ({formatCount(hiddenCount)} hidden)
        </button>
      )}

      {searching && pinned.length + rest.length === 0 && (
        <p className="px-3 py-4 text-xs text-content-tertiary">
          No {plural} match “{query.trim()}”
        </p>
      )}

      <p role="status" aria-live="polite" className="sr-only">
        {announcement}
      </p>
    </div>
  )
}
