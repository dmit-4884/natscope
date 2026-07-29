import { useState, useRef, useEffect, lazy, Suspense } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import type { SavedConnection } from '@/api/connections'
import { useConnections , useConnectionHealth } from '@/contexts/connection'
import { CONNECTION_QUERY_PREFIX } from '@/hooks/useConnectionQuery'
import { usePreferencesStore } from '@/stores/preferencesStore'
import { BoltIcon, ChevronDownIcon, PlusIcon, RefreshIcon } from '@/components/ui'
import type { ConnectionSummary } from './ConnectedLayout'
import { SIDEBAR_PANEL_ID } from './Sidebar'
import Tooltip from './Tooltip'

const ServerInfo = lazy(() => import('../streams/ServerInfo'))

interface CompactHeaderProps {
  connectionId: string | null
  currentConnection: ConnectionSummary | null
  onSwitchConnection: (connection: SavedConnection) => void
  onDisconnect: () => void
  onOpenConnections: () => void
  onConnectionResolved?: (connection: SavedConnection) => void
}

export default function CompactHeader({
  connectionId,
  currentConnection,
  onSwitchConnection,
  onOpenConnections,
  onConnectionResolved,
}: CompactHeaderProps) {
  const [showConnectionDropdown, setShowConnectionDropdown] = useState(false)
  const [hasOpenedDropdown, setHasOpenedDropdown] = useState(false)
  const [showServerInfo, setShowServerInfo] = useState(false)

  // Connection health
  const { serverVersion, refetch: refetchHealth } = useConnectionHealth(connectionId)
  const queryClient = useQueryClient()
  const [isRefreshing, setIsRefreshing] = useState(false)

  const handleRefresh = async () => {
    setIsRefreshing(true)
    try {
      await Promise.all([
        refetchHealth(),
        // Invalidate every connection-scoped cache under the prefix.
        queryClient.invalidateQueries({
          queryKey: [CONNECTION_QUERY_PREFIX, connectionId],
        }),
        queryClient.invalidateQueries({ queryKey: ['mappings'] }),
      ])
    } finally {
      setIsRefreshing(false)
    }
  }

  // Lazy-loaded on first dropdown open.
  const { data: savedConnections = [], refetch: refetchConnections, isLoading: isLoadingConnections } = useConnections({ enabled: hasOpenedDropdown })

  // Resolve full currentConnection once connections load.
  useEffect(() => {
    if (!connectionId || savedConnections.length === 0 || !onConnectionResolved) return
    const found = savedConnections.find(c => c.id === connectionId)
    if (found) {
      onConnectionResolved(found)
    }
  }, [connectionId, savedConnections, onConnectionResolved])

  const dropdownRef = useRef<HTMLDivElement>(null)

  // Close dropdown when clicking outside
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
        setShowConnectionDropdown(false)
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [])

  const handleConnectToSaved = (connection: SavedConnection) => {
    onSwitchConnection(connection)
    setShowConnectionDropdown(false)
  }


  const sidebarCollapsed = usePreferencesStore((s) => s.isPanelCollapsed(SIDEBAR_PANEL_ID))
  const togglePanelCollapsed = usePreferencesStore((s) => s.togglePanelCollapsed)

  return (
    <header className="bg-surface-primary border-b border-border h-12 flex items-center justify-between px-4 text-sm">
      {/* Left side - Logo and connection */}
      <div className="flex items-center gap-3">
        <Tooltip content={sidebarCollapsed ? 'Show sidebar' : 'Hide sidebar'}>
          <button
            type="button"
            onClick={() => togglePanelCollapsed(SIDEBAR_PANEL_ID)}
            className="p-1.5 text-content-tertiary hover:text-content-primary hover:bg-surface-tertiary rounded transition-colors"
            aria-label={sidebarCollapsed ? 'Show sidebar' : 'Hide sidebar'}
            aria-pressed={!sidebarCollapsed}
          >
            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h16M4 18h16" />
            </svg>
          </button>
        </Tooltip>

        {/* App icon and title */}
        <div className="flex items-center gap-2">
          <div className="w-8 h-8 bg-gradient-to-br from-blue-500 to-blue-600 rounded-lg flex items-center justify-center shadow-sm">
            <BoltIcon className="w-5 h-5 text-content-inverse" />
          </div>
          <span className="font-semibold text-content-primary">Natscope</span>
        </div>

        {/* Divider */}
        <div className="w-px h-6 bg-surface-hover" />

        {/* Connection info / selector */}
        <div className="relative" ref={dropdownRef}>
          <button
            onClick={() => {
              const willOpen = !showConnectionDropdown
              setShowConnectionDropdown(willOpen)
              // Fetch connections when opening dropdown
              if (willOpen) {
                setHasOpenedDropdown(true)
                refetchConnections()
              }
            }}
            className="flex items-center gap-2 px-2 py-1.5 hover:bg-surface-tertiary rounded-md transition-colors max-w-full border border-transparent hover:border-border"
          >
            {/* Status dot */}
            <div
              role="img"
              aria-label={currentConnection ? 'Connected' : 'Not connected'}
              className={`w-2 h-2 rounded-full shrink-0 ${currentConnection ? 'bg-green-500' : 'bg-gray-400'}`}
            />

            {currentConnection ? (
              <>
                <span className="font-medium text-content-primary truncate">{currentConnection.name}</span>
                <span className="text-content-tertiary font-mono text-xs truncate">
                  {currentConnection.urls[0]}
                </span>
              </>
            ) : (
              <span className="text-content-tertiary">Click to connect...</span>
            )}

            {serverVersion && (
              <span className="text-2xs font-medium text-accent-text bg-accent-light border border-blue-200 rounded px-1.5 py-0.5 shrink-0">
                v{serverVersion}
              </span>
            )}

            {/* Dropdown arrow */}
            <ChevronDownIcon className="w-4 h-4 text-content-muted shrink-0" />
          </button>

          {/* Connection dropdown */}
          {showConnectionDropdown && (
            <div className="absolute top-full left-0 mt-1 w-80 bg-surface-primary border border-border rounded-lg shadow-lg z-50">
                  {/* Saved connections list */}
                  <div className="max-h-64 overflow-auto">
                    {isLoadingConnections ? (
                      <div className="px-3 py-4 text-center text-content-tertiary text-sm flex items-center justify-center gap-2">
                        <svg className="animate-spin h-4 w-4 text-blue-500" fill="none" viewBox="0 0 24 24">
                          <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                          <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
                        </svg>
                        Loading connections...
                      </div>
                    ) : savedConnections.length > 0 ? (
                      savedConnections.map((conn) => (
                        <button
                          key={conn.id}
                          onClick={() => handleConnectToSaved(conn)}
                          className={`w-full px-3 py-2.5 text-left hover:bg-surface-secondary flex items-center gap-3 border-b border-gray-100 last:border-0 ${
                            currentConnection?.id === conn.id ? 'bg-accent-light' : ''
                          }`}
                        >
                          <div
                            role="img"
                            aria-label={currentConnection?.id === conn.id ? 'Active connection' : 'Inactive connection'}
                            className={`w-2 h-2 rounded-full shrink-0 ${currentConnection?.id === conn.id ? 'bg-green-500' : 'bg-gray-300'}`}
                          />
                          <div className="flex-1 min-w-0">
                            <div className="font-medium text-content-primary truncate">{conn.name}</div>
                            <div className="text-xs text-content-tertiary font-mono truncate">{conn.urls[0]}</div>
                          </div>
                          {conn.auth?.username && (
                            <span className="text-xs text-content-tertiary px-1.5 py-0.5 bg-surface-tertiary rounded">
                              {conn.auth.username}
                            </span>
                          )}
                        </button>
                      ))
                    ) : (
                      <div className="px-3 py-4 text-center text-content-tertiary text-sm">
                        No saved connections
                      </div>
                    )}
                  </div>

                  {/* New connection / Manage connections */}
                  <div className="p-2 border-t border-border bg-surface-secondary">
                    <button
                      onClick={() => {
                        setShowConnectionDropdown(false)
                        onOpenConnections()
                      }}
                      className="w-full px-3 py-2 text-sm text-accent hover:bg-accent-light rounded-md flex items-center justify-center gap-2 font-medium"
                    >
                      <PlusIcon className="w-4 h-4" />
                      Manage Connections
                    </button>
                  </div>
            </div>
          )}
        </div>

        {currentConnection && (
          <div className="flex items-center gap-0.5">
            {/* Server info */}
            <Tooltip content="Server information">
              <button
                onClick={() => setShowServerInfo(true)}
                className="p-1.5 text-content-muted hover:text-accent hover:bg-accent-light rounded-md transition-colors"
                aria-label="Server information"
              >
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2m-2-4h.01M17 16h.01" />
                </svg>
              </button>
            </Tooltip>
            {/* Refresh */}
            <Tooltip content="Refresh">
              <button
                onClick={handleRefresh}
                disabled={isRefreshing}
                className="p-1.5 text-content-muted hover:text-content-secondary hover:bg-surface-tertiary rounded-md transition-colors"
                aria-label="Refresh"
              >
                <RefreshIcon className={`w-4 h-4 ${isRefreshing ? 'animate-spin' : ''}`} />
              </button>
            </Tooltip>
          </div>
        )}
      </div>

      {/* Right side - Settings */}
      <div className="flex items-center gap-2">
        <Tooltip content="Open settings">
          <button
            onClick={onOpenConnections}
            className="px-3 py-1.5 text-sm font-medium text-gray-700 bg-surface-primary border border-border hover:border-blue-300 hover:bg-accent-light hover:text-accent-text rounded-md transition-colors flex items-center gap-2 shadow-sm"
          >
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
            </svg>
            <span>Settings</span>
          </button>
        </Tooltip>
      </div>
      {/* Server Info Modal */}
      {showServerInfo && connectionId && (
        <Suspense fallback={null}>
          <ServerInfo connectionId={connectionId} onClose={() => setShowServerInfo(false)} />
        </Suspense>
      )}
    </header>
  )
}
