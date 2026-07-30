import { Button, JsonEditor, ClipboardIcon, Tabs, tabPanelProps } from '@/components/ui'
import { ConsumerFormFields } from '@/components/common/forms'
import type { ConsumerCreateRequest } from '@/types/management'
import { CONSUMER_IMMUTABLE_FIELDS } from '@/types/management'

type EditorMode = 'form' | 'json'

const EDITOR_TABS_PREFIX = 'consumer-editor'

interface Props {
  title: string
  subtitle: string
  mode: EditorMode
  onModeChange: (mode: EditorMode) => void
  value: ConsumerCreateRequest
  onChange: (value: ConsumerCreateRequest) => void
  originalValue?: ConsumerCreateRequest | null
  isEditMode: boolean
  hasChanges?: boolean
  isSaving: boolean
  onCancel: () => void
  onShowDiff?: () => void
  onCreate?: () => void
}

export function ConsumerEditor({
  title,
  subtitle,
  mode,
  onModeChange,
  value,
  onChange,
  originalValue,
  isEditMode,
  hasChanges,
  isSaving,
  onCancel,
  onShowDiff,
  onCreate,
}: Props) {
  return (
    <>
      <div className="p-4 border-b bg-surface-secondary shrink-0">
        <div className="flex items-center justify-between">
          <div>
            <h3 className="font-semibold text-content-primary">{title}</h3>
            <p className="text-sm text-content-tertiary mt-1">{subtitle}</p>
          </div>
          {isEditMode && onShowDiff && (
            <div className="flex gap-2">
              <Button variant="secondary" size="sm" onClick={onCancel}>
                Cancel
              </Button>
              {hasChanges && (
                <Button variant="secondary" size="sm" onClick={onShowDiff}>
                  <ClipboardIcon className="w-4 h-4 mr-1.5" />
                  Show Diff
                </Button>
              )}
              <Button size="sm" onClick={onShowDiff} disabled={isSaving || !hasChanges}>
                {isSaving ? 'Saving...' : 'Save Changes'}
              </Button>
            </div>
          )}
        </div>
      </div>

      <div className="flex items-center justify-between p-3 border-b bg-surface-secondary shrink-0">
        <Tabs
          variant="pills"
          label="Editor mode"
          idPrefix={EDITOR_TABS_PREFIX}
          className="w-fit"
          value={mode}
          onChange={(next) => onModeChange(next as EditorMode)}
          tabs={[
            { value: 'form', label: 'Form View' },
            { value: 'json', label: 'JSON View' },
          ]}
        />
      </div>

      <div
        {...tabPanelProps(EDITOR_TABS_PREFIX, 'form')}
        hidden={mode !== 'form'}
        className={mode === 'form' ? 'flex-1 overflow-auto p-4 min-h-0' : undefined}
      >
        <ConsumerFormFields
          value={value}
          onChange={onChange}
          isEditMode={isEditMode}
          immutableFields={isEditMode ? (CONSUMER_IMMUTABLE_FIELDS as unknown as string[]) : undefined}
        />
      </div>
      <div
        {...tabPanelProps(EDITOR_TABS_PREFIX, 'json')}
        hidden={mode !== 'json'}
        className={mode === 'json' ? 'flex-1 flex flex-col p-4 min-h-0' : undefined}
      >
        <JsonEditor
          value={value}
          onChange={onChange}
          immutableFields={isEditMode ? CONSUMER_IMMUTABLE_FIELDS : undefined}
          originalValue={originalValue ?? undefined}
        />
      </div>

      {!isEditMode && onCreate && (
        <div className="flex justify-end gap-2 p-3 border-t bg-surface-secondary shrink-0">
          <Button variant="secondary" onClick={onCancel}>
            Cancel
          </Button>
          <Button onClick={onCreate} disabled={isSaving}>
            {isSaving ? 'Creating...' : 'Create Consumer'}
          </Button>
        </div>
      )}
    </>
  )
}
