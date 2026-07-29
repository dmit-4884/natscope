import { useEffect } from 'react'
import { NavLink, Outlet, useLocation, useNavigate, useOutletContext } from 'react-router-dom'
import type { ConnectionOutletContext } from '@/components/common/ConnectedLayout'
import { CloseIcon, BoltIcon, DocumentIcon, CopyIcon } from '@/components/ui'
import Tooltip from '@/components/common/Tooltip'
import { safeSetItem } from '@/utils/safeStorage'
import { readSettingsReturn, clearSettingsReturn, SETTINGS_LAST_TAB_KEY } from './settingsNav'

const TAB_PATTERN = /^\/settings\/([^/]+)/

function tabFromPath(pathname: string): string | null {
  const m = pathname.match(TAB_PATTERN)
  return m ? m[1] : null
}

interface NavItem {
  to: string
  label: string
  icon: React.ReactNode
}

function MappingsIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" stroke="currentColor" viewBox="0 0 24 24">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.75} d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1" />
    </svg>
  )
}

function PreferencesIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" stroke="currentColor" viewBox="0 0 24 24">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.75} d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.066 2.573c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.573 1.066c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.066-2.573c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.75} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
    </svg>
  )
}

function WorkspaceIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" stroke="currentColor" viewBox="0 0 24 24">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.75} d="M4 7v10a2 2 0 002 2h12a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H6a2 2 0 00-2 2z" />
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.75} d="M12 11v5m0 0l-2-2m2 2l2-2" />
    </svg>
  )
}

export default function SettingsLayout() {
  const location = useLocation()
  const navigate = useNavigate()
  // React Router does not propagate outlet context through a nested `<Outlet>`
  // automatically — forward it explicitly or child pages lose `handleSwitchConnection`.
  const parentCtx = useOutletContext<ConnectionOutletContext | undefined>()

  // Persist the last visited tab so reopening Settings drops the user where
  // they left off, regardless of how they got here (button or direct URL).
  useEffect(() => {
    const tab = tabFromPath(location.pathname)
    if (tab) {
      safeSetItem(SETTINGS_LAST_TAB_KEY, tab)
    }
  }, [location.pathname])

  const handleClose = () => {
    const returnTo = readSettingsReturn()
    clearSettingsReturn()
    navigate(returnTo && !returnTo.startsWith('/settings') ? returnTo : '/streams')
  }

  const items: NavItem[] = [
    {
      to: '/settings/connections',
      label: 'Connections',
      icon: <BoltIcon className="w-4 h-4 shrink-0" />,
    },
    {
      to: '/settings/proto',
      label: 'Proto Files',
      icon: <DocumentIcon className="w-4 h-4 shrink-0" />,
    },
    {
      to: '/settings/mappings',
      label: 'Mappings',
      icon: <MappingsIcon className="w-4 h-4 shrink-0" />,
    },
    {
      to: '/settings/templates',
      label: 'Templates',
      icon: <CopyIcon className="w-4 h-4 shrink-0" />,
    },
    {
      to: '/settings/preferences',
      label: 'Preferences',
      icon: <PreferencesIcon className="w-4 h-4 shrink-0" />,
    },
    {
      to: '/settings/workspace',
      label: 'Workspace',
      icon: <WorkspaceIcon className="w-4 h-4 shrink-0" />,
    },
  ]

  return (
    <div className="flex-1 flex flex-col overflow-hidden bg-surface-secondary">
      {/* Top tab bar — scrolls horizontally on narrow screens instead of wrapping */}
      <nav aria-label="Settings" className="shrink-0 bg-surface-primary border-b border-border px-2 sm:px-4">
        <div className="flex items-center gap-1 overflow-x-auto overflow-y-hidden">
          <Tooltip content="Close settings">
            <button
              type="button"
              onClick={handleClose}
              className="mr-1 p-1.5 shrink-0 text-content-muted hover:text-gray-700 hover:bg-surface-tertiary rounded transition-colors"
              aria-label="Close settings"
            >
              <CloseIcon className="w-4 h-4" />
            </button>
          </Tooltip>
          <div className="h-5 w-px bg-surface-hover mr-1 shrink-0" />
          {items.map((item) => (
            <NavLink
              key={item.to}
              to={item.to}
              end={false}
              className={({ isActive }) =>
                `group relative flex items-center gap-2 px-3 h-11 shrink-0 whitespace-nowrap text-sm font-medium transition-colors border-b-2 -mb-px ${
                  isActive
                    ? 'border-accent text-accent-text'
                    : 'border-transparent text-content-secondary hover:text-content-primary hover:border-border-strong'
                }`
              }
            >
              {item.icon}
              <span>{item.label}</span>
            </NavLink>
          ))}
        </div>
      </nav>

      {/* Section content */}
      <main className="flex-1 overflow-hidden flex flex-col">
        <Outlet context={parentCtx} />
      </main>
    </div>
  )
}
