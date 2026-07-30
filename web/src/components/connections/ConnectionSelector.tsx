import { useState, useRef, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { toast } from '@/utils/toast'
import { getErrorMessage, stripErrorCodePrefix } from '@/api/errors'
import { useRowKeys } from '@/hooks/useRowKeys'
import type { SavedConnection } from '@/api/connections'
import { safeSetItem } from '@/utils/safeStorage'
import {
  ACTIVE_CONNECTION_INFO_KEY,
  AuthConfig,
  clearActiveConnection,
  getActiveConnectionId,
  NatsUrl,
  setActiveConnectionId,
  toApiAuthConfig,
  useConnections,
  useCreateConnection,
  useTestConnection,
  type AuthMethod,
} from '@/contexts/connection'
import { BoltIcon, CloseIcon, EyeIcon, EyeOffIcon, Input, PlusIcon, Spinner } from '@/components/ui'
import { AUTH_LABELS } from './manager/connectionFormData'

type AuthMethodTab = AuthMethod

export default function ConnectionSelector() {
  const navigate = useNavigate()
  const { data: savedConnections = [], isLoading: loadingConnections } = useConnections()
  const createConnectionMutation = useCreateConnection()
  const testConnectionMutation = useTestConnection()

  const [showNewForm, setShowNewForm] = useState(false)
  const [connectingId, setConnectingId] = useState<string | null>(null)
  const [error, setError] = useState<{ id: string; message: string } | null>(null)

  // New connection form state
  const [newName, setNewName] = useState('')
  const [nameError, setNameError] = useState<string | undefined>(undefined)
  const [newUrls, setNewUrls] = useState([''])
  const [urlErrors, setUrlErrors] = useState<(string | undefined)[]>([])
  const urlKeys = useRowKeys(newUrls.length)
  const [authMethod, setAuthMethod] = useState<AuthMethodTab>('none')
  const [newUsername, setNewUsername] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [newToken, setNewToken] = useState('')
  const [newNkeySeed, setNewNkeySeed] = useState('')
  const [showPassword, setShowPassword] = useState(false)
  const [showToken, setShowToken] = useState(false)
  const [newCredentials, setNewCredentials] = useState('')

  // Auto-reconnect
  const hasCheckedRef = useRef(false)
  useEffect(() => {
    if (hasCheckedRef.current || loadingConnections) return
    hasCheckedRef.current = true

    const activeConnId = getActiveConnectionId()
    if (!activeConnId) return

    const savedConn = savedConnections.find(c => c.id === activeConnId)
    if (!savedConn) {
      clearActiveConnection()
      return
    }

    navigate('/streams')
  }, [loadingConnections, savedConnections, navigate])

  // Probe the server before entering the app: a dead server used to be
  // discovered only after navigating, which bounced the user back silently.
  const handleConnectToSaved = async (connection: SavedConnection) => {
    setConnectingId(connection.id)
    setError(null)
    try {
      const result = await testConnectionMutation.mutateAsync({
        urls: connection.urls,
        connectionId: connection.id,
      })
      if (!result.success) {
        const message = result.error ? stripErrorCodePrefix(result.error) : 'Connection failed'
        setError({ id: connection.id, message })
        toast.error(`${connection.name}: ${message}`)
        return
      }

      setActiveConnectionId(connection.id)
      safeSetItem(ACTIVE_CONNECTION_INFO_KEY, JSON.stringify({
        id: connection.id, name: connection.name, urls: connection.urls,
      }))
      navigate('/streams')
    } catch (err) {
      // The global MutationCache.onError already toasts; keep the row inline.
      setError({ id: connection.id, message: getErrorMessage(err) })
    } finally {
      setConnectingId(null)
    }
  }

  const buildAuthConfig = (): AuthConfig | undefined => {
    switch (authMethod) {
      case 'userpass':
        if (!newUsername && !newPassword) return undefined
        return AuthConfig.fromTrusted({ method: 'userpass', username: newUsername || undefined, password: newPassword || undefined })
      case 'token':
        if (!newToken) return undefined
        return AuthConfig.fromTrusted({ method: 'token', token: newToken })
      case 'nkey':
        if (!newNkeySeed) return undefined
        return AuthConfig.fromTrusted({ method: 'nkey', nkeySeed: newNkeySeed })
      case 'credentials':
        if (!newCredentials) return undefined
        return AuthConfig.fromTrusted({ method: 'credentials', credentials: newCredentials })
      default:
        return undefined
    }
  }

  const handleTestConnection = async () => {
    if (!validateUrls()) return
    setError(null)
    try {
      const result = await testConnectionMutation.mutateAsync({
        urls: newUrls.filter(Boolean),
        auth: toApiAuthConfig(buildAuthConfig()),
      })
      if (result.success) {
        toast.success(`Connected to ${result.serverName || result.connectedUrl} (v${result.serverVersion}, RTT: ${result.rttMs}ms${result.jetstreamEnabled ? ', JetStream' : ''})`)
      } else {
        const msg = result.error ? stripErrorCodePrefix(result.error) : 'Connection failed'
        toast.error(msg)
        setError({ id: '__test__', message: msg })
      }
    } catch (err) {
      setError({ id: '__test__', message: getErrorMessage(err) || 'Test failed' })
    }
  }

  const validateName = (): boolean => {
    const blankButTyped = newName !== '' && newName.trim() === ''
    setNameError(blankButTyped ? 'Name is required' : undefined)
    return !blankButTyped
  }

  const validateUrls = (): boolean => {
    const nextUrlErrors = newUrls.map((u) => {
      if (!u.trim()) return undefined
      const result = NatsUrl.create(u)
      return result.isErr() ? result.error.message : undefined
    })
    if (!newUrls.some((u) => u.trim() !== '')) {
      nextUrlErrors[0] = 'Add at least one server URL'
    }
    setUrlErrors(nextUrlErrors)
    return nextUrlErrors.every((e) => !e)
  }

  const handleNewConnection = async () => {
    const nameOk = validateName()
    if (!validateUrls() || !nameOk) return
    const urls = newUrls.filter(Boolean)
    setConnectingId('__new__')
    setError(null)
    try {
      const savedConnection = await createConnectionMutation.mutateAsync({
        name: newName.trim() || urls[0],
        urls,
        auth: toApiAuthConfig(buildAuthConfig()),
      })

      setActiveConnectionId(savedConnection.id)
      safeSetItem(ACTIVE_CONNECTION_INFO_KEY, JSON.stringify({
        id: savedConnection.id, name: savedConnection.name, urls: savedConnection.urls,
      }))
      navigate('/streams')
    } catch (err) {
      setError({ id: '__new__', message: getErrorMessage(err) || 'Connection failed' })
    } finally {
      setConnectingId(null)
    }
  }

  const handleAddUrl = () => {
    urlKeys.registerAdd()
    setNewUrls([...newUrls, ''])
  }
  const handleRemoveUrl = (index: number) => {
    urlKeys.registerRemove(index)
    setNewUrls(newUrls.filter((_, i) => i !== index))
  }
  const handleUrlChange = (index: number, value: string) => {
    const updated = [...newUrls]
    updated[index] = value
    setNewUrls(updated)
  }

  const handleFileUpload = (setter: (v: string) => void) => {
    const input = document.createElement('input')
    input.type = 'file'
    input.onchange = (e) => {
      const file = (e.target as HTMLInputElement).files?.[0]
      if (file) {
        const reader = new FileReader()
        reader.onload = () => setter(reader.result as string)
        reader.readAsText(file)
      }
    }
    input.click()
  }

  return (
    <div className="flex-1 flex items-center justify-center bg-surface-secondary">
      <div className="max-w-lg w-full mx-4">
        {/* Header */}
        <div className="text-center mb-8">
          <div className="mx-auto w-16 h-16 bg-gradient-to-br from-blue-500 to-blue-600 rounded-xl flex items-center justify-center shadow-lg mb-4">
            <BoltIcon className="w-10 h-10 text-content-inverse" />
          </div>
          <h1 className="text-2xl font-bold text-content-primary">Natscope</h1>
          <p className="mt-2 text-sm text-content-tertiary">Connect to your NATS server to get started</p>
        </div>

        {!showNewForm ? (
          <>
            {/* Saved Connections */}
            {loadingConnections ? (
              <div className="bg-surface-primary rounded-lg shadow-sm border border-border p-6 text-center text-content-tertiary">
                <svg className="animate-spin h-6 w-6 mx-auto text-blue-500 mb-2" fill="none" viewBox="0 0 24 24">
                  <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                  <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
                </svg>
                Loading connections...
              </div>
            ) : savedConnections.length > 0 ? (
              <div className="bg-surface-primary rounded-lg shadow-sm border border-border overflow-hidden mb-4">
                <div className="px-4 py-3 bg-surface-secondary border-b border-border">
                  <h2 className="text-sm font-semibold text-gray-700">Saved Connections</h2>
                </div>
                <div className="divide-y divide-gray-100">
                  {savedConnections.map((conn) => (
                    <button
                      key={conn.id}
                      onClick={() => void handleConnectToSaved(conn)}
                      disabled={connectingId !== null}
                      aria-busy={connectingId === conn.id}
                      className={`w-full px-4 py-3 text-left hover:bg-surface-secondary flex items-center gap-3 transition-colors disabled:opacity-50 ${
                        error?.id === conn.id ? 'bg-status-error-bg' : ''
                      }`}
                    >
                      <div
                        role="img"
                        aria-label={
                          connectingId === conn.id
                            ? 'Connecting'
                            : error?.id === conn.id
                              ? 'Connection error'
                              : 'Saved connection'
                        }
                        className={`w-2 h-2 rounded-full shrink-0 ${
                          connectingId === conn.id
                            ? 'bg-amber-500 animate-pulse'
                            : error?.id === conn.id
                              ? 'bg-red-500'
                              : 'bg-gray-300'
                        }`}
                      />
                      <div className="flex-1 min-w-0">
                        <div className="font-medium text-content-primary truncate">{conn.name}</div>
                        <div className="text-xs text-content-tertiary font-mono truncate">{conn.urls.join(', ')}</div>
                        {error?.id === conn.id && (
                          <div className="text-xs text-status-error-text mt-0.5">{error.message}</div>
                        )}
                      </div>
                      {conn.auth && conn.auth.method !== 'none' && (
                        <span className="text-xs text-content-tertiary px-2 py-1 bg-surface-tertiary rounded">
                          {conn.auth.method === 'userpass' ? conn.auth.username : conn.auth.method}
                        </span>
                      )}
                      {conn.urls.length > 1 && (
                        <span className="text-xs text-blue-500 px-1.5 py-0.5 bg-accent-light rounded">
                          {conn.urls.length} servers
                        </span>
                      )}
                      {connectingId === conn.id ? (
                        <Spinner size="sm" className="shrink-0" />
                      ) : (
                        <svg className="w-4 h-4 text-content-muted shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
                        </svg>
                      )}
                    </button>
                  ))}
                </div>
              </div>
            ) : (
              <div className="bg-surface-primary rounded-lg shadow-sm border border-border p-6 text-center text-content-tertiary mb-4">
                <svg className="mx-auto h-12 w-12 text-content-muted mb-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7M4 7c0 2.21 3.582 4 8 4s8-1.79 8-4M4 7c0-2.21 3.582-4 8-4s8 1.79 8 4" />
                </svg>
                <p className="text-sm">No saved connections</p>
                <p className="text-xs text-content-muted mt-1">Create a new connection to get started</p>
              </div>
            )}

            <button
              onClick={() => setShowNewForm(true)}
              className="w-full px-4 py-3 bg-accent hover:bg-accent-hover text-content-inverse font-medium rounded-lg flex items-center justify-center gap-2 transition-colors"
            >
              <PlusIcon className="w-5 h-5" />
              New Connection
            </button>
          </>
        ) : (
          /* New Connection Form */
          <div className="bg-surface-primary rounded-lg shadow-sm border border-border overflow-hidden">
            <div className="px-4 py-3 bg-surface-secondary border-b border-border flex items-center justify-between">
              <h2 className="text-sm font-semibold text-gray-700">New Connection</h2>
              <button
                onClick={() => setShowNewForm(false)}
                aria-label="Cancel"
                className="text-content-muted hover:text-content-secondary"
              >
                <CloseIcon className="w-5 h-5" />
              </button>
            </div>
            <div className="p-4 space-y-4">
              {(error?.id === '__new__' || error?.id === '__test__') && (
                <div className="px-3 py-2 bg-status-error-bg border border-red-200 rounded-lg text-status-error-text text-sm">
                  {error.message}
                </div>
              )}

              {/* Name */}
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1" htmlFor="new-connection-name">Name</label>
                <Input
                  id="new-connection-name"
                  type="text"
                  value={newName}
                  onChange={(e) => setNewName(e.target.value)}
                  placeholder="My NATS Server"
                  error={!!nameError}
                  errorMessage={nameError}
                />
              </div>

              {/* Servers */}
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Servers</label>
                {newUrls.map((url, i) => (
                  <div key={urlKeys.keys[i]} className="flex gap-2 mb-2 items-start">
                    <Input
                      type="text"
                      aria-label={`Server URL ${i + 1}`}
                      value={url}
                      onChange={(e) => handleUrlChange(i, e.target.value)}
                      placeholder="nats://localhost:4222"
                      mono
                      error={!!urlErrors[i]}
                      errorMessage={urlErrors[i]}
                      className="flex-1"
                    />
                    {newUrls.length > 1 && (
                      <button
                        onClick={() => handleRemoveUrl(i)}
                        aria-label={`Remove server URL ${i + 1}`}
                        className="text-content-muted hover:text-red-500 px-2"
                      >
                        <CloseIcon className="w-4 h-4" />
                      </button>
                    )}
                  </div>
                ))}
                <button
                  onClick={handleAddUrl}
                  className="text-xs text-accent hover:text-accent-text flex items-center gap-1"
                >
                  <PlusIcon className="w-3 h-3" />
                  Add server
                </button>
              </div>

              {/* Auth Method */}
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">Authentication</label>
                <div className="flex gap-1 mb-3">
                  {(['none', 'userpass', 'token', 'nkey', 'credentials'] as AuthMethodTab[]).map((m) => (
                    <button
                      key={m}
                      onClick={() => setAuthMethod(m)}
                      className={`px-3 py-1.5 text-xs rounded-md transition-colors ${
                        authMethod === m
                          ? 'bg-accent-muted text-accent-text font-medium'
                          : 'bg-surface-tertiary text-content-secondary hover:bg-surface-hover'
                      }`}
                    >
                      {AUTH_LABELS[m]}
                    </button>
                  ))}
                </div>

                {authMethod === 'userpass' && (
                  <div className="grid grid-cols-2 gap-3">
                    <input type="text" value={newUsername} onChange={(e) => setNewUsername(e.target.value)}
                      placeholder="Username" className="w-full px-3 py-2 border border-border-strong rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-border-focus" />
                    <div className="relative">
                      <input type={showPassword ? 'text' : 'password'} value={newPassword} onChange={(e) => setNewPassword(e.target.value)}
                        placeholder="Password" className="w-full px-3 py-2 pr-9 border border-border-strong rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-border-focus" />
                      <button type="button" onClick={() => setShowPassword(!showPassword)}
                        aria-label={showPassword ? 'Hide password' : 'Show password'}
                        className="absolute right-2.5 top-1/2 -translate-y-1/2 text-content-muted hover:text-content-secondary">
                        {showPassword ? (
                          <EyeOffIcon className="w-4 h-4" />
                        ) : (
                          <EyeIcon className="w-4 h-4" />
                        )}
                      </button>
                    </div>
                  </div>
                )}

                {authMethod === 'token' && (
                  <div className="relative">
                    <input type={showToken ? 'text' : 'password'} value={newToken} onChange={(e) => setNewToken(e.target.value)}
                      placeholder="Token" className="w-full px-3 py-2 pr-9 border border-border-strong rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-border-focus" />
                    <button type="button" onClick={() => setShowToken(!showToken)}
                      aria-label={showToken ? 'Hide token' : 'Show token'}
                      className="absolute right-2.5 top-1/2 -translate-y-1/2 text-content-muted hover:text-content-secondary">
                      {showToken ? (
                        <EyeOffIcon className="w-4 h-4" />
                      ) : (
                        <EyeIcon className="w-4 h-4" />
                      )}
                    </button>
                  </div>
                )}

                {authMethod === 'nkey' && (
                  <div>
                    <textarea value={newNkeySeed} onChange={(e) => setNewNkeySeed(e.target.value)}
                      placeholder="Paste NKey seed (SUAM...)" rows={3}
                      className="w-full px-3 py-2 border border-border-strong rounded-lg text-sm font-mono focus:outline-none focus:ring-2 focus:ring-border-focus" />
                    <button onClick={() => handleFileUpload(setNewNkeySeed)}
                      className="text-xs text-accent hover:text-accent-text mt-1">Upload .nk file</button>
                  </div>
                )}

                {authMethod === 'credentials' && (
                  <div>
                    <textarea value={newCredentials} onChange={(e) => setNewCredentials(e.target.value)}
                      placeholder="Paste credentials file content" rows={4}
                      className="w-full px-3 py-2 border border-border-strong rounded-lg text-sm font-mono focus:outline-none focus:ring-2 focus:ring-border-focus" />
                    <button onClick={() => handleFileUpload(setNewCredentials)}
                      className="text-xs text-accent hover:text-accent-text mt-1">Upload .creds file</button>
                  </div>
                )}
              </div>

              {/* Actions */}
              <div className="flex gap-2">
                <button
                  onClick={handleTestConnection}
                  disabled={testConnectionMutation.isPending || newUrls.filter(Boolean).length === 0}
                  className="flex-1 px-4 py-2.5 border border-border-strong text-gray-700 font-medium rounded-lg hover:bg-surface-secondary disabled:opacity-50 transition-colors flex items-center justify-center gap-2 text-sm"
                >
                  {testConnectionMutation.isPending ? (
                    <svg className="animate-spin h-4 w-4" fill="none" viewBox="0 0 24 24">
                      <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                      <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
                    </svg>
                  ) : null}
                  Test
                </button>
                <button
                  onClick={handleNewConnection}
                  disabled={connectingId !== null}
                  className="flex-1 px-4 py-2.5 bg-accent hover:bg-accent-hover disabled:bg-gray-300 text-content-inverse font-medium rounded-lg transition-colors flex items-center justify-center gap-2 text-sm"
                >
                  {connectingId === '__new__' && (
                    <svg className="animate-spin h-4 w-4" fill="none" viewBox="0 0 24 24">
                      <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                      <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
                    </svg>
                  )}
                  {connectingId === '__new__' ? 'Creating...' : 'Connect'}
                </button>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
