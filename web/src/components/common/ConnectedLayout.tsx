import { useEffect, useState, useCallback, useRef, useMemo } from 'react'
import { z } from 'zod'
import { useLocation, useNavigate, Outlet } from 'react-router-dom'
import { useQueryClient } from '@tanstack/react-query'
import {
  ACTIVE_CONNECTION_INFO_KEY,
  clearActiveConnection,
  getActiveConnectionId,
  setActiveConnectionId,
  useConnectionValidation,
} from '@/contexts/connection'
import type { SavedConnection } from '@/api/connections'
import { CONNECTION_QUERY_PREFIX } from '@/hooks/useConnectionQuery'
import { resetAllStores } from '@/stores/resetAllStores'
import { logger } from '@/utils/logger'
import { safeGetItem, safeSetItem } from '@/utils/safeStorage'
import {
  rememberSettingsReturn,
  isSettingsTab,
  SETTINGS_LAST_TAB_KEY,
  type SettingsTab,
} from '@/components/settings/page/settingsNav'
import CompactHeader from './CompactHeader'
import Sidebar from './Sidebar'

function getLastSettingsTab(): SettingsTab {
  const v = safeGetItem(SETTINGS_LAST_TAB_KEY)
  return isSettingsTab(v) ? v : 'connections'
}

/** Minimal connection info stored in localStorage for instant header display */
const storedConnectionInfoSchema = z.object({
  id: z.string(),
  name: z.string(),
  urls: z.array(z.string()),
})
type StoredConnectionInfo = z.infer<typeof storedConnectionInfoSchema>

function getStoredConnectionInfo(): StoredConnectionInfo | null {
  const raw = safeGetItem(ACTIVE_CONNECTION_INFO_KEY)
  if (!raw) return null
  try {
    const parsed = storedConnectionInfoSchema.safeParse(JSON.parse(raw))
    return parsed.success ? parsed.data : null
  } catch {
    return null
  }
}

function storeConnectionInfo(connection: SavedConnection) {
  safeSetItem(ACTIVE_CONNECTION_INFO_KEY, JSON.stringify({
    id: connection.id,
    name: connection.name,
    urls: connection.urls,
  }))
}

export type ConnectionSummary = Pick<SavedConnection, 'id' | 'name' | 'urls'>

export interface ConnectionOutletContext {
  connectionId: string
  currentConnection: ConnectionSummary | null
  handleOpenMappings: (subjectPattern?: string) => void
  handleSwitchConnection: (connection: SavedConnection) => void
  handleDisconnect: () => Promise<void> | void
}

export default function ConnectedLayout() {
  const navigate = useNavigate()
  const queryClient = useQueryClient()

  const [connectionLost, setConnectionLost] = useState(false)
  const location = useLocation()

  // Initialize currentConnection from localStorage for instant display
  const [currentConnection, setCurrentConnection] = useState<ConnectionSummary | null>(() => {
    const stored = getStoredConnectionInfo()
    return stored ? { id: stored.id, name: stored.name, urls: stored.urls } : null
  })

  // connectionId from localStorage (savedConnection.Id)
  const [connectionId, setConnectionId] = useState<string | null>(() =>
    getActiveConnectionId()
  )

  const connectionLostTimeoutRef = useRef<ReturnType<typeof setTimeout>>()

  useEffect(() => {
    return () => {
      if (connectionLostTimeoutRef.current) clearTimeout(connectionLostTimeoutRef.current)
    }
  }, [])

  // If no connectionId, redirect to connection selector
  useEffect(() => {
    if (!connectionId) {
      navigate('/', { replace: true })
    }
  }, [connectionId, navigate])

  const handleDisconnect = useCallback(async () => {
    setCurrentConnection(null)
    setConnectionId(null)
    clearActiveConnection()
    resetAllStores()
    // Every connection-scoped query lives under [CONNECTION_QUERY_PREFIX, ...],
    // so cancel+removeQueries on that prefix drops them all; global list stays.
    const filter = { queryKey: [CONNECTION_QUERY_PREFIX] }
    await queryClient.cancelQueries(filter)
    queryClient.removeQueries(filter)
    navigate('/', { replace: true })
  }, [queryClient, navigate])

  // Handle invalid connection (lazy connect failed on backend)
  const handleInvalidConnection = useCallback(() => {
    logger.warn('Connection is no longer valid, disconnecting...')
    setConnectionLost(true)
    handleDisconnect()
    if (connectionLostTimeoutRef.current) clearTimeout(connectionLostTimeoutRef.current)
    connectionLostTimeoutRef.current = setTimeout(() => setConnectionLost(false), 3000)
  }, [handleDisconnect])

  useConnectionValidation(connectionId, handleInvalidConnection)

  const handleSwitchConnection = useCallback(async (connection: SavedConnection) => {
    // Cancel + drop every connection-scoped query in one call — any hook
    // built on useConnectionQuery is auto-caught via CONNECTION_QUERY_PREFIX.
    const filter = { queryKey: [CONNECTION_QUERY_PREFIX] }
    await queryClient.cancelQueries(filter)
    queryClient.removeQueries(filter)
    resetAllStores()

    setActiveConnectionId(connection.id)
    storeConnectionInfo(connection)
    setConnectionId(connection.id)
    setCurrentConnection(connection)
    setConnectionLost(false)

    // Always navigate to streams list to avoid stale stream from previous connection
    navigate('/streams')
  }, [queryClient, navigate])

  // When CompactHeader resolves full connection data from lazy-loaded connections
  const handleConnectionResolved = useCallback((connection: SavedConnection) => {
    setCurrentConnection(connection)
    storeConnectionInfo(connection)
  }, [])

  // Remembers where to return via location, plus the last-visited settings
  // tab (localStorage) so reopening drops the user back where they were.
  const openSettings = useCallback((tab?: SettingsTab) => {
    const target = tab ?? getLastSettingsTab()
    rememberSettingsReturn(location.pathname + location.search)
    navigate(`/settings/${target}`)
  }, [location.pathname, location.search, navigate])

  const handleOpenMappings = useCallback((subjectPattern?: string) => {
    rememberSettingsReturn(location.pathname + location.search)
    const query = subjectPattern ? `?subject=${encodeURIComponent(subjectPattern)}` : ''
    navigate(`/settings/mappings${query}`)
  }, [location.pathname, location.search, navigate])

  const outletContext = useMemo<ConnectionOutletContext>(() => ({
    connectionId: connectionId || '',
    currentConnection,
    handleOpenMappings,
    handleSwitchConnection,
    handleDisconnect,
  }), [connectionId, currentConnection, handleOpenMappings, handleSwitchConnection, handleDisconnect])

  if (!connectionId) {
    return null
  }

  return (
    <>
      {/* Connection Lost Notification */}
      {connectionLost && (
        <div className="fixed top-4 right-4 z-50 bg-red-500 text-content-inverse px-4 py-2 rounded-lg shadow-lg text-sm">
          Connection lost - please reconnect
        </div>
      )}

      {/* Compact Header */}
      <CompactHeader
        connectionId={connectionId}
        currentConnection={currentConnection}
        onSwitchConnection={handleSwitchConnection}
        onDisconnect={handleDisconnect}
        onOpenConnections={() => openSettings()}
        onConnectionResolved={handleConnectionResolved}
      />

      {/* Main Content Area */}
      <div className="flex-1 flex overflow-hidden">
        {!location.pathname.startsWith('/settings') && <Sidebar connectionId={connectionId} />}

        {/* Main Content */}
        <Outlet context={outletContext} />
      </div>
    </>
  )
}
