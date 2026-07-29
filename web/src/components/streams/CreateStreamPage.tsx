import { useState } from 'react'
import { useNavigate, useOutletContext } from 'react-router-dom'
import type { StreamCreateRequest } from '@/types/management'
import { useCreateStream } from '@/contexts/streams'
import { Button, JsonEditor, Tabs, tabPanelProps } from '@/components/ui'
import { StreamFormFields } from '@/components/common/forms'
import type { ConnectionOutletContext } from '../common/ConnectedLayout'
import { buildStreamCreatePayload } from './config/streamFieldDefinitions'

const defaultStreamConfig: StreamCreateRequest = {
  name: '',
  subjects: [''],
  retention: 'limits',
  storage: 'file',
  discard: 'old',
  max_msgs: -1,
  max_bytes: -1,
  max_age: 0,
  num_replicas: 1,
}

export default function CreateStreamPage() {
  const { connectionId, currentConnection } = useOutletContext<ConnectionOutletContext>()
  const navigate = useNavigate()

  const [formValue, setFormValue] = useState<StreamCreateRequest>(defaultStreamConfig)
  const [editorMode, setEditorMode] = useState<'form' | 'json'>('form')

  const createStream = useCreateStream(connectionId || undefined)

  const handleCreate = async () => {
    try {
      const result = await createStream.mutateAsync(buildStreamCreatePayload(formValue))
      // Navigate to the new stream
      navigate(`/streams/${encodeURIComponent(result.name)}/messages`)
    } catch {
      /* toasted by the mutation hook; stay on the form */
    }
  }

  const handleCancel = () => {
    navigate(`/streams`)
  }

  return (
    <div className="flex-1 bg-surface-primary flex flex-col overflow-hidden">
      {/* Header */}
      <div className="px-4 pt-4 pb-3 border-b bg-surface-secondary">
        <h3 className="font-semibold text-content-primary">Create New Stream</h3>
        <p className="text-sm text-content-tertiary mt-0.5 mb-3">
          {currentConnection?.name || currentConnection?.urls[0] || 'Unknown connection'}
        </p>

        {/* Mode Toggle Tabs */}
        <Tabs
          variant="pills"
          label="Editor mode"
          idPrefix="stream-editor"
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
      <div {...tabPanelProps('stream-editor', editorMode)} className="flex-1 overflow-auto p-6">
        {editorMode === 'form' ? (
          <div className="max-w-4xl">
            <StreamFormFields
              value={formValue}
              onChange={setFormValue}
              isEditMode={false}
              defaultExpanded={true}
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
          disabled={
            createStream.isPending ||
            !formValue.name ||
            !formValue.subjects?.some((s) => s.trim() !== '')
          }
        >
          {createStream.isPending ? 'Creating...' : 'Create Stream'}
        </Button>
      </div>
    </div>
  )
}
