import { useLayoutEffect, useRef, useState, type ReactNode } from 'react'
import { NavLink, useLocation, useNavigate } from 'react-router-dom'
import { useConnectionPolicy } from '@/contexts/connection'
import { ChevronDownIcon, OverflowMenu, RelationsIcon, UsersIcon } from '@/components/ui'
import { cn } from '@/utils/cn'

interface StreamTab {
  path: string
  label: string
  icon: ReactNode
}

const ALL_TABS: StreamTab[] = [
  {
    path: 'messages',
    label: 'Messages',
    icon: (
      <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z" />
      </svg>
    ),
  },
  {
    path: 'config',
    label: 'Config',
    icon: (
      <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
      </svg>
    ),
  },
  { path: 'consumers', label: 'Consumers', icon: <UsersIcon className="w-4 h-4" /> },
  { path: 'relations', label: 'Relations', icon: <RelationsIcon className="w-4 h-4" /> },
  {
    path: 'publish',
    label: 'Publish',
    icon: (
      <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 19l9 2-9-18-9 18 9-2zm0 0v-8" />
      </svg>
    ),
  },
]

const READ_ONLY_TABS = ALL_TABS.filter((t) => t.path !== 'publish')

const GAP_PX = 4
const MORE_PX = 36

const tabClass = (active: boolean) =>
  cn(
    'px-4 py-2 text-sm font-medium rounded-t transition-colors flex items-center gap-1.5 whitespace-nowrap',
    active
      ? 'bg-surface-primary text-accent border-t border-x border-border'
      : 'text-content-secondary hover:text-content-primary hover:bg-surface-tertiary',
  )

export default function StreamTabs({ baseUrl }: { baseUrl: string }) {
  const { readOnly } = useConnectionPolicy()
  const tabs = readOnly ? READ_ONLY_TABS : ALL_TABS
  const rowRef = useRef<HTMLDivElement>(null)
  const measureRefs = useRef<(HTMLSpanElement | null)[]>([])
  const [visibleCount, setVisibleCount] = useState(tabs.length)
  const location = useLocation()
  const navigate = useNavigate()

  useLayoutEffect(() => {
    const row = rowRef.current
    if (!row) return
    const fit = () => {
      const available = row.clientWidth
      if (available === 0) return
      const widths = measureRefs.current.map((el) => el?.offsetWidth ?? 0)
      const total = widths.reduce((sum, w) => sum + w, 0) + GAP_PX * (widths.length - 1)
      if (total <= available) {
        setVisibleCount(tabs.length)
        return
      }
      let used = MORE_PX
      let count = 0
      while (count < widths.length && used + widths[count] + GAP_PX <= available) {
        used += widths[count] + GAP_PX
        count++
      }
      setVisibleCount(count)
    }
    fit()
    if (typeof ResizeObserver === 'undefined') return
    const observer = new ResizeObserver(fit)
    observer.observe(row)
    return () => observer.disconnect()
  }, [tabs])

  const activeIndex = tabs.findIndex((t) => location.pathname.endsWith(`/${t.path}`))
  const hidden = tabs.slice(visibleCount)

  return (
    <div className="relative">
      <div ref={rowRef} data-testid="stream-tabs" className="flex gap-1">
        {tabs.slice(0, visibleCount).map((t) => (
          <NavLink
            key={t.path}
            to={`${baseUrl}/${t.path}`}
            end={t.path === 'messages'}
            className={({ isActive }) => tabClass(isActive)}
          >
            {t.icon}
            {t.label}
          </NavLink>
        ))}
        {hidden.length > 0 && (
          <OverflowMenu
            label="More tabs"
            className="self-center"
            icon={<ChevronDownIcon className="w-4 h-4" />}
            highlighted={activeIndex >= visibleCount}
            items={hidden.map((t) => ({
              label: t.label,
              icon: t.icon,
              selected: tabs.indexOf(t) === activeIndex,
              onSelect: () => navigate(`${baseUrl}/${t.path}`),
            }))}
          />
        )}
      </div>
      <div aria-hidden="true" className="invisible absolute left-0 top-0 h-0 w-0 overflow-hidden">
        {tabs.map((t, i) => (
          <span
            key={t.path}
            ref={(el) => {
              measureRefs.current[i] = el
            }}
            data-tab={t.path}
            className={cn(tabClass(true), 'w-max')}
          >
            {t.icon}
            {t.label}
          </span>
        ))}
      </div>
    </div>
  )
}
