import { useState, useEffect } from 'react'
import { Link, useParams, useNavigate, useOutletContext } from 'react-router-dom'
import { useConnectionPolicy } from '@/contexts/connection'
import {
  useKVBuckets,
  useKVKeys,
  useKVKey,
  useKVKeyHistory,
  usePutKVKey,
  useDeleteKVKey,
  usePurgeKVKey,
  useDeleteKVBucket,
  usePurgeKVBucket,
  useKVWatch,
} from '@/contexts/kv'
import { decodeBase64, type KVProtoValue } from '@/api/management'
import { getErrorMessage } from '@/api/errors'
import { formatBytes, formatCount, formatDateTime, formatNsDuration, formatTime } from '@/utils/formatters'
import { decodeBase64ToBytes } from '@/utils/base64'
import { plural } from '@/utils/plural'
import { useConfirmation } from '@/contexts/settings'
import type { KVChange, KVEntry } from '@/types/management'
import {
  Button,
  Modal,
  Input,
  Badge,
  Alert,
  EmptyState,
  Spinner,
  SearchInput,
  CloseIcon,
  PlusIcon,
  RefreshIcon,
  OverflowMenu,
  Tabs,
  Toggle,
  tabPanelProps,
} from '@/components/ui'
import Tooltip from '../common/Tooltip'
import type { ConnectionOutletContext } from '../common/ConnectedLayout'
import { WireView } from '../messages/WireView'
import { KVProtoBar } from './KVProtoBar'
import { useKVProtoTarget, type KVProtoTarget } from './useKVProtoTarget'
import { keyTtlRaisedTo, parseKeyTtl } from './keyTtl'

function editableValue(entry: KVEntry, asProto: boolean): string {
  if (!asProto) return decodeBase64(entry.value)
  return entry.decoded?.data === undefined ? '{}' : JSON.stringify(entry.decoded.data, null, 2)
}

function storedValue(target: KVProtoTarget | null, text: string): string | KVProtoValue {
  return target
    ? {
        messageType: target.messageType,
        sourceId: target.sourceId,
        framing: target.framing,
        fingerprint: target.pinnedFingerprint,
        json: text,
      }
    : text
}

const NO_KEYS: string[] = []
const PREVIEW_CHARS = 80

function isPrintable(text: string): boolean {
  for (let i = 0; i < text.length; i++) {
    const code = text.charCodeAt(i)
    if (code < 32 && code !== 9 && code !== 10 && code !== 13) return false
  }
  return true
}

function changePreview(change: KVChange): string {
  try {
    const text = new TextDecoder('utf-8', { fatal: true }).decode(decodeBase64ToBytes(change.value))
    if (isPrintable(text)) {
      const line = text.replace(/\s+/g, ' ')
      return line.length > PREVIEW_CHARS || change.size > text.length ? `${line.slice(0, PREVIEW_CHARS)}…` : line
    }
  } catch {
    /* binary values fall through to their size */
  }
  return formatBytes(change.size)
}

function isKeyPattern(text: string): boolean {
  return /[*>]/.test(text)
}

function isValidKeyPattern(text: string): boolean {
  const tokens = text.split('.')
  return tokens.every(
    (token, i) => token !== '' && (!/[*>]/.test(token) || token === '*' || (token === '>' && i === tokens.length - 1)),
  )
}

type KVConfirmAction = {
  type: 'delete-bucket' | 'clear-bucket' | 'delete-key' | 'purge-key'
  name: string
  confirmText: string
}

export default function KVStorePage() {
  const { bucketName } = useParams<{ bucketName: string }>()
  const navigate = useNavigate()
  const { connectionId, currentConnection } = useOutletContext<ConnectionOutletContext>()

  const [selectedKey, setSelectedKey] = useState<string | null>(null)
  const [isCreatingKey, setIsCreatingKey] = useState(false)
  const [keySearchQuery, setKeySearchQuery] = useState('')
  const [newKeyName, setNewKeyName] = useState('')
  const [newKeyValue, setNewKeyValue] = useState('')
  const [newKeyTtl, setNewKeyTtl] = useState('')
  const [draft, setDraft] = useState<{ key: string | null; value: string; revision?: number } | null>(null)
  const [live, setLive] = useState(false)
  const [listView, setListView] = useState<'keys' | 'changes'>('keys')
  const [showHistory, setShowHistory] = useState(false)
  const [rawChoice, setRawChoice] = useState<{ key: string | null; revision?: number; on: boolean } | null>(null)
  const [confirmAction, setConfirmAction] = useState<KVConfirmAction | null>(null)

  const deleteKeyConfirmation = useConfirmation('deleteKvKey')
  const purgeKeyConfirmation = useConfirmation('purgeKvHistory')

  // Fetch bucket info
  const { data: buckets = [] } = useKVBuckets(connectionId)
  const bucketInfo = buckets.find(b => b.bucket === bucketName)
  const allowsKeyTtl = !!bucketInfo?.limit_marker_ttl
  const mirrorOf = bucketInfo?.mirror_of
  const keyTtl = allowsKeyTtl ? parseKeyTtl(newKeyTtl) : {}
  const raisedTtlNs = keyTtlRaisedTo(keyTtl.ns, bucketInfo?.limit_marker_ttl)

  const searchText = keySearchQuery.trim()
  const badPattern = isKeyPattern(searchText) && !isValidKeyPattern(searchText)
  const keyPattern = isKeyPattern(searchText) && !badPattern ? searchText : ''
  const { data: keyList, isLoading: keysLoading, error: keysError, refetch: refetchKeys } = useKVKeys(
    connectionId,
    bucketName,
    keyPattern
  )
  const keys = keyList?.keys ?? NO_KEYS
  const lookupKey = keyList?.truncated && searchText && !isKeyPattern(searchText) && searchText.split('.').every(Boolean) ? searchText : ''
  const { data: lookup } = useKVKeys(connectionId, bucketName, lookupKey, lookupKey !== '')
  const exactKey = lookupKey && lookup?.keys.includes(lookupKey) ? lookupKey : ''
  const watch = useKVWatch(connectionId, bucketName, keyPattern, live)

  // Fetch selected key value
  const { data: keyEntry, isLoading: keyLoading } = useKVKey(
    connectionId,
    bucketName,
    selectedKey ?? undefined
  )

  // History is fetched only while its modal is open (server-side it spins up
  // an ephemeral consumer per request).
  const {
    data: keyHistory,
    isLoading: historyLoading,
    error: historyError,
  } = useKVKeyHistory(
    connectionId,
    bucketName,
    showHistory && selectedKey ? selectedKey : undefined
  )

  const target = useKVProtoTarget(bucketName ?? '', selectedKey ?? '', keyEntry?.decoded)
  const newKeyTarget = useKVProtoTarget(bucketName ?? '', newKeyName.trim())

  const { readOnly } = useConnectionPolicy()

  // Mutations
  const deleteBucket = useDeleteKVBucket(connectionId)
  const purgeBucket = usePurgeKVBucket(connectionId)
  const putKey = usePutKVKey(connectionId, bucketName)
  const deleteKey = useDeleteKVKey(connectionId, bucketName)
  const purgeKey = usePurgeKVKey(connectionId, bucketName)

  const textMatches = keyPattern ? keys : keys.filter((k) => k.toLowerCase().includes(searchText.toLowerCase()))
  const filteredKeys = exactKey && !textMatches.includes(exactKey) ? [exactKey, ...textMatches] : textMatches
  const partial = !!keyList?.truncated

  const valueDirty = draft !== null && draft.key === selectedKey
  const editingValue = valueDirty ? draft.value : keyEntry && !isCreatingKey ? editableValue(keyEntry, !!target) : ''
  const setEditingValue = (value: string) =>
    setDraft((prev) => ({
      key: selectedKey,
      value,
      revision: prev && prev.key === selectedKey ? prev.revision : keyEntry?.revision,
    }))
  const changedWhileEditing =
    valueDirty && draft.revision !== undefined && !!keyEntry && keyEntry.revision !== draft.revision

  const undecodable = !!keyEntry?.decoded?.error && keyEntry.decoded.data === undefined
  const showRaw = rawChoice && rawChoice.key === selectedKey && rawChoice.revision === keyEntry?.revision ? rawChoice.on : undecodable

  // Reset selection AND search on bucket change — a stale search filter
  // would hide all keys in the new bucket, looking like it's empty.
  useEffect(() => {
    setSelectedKey(null)
    setIsCreatingKey(false)
    setKeySearchQuery('')
  }, [bucketName])

  useEffect(() => {
    setShowHistory(false)
  }, [selectedKey, bucketName])

  // Handle create/update key
  const handleSaveKey = async () => {
    try {
      if (isCreatingKey) {
        if (!newKeyName.trim() || keyTtl.error) return
        await putKey.mutateAsync({ key: newKeyName, value: storedValue(newKeyTarget, newKeyValue), ttl: keyTtl.ns })
        setIsCreatingKey(false)
        setNewKeyName('')
        setNewKeyValue('')
        setNewKeyTtl('')
      } else if (selectedKey) {
        await putKey.mutateAsync({
          key: selectedKey,
          value: storedValue(target, editingValue),
          expectedRevision: valueDirty ? draft.revision : keyEntry?.revision,
        })
        setDraft(null)
      }
      refetchKeys()
    } catch {
      /* toasted by usePutKVKey */
    }
  }

  const performAction = async (action: KVConfirmAction) => {
    try {
      if (action.type === 'delete-bucket') {
        await deleteBucket.mutateAsync(action.name)
        navigate('/kv')
      } else if (action.type === 'clear-bucket') {
        await purgeBucket.mutateAsync(action.name)
        setSelectedKey(null)
        setDraft(null)
        refetchKeys()
      } else if (action.type === 'delete-key') {
        await deleteKey.mutateAsync(action.name)
        if (selectedKey === action.name) {
          setSelectedKey(null)
        }
        refetchKeys()
      } else if (action.type === 'purge-key') {
        await purgeKey.mutateAsync(action.name)
        if (selectedKey === action.name) {
          setSelectedKey(null)
        }
        refetchKeys()
      }
    } catch {
      /* toasted by the mutation hooks */
    } finally {
      setConfirmAction(null)
    }
  }

  const requestAction = (action: KVConfirmAction) => {
    if (action.type === 'delete-key' && !deleteKeyConfirmation.enabled) {
      void performAction(action)
      return
    }
    if (action.type === 'purge-key' && !purgeKeyConfirmation.enabled) {
      void performAction(action)
      return
    }
    setConfirmAction(action)
  }

  const handleConfirmAction = () => {
    if (!confirmAction) return
    void performAction(confirmAction)
  }

  if (!connectionId || !bucketName) {
    return null
  }

  return (
    <div className="flex-1 flex flex-col overflow-hidden bg-surface-primary">
      {/* Header */}
      <div className="px-4 py-3 border-b bg-surface-secondary">
        <div className="flex items-center justify-between">
          <div>
            <h3 className="font-semibold text-content-primary">{bucketName}</h3>
            <p className="text-sm text-content-tertiary mt-0.5">
              {currentConnection?.name || currentConnection?.urls[0] || 'Unknown connection'}
              {bucketInfo && (
                <span className="ml-2">
                  | {plural(bucketInfo.values, 'revision')} | {formatBytes(bucketInfo.bytes)}
                </span>
              )}
            </p>
          </div>
          {!readOnly && (
            <OverflowMenu
              label="Bucket actions"
              items={[
                ...(mirrorOf ? [] : [
                  {
                    label: 'Edit bucket…',
                    onSelect: () => navigate(`/kv/${encodeURIComponent(bucketName)}/edit`),
                  },
                  {
                    label: 'Clear bucket…',
                    destructive: true,
                    onSelect: () => setConfirmAction({
                      type: 'clear-bucket',
                      name: bucketName,
                      confirmText: '',
                    }),
                  },
                ]),
                {
                  label: 'Delete bucket…',
                  destructive: true,
                  onSelect: () => setConfirmAction({
                    type: 'delete-bucket',
                    name: bucketName,
                    confirmText: '',
                  }),
                },
              ]}
            />
          )}
        </div>
      </div>

      {mirrorOf ? (
        <EmptyState
          title={`Mirror of ${mirrorOf}`}
          description="NATS serves a mirror's keys under the name of the bucket it mirrors, so they are read and written there."
          action={
            <Link to={`/kv/${encodeURIComponent(mirrorOf)}`} className="text-accent hover:text-accent-text">
              {mirrorOf}
            </Link>
          }
        />
      ) : (
        <div className="flex-1 flex min-h-0">
          {/* Keys List */}
          <div className="w-72 border-r bg-surface-primary flex flex-col">
            <div className="p-3 border-b">
              <SearchInput
                placeholder="Search keys, or a pattern like orders.>"
                value={keySearchQuery}
                onChange={setKeySearchQuery}
                debounce={200}
                size="sm"
                className="mb-2"
              />
              <div className="flex items-center justify-between">
                <span className="text-xs text-content-tertiary">
                  {partial ? `${formatCount(filteredKeys.length)}+ keys` : plural(filteredKeys.length, 'key')}
                </span>
                {!readOnly && (
                  <Button
                    size="sm"
                    variant="ghost"
                    aria-label="New key"
                    onClick={() => {
                      setIsCreatingKey(true)
                      setSelectedKey(null)
                      setNewKeyName('')
                      setNewKeyValue('')
                      setNewKeyTtl('')
                    }}
                  >
                    <PlusIcon className="w-4 h-4" />
                  </Button>
                )}
              </div>
              {badPattern && (
                <p className="mt-1 text-xs text-status-warning-text">
                  Use * or &gt; as a whole part of the key, like <span className="font-mono">orders.*</span> or{' '}
                  <span className="font-mono">orders.&gt;</span>
                </p>
              )}
              {keyList?.truncated && (
                <p className="mt-1 text-xs text-content-tertiary" data-testid="kv-keys-truncated">
                  Only the first {plural(keys.length, 'key')} are loaded. Narrow the search with a pattern like{' '}
                  <span className="font-mono">orders.&gt;</span>
                  {searchText && !keyPattern && <> The text filter only searched the {plural(keys.length, 'loaded key')}.</>}
                </p>
              )}
              <div className="mt-2 flex items-center gap-2 text-xs">
                <Toggle
                  size="xs"
                  checked={live}
                  onChange={(on) => {
                    setLive(on)
                    if (!on) setListView('keys')
                  }}
                  label="Live updates"
                />
                <span className="text-content-secondary">Live updates</span>
                {watch.status === 'starting' && <span className="text-content-tertiary">Starting…</span>}
                {watch.status === 'reconnecting' && (
                  <span className="min-w-0 truncate text-status-warning-text" title={watch.error}>
                    Reconnecting… ({watch.error})
                  </span>
                )}
                {watch.status === 'live' && (
                  <span className="flex items-center gap-1 text-status-success-text">
                    <span className="w-1.5 h-1.5 rounded-full bg-status-success-text" aria-hidden="true" />
                    <span>Live</span>
                  </span>
                )}
              </div>
              {watch.status === 'stopped' && (
                <div className="mt-1 flex items-center gap-2 text-xs text-status-error-text">
                  <span className="min-w-0 truncate" title={watch.error}>Stopped: {watch.error}</span>
                  <Button size="sm" variant="ghost" onClick={watch.restart}>
                    Restart
                  </Button>
                </div>
              )}
              {live && (
                <Tabs
                  variant="pills"
                  label="Key list view"
                  idPrefix="kv-list"
                  className="mt-2 w-fit"
                  value={listView}
                  onChange={(view) => setListView(view as 'keys' | 'changes')}
                  tabs={[
                    { value: 'keys', label: 'Keys' },
                    { value: 'changes', label: `Changes (${watch.changes.length})` },
                  ]}
                />
              )}
            </div>

            <div className="flex-1 overflow-auto" {...(live ? tabPanelProps('kv-list', listView) : {})}>
              {live && listView === 'changes' ? (
                watch.changes.length === 0 ? (
                  <p className="p-4 text-center text-sm text-content-tertiary">
                    No changes yet. Changes made from now on show up here.
                  </p>
                ) : (
                  <ul>
                    {watch.changes.map((change) => (
                      <li key={`${change.revision}-${change.key}`} className="border-b">
                        <button
                          type="button"
                          className={`w-full text-left p-2 hover:bg-surface-secondary focus:outline-none focus-visible:ring-2 focus-visible:ring-border-focus ${
                            selectedKey === change.key ? 'bg-accent-light' : ''
                          }`}
                          onClick={() => {
                            setSelectedKey(change.key)
                            setIsCreatingKey(false)
                          }}
                        >
                          <span className="flex items-center gap-2">
                            <span className="font-mono text-sm truncate min-w-0">{change.key}</span>
                            <Badge size="sm" variant={change.operation === 'put' ? 'success' : change.operation === 'delete' ? 'warning' : 'error'}>
                              {change.operation}
                            </Badge>
                          </span>
                          <span className="block text-2xs text-content-tertiary truncate">
                            rev {change.revision} · {formatTime(change.created)}
                            {change.operation === 'put' && ` · ${changePreview(change)}`}
                          </span>
                        </button>
                      </li>
                    ))}
                  </ul>
                )
              ) : keysLoading ? (
                <div className="flex items-center justify-center p-4">
                  <Spinner size="sm" />
                </div>
              ) : keysError ? (
                <div className="p-3">
                  <Alert variant="error">
                    {getErrorMessage(keysError)}
                  </Alert>
                </div>
              ) : (
                <>
                  {filteredKeys.map((key: string) => (
                    <div
                      key={key}
                      className={`pr-2 border-b hover:bg-surface-secondary flex items-center justify-between group ${
                        selectedKey === key ? 'bg-accent-light' : ''
                      }`}
                    >
                      <button
                        type="button"
                        aria-current={selectedKey === key ? 'true' : undefined}
                        className="w-full text-left p-2 text-sm truncate flex-1 min-w-0 focus:outline-none focus-visible:ring-2 focus-visible:ring-border-focus rounded"
                        onClick={() => {
                          setSelectedKey(key)
                          setIsCreatingKey(false)
                        }}
                      >
                        {key}
                      </button>
                      {!readOnly && (
                        <div className="flex gap-1 shrink-0 opacity-0 group-hover:opacity-100 focus-within:opacity-100">
                          <Tooltip content="Purge all revisions">
                            <button
                              onClick={(e) => {
                                e.stopPropagation()
                                requestAction({
                                  type: 'purge-key',
                                  name: key,
                                  confirmText: '',
                                })
                              }}
                              className="p-1 hover:text-orange-600"
                              aria-label={`Purge all revisions of ${key}`}
                            >
                              <RefreshIcon className="w-3.5 h-3.5" />
                            </button>
                          </Tooltip>
                          <Tooltip content="Delete key">
                            <button
                              onClick={(e) => {
                                e.stopPropagation()
                                requestAction({
                                  type: 'delete-key',
                                  name: key,
                                  confirmText: '',
                                })
                              }}
                              className="p-1 hover:text-status-error-text"
                              aria-label={`Delete key ${key}`}
                            >
                              <CloseIcon className="w-3.5 h-3.5" />
                            </button>
                          </Tooltip>
                        </div>
                      )}
                    </div>
                  ))}

                  {filteredKeys.length === 0 && (
                    <div className="p-4 text-center text-sm text-content-tertiary">
                      {keySearchQuery ? `No keys match your search${partial && !keyPattern ? ' among the loaded keys' : ''}` : 'No keys in this bucket'}
                    </div>
                  )}
                </>
              )}
            </div>
          </div>

          {/* Key Editor */}
          <div className="flex-1 flex flex-col overflow-hidden">
            {isCreatingKey && !readOnly ? (
              <>
                <div className="p-4 border-b bg-surface-secondary">
                  <h3 className="font-semibold text-content-primary">Create New Key</h3>
                  <p className="text-sm text-content-tertiary mt-1">
                    Add a new key to bucket "{bucketName}"
                  </p>
                </div>

                <div className="flex-1 overflow-auto p-4 space-y-4">
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1">Key Name</label>
                    <Input
                      value={newKeyName}
                      onChange={(e) => setNewKeyName(e.target.value)}
                      placeholder="my.key.name"
                    />
                    {newKeyTarget && (
                      <p className="mt-1 text-xs text-content-tertiary" data-testid="kv-new-proto">
                        Stored as Protobuf <span className="font-mono">{newKeyTarget.messageType}</span> (mapping{' '}
                        <span className="font-mono">{newKeyTarget.pattern}</span>)
                      </p>
                    )}
                  </div>

                  {allowsKeyTtl ? (
                    <div>
                      <label htmlFor="kv-new-key-ttl" className="block text-sm font-medium text-gray-700 mb-1">TTL</label>
                      <Input
                        id="kv-new-key-ttl"
                        value={newKeyTtl}
                        onChange={(e) => setNewKeyTtl(e.target.value)}
                        placeholder="e.g. 30s, 5m or 1h; empty keeps the key"
                        error={!!keyTtl.error}
                        errorMessage={keyTtl.error}
                      />
                      {raisedTtlNs !== undefined && (
                        <p className="mt-1 text-xs text-status-warning-text">
                          This bucket&apos;s marker TTL is {formatNsDuration(raisedTtlNs)}; NATS raises shorter key TTLs to it.
                        </p>
                      )}
                    </div>
                  ) : (
                    <p className="text-xs text-content-tertiary">
                      To give keys a TTL, set a key TTL marker in the{' '}
                      <Link to={`/kv/${encodeURIComponent(bucketName)}/edit`} className="text-accent hover:text-accent-text">
                        bucket settings
                      </Link>
                      .
                    </p>
                  )}

                  <div className="flex-1 flex flex-col">
                    <label className="block text-sm font-medium text-gray-700 mb-1">Value</label>
                    <textarea
                      value={newKeyValue}
                      onChange={(e) => setNewKeyValue(e.target.value)}
                      className="flex-1 w-full p-3 font-mono text-sm border border-border-strong rounded resize-none focus:outline-none focus:ring-2 focus:ring-border-focus"
                      placeholder={newKeyTarget ? `JSON for ${newKeyTarget.messageType}` : 'Enter value (text or JSON)'}
                      spellCheck={false}
                    />
                  </div>
                </div>

                <div className="flex justify-end gap-2 p-3 border-t bg-surface-secondary">
                  <Button variant="secondary" onClick={() => setIsCreatingKey(false)}>
                    Cancel
                  </Button>
                  <Button
                    onClick={handleSaveKey}
                    disabled={putKey.isPending || !newKeyName.trim() || !!keyTtl.error}
                  >
                    {putKey.isPending ? 'Creating...' : 'Create Key'}
                  </Button>
                </div>
              </>
            ) : selectedKey ? (
              <>
                <div className="p-4 border-b bg-surface-secondary">
                  <div className="flex items-center justify-between">
                    <div>
                      <h3 className="font-semibold text-content-primary">{selectedKey}</h3>
                      <p className="text-sm text-content-tertiary mt-1">
                        {keyEntry && (
                          <span className="flex gap-2">
                            <Badge variant="default" size="sm">Rev {keyEntry.revision}</Badge>
                            <span>Last updated: {formatDateTime(keyEntry.created)}</span>
                            {keyEntry.ttl && (
                              <span data-testid="kv-key-expiry">
                                TTL {formatNsDuration(keyEntry.ttl)}, expires {formatDateTime(keyEntry.created + keyEntry.ttl / 1_000_000)}
                              </span>
                            )}
                          </span>
                        )}
                      </p>
                    </div>
                    <div className="flex gap-2">
                      <Button
                        variant="secondary"
                        size="sm"
                        onClick={() => setShowHistory(true)}
                      >
                        History
                      </Button>
                      {!readOnly && (
                        <>
                          <Button
                            variant="secondary"
                            size="sm"
                            onClick={() => requestAction({
                              type: 'purge-key',
                              name: selectedKey,
                              confirmText: '',
                            })}
                          >
                            Purge
                          </Button>
                          <Button
                            variant="danger"
                            size="sm"
                            onClick={() => requestAction({
                              type: 'delete-key',
                              name: selectedKey,
                              confirmText: '',
                            })}
                          >
                            Delete
                          </Button>
                        </>
                      )}
                    </div>
                  </div>
                </div>

                {target && keyEntry && (
                  <KVProtoBar
                    bucket={bucketName}
                    target={target}
                    decoded={keyEntry.decoded}
                    showRaw={showRaw}
                    onToggleRaw={() => setRawChoice({ key: selectedKey, revision: keyEntry.revision, on: !showRaw })}
                  />
                )}

                <div className="flex-1 overflow-auto p-4 flex flex-col gap-3">
                  {keyEntry?.ttl && !readOnly && (
                    <p className="text-xs text-content-tertiary">
                      A TTL can only be set when a key is created, so saving writes a revision without one and the key
                      will stop expiring.
                    </p>
                  )}
                  {changedWhileEditing && (
                    <Alert variant="warning">
                      This key changed on the server (revision {keyEntry.revision}) while you were editing. Saving will
                      fail; Reset loads the new value.
                    </Alert>
                  )}
                  {target && keyEntry?.decoded?.error && (
                    <Alert variant="warning">
                      The stored value does not decode as {target.messageType}: {keyEntry.decoded.error}.
                      {!showRaw && ` Saving encodes the JSON below as ${target.messageType}.`}
                    </Alert>
                  )}
                  {keyLoading ? (
                    <div className="flex items-center justify-center h-full">
                      <Spinner size="lg" />
                    </div>
                  ) : target && showRaw && keyEntry ? (
                    <WireView dataBase64={keyEntry.value} totalBytes={decodeBase64ToBytes(keyEntry.value).length} />
                  ) : (
                    <textarea
                      value={editingValue}
                      onChange={(e) => setEditingValue(e.target.value)}
                      aria-label="Key value"
                      readOnly={readOnly}
                      className="w-full flex-1 min-h-0 p-3 font-mono text-sm border border-border-strong rounded resize-none focus:outline-none focus:ring-2 focus:ring-border-focus"
                      spellCheck={false}
                    />
                  )}
                </div>

                {!readOnly && (
                  <div className="flex justify-end gap-2 p-3 border-t bg-surface-secondary">
                    <Button
                      variant="secondary"
                      onClick={() => setDraft(null)}
                    >
                      Reset
                    </Button>
                    <Button
                      onClick={handleSaveKey}
                      disabled={putKey.isPending || showRaw}
                    >
                      {putKey.isPending ? 'Saving...' : 'Save Value'}
                    </Button>
                  </div>
                )}
              </>
            ) : (
              <div className="flex-1 flex items-center justify-center text-content-tertiary">
                <div className="text-center">
                  <svg className="mx-auto h-12 w-12 text-content-muted mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1121 9z" />
                  </svg>
                  <p className="text-sm mb-4">Select a key to view/edit</p>
                  {!readOnly && (
                    <Button
                      onClick={() => {
                        setIsCreatingKey(true)
                        setSelectedKey(null)
                        setNewKeyName('')
                        setNewKeyValue('')
                        setNewKeyTtl('')
                      }}
                    >
                      <PlusIcon className="w-4 h-4 mr-2" />
                      Create New Key
                    </Button>
                  )}
                </div>
              </div>
            )}
          </div>
        </div>
      )}

      {/* Key History */}
      {showHistory && selectedKey && (
        <Modal
          isOpen={true}
          onClose={() => setShowHistory(false)}
          title={`History: ${selectedKey}`}
        >
          <Modal.Body>
            {historyLoading ? (
              <div className="flex items-center justify-center p-6">
                <Spinner size="md" />
              </div>
            ) : historyError ? (
              <Alert variant="error">{getErrorMessage(historyError)}</Alert>
            ) : !keyHistory || keyHistory.length === 0 ? (
              <p className="text-sm text-content-tertiary text-center py-6">
                No history available for this key
              </p>
            ) : (
              <div className="space-y-3 max-h-96 overflow-auto">
                {[...keyHistory].reverse().map((entry) => (
                  <div key={entry.revision} className="border rounded p-3">
                    <div className="flex items-center gap-2 mb-2">
                      <Badge variant="default" size="sm">Rev {entry.revision}</Badge>
                      <Badge
                        variant={
                          entry.operation === 'put'
                            ? 'success'
                            : entry.operation === 'delete'
                            ? 'warning'
                            : 'error'
                        }
                        size="sm"
                      >
                        {entry.operation}
                      </Badge>
                      <span className="text-xs text-content-tertiary">
                        {formatDateTime(entry.created)}
                      </span>
                    </div>
                    {entry.operation === 'put' ? (
                      <pre className="text-xs font-mono bg-surface-secondary rounded p-2 max-h-32 overflow-auto whitespace-pre-wrap break-all">
                        {entry.decoded?.data !== undefined && !entry.decoded.error
                          ? JSON.stringify(entry.decoded.data, null, 2)
                          : decodeBase64(entry.value)}
                      </pre>
                    ) : (
                      <p className="text-xs text-content-muted italic">
                        {entry.operation === 'delete' ? 'Delete marker' : 'Purge marker'}
                      </p>
                    )}
                  </div>
                ))}
              </div>
            )}
          </Modal.Body>
          <Modal.Footer>
            <Button variant="secondary" onClick={() => setShowHistory(false)}>
              Close
            </Button>
          </Modal.Footer>
        </Modal>
      )}

      {/* Confirm Dialog */}
      {confirmAction && (
        <Modal
          isOpen={true}
          onClose={() => setConfirmAction(null)}
          title={
            confirmAction.type === 'delete-bucket'
              ? 'Delete KV Store'
              : confirmAction.type === 'clear-bucket'
              ? 'Clear KV Store'
              : confirmAction.type === 'delete-key'
              ? 'Delete Key'
              : 'Purge Key'
          }
        >
          <Modal.Body>
            <Alert variant="warning">
              {confirmAction.type === 'delete-bucket' && (
                <span>This will permanently delete the KV store <strong>{confirmAction.name}</strong> and all its keys. This action cannot be undone.</span>
              )}
              {confirmAction.type === 'clear-bucket' && (
                <span>
                  This removes every key and every revision of the KV store <strong>{confirmAction.name}</strong> at once and
                  keeps the bucket. Apps watching the bucket are not told: no delete markers are left, so the values they
                  cached stay until they reload. This action cannot be undone.
                </span>
              )}
              {confirmAction.type === 'delete-key' && (
                <span>This will delete the key <strong>{confirmAction.name}</strong>. The deletion will be recorded in the key's history.</span>
              )}
              {confirmAction.type === 'purge-key' && (
                <span>This will permanently purge all revisions of the key <strong>{confirmAction.name}</strong>. This action cannot be undone.</span>
              )}
            </Alert>

            {(confirmAction.type === 'delete-bucket' || confirmAction.type === 'clear-bucket') && (
              <div className="mt-4">
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Type "{confirmAction.name}" to confirm
                </label>
                <Input
                  value={confirmAction.confirmText}
                  onChange={(e) => setConfirmAction({ ...confirmAction, confirmText: e.target.value })}
                  placeholder={confirmAction.name}
                />
              </div>
            )}
          </Modal.Body>

          <Modal.Footer>
            <Button variant="secondary" onClick={() => setConfirmAction(null)}>
              Cancel
            </Button>
            <Button
              variant="danger"
              onClick={handleConfirmAction}
              disabled={
                ((confirmAction.type === 'delete-bucket' || confirmAction.type === 'clear-bucket') &&
                  confirmAction.confirmText !== confirmAction.name) ||
                deleteBucket.isPending || purgeBucket.isPending || deleteKey.isPending || purgeKey.isPending
              }
            >
              {confirmAction.type === 'delete-bucket'
                ? deleteBucket.isPending ? 'Deleting...' : 'Delete KV Store'
                : confirmAction.type === 'clear-bucket'
                ? purgeBucket.isPending ? 'Clearing...' : 'Clear KV Store'
                : confirmAction.type === 'delete-key'
                ? deleteKey.isPending ? 'Deleting...' : 'Delete Key'
                : purgeKey.isPending ? 'Purging...' : 'Purge Key'}
            </Button>
          </Modal.Footer>
        </Modal>
      )}
    </div>
  )
}
