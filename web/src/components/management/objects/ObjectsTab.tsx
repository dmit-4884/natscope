import { useState, useEffect, useMemo, useRef } from 'react'
import { useParams, useNavigate, useOutletContext } from 'react-router-dom'
import type { ObjectBucketConfig, ObjectBucketInfo, ObjectInfo } from '@/types/management'
import {
  useObjectBuckets,
  useCreateObjectBucket,
  useDeleteObjectBucket,
  useSealObjectBucket,
  useObjects,
  useObject,
  usePutObject,
  useDeleteObject,
} from '@/contexts/objects'
import { decodeBase64ToBytes } from '@/utils/base64'
import { toast } from '@/utils/toast'
import { plural } from '@/utils/plural'
import { formatBytes, formatDateTime } from '@/utils/formatters'
import { useConfirmation } from '@/contexts/settings'
import { Button, Modal, Input, Badge, Alert, Spinner, JsonEditor, CloseIcon, Tabs, tabPanelProps, OverflowMenu } from '@/components/ui'
import type { ConnectionOutletContext } from '@/components/common/ConnectedLayout'
import Tooltip from '@/components/common/Tooltip'
import { ObjectBucketFormFields } from './ObjectBucketFormFields'

const defaultBucketConfig: ObjectBucketConfig = {
  bucket: '',
  storage: 'file',
  num_replicas: 1,
}

interface ObjectsTabProps {
  createMode?: boolean
}

type ObjectConfirmAction = {
  type: 'delete-bucket' | 'seal-bucket' | 'delete-object'
  name: string
  confirmText: string
}

function ObjectsTab({ createMode = false }: ObjectsTabProps) {
  const { bucketName } = useParams<{ bucketName: string }>()
  const navigate = useNavigate()
  const { connectionId, currentConnection } = useOutletContext<ConnectionOutletContext>()
  const fileInputRef = useRef<HTMLInputElement>(null)

  const [selectedObject, setSelectedObject] = useState<ObjectInfo | null>(null)
  const [objectSearchQuery, setObjectSearchQuery] = useState('')
  const [formValue, setFormValue] = useState<ObjectBucketConfig>(defaultBucketConfig)
  const [editorMode, setEditorMode] = useState<'form' | 'json'>('form')
  const [uploadDescription, setUploadDescription] = useState('')
  const [confirmAction, setConfirmAction] = useState<ObjectConfirmAction | null>(null)

  const deleteObjectConfirmation = useConfirmation('deleteObject')

  // Fetch buckets to get bucket info
  const { data: buckets = [], refetch: refetchBuckets } = useObjectBuckets(connectionId)
  const selectedBucket = bucketName
    ? buckets.find((b: ObjectBucketInfo) => b.bucket === bucketName) || null
    : null

  // Fetch objects for selected bucket
  const { data: objects = [], isLoading: objectsLoading, refetch: refetchObjects } = useObjects(
    connectionId,
    bucketName
  )

  // Fetch selected object data
  const { data: objectData, isLoading: objectLoading } = useObject(
    connectionId,
    bucketName,
    selectedObject?.name
  )

  // Mutations
  const createBucket = useCreateObjectBucket(connectionId)
  const deleteBucket = useDeleteObjectBucket(connectionId)
  const sealBucket = useSealObjectBucket(connectionId)
  const putObject = usePutObject(connectionId, bucketName)
  const deleteObject = useDeleteObject(connectionId, bucketName)

  // Filter objects
  const filteredObjects = (objects ?? []).filter((o: ObjectInfo) =>
    o.name.toLowerCase().includes(objectSearchQuery.toLowerCase())
  )

  // Reset selection when bucket changes
  useEffect(() => {
    setSelectedObject(null)
    setObjectSearchQuery('')
  }, [bucketName])

  const previewContent = useMemo(() => {
    if (!objectData) return null
    try {
      const bytes = decodeBase64ToBytes(objectData.data)
      if (bytes.length >= 100000) return null
      return new TextDecoder('utf-8', { fatal: true }).decode(bytes)
    } catch {
      return null
    }
  }, [objectData])

  const handleCreateBucket = async () => {
    try {
      const result = await createBucket.mutateAsync(formValue)
      navigate(`/objects/${encodeURIComponent(result.bucket)}`)
    } catch {
      /* toasted by useCreateObjectBucket */
    }
  }

  const handleCancel = () => {
    navigate(`/objects`)
  }

  // Handle file upload
  const handleFileUpload = async (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0]
    if (!file || !bucketName) return

    const reader = new FileReader()
    reader.onload = async () => {
      const arrayBuffer = reader.result as ArrayBuffer
      const uint8Array = new Uint8Array(arrayBuffer)
      try {
        await putObject.mutateAsync({
          name: file.name,
          data: uint8Array,
          options: uploadDescription ? { description: uploadDescription } : undefined,
        })
        setUploadDescription('')
        refetchObjects()
      } catch {
        /* toasted by usePutObject */
      }
    }
    reader.onerror = () => {
      toast.error(`Failed to read ${file.name}`)
    }
    reader.readAsArrayBuffer(file)

    if (fileInputRef.current) {
      fileInputRef.current.value = ''
    }
  }

  // Handle download
  const handleDownload = () => {
    if (!objectData || !selectedObject) return
    const blob = new Blob([decodeBase64ToBytes(objectData.data)], { type: 'application/octet-stream' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = selectedObject.name
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)
  }

  const performAction = async (action: ObjectConfirmAction) => {
    try {
      if (action.type === 'delete-bucket') {
        await deleteBucket.mutateAsync(action.name)
        navigate(`/objects`)
        refetchBuckets()
      } else if (action.type === 'seal-bucket') {
        await sealBucket.mutateAsync(action.name)
        refetchBuckets()
      } else if (action.type === 'delete-object') {
        await deleteObject.mutateAsync(action.name)
        if (selectedObject?.name === action.name) {
          setSelectedObject(null)
        }
        refetchObjects()
      }
    } catch {
      /* toasted by the mutation hooks */
    } finally {
      setConfirmAction(null)
    }
  }

  const requestAction = (action: ObjectConfirmAction) => {
    if (action.type === 'delete-object' && !deleteObjectConfirmation.enabled) {
      void performAction(action)
      return
    }
    setConfirmAction(action)
  }

  const handleConfirmAction = () => {
    if (!confirmAction) return
    void performAction(confirmAction)
  }

  if (!connectionId) return null

  // Create mode
  if (createMode) {
    return (
      <div className="flex-1 bg-surface-primary flex flex-col overflow-hidden">
        {/* Header */}
        <div className="px-4 pt-4 pb-3 border-b bg-surface-secondary">
          <h3 className="font-semibold text-content-primary">Create New Object Store</h3>
          <p className="text-sm text-content-tertiary mt-0.5 mb-3">
            {currentConnection?.name || currentConnection?.urls[0] || 'Unknown connection'}
          </p>

          {/* Mode Toggle Tabs */}
          <Tabs
            variant="pills"
            label="Editor mode"
            idPrefix="object-editor"
            className="w-fit"
            value={editorMode}
            onChange={(mode) => setEditorMode(mode as 'form' | 'json')}
            tabs={[
              { value: 'form', label: 'Form View' },
              { value: 'json', label: 'JSON View' },
            ]}
          />
        </div>

        {/* Form Content */}
        <div {...tabPanelProps('object-editor', editorMode)} className="flex-1 overflow-auto p-6">
          {editorMode === 'form' ? (
            <div className="max-w-4xl">
              <ObjectBucketFormFields
                value={formValue}
                onChange={setFormValue}
                isEditMode={false}
              />
            </div>
          ) : (
            <div className="h-full max-w-4xl">
              <JsonEditor
                value={formValue}
                onChange={setFormValue}
              />
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="flex justify-end gap-3 px-6 py-4 border-t bg-surface-secondary">
          <Button variant="secondary" onClick={handleCancel}>
            Cancel
          </Button>
          <Button
            onClick={handleCreateBucket}
            disabled={createBucket.isPending || !formValue.bucket}
          >
            {createBucket.isPending ? 'Creating...' : 'Create Object Store'}
          </Button>
        </div>
      </div>
    )
  }

  // Bucket view
  return (
    <div className="flex-1 flex flex-col overflow-hidden bg-surface-primary">
      {/* Header */}
      <div className="px-4 py-3 border-b bg-surface-secondary">
        <div className="flex items-center justify-between">
          <div>
            <div className="flex items-center gap-2">
              <h3 className="font-semibold text-content-primary">{bucketName}</h3>
              {selectedBucket?.sealed && (
                <Badge variant="warning" size="sm">Sealed</Badge>
              )}
            </div>
            <p className="text-sm text-content-tertiary mt-0.5">
              {currentConnection?.name || currentConnection?.urls[0] || 'Unknown connection'}
              {selectedBucket && (
                <span className="ml-2">
                  | {plural(selectedBucket.objects, 'object')} | {formatBytes(selectedBucket.size)}
                </span>
              )}
            </p>
          </div>
          <OverflowMenu
            label="Bucket actions"
            items={[
              ...(selectedBucket && !selectedBucket.sealed
                ? [{
                    label: 'Seal bucket…',
                    onSelect: () => setConfirmAction({
                      type: 'seal-bucket' as const,
                      name: bucketName!,
                      confirmText: '',
                    }),
                  }]
                : []),
              {
                label: 'Delete bucket…',
                destructive: true,
                onSelect: () => setConfirmAction({
                  type: 'delete-bucket',
                  name: bucketName!,
                  confirmText: '',
                }),
              },
            ]}
          />
        </div>
      </div>

      {/* Main content */}
      <div className="flex-1 flex min-h-0">
        {/* Objects List */}
        <div className="w-72 border-r bg-surface-primary flex flex-col">
          <div className="p-3 border-b">
            <div className="flex gap-2 mb-2">
              <Input
                placeholder="Search objects..."
                value={objectSearchQuery}
                onChange={(e) => setObjectSearchQuery(e.target.value)}
                className="flex-1"
              />
              <input
                ref={fileInputRef}
                type="file"
                onChange={handleFileUpload}
                className="hidden"
                disabled={selectedBucket?.sealed}
              />
              <Tooltip content={selectedBucket?.sealed ? 'Object store is sealed' : 'Upload file'}>
                <Button
                  onClick={() => fileInputRef.current?.click()}
                  size="sm"
                  disabled={selectedBucket?.sealed}
                  aria-label={selectedBucket?.sealed ? 'Object store is sealed' : 'Upload file'}
                >
                  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12" />
                  </svg>
                </Button>
              </Tooltip>
            </div>
            <div className="text-xs text-content-tertiary">
              {plural(filteredObjects.length, 'object')}
            </div>
          </div>

          <div className="flex-1 overflow-auto">
            {objectsLoading ? (
              <div className="flex items-center justify-center p-4">
                <Spinner size="sm" />
              </div>
            ) : (
              <>
                {filteredObjects.map((obj: ObjectInfo) => (
                  <div
                    key={obj.nuid}
                    className={`p-2 border-b cursor-pointer hover:bg-surface-secondary group ${
                      selectedObject?.name === obj.name ? 'bg-accent-light' : ''
                    }`}
                    onClick={() => setSelectedObject(obj)}
                  >
                    <div className="flex items-center justify-between mb-1">
                      <span className="text-sm truncate flex-1" title={obj.name}>{obj.name}</span>
                      <Tooltip content="Delete object">
                        <button
                          onClick={(e) => {
                            e.stopPropagation()
                            requestAction({
                              type: 'delete-object',
                              name: obj.name,
                              confirmText: '',
                            })
                          }}
                          className="p-1 opacity-0 group-hover:opacity-100 hover:text-status-error-text"
                          aria-label={`Delete object ${obj.name}`}
                        >
                          <CloseIcon className="w-3.5 h-3.5" />
                        </button>
                      </Tooltip>
                    </div>
                    <div className="text-xs text-content-tertiary flex gap-2">
                      <span>{formatBytes(obj.size)}</span>
                      <span>{obj.chunks} chunks</span>
                    </div>
                  </div>
                ))}

                {filteredObjects.length === 0 && (
                  <div className="p-4 text-center text-sm text-content-tertiary">
                    {objectSearchQuery ? 'No objects match your search' : 'No objects in this store'}
                  </div>
                )}
              </>
            )}
          </div>
        </div>

        {/* Object Preview */}
        <div className="flex-1 flex flex-col overflow-hidden">
          {selectedObject ? (
            <>
              <div className="p-4 border-b bg-surface-secondary">
                <div className="flex items-center justify-between">
                  <div>
                    <h3 className="font-semibold text-content-primary">{selectedObject.name}</h3>
                    <p className="text-sm text-content-tertiary mt-1">
                      {formatBytes(selectedObject.size)} | {selectedObject.chunks} chunks | Modified: {formatDateTime(selectedObject.mod_time)}
                    </p>
                  </div>
                  <div className="flex gap-2">
                    <Button variant="secondary" size="sm" onClick={handleDownload}>
                      <svg className="w-4 h-4 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
                      </svg>
                      Download
                    </Button>
                    <Button
                      variant="danger"
                      size="sm"
                      onClick={() => requestAction({
                        type: 'delete-object',
                        name: selectedObject.name,
                        confirmText: '',
                      })}
                    >
                      Delete
                    </Button>
                  </div>
                </div>
              </div>

              <div className="flex-1 overflow-auto p-4">
                {objectLoading ? (
                  <div className="flex items-center justify-center h-full">
                    <Spinner size="lg" />
                  </div>
                ) : previewContent !== null ? (
                  <pre className="w-full h-full p-3 font-mono text-sm border border-border-strong rounded bg-surface-secondary overflow-auto whitespace-pre-wrap">
                    {previewContent}
                  </pre>
                ) : (
                  <div className="flex items-center justify-center h-full text-content-tertiary">
                    <div className="text-center">
                      <svg className="mx-auto h-12 w-12 text-content-muted mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
                      </svg>
                      <p className="text-sm">Binary content - download to view</p>
                      <p className="text-xs text-content-muted mt-1">Digest: {selectedObject.digest}</p>
                    </div>
                  </div>
                )}
              </div>

              {selectedObject.description && (
                <div className="p-3 border-t bg-surface-secondary">
                  <p className="text-sm text-content-secondary">{selectedObject.description}</p>
                </div>
              )}
            </>
          ) : (
            <div className="flex-1 flex items-center justify-center text-content-tertiary">
              <div className="text-center">
                <svg className="mx-auto h-12 w-12 text-content-muted mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M7 21h10a2 2 0 002-2V9.414a1 1 0 00-.293-.707l-5.414-5.414A1 1 0 0012.586 3H7a2 2 0 00-2 2v14a2 2 0 002 2z" />
                </svg>
                <p className="text-sm mb-4">Select an object to view or upload a file</p>
                {!selectedBucket?.sealed && (
                  <div>
                    <Input
                      placeholder="Description (optional)"
                      value={uploadDescription}
                      onChange={(e) => setUploadDescription(e.target.value)}
                      className="mb-2"
                    />
                    <Button onClick={() => fileInputRef.current?.click()}>
                      <svg className="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12" />
                      </svg>
                      Upload File
                    </Button>
                  </div>
                )}
              </div>
            </div>
          )}
        </div>
      </div>

      {/* Confirm Dialog */}
      {confirmAction && (
        <Modal
          isOpen={true}
          onClose={() => setConfirmAction(null)}
          title={
            confirmAction.type === 'delete-bucket'
              ? 'Delete Object Store'
              : confirmAction.type === 'seal-bucket'
              ? 'Seal Object Store'
              : 'Delete Object'
          }
        >
          <Modal.Body>
            <Alert variant="warning">
              {confirmAction.type === 'delete-bucket' && (
                <span>This will permanently delete the object store <strong>{confirmAction.name}</strong> and all its objects. This action cannot be undone.</span>
              )}
              {confirmAction.type === 'seal-bucket' && (
                <span>This will seal the object store <strong>{confirmAction.name}</strong>, making it read-only. This action cannot be undone.</span>
              )}
              {confirmAction.type === 'delete-object' && (
                <span>This will delete the object <strong>{confirmAction.name}</strong>. This action cannot be undone.</span>
              )}
            </Alert>

            {(confirmAction.type === 'delete-bucket' || confirmAction.type === 'seal-bucket') && (
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
              variant={confirmAction.type === 'seal-bucket' ? 'primary' : 'danger'}
              onClick={handleConfirmAction}
              disabled={
                ((confirmAction.type === 'delete-bucket' || confirmAction.type === 'seal-bucket') &&
                  confirmAction.confirmText !== confirmAction.name) ||
                deleteBucket.isPending || sealBucket.isPending || deleteObject.isPending
              }
            >
              {confirmAction.type === 'delete-bucket'
                ? deleteBucket.isPending ? 'Deleting...' : 'Delete Object Store'
                : confirmAction.type === 'seal-bucket'
                ? sealBucket.isPending ? 'Sealing...' : 'Seal Object Store'
                : deleteObject.isPending ? 'Deleting...' : 'Delete Object'}
            </Button>
          </Modal.Footer>
        </Modal>
      )}
    </div>
  )
}

export default ObjectsTab
