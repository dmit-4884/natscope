import { useState, useEffect } from 'react'
import { useParams, useNavigate, useOutletContext } from 'react-router-dom'
import {
  useKVBuckets,
  useKVKeys,
  useKVKey,
  useKVKeyHistory,
  usePutKVKey,
  useDeleteKVKey,
  usePurgeKVKey,
  useDeleteKVBucket,
} from '@/contexts/kv'
import { decodeBase64 } from '@/api/management'
import { getErrorMessage } from '@/api/errors'
import { formatBytes, formatDateTime } from '@/utils/formatters'
import { plural } from '@/utils/plural'
import { useConfirmation } from '@/contexts/settings'
import { Button, Modal, Input, Badge, Alert, Spinner, SearchInput, CloseIcon, PlusIcon, RefreshIcon, OverflowMenu } from '@/components/ui'
import Tooltip from '../common/Tooltip'
import type { ConnectionOutletContext } from '../common/ConnectedLayout'

type KVConfirmAction = {
  type: 'delete-bucket' | 'delete-key' | 'purge-key'
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
  const [editingValue, setEditingValue] = useState('')
  const [valueDirty, setValueDirty] = useState(false)
  const [showHistory, setShowHistory] = useState(false)
  const [confirmAction, setConfirmAction] = useState<KVConfirmAction | null>(null)

  const deleteKeyConfirmation = useConfirmation('deleteKvKey')
  const purgeKeyConfirmation = useConfirmation('purgeKvHistory')

  // Fetch bucket info
  const { data: buckets = [] } = useKVBuckets(connectionId)
  const bucketInfo = buckets.find(b => b.bucket === bucketName)

  // Fetch keys for bucket
  const { data: keys = [], isLoading: keysLoading, error: keysError, refetch: refetchKeys } = useKVKeys(
    connectionId,
    bucketName
  )

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

  // Mutations
  const deleteBucket = useDeleteKVBucket(connectionId)
  const putKey = usePutKVKey(connectionId, bucketName)
  const deleteKey = useDeleteKVKey(connectionId, bucketName)
  const purgeKey = usePurgeKVKey(connectionId, bucketName)

  // Filter keys
  const filteredKeys = (keys ?? []).filter((k: string) =>
    k.toLowerCase().includes(keySearchQuery.toLowerCase())
  )

  useEffect(() => {
    if (keyEntry && !isCreatingKey && !valueDirty) {
      setEditingValue(decodeBase64(keyEntry.value))
    }
  }, [keyEntry, isCreatingKey, valueDirty])

  useEffect(() => {
    setValueDirty(false)
    setEditingValue('')
  }, [selectedKey])

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
        if (!newKeyName.trim()) return
        await putKey.mutateAsync({ key: newKeyName, value: newKeyValue })
        setIsCreatingKey(false)
        setNewKeyName('')
        setNewKeyValue('')
      } else if (selectedKey) {
        await putKey.mutateAsync({ key: selectedKey, value: editingValue })
        setValueDirty(false)
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
                  | {plural(bucketInfo.values, 'value')} | {formatBytes(bucketInfo.bytes)}
                </span>
              )}
            </p>
          </div>
          <OverflowMenu
            label="Bucket actions"
            items={[
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
        </div>
      </div>

      {/* Main content */}
      <div className="flex-1 flex min-h-0">
        {/* Keys List */}
        <div className="w-72 border-r bg-surface-primary flex flex-col">
          <div className="p-3 border-b">
            <SearchInput
              placeholder="Search keys..."
              value={keySearchQuery}
              onChange={setKeySearchQuery}
              debounce={200}
              size="sm"
              className="mb-2"
            />
            <div className="flex items-center justify-between">
              <span className="text-xs text-content-tertiary">
                {plural(filteredKeys.length, 'key')}
              </span>
              <Button
                size="sm"
                variant="ghost"
                aria-label="New key"
                onClick={() => {
                  setIsCreatingKey(true)
                  setSelectedKey(null)
                  setNewKeyName('')
                  setNewKeyValue('')
                }}
              >
                <PlusIcon className="w-4 h-4" />
              </Button>
            </div>
          </div>

          <div className="flex-1 overflow-auto">
            {keysLoading ? (
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
                    className={`p-2 border-b cursor-pointer hover:bg-surface-secondary flex items-center justify-between group ${
                      selectedKey === key ? 'bg-accent-light' : ''
                    }`}
                    onClick={() => {
                      setSelectedKey(key)
                      setIsCreatingKey(false)
                    }}
                  >
                    <span className="text-sm truncate flex-1">{key}</span>
                    <div className="flex gap-1 opacity-0 group-hover:opacity-100">
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
                  </div>
                ))}

                {filteredKeys.length === 0 && (
                  <div className="p-4 text-center text-sm text-content-tertiary">
                    {keySearchQuery ? 'No keys match your search' : 'No keys in this bucket'}
                  </div>
                )}
              </>
            )}
          </div>
        </div>

        {/* Key Editor */}
        <div className="flex-1 flex flex-col overflow-hidden">
          {isCreatingKey ? (
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
                </div>

                <div className="flex-1 flex flex-col">
                  <label className="block text-sm font-medium text-gray-700 mb-1">Value</label>
                  <textarea
                    value={newKeyValue}
                    onChange={(e) => setNewKeyValue(e.target.value)}
                    className="flex-1 w-full p-3 font-mono text-sm border border-border-strong rounded resize-none focus:outline-none focus:ring-2 focus:ring-border-focus"
                    placeholder="Enter value (text or JSON)"
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
                  disabled={putKey.isPending || !newKeyName.trim()}
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
                  </div>
                </div>
              </div>

              <div className="flex-1 overflow-auto p-4">
                {keyLoading ? (
                  <div className="flex items-center justify-center h-full">
                    <Spinner size="lg" />
                  </div>
                ) : (
                  <textarea
                    value={editingValue}
                    onChange={(e) => {
                      setEditingValue(e.target.value)
                      setValueDirty(true)
                    }}
                    aria-label="Key value"
                    className="w-full h-full p-3 font-mono text-sm border border-border-strong rounded resize-none focus:outline-none focus:ring-2 focus:ring-border-focus"
                    spellCheck={false}
                  />
                )}
              </div>

              <div className="flex justify-end gap-2 p-3 border-t bg-surface-secondary">
                <Button
                  variant="secondary"
                  onClick={() => {
                    if (keyEntry) {
                      setEditingValue(decodeBase64(keyEntry.value))
                      setValueDirty(false)
                    }
                  }}
                >
                  Reset
                </Button>
                <Button
                  onClick={handleSaveKey}
                  disabled={putKey.isPending}
                >
                  {putKey.isPending ? 'Saving...' : 'Save Value'}
                </Button>
              </div>
            </>
          ) : (
            <div className="flex-1 flex items-center justify-center text-content-tertiary">
              <div className="text-center">
                <svg className="mx-auto h-12 w-12 text-content-muted mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1121 9z" />
                </svg>
                <p className="text-sm mb-4">Select a key to view/edit</p>
                <Button
                  onClick={() => {
                    setIsCreatingKey(true)
                    setSelectedKey(null)
                    setNewKeyName('')
                    setNewKeyValue('')
                  }}
                >
                  <PlusIcon className="w-4 h-4 mr-2" />
                  Create New Key
                </Button>
              </div>
            </div>
          )}
        </div>
      </div>

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
                        {decodeBase64(entry.value)}
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
              {confirmAction.type === 'delete-key' && (
                <span>This will delete the key <strong>{confirmAction.name}</strong>. The deletion will be recorded in the key's history.</span>
              )}
              {confirmAction.type === 'purge-key' && (
                <span>This will permanently purge all revisions of the key <strong>{confirmAction.name}</strong>. This action cannot be undone.</span>
              )}
            </Alert>

            {confirmAction.type === 'delete-bucket' && (
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
                (confirmAction.type === 'delete-bucket' && confirmAction.confirmText !== confirmAction.name) ||
                deleteBucket.isPending || deleteKey.isPending || purgeKey.isPending
              }
            >
              {confirmAction.type === 'delete-bucket'
                ? deleteBucket.isPending ? 'Deleting...' : 'Delete KV Store'
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
