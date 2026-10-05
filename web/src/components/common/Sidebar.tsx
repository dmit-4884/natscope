import { useCallback } from 'react'
import { Link, useNavigate, useParams, useLocation } from 'react-router-dom'
import { useQueryClient } from '@tanstack/react-query'
import { CONNECTION_QUERY_PREFIX } from '@/hooks/useConnectionQuery'
import { usePreferencesStore } from '@/stores/preferencesStore'
import { PlusIcon, ServicesIcon, SignalIcon, SwitchHorizontalIcon, UsersIcon } from '@/components/ui'
import { useSubscribeStatus, type SubscribeStatus } from '../subscribe/subscribeSession'
import StreamList from '../streams/StreamList'
import KVList from '../kv/KVList'
import ObjectList from '../objects/ObjectList'
import CollapsibleSection from './CollapsibleSection'
import Tooltip from './Tooltip'

function CreateLink({ to, label }: { to: string; label: string }) {
  return (
    <Tooltip content={label}>
      <Link
        to={to}
        aria-label={label}
        className="flex items-center justify-center px-2 py-2 text-content-muted hover:text-accent hover:bg-accent-light transition-colors"
      >
        <PlusIcon className="w-4 h-4" />
      </Link>
    </Tooltip>
  )
}

interface SidebarProps {
  connectionId: string
}

interface PageEntry {
  to: string
  label: string
  icon: React.ReactNode
}

const PAGE_ENTRIES: PageEntry[] = [
  { to: '/consumers', label: 'Consumers', icon: <UsersIcon /> },
  { to: '/request', label: 'Request / Reply', icon: <SwitchHorizontalIcon /> },
  { to: '/subscribe', label: 'Subscribe', icon: <SignalIcon /> },
  { to: '/services', label: 'Services', icon: <ServicesIcon /> },
]

type Listening = Extract<SubscribeStatus, 'live' | 'connecting'> | null

const LISTENING_TEXT: Record<NonNullable<Listening>, string> = {
  live: 'Subscription is running',
  connecting: 'Subscription is reconnecting',
}

function ListeningDot({ state, className }: { state: NonNullable<Listening>; className?: string }) {
  const color = state === 'live' ? 'bg-status-success-border' : 'bg-status-warning-border'
  return <span aria-hidden="true" className={`w-1.5 h-1.5 rounded-full animate-pulse ${color} ${className ?? ''}`} />
}

interface PageEntryProps {
  entry: PageEntry
  active: boolean
  listening: Listening
}

function RailButton({ entry, active, listening, onClick }: PageEntryProps & { onClick: () => void }) {
  return (
    <Tooltip content={listening ? `${entry.label} · ${LISTENING_TEXT[listening]}` : entry.label}>
      <button
        type="button"
        onClick={onClick}
        className={`relative p-1.5 rounded transition-colors ${
          active ? 'text-accent bg-accent-muted/70' : 'text-content-tertiary hover:text-content-primary hover:bg-surface-hover/60'
        }`}
        aria-label={listening ? `${entry.label}, ${LISTENING_TEXT[listening].toLowerCase()}` : entry.label}
        aria-current={active ? 'page' : undefined}
      >
        {entry.icon}
        {listening && <ListeningDot state={listening} className="absolute top-1 right-1" />}
      </button>
    </Tooltip>
  )
}

function PageLink({ entry, active, listening }: PageEntryProps) {
  return (
    <Link
      to={entry.to}
      aria-current={active ? 'page' : undefined}
      className={`flex items-center gap-2 px-3 py-2 border-b border-border transition-colors ${
        active ? 'bg-accent-light text-accent-text' : 'bg-surface-secondary text-gray-700 hover:bg-surface-tertiary'
      }`}
    >
      {entry.icon}
      <span className="text-xs font-semibold uppercase tracking-wide">{entry.label}</span>
      {listening && (
        <Tooltip content={LISTENING_TEXT[listening]}>
          <span className="ml-auto flex items-center p-1" data-testid="subscribe-listening" data-state={listening}>
            <ListeningDot state={listening} />
            <span className="sr-only">{LISTENING_TEXT[listening]}</span>
          </span>
        </Tooltip>
      )}
    </Link>
  )
}

export const SIDEBAR_PANEL_ID = 'main-sidebar'

const STREAMS_ICON = (
  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
  </svg>
)
const KV_ICON = (
  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7M4 7c0 2.21 3.582 4 8 4s8-1.79 8-4M4 7c0-2.21 3.582-4 8-4s8 1.79 8 4" />
  </svg>
)
const OBJECTS_ICON = (
  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 8h14M5 8a2 2 0 110-4h14a2 2 0 110 4M5 8v10a2 2 0 002 2h10a2 2 0 002-2V8m-9 4h4" />
  </svg>
)

export default function Sidebar({ connectionId }: SidebarProps) {
  const { streamName } = useParams()
  const location = useLocation()
  const navigate = useNavigate()
  const queryClient = useQueryClient()

  // Collapse toggled from CompactHeader; persisted in preferences so it
  // survives reloads.
  const collapsed = usePreferencesStore((s) => s.isPanelCollapsed(SIDEBAR_PANEL_ID))

  const invalidateStreams = useCallback(() => {
    queryClient.invalidateQueries({
      queryKey: [CONNECTION_QUERY_PREFIX, connectionId, 'streams'],
    })
  }, [queryClient, connectionId])

  const invalidateObjects = useCallback(() => {
    queryClient.invalidateQueries({
      queryKey: [CONNECTION_QUERY_PREFIX, connectionId, 'objects', 'buckets'],
    })
  }, [queryClient, connectionId])

  // Determine active section based on URL
  const isStreams = location.pathname.includes('/streams/') && !!streamName
  const isKV = location.pathname.includes('/kv')
  const isObjects = location.pathname.includes('/objects')
  const isPage = (entry: PageEntry) => location.pathname.startsWith(entry.to)
  const subscribeStatus = useSubscribeStatus()
  const isListening = (entry: PageEntry): Listening =>
    entry.to === '/subscribe' && (subscribeStatus === 'live' || subscribeStatus === 'connecting') ? subscribeStatus : null

  // Mini-rail mode: narrow strip of section icons that navigate on click.
  if (collapsed) {
    return (
      <nav
        className="w-10 bg-surface-secondary border-r border-border flex flex-col items-center py-1.5 gap-0.5 shrink-0"
        role="navigation"
        aria-label="Sections"
      >
        <Tooltip content="Streams">
          <button
            type="button"
            onClick={() => navigate('/streams')}
            className={`p-1.5 rounded transition-colors ${
              isStreams
                ? 'text-accent bg-accent-muted/70'
                : 'text-content-tertiary hover:text-content-primary hover:bg-surface-hover/60'
            }`}
            aria-label="Streams"
          >
            {STREAMS_ICON}
          </button>
        </Tooltip>
        <Tooltip content="KV Stores">
          <button
            type="button"
            onClick={() => navigate('/kv')}
            className={`p-1.5 rounded transition-colors ${
              isKV
                ? 'text-accent bg-accent-muted/70'
                : 'text-content-tertiary hover:text-content-primary hover:bg-surface-hover/60'
            }`}
            aria-label="KV Stores"
          >
            {KV_ICON}
          </button>
        </Tooltip>
        <Tooltip content="Object Store">
          <button
            type="button"
            onClick={() => navigate('/objects')}
            className={`p-1.5 rounded transition-colors ${
              isObjects
                ? 'text-accent bg-accent-muted/70'
                : 'text-content-tertiary hover:text-content-primary hover:bg-surface-hover/60'
            }`}
            aria-label="Object Store"
          >
            {OBJECTS_ICON}
          </button>
        </Tooltip>
        {PAGE_ENTRIES.map((entry) => (
          <RailButton
            key={entry.to}
            entry={entry}
            active={isPage(entry)}
            listening={isListening(entry)}
            onClick={() => navigate(entry.to)}
          />
        ))}
      </nav>
    )
  }

  return (
    <nav
      className="w-64 bg-surface-primary border-r flex flex-col overflow-hidden shrink-0"
      role="navigation"
      aria-label="Streams navigation"
    >
      <div className="flex-1 overflow-auto scrollbar-thin">
        <CollapsibleSection
          title="Streams"
          icon={STREAMS_ICON}
          defaultOpen={true}
          isActive={isStreams}
          onOpen={invalidateStreams}
          actions={<CreateLink to="/streams/new" label="Create stream" />}
        >
          <StreamList connectionId={connectionId} />
        </CollapsibleSection>

        <CollapsibleSection
          title="KV Stores"
          icon={KV_ICON}
          defaultOpen={false}
          isActive={isKV}
          onOpen={invalidateStreams}
          actions={<CreateLink to="/kv/new" label="Create KV bucket" />}
        >
          <KVList connectionId={connectionId} />
        </CollapsibleSection>

        <CollapsibleSection
          title="Object Store"
          icon={OBJECTS_ICON}
          defaultOpen={false}
          isActive={isObjects}
          onOpen={invalidateObjects}
          actions={<CreateLink to="/objects/new" label="Create object bucket" />}
        >
          <ObjectList connectionId={connectionId} />
        </CollapsibleSection>

        {PAGE_ENTRIES.map((entry) => (
          <PageLink key={entry.to} entry={entry} active={isPage(entry)} listening={isListening(entry)} />
        ))}
      </div>
    </nav>
  )
}
