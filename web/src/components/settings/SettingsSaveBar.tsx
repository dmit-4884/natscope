import { Button } from '@/components/ui'

interface Props {
  canSave: boolean
  isSaving: boolean
  isResetting: boolean
  onSave: () => void
  onReset: () => void
}

export function SettingsSaveBar({ canSave, isSaving, isResetting, onSave, onReset }: Props) {
  return (
    <div className="shrink-0 flex flex-wrap items-center justify-between gap-3 pt-4 mt-4 border-t border-border">
      <Button variant="secondary" onClick={onReset} loading={isResetting}>
        Reset to defaults
      </Button>
      <Button onClick={onSave} disabled={!canSave} loading={isSaving}>
        Save changes
      </Button>
    </div>
  )
}
