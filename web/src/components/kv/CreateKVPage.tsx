import { useState } from 'react'
import { useNavigate, useOutletContext } from 'react-router-dom'
import type { KVBucketConfig } from '@/types/management'
import { useCreateKVBucket } from '@/contexts/kv'
import { Button, JsonEditor, Tabs, tabPanelProps } from '@/components/ui'
import { KVBucketFormFields } from '@/components/management/kv/KVBucketFormFields'
import type { ConnectionOutletContext } from '../common/ConnectedLayout'

const defaultBucketConfig: KVBucketConfig = {
  bucket: '',
  history: 1,
  storage: 'file',
  num_replicas: 1,
}

export default function CreateKVPage() {
  const { connectionId, currentConnection } = useOutletContext<ConnectionOutletContext>()
  const navigate = useNavigate()

  const [formValue, setFormValue] = useState<KVBucketConfig>(defaultBucketConfig)
  const [editorMode, setEditorMode] = useState<'form' | 'json'>('form')

  const createBucket = useCreateKVBucket(connectionId || undefined)

  const handleCreate = async () => {
    try {
      await createBucket.mutateAsync(formValue)
      // Navigate to the new KV store
      navigate(`/kv/${encodeURIComponent(formValue.bucket)}`)
    } catch {
      /* toasted by useCreateKVBucket; stay on the form */
    }
  }

  const handleCancel = () => {
    navigate(`/kv`)
  }

  return (
    <div className="flex-1 bg-surface-primary flex flex-col overflow-hidden">
      {/* Header */}
      <div className="px-4 pt-4 pb-3 border-b bg-surface-secondary">
        <h3 className="font-semibold text-content-primary">Create New KV Store</h3>
        <p className="text-sm text-content-tertiary mt-0.5 mb-3">
          {currentConnection?.name || currentConnection?.urls[0] || 'Unknown connection'}
        </p>

        {/* Mode Toggle Tabs */}
        <Tabs
          variant="pills"
          label="Editor mode"
          idPrefix="kv-editor"
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
      <div {...tabPanelProps('kv-editor', editorMode)} className="flex-1 overflow-auto p-6">
        {editorMode === 'form' ? (
          <div className="max-w-4xl">
            <KVBucketFormFields
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
          onClick={handleCreate}
          disabled={createBucket.isPending || !formValue.bucket}
        >
          {createBucket.isPending ? 'Creating...' : 'Create KV Store'}
        </Button>
      </div>
    </div>
  )
}
