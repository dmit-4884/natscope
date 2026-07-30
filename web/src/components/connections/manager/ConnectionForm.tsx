import { useEffect, useState } from 'react'
import { useRowKeys } from '@/hooks/useRowKeys'
import { CloseIcon, EyeIcon, EyeOffIcon, Input, LockClosedIcon, PlusIcon } from '@/components/ui'
import {
  AUTH_LABELS,
  emptyTls,
  type ConnectionFormData,
  type AuthMethodTab,
} from './connectionFormData'

interface Props {
  value: ConnectionFormData
  onChange: (value: ConnectionFormData) => void
  onCancel: () => void
  onTest: () => void
  isTesting: boolean
  mode?: 'create' | 'edit'
  nameError?: string
  urlErrors?: (string | undefined)[]
}

function uploadFile(onChange: (value: string) => void) {
  const input = document.createElement('input')
  input.type = 'file'
  input.onchange = (e) => {
    const file = (e.target as HTMLInputElement).files?.[0]
    if (file) {
      const reader = new FileReader()
      reader.onload = () => onChange(reader.result as string)
      reader.readAsText(file)
    }
  }
  input.click()
}

/** True when any TLS field carries user input. Drives "is TLS configured?" UX. */
function hasTlsContent(tls: ConnectionFormData['tls']): boolean {
  return (
    tls.caCert !== '' ||
    tls.clientCert !== '' ||
    tls.clientKey !== '' ||
    tls.skipVerify ||
    tls.tlsFirst
  )
}

export function ConnectionForm({
  value,
  onChange,
  onCancel,
  onTest,
  isTesting,
  mode = 'create',
  nameError,
  urlErrors,
}: Props) {
  const [showPassword, setShowPassword] = useState(false)
  const [showToken, setShowToken] = useState(false)
  // TLS section is opt-in: shown only when opened or already configured.
  // "Remove" collapses it and clears TLS state so no discarded fields linger.
  const [tlsOpen, setTlsOpen] = useState(hasTlsContent(value.tls))

  // Parent hydrates value.tls asynchronously after connections load, so the
  // initial useState misses it — open here when it appears; never auto-close.
  useEffect(() => {
    if (hasTlsContent(value.tls)) setTlsOpen(true)
    // Keyed on primitive fields, not object identity — parent recreates the
    // object every keystroke, which would refire this effect forever.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [value.tls.caCert, value.tls.clientCert, value.tls.clientKey, value.tls.skipVerify, value.tls.tlsFirst])

  const urlKeys = useRowKeys(value.urls.length)
  const update = (patch: Partial<ConnectionFormData>) => onChange({ ...value, ...patch })
  const updateTls = (patch: Partial<ConnectionFormData['tls']>) =>
    onChange({ ...value, tls: { ...value.tls, ...patch } })

  const removeTls = () => {
    onChange({ ...value, tls: { ...emptyTls } })
    setTlsOpen(false)
  }

  return (
    <div className="mb-4 bg-surface-primary rounded-lg border border-border overflow-hidden">
      <div className="px-4 py-3 bg-surface-secondary border-b border-border flex items-center justify-between">
        <h3 className="text-sm font-semibold text-gray-700">{mode === 'edit' ? 'Edit Connection' : 'New Connection'}</h3>
        <button onClick={onCancel} aria-label="Cancel" className="text-content-muted hover:text-content-secondary">
          <CloseIcon className="w-5 h-5" />
        </button>
      </div>
      <div className="p-4 space-y-4">
        <label className="block">
          <span className="block text-sm font-medium text-gray-700 mb-1">Name</span>
          <Input
            type="text"
            value={value.name}
            onChange={(e) => update({ name: e.target.value })}
            placeholder="My NATS Server"
            error={!!nameError}
            errorMessage={nameError}
          />
        </label>

        <label className="block">
          <span className="block text-sm font-medium text-gray-700 mb-1">
            Description <span className="text-content-muted font-normal">(optional)</span>
          </span>
          <input
            type="text"
            value={value.description}
            onChange={(e) => update({ description: e.target.value })}
            placeholder="Notes — environment, owner, anything that helps tell servers apart"
            className="w-full px-3 py-2 border border-border-strong rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-border-focus focus:border-border-focus"
          />
        </label>

        <fieldset className="border-0 p-0 m-0">
          <legend className="block text-sm font-medium text-gray-700 mb-1">Servers</legend>
          {value.urls.map((url, i) => (
            <div key={urlKeys.keys[i]} className="flex gap-2 mb-2 items-start">
              <Input
                type="text"
                aria-label={`Server URL ${i + 1}`}
                value={url}
                onChange={(e) => {
                  const u = [...value.urls]
                  u[i] = e.target.value
                  update({ urls: u })
                }}
                placeholder="nats://localhost:4222"
                mono
                error={!!urlErrors?.[i]}
                errorMessage={urlErrors?.[i]}
                className="flex-1"
              />
              {value.urls.length > 1 && (
                <button
                  onClick={() => {
                    urlKeys.registerRemove(i)
                    update({ urls: value.urls.filter((_, idx) => idx !== i) })
                  }}
                  aria-label={`Remove server URL ${i + 1}`}
                  className="text-content-muted hover:text-red-500 px-2"
                >
                  <CloseIcon className="w-4 h-4" />
                </button>
              )}
            </div>
          ))}
          <button
            onClick={() => {
              urlKeys.registerAdd()
              update({ urls: [...value.urls, ''] })
            }}
            className="text-xs text-accent hover:text-accent-text flex items-center gap-1"
          >
            <PlusIcon className="w-3 h-3" />
            Add server
          </button>
        </fieldset>

        <div>
          <label className="block text-sm font-medium text-gray-700 mb-2">Authentication</label>
          <div className="flex gap-1 mb-3">
            {(['none', 'userpass', 'token', 'nkey', 'credentials'] as AuthMethodTab[]).map((m) => (
              <button
                key={m}
                onClick={() => update({ authMethod: m })}
                className={`px-3 py-1.5 text-xs rounded-md transition-colors ${
                  value.authMethod === m
                    ? 'bg-accent-muted text-accent-text font-medium'
                    : 'bg-surface-tertiary text-content-secondary hover:bg-surface-hover'
                }`}
              >
                {AUTH_LABELS[m]}
              </button>
            ))}
          </div>

          {value.authMethod === 'userpass' && (
            <div className="grid grid-cols-2 gap-3">
              <input
                type="text"
                aria-label="Username"
                value={value.username}
                onChange={(e) => update({ username: e.target.value })}
                placeholder="Username"
                className="w-full px-3 py-2 border border-border-strong rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-border-focus"
              />
              <div className="relative">
                <input
                  type={showPassword ? 'text' : 'password'}
                  aria-label="Password"
                  value={value.password}
                  onChange={(e) => update({ password: e.target.value })}
                  placeholder={value.secretsSet.password ? 'Stored — leave blank to keep' : 'Password'}
                  className="w-full px-3 py-2 pr-9 border border-border-strong rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-border-focus"
                />
                <button
                  type="button"
                  onClick={() => setShowPassword(!showPassword)}
                  aria-label={showPassword ? 'Hide password' : 'Show password'}
                  className="absolute right-2.5 top-1/2 -translate-y-1/2 text-content-muted hover:text-content-secondary"
                >
                  {showPassword ? <EyeOffIcon className="w-4 h-4" /> : <EyeIcon className="w-4 h-4" />}
                </button>
              </div>
            </div>
          )}
          {value.authMethod === 'token' && (
            <div className="relative">
              <input
                type={showToken ? 'text' : 'password'}
                aria-label="Auth token"
                value={value.token}
                onChange={(e) => update({ token: e.target.value })}
                placeholder={value.secretsSet.token ? 'Stored — leave blank to keep' : 'Token'}
                className="w-full px-3 py-2 pr-9 border border-border-strong rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-border-focus"
              />
              <button
                type="button"
                onClick={() => setShowToken(!showToken)}
                aria-label={showToken ? 'Hide token' : 'Show token'}
                className="absolute right-2.5 top-1/2 -translate-y-1/2 text-content-muted hover:text-content-secondary"
              >
                {showToken ? <EyeOffIcon className="w-4 h-4" /> : <EyeIcon className="w-4 h-4" />}
              </button>
            </div>
          )}
          {value.authMethod === 'nkey' && (
            <div>
              <textarea
                aria-label="NKey seed"
                value={value.nkeySeed}
                onChange={(e) => update({ nkeySeed: e.target.value })}
                placeholder={value.secretsSet.nkeySeed ? 'Stored — leave blank to keep' : 'Paste NKey seed (SUAM...)'}
                rows={3}
                className="w-full px-3 py-2 border border-border-strong rounded-lg text-sm font-mono focus:outline-none focus:ring-2 focus:ring-border-focus"
              />
              <button
                onClick={() => uploadFile((v) => update({ nkeySeed: v }))}
                className="text-xs text-accent hover:text-accent-text mt-1"
              >
                Upload .nk file
              </button>
            </div>
          )}
          {value.authMethod === 'credentials' && (
            <div>
              <textarea
                aria-label="Credentials file contents"
                value={value.credentials}
                onChange={(e) => update({ credentials: e.target.value })}
                placeholder={value.secretsSet.credentials ? 'Stored — leave blank to keep' : 'Paste credentials file content'}
                rows={4}
                className="w-full px-3 py-2 border border-border-strong rounded-lg text-sm font-mono focus:outline-none focus:ring-2 focus:ring-border-focus"
              />
              <button
                onClick={() => uploadFile((v) => update({ credentials: v }))}
                className="text-xs text-accent hover:text-accent-text mt-1"
              >
                Upload .creds file
              </button>
            </div>
          )}
        </div>

        {/* TLS — opt-in. Hidden until the user adds it; "Remove" wipes any
            entered fields so closing the section means closing it for real. */}
        {!tlsOpen ? (
          <button
            type="button"
            onClick={() => setTlsOpen(true)}
            className="text-xs text-accent hover:text-accent-text inline-flex items-center gap-1"
          >
            <PlusIcon className="w-3.5 h-3.5" />
            Add TLS configuration
          </button>
        ) : (
          <div className="border border-border rounded-lg overflow-hidden">
            <div className="flex items-center justify-between px-3 py-2 bg-surface-secondary border-b border-border">
              <span className="flex items-center gap-2 text-sm font-medium text-gray-700">
                <LockClosedIcon className="w-4 h-4 text-content-tertiary" />
                TLS
                {(value.tls.caCert || value.tls.clientCert || value.tls.clientKey) && (
                  <span className="text-2xs px-1.5 py-0.5 rounded-full bg-status-success-light text-green-700 font-semibold">
                    configured
                  </span>
                )}
                {value.tls.skipVerify && (
                  <span className="text-2xs px-1.5 py-0.5 rounded-full bg-status-warning-light text-amber-700 font-semibold">
                    skip verify
                  </span>
                )}
              </span>
              <button
                type="button"
                onClick={removeTls}
                className="text-xs text-content-tertiary hover:text-status-error-text"
              >
                Remove
              </button>
            </div>
            <div className="p-3 space-y-3">
              <label className="block">
                <span className="block text-xs font-medium text-gray-700 mb-1">CA certificate (PEM)</span>
                <textarea
                  value={value.tls.caCert}
                  onChange={(e) => updateTls({ caCert: e.target.value })}
                  placeholder="-----BEGIN CERTIFICATE-----"
                  rows={3}
                  className="w-full px-3 py-2 border border-border-strong rounded-lg text-xs font-mono focus:outline-none focus:ring-2 focus:ring-border-focus"
                />
                <button
                  onClick={() => uploadFile((v) => updateTls({ caCert: v }))}
                  className="text-xs text-accent hover:text-accent-text mt-1"
                >
                  Upload .pem / .crt
                </button>
              </label>

              <div className="grid grid-cols-2 gap-3">
                <label className="block">
                  <span className="block text-xs font-medium text-gray-700 mb-1">Client certificate</span>
                  <textarea
                    value={value.tls.clientCert}
                    onChange={(e) => updateTls({ clientCert: e.target.value })}
                    placeholder="-----BEGIN CERTIFICATE-----"
                    rows={3}
                    className="w-full px-3 py-2 border border-border-strong rounded-lg text-xs font-mono focus:outline-none focus:ring-2 focus:ring-border-focus"
                  />
                  <button
                    onClick={() => uploadFile((v) => updateTls({ clientCert: v }))}
                    className="text-xs text-accent hover:text-accent-text mt-1"
                  >
                    Upload .crt
                  </button>
                </label>
                <label className="block">
                  <span className="block text-xs font-medium text-gray-700 mb-1">Client key</span>
                  <textarea
                    value={value.tls.clientKey}
                    onChange={(e) => updateTls({ clientKey: e.target.value })}
                    placeholder={value.secretsSet.clientKey ? 'Stored — leave blank to keep' : '-----BEGIN PRIVATE KEY-----'}
                    rows={3}
                    className="w-full px-3 py-2 border border-border-strong rounded-lg text-xs font-mono focus:outline-none focus:ring-2 focus:ring-border-focus"
                  />
                  <button
                    onClick={() => uploadFile((v) => updateTls({ clientKey: v }))}
                    className="text-xs text-accent hover:text-accent-text mt-1"
                  >
                    Upload .key
                  </button>
                </label>
              </div>

              <p className="text-2xs text-content-tertiary -mt-1">
                Both fields are required together for mTLS — leave blank to use server-only TLS.
              </p>

              <div className="flex flex-col gap-2 pt-1">
                <label className="flex items-center gap-2 text-xs text-gray-700 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={value.tls.skipVerify}
                    onChange={(e) => updateTls({ skipVerify: e.target.checked })}
                    className="rounded border-border-strong text-accent focus:ring-border-focus"
                  />
                  <span>
                    <span className="font-medium">Skip certificate verification</span>
                    <span className="text-content-tertiary"> — accept self-signed certs (dev only)</span>
                  </span>
                </label>

                <label className="flex items-center gap-2 text-xs text-gray-700 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={value.tls.tlsFirst}
                    onChange={(e) => updateTls({ tlsFirst: e.target.checked })}
                    className="rounded border-border-strong text-accent focus:ring-border-focus"
                  />
                  <span>
                    <span className="font-medium">TLS handshake first</span>
                    <span className="text-content-tertiary"> — required by NATS 2.10+ servers with <code>tls_handshake_first</code></span>
                  </span>
                </label>
              </div>
            </div>
          </div>
        )}

        <div className="flex gap-2">
          <button
            onClick={onTest}
            disabled={isTesting || value.urls.filter(Boolean).length === 0}
            className="px-4 py-2 text-sm border border-border-strong text-gray-700 rounded-lg hover:bg-surface-secondary disabled:opacity-50 transition-colors flex items-center gap-2"
          >
            {isTesting && (
              <svg className="animate-spin h-3.5 w-3.5" fill="none" viewBox="0 0 24 24">
                <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                <path
                  className="opacity-75"
                  fill="currentColor"
                  d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
                />
              </svg>
            )}
            Test
          </button>
        </div>
      </div>
    </div>
  )
}
