import { Button, JsonEditor } from '@/components/ui'
import { StreamFormFields } from '@/components/common/forms'
import type { StreamCreateRequest } from '@/types/management'
import { getStreamUpdateLockedKeys } from './streamFieldDefinitions'

interface Props {
  editorMode: 'form' | 'json'
  onModeChange: (mode: 'form' | 'json') => void
  value: StreamCreateRequest
  onChange: (value: StreamCreateRequest) => void
  originalValue: StreamCreateRequest | null
  isSaving: boolean
  hasChanges: boolean
  onCancel: () => void
  onSave: () => void
}

const LOCKED_UPDATE_KEYS = getStreamUpdateLockedKeys()

export function StreamConfigEditor({
  editorMode,
  onModeChange,
  value,
  onChange,
  originalValue,
  isSaving,
  hasChanges,
  onCancel,
  onSave,
}: Props) {
  return (
    <>
      <div className="flex items-center justify-between px-4 py-2 border-b bg-surface-secondary shrink-0 gap-3">
        <div className="flex gap-1 bg-surface-hover p-0.5 rounded">
          <button
            onClick={() => onModeChange('form')}
            className={`px-3 py-1.5 text-sm font-medium rounded transition-colors ${
              editorMode === 'form' ? 'bg-surface-primary text-content-primary shadow-sm' : 'text-content-secondary hover:text-content-primary'
            }`}
          >
            Form View
          </button>
          <button
            onClick={() => onModeChange('json')}
            className={`px-3 py-1.5 text-sm font-medium rounded transition-colors ${
              editorMode === 'json' ? 'bg-surface-primary text-content-primary shadow-sm' : 'text-content-secondary hover:text-content-primary'
            }`}
          >
            JSON View
          </button>
        </div>
        <div className="flex items-center gap-2">
          <span className="text-xs text-content-tertiary hidden sm:inline">
            Some fields cannot be changed after creation.
          </span>
          <Button variant="ghost" size="sm" onClick={onCancel}>
            Cancel
          </Button>
          <Button size="sm" onClick={onSave} disabled={isSaving || !hasChanges}>
            {isSaving ? 'Saving...' : 'Save Changes'}
          </Button>
        </div>
      </div>
      {editorMode === 'form' ? (
        <div className="p-4">
          <StreamFormFields
            value={value}
            onChange={onChange}
            isEditMode={true}
          />
        </div>
      ) : (
        // JSON view fills remaining width/height below the toolbar; the
        // textarea handles its own scrolling.
        <div className="flex-1 flex flex-col min-h-0 w-full px-4 pb-4 pt-3">
          <JsonEditor
            value={value}
            onChange={onChange}
            immutableFields={LOCKED_UPDATE_KEYS}
            originalValue={originalValue}
          />
        </div>
      )}
    </>
  )
}
