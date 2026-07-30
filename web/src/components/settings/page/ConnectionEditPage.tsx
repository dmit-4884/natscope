import { useEffect, useMemo, useRef, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { Alert, Button, Spinner, DestructiveConfirm, WarningIcon } from '@/components/ui'
import Tooltip from '@/components/common/Tooltip'
import { formatBytes, formatDateTime } from '@/utils/formatters'
import { toast } from '@/utils/toast'
import { getErrorMessage, stripErrorCodePrefix } from '@/api/errors'
import {
  AuthConfig,
  NatsUrl,
  toApiAuthConfig,
  toApiTlsConfig,
  useConnections,
  useCreateConnection,
  useUpdateConnection,
  useTestConnection,
} from '@/contexts/connection'
import { ConnectionForm } from '@/components/connections/manager/ConnectionForm'
import { emptyTls, noSecretsSet, type ConnectionFormData } from '@/components/connections/manager/connectionFormData'
import type { SavedConnection, TestConnectionResponse } from '@/api/connections'

interface Props {
  mode: 'create' | 'edit'
}

const blankFormData: ConnectionFormData = {
  name: '',
  description: '',
  urls: [''],
  authMethod: 'none',
  username: '',
  password: '',
  token: '',
  nkeySeed: '',
  credentials: '',
  tls: { ...emptyTls },
  secretsSet: { ...noSecretsSet },
}

// TLS proto construction lives in the connection adapter (`toApiTlsConfig`),
// not here.

/** Hydrate the form from a saved connection (edit mode pre-fill). */
function formFromConnection(c: SavedConnection): ConnectionFormData {
  return {
    name: c.name,
    description: c.description ?? '',
    urls: c.urls.length > 0 ? [...c.urls] : [''],
    authMethod: c.auth?.method ?? 'none',
    username: c.auth?.username ?? '',
    password: c.auth?.password ?? '',
    token: c.auth?.token ?? '',
    nkeySeed: c.auth?.nkeySeed ?? '',
    credentials: c.auth?.credentials ?? '',
    tls: {
      caCert: c.tls?.caCert ?? '',
      clientCert: c.tls?.clientCert ?? '',
      clientKey: c.tls?.clientKey ?? '',
      skipVerify: c.tls?.skipVerify ?? false,
      tlsFirst: c.tls?.tlsFirst ?? false,
    },
    secretsSet: {
      password: c.auth?.hasPassword ?? false,
      token: c.auth?.hasToken ?? false,
      nkeySeed: c.auth?.hasNkeySeed ?? false,
      credentials: c.auth?.hasCredentials ?? false,
      clientKey: c.tls?.hasClientKey ?? false,
    },
  }
}

/** Full-page editor for a saved NATS connection. */
export default function ConnectionEditPage({ mode }: Props) {
  const navigate = useNavigate()
  const { id } = useParams<{ id: string }>()

  const isEdit = mode === 'edit'
  const { data: connections, isLoading: connectionsLoading } = useConnections()
  const existing = useMemo(
    () => (isEdit && id ? connections?.find((c) => c.id === id) : undefined),
    [isEdit, id, connections],
  )

  // A deep-link refresh hits this component before the List response lands;
  // track that separately so we show a spinner instead of a flickering blank form.
  const isHydrating = isEdit && (connectionsLoading || (!existing && !connections))
  const notFound = isEdit && !!connections && !existing

  const [form, setForm] = useState<ConnectionFormData>(blankFormData)
  const [error, setError] = useState<string | null>(null)
  const [nameError, setNameError] = useState<string | undefined>(undefined)
  const [urlErrors, setUrlErrors] = useState<(string | undefined)[]>([])
  const [testResult, setTestResult] = useState<{ ok: boolean; data: TestConnectionResponse } | null>(null)
  // Snapshot right after pre-fill (or blank for create) — lets us tell a real
  // edit from an incidental re-render, for the Discard prompt and stale-test flag.
  const [snapshot, setSnapshot] = useState<ConnectionFormData>(blankFormData)
  const lastHydratedFor = useRef<string | null>(null)

  const createMutation = useCreateConnection()
  const updateMutation = useUpdateConnection()
  const testMutation = useTestConnection()

  // Wait for connections to arrive (cache can be empty on a deep-link refresh),
  // then hydrate once per id so a later refetch doesn't clobber the user's edits.
  useEffect(() => {
    if (!isEdit) {
      lastHydratedFor.current = null
      return
    }
    if (!existing) return
    if (lastHydratedFor.current === existing.id) return
    const next = formFromConnection(existing)
    setForm(next)
    setSnapshot(next)
    lastHydratedFor.current = existing.id
  }, [isEdit, existing])

  // Any divergence from the snapshot counts as "dirty". Cheap deep-equal via
  // JSON — the form tree is small (no Date / Map fields) and stable in shape.
  const isDirty = useMemo(() => JSON.stringify(form) !== JSON.stringify(snapshot), [form, snapshot])

  // The sidebar probe reflects the form as it was at Test time; once the user
  // edits again that snapshot is stale, so we flag it instead of hiding the mismatch.
  const [testFormSnapshot, setTestFormSnapshot] = useState<string | null>(null)
  const [showDiscardConfirm, setShowDiscardConfirm] = useState(false)
  const isTestStale = !!testResult && testFormSnapshot !== null && testFormSnapshot !== JSON.stringify(form)

  // `explicit`: Save/Update emits even an empty-`none` AuthConfig so the backend
  // can clear previously-set auth; Test on an unsaved connection skips empty configs.
  const buildAuth = (opts: { explicit: boolean }): AuthConfig | undefined => {
    switch (form.authMethod) {
      case 'userpass':
        if (!form.username && !form.password) {
          return opts.explicit ? AuthConfig.fromTrusted({ method: 'userpass' }) : undefined
        }
        return AuthConfig.fromTrusted({
          method: 'userpass',
          username: form.username || undefined,
          password: form.password || undefined,
        })
      case 'token':
        if (!form.token) {
          return opts.explicit ? AuthConfig.fromTrusted({ method: 'token' }) : undefined
        }
        return AuthConfig.fromTrusted({ method: 'token', token: form.token })
      case 'nkey':
        if (!form.nkeySeed) {
          return opts.explicit ? AuthConfig.fromTrusted({ method: 'nkey' }) : undefined
        }
        return AuthConfig.fromTrusted({ method: 'nkey', nkeySeed: form.nkeySeed })
      case 'credentials':
        if (!form.credentials) {
          return opts.explicit ? AuthConfig.fromTrusted({ method: 'credentials' }) : undefined
        }
        return AuthConfig.fromTrusted({ method: 'credentials', credentials: form.credentials })
      default:
        // Edit-mode Save must explicitly send method=NONE to wipe a previously-set
        // auth; otherwise undefined means "leave as is".
        return opts.explicit ? AuthConfig.fromTrusted({ method: 'none' }) : undefined
    }
  }

  const handleTest = async () => {
    if (!validateUrls()) return
    const urls = form.urls.filter(Boolean)
    setError(null)
    // Snapshot exactly what the user sent so we can detect later edits.
    const snap = JSON.stringify(form)
    try {
      const result = await testMutation.mutateAsync({
        urls,
        auth: toApiAuthConfig(buildAuth({ explicit: false })),
        tls: toApiTlsConfig(form.tls, { explicit: false }),
        connectionId: isEdit ? id : undefined,
      })
      setTestResult({ ok: !!result.success, data: result })
      setTestFormSnapshot(snap)
      if (!result.success) {
        setError(result.error ? stripErrorCodePrefix(result.error) : 'Connection failed')
      }
    } catch (err) {
      const msg = getErrorMessage(err) || 'Test failed'
      setError(msg)
      setTestResult(null)
      setTestFormSnapshot(null)
    }
  }

  const handleCancel = () => {
    if (isDirty) {
      setShowDiscardConfirm(true)
      return
    }
    navigate('/settings/connections')
  }

  const validateUrls = (): boolean => {
    const nextUrlErrors = form.urls.map((u) => {
      if (!u.trim()) return undefined
      const result = NatsUrl.create(u)
      return result.isErr() ? result.error.message : undefined
    })
    if (!form.urls.some((u) => u.trim() !== '')) {
      nextUrlErrors[0] = 'At least one server URL is required'
    }

    setUrlErrors(nextUrlErrors)
    return nextUrlErrors.every((e) => !e)
  }

  const validate = (): boolean => {
    const nextNameError = form.name.trim() ? undefined : 'Name is required'
    setNameError(nextNameError)
    const urlsOk = validateUrls()

    return !nextNameError && urlsOk
  }

  const handleSave = async () => {
    if (!validate()) return
    const urls = form.urls.filter(Boolean)
    setError(null)
    try {
      const name = form.name.trim()
      const description = form.description.trim() || undefined
      // Edit mode sends explicit auth/tls (even empty) so the user can clear
      // them; Create mode only sends what's actually filled in.
      if (isEdit && id) {
        await updateMutation.mutateAsync({
          id,
          connection: {
            name,
            description,
            urls,
            auth: toApiAuthConfig(buildAuth({ explicit: true })),
            tls: toApiTlsConfig(form.tls, { explicit: true }),
          },
        })
        toast.success('Connection updated')
      } else {
        await createMutation.mutateAsync({
          name,
          description,
          urls,
          auth: toApiAuthConfig(buildAuth({ explicit: false })),
          tls: toApiTlsConfig(form.tls, { explicit: false }),
        })
        toast.success('Connection saved')
      }
      navigate('/settings/connections')
    } catch (err) {
      const msg = getErrorMessage(err) || 'Save failed'
      setError(msg)
    }
  }

  const isSaving = createMutation.isPending || updateMutation.isPending

  if (isHydrating) {
    return (
      <div className="flex-1 flex items-center justify-center bg-surface-secondary">
        <div className="flex flex-col items-center gap-3 text-content-tertiary">
          <Spinner size="lg" />
          <p className="text-sm">Loading connection…</p>
        </div>
      </div>
    )
  }

  if (notFound) {
    return (
      <div className="flex-1 flex items-center justify-center bg-surface-secondary">
        <div className="flex flex-col items-center gap-3 text-content-tertiary max-w-md text-center px-6">
          <WarningIcon className="w-10 h-10 text-gray-300" />
          <p className="text-sm font-medium text-content-primary">Connection not found</p>
          <p className="text-xs">It may have been deleted. Return to the list to pick another.</p>
          <Link
            to="/settings/connections"
            className="text-sm text-accent hover:text-accent-text"
          >
            Back to connections
          </Link>
        </div>
      </div>
    )
  }

  return (
    <div className="flex-1 flex flex-col overflow-hidden">
      <div className="flex items-center gap-3 px-4 sm:px-6 lg:px-8 py-4 border-b border-border bg-surface-primary">
        <button
          type="button"
          onClick={handleCancel}
          className="text-sm text-content-tertiary hover:text-gray-700 inline-flex items-center gap-1"
        >
          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
          </svg>
          Back
        </button>
        <div className="h-5 w-px bg-surface-hover" />
        <h2 className="text-base font-semibold text-content-primary">
          {isEdit ? 'Edit Connection' : 'New Connection'}
        </h2>
        {isDirty && (
          <span className="text-2xs px-1.5 py-0.5 rounded-full bg-status-warning-light text-amber-700 font-medium">
            unsaved changes
          </span>
        )}
      </div>

      <div className="flex-1 overflow-y-auto bg-surface-secondary">
        <div className="max-w-6xl mx-auto px-4 sm:px-6 lg:px-8 py-6 grid grid-cols-1 lg:grid-cols-[minmax(0,1fr)_320px] gap-6">
          <div>
            {error && (
              <div className="mb-4">
                <Alert variant="error">{error}</Alert>
              </div>
            )}
            <ConnectionForm
              value={form}
              onChange={setForm}
              onCancel={handleCancel}
              onTest={handleTest}
              isTesting={testMutation.isPending}
              mode={mode}
              nameError={nameError}
              urlErrors={urlErrors}
            />
          </div>

          <aside className="space-y-4">
            <div className="bg-surface-primary border border-border rounded-lg p-4">
              <h3 className="text-sm font-semibold text-content-primary mb-2">Probe</h3>
              <p className="text-xs text-content-tertiary mb-3">
                Click <span className="font-medium">Test</span> to attempt a connection without saving.
                The result below comes straight from the broker.
              </p>
              {!testResult && (
                <p className="text-xs text-content-muted">No probe yet.</p>
              )}
              {isTestStale && (
                <div className="mb-2 flex items-start gap-2 px-2 py-1.5 bg-status-warning-bg border border-amber-200 rounded text-2xs text-amber-800">
                  <WarningIcon className="w-3.5 h-3.5 mt-px shrink-0" />
                  <span>Form changed since last test — re-run Test to verify.</span>
                </div>
              )}
              {testResult && testResult.ok && (
                <ul className="text-xs space-y-1.5">
                  <li className="flex justify-between gap-2">
                    <span className="text-content-tertiary">Status</span>
                    <span className="font-medium text-emerald-700">Success</span>
                  </li>
                  {testResult.data.connectedUrl && (
                    <li className="flex justify-between gap-2">
                      <span className="text-content-tertiary">Connected URL</span>
                      <span className="font-mono truncate">{testResult.data.connectedUrl}</span>
                    </li>
                  )}
                  {testResult.data.serverVersion && (
                    <li className="flex justify-between gap-2">
                      <span className="text-content-tertiary">Server version</span>
                      <span className="font-mono">v{testResult.data.serverVersion}</span>
                    </li>
                  )}
                  {testResult.data.serverName && (
                    <li className="flex justify-between gap-2">
                      <span className="text-content-tertiary">Server name</span>
                      <span className="font-mono truncate">{testResult.data.serverName}</span>
                    </li>
                  )}
                  {testResult.data.clusterName && (
                    <li className="flex justify-between gap-2">
                      <span className="text-content-tertiary">Cluster</span>
                      <span className="font-mono truncate">{testResult.data.clusterName}</span>
                    </li>
                  )}
                  {testResult.data.rttMs !== undefined && (
                    <li className="flex justify-between gap-2">
                      <span className="text-content-tertiary">RTT</span>
                      <span className="font-mono">{Number(testResult.data.rttMs)} ms</span>
                    </li>
                  )}
                  <li className="flex justify-between gap-2">
                    <span className="text-content-tertiary">JetStream</span>
                    <span className="font-mono">{testResult.data.jetstreamEnabled ? 'enabled' : 'no'}</span>
                  </li>
                  {testResult.data.maxPayload !== undefined && Number(testResult.data.maxPayload) > 0 && (
                    <li className="flex justify-between gap-2">
                      <span className="text-content-tertiary">Max payload</span>
                      <span className="font-mono">{formatBytes(Number(testResult.data.maxPayload))}</span>
                    </li>
                  )}
                  {testResult.data.discoveredServers && testResult.data.discoveredServers.length > 0 && (
                    <li className="pt-1 border-t border-gray-100 mt-1">
                      <div className="text-content-tertiary mb-1">Discovered ({testResult.data.discoveredServers.length})</div>
                      <ul className="space-y-0.5 font-mono text-2xs text-content-secondary">
                        {testResult.data.discoveredServers.map((u) => (
                          <li key={u} className="truncate">{u}</li>
                        ))}
                      </ul>
                    </li>
                  )}
                </ul>
              )}
              {testResult && !testResult.ok && (
                <div className="text-xs text-rose-700 break-words">
                  <span className="font-medium">Failed:</span> {testResult.data.error ? stripErrorCodePrefix(testResult.data.error) : 'unknown error'}
                </div>
              )}
              {isEdit && existing?.meta && (
                <p className="mt-3 pt-3 border-t border-gray-100 text-2xs text-content-tertiary">
                  Last persisted probe: {formatDateTime(existing.meta.lastTestedAt)}
                </p>
              )}
            </div>

            <div className="bg-accent-light border border-blue-100 rounded-lg p-4 text-xs text-blue-900 space-y-1">
              <p className="font-medium">Storage</p>
              <p>Saved to the local database.</p>
              <p>Secrets go to the OS keychain or an encrypted file vault, never the database.</p>
            </div>
          </aside>
        </div>
      </div>

      <div className="border-t border-border bg-surface-primary px-4 sm:px-6 lg:px-8 py-3 flex justify-end gap-2">
        <Button variant="secondary" onClick={handleCancel}>
          Cancel
        </Button>
        <Tooltip content={isEdit && !isDirty ? 'No changes to save' : ''}>
          <Button
            onClick={handleSave}
            loading={isSaving}
            disabled={isEdit && !isDirty}
          >
            {isEdit ? 'Save changes' : 'Create connection'}
          </Button>
        </Tooltip>
      </div>

      <DestructiveConfirm
        isOpen={showDiscardConfirm}
        title="Discard unsaved changes"
        description={<span>You have unsaved edits. Leave without saving?</span>}
        tone="warning"
        confirmLabel="Discard"
        onCancel={() => setShowDiscardConfirm(false)}
        onConfirm={() => {
          setShowDiscardConfirm(false)
          navigate('/settings/connections')
        }}
      />
    </div>
  )
}
