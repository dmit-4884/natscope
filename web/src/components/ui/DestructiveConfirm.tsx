import { useEffect, useState, type ReactNode } from 'react'
import { Modal } from './Modal'
import { Button } from './Button'
import { Input } from './Input'
import { Alert } from './Alert'

export interface DestructiveConfirmProps {
  /** Whether the dialog is open. */
  isOpen: boolean
  /** Title shown in the modal header. */
  title: string
  /**
   * The body of the warning. Pass a string for a simple line, or a node to
   * embed the entity name with formatting (e.g. `<strong>{name}</strong>`).
   */
  description: ReactNode
  /** Label for the confirm button. Defaults to "Delete". */
  confirmLabel?: string
  /** Label for the cancel button. Defaults to "Cancel". */
  cancelLabel?: string
  /** Visual severity: "danger" (default, red) or "warning" (amber, for
   * reversible-but-impactful actions). */
  tone?: 'danger' | 'warning'
  /** Require the user to type this exact string to unlock confirm; use for
   * irreversible deletes to prevent muscle-memory clicks. */
  requireTypedName?: string
  /** Disables both buttons and shows the confirm button as loading. */
  isPending?: boolean
  /** Disables confirm without the loading treatment, e.g. while `extra` holds invalid input. */
  confirmDisabled?: boolean
  /** Extra content rendered between the description and the confirm row. */
  extra?: ReactNode
  /** Called when the dialog should close without confirming. */
  onCancel: () => void
  /** Called when the user clicks the confirm button. */
  onConfirm: () => void
}

// Both tones render as an amber Alert; the Alert palette has no dedicated
// "destructive" variant, so severity comes from the confirm button color.
const ALERT_VARIANT = 'warning' as const

/**
 * `DestructiveConfirm` — single source of truth for "are you sure?" dialogs.
 *
 * With `requireTypedName`, the confirm button stays disabled until the typed
 * value matches exactly (case-sensitive).
 */
export function DestructiveConfirm({
  isOpen,
  title,
  description,
  confirmLabel = 'Delete',
  cancelLabel = 'Cancel',
  tone = 'danger',
  requireTypedName,
  isPending,
  confirmDisabled,
  extra,
  onCancel,
  onConfirm,
}: DestructiveConfirmProps) {
  const [typed, setTyped] = useState('')

  // Clear the typed-name input every time the dialog re-opens so a
  // previous half-finished entry doesn't pre-arm the confirm button.
  useEffect(() => {
    if (isOpen) setTyped('')
  }, [isOpen])

  const typedMatches = !requireTypedName || typed === requireTypedName
  const isConfirmDisabled = isPending || confirmDisabled || !typedMatches

  return (
    <Modal isOpen={isOpen} onClose={onCancel} title={title}>
      <Modal.Body>
        <div className="space-y-4">
          <Alert variant={ALERT_VARIANT}>{description}</Alert>
          {extra}
          {requireTypedName && (
            <div>
              <label htmlFor="destructive-confirm-typed" className="block text-sm font-medium text-gray-700 mb-1">
                Type <span className="font-mono">{requireTypedName}</span> to confirm
              </label>
              <Input
                id="destructive-confirm-typed"
                value={typed}
                onChange={(e) => setTyped(e.target.value)}
                placeholder={requireTypedName}
                autoComplete="off"
                autoFocus
              />
            </div>
          )}
        </div>
      </Modal.Body>
      <Modal.Footer>
        <Button variant="secondary" onClick={onCancel} disabled={isPending}>
          {cancelLabel}
        </Button>
        <Button
          variant={tone === 'danger' ? 'danger' : 'primary'}
          onClick={onConfirm}
          disabled={isConfirmDisabled}
          loading={isPending}
        >
          {confirmLabel}
        </Button>
      </Modal.Footer>
    </Modal>
  )
}
