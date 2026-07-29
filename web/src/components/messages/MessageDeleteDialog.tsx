import { useState, useEffect } from 'react'
import { Button, Modal, Alert } from '@/components/ui'
import { DontAskAgainCheckbox } from '@/components/common/DontAskAgainCheckbox'

interface Props {
  isOpen: boolean
  sequence: number
  /** Pre-selected erase mode (from behavior.secureDeleteDefault). */
  defaultSecure: boolean
  isPending: boolean
  onCancel: () => void
  onConfirm: (opts: { secure: boolean; dontAskAgain: boolean }) => void
}

/**
 * Confirm dialog for deleting a stream message: normal delete vs secure erase
 * (overwrites first, irreversible).
 */
export function MessageDeleteDialog({ isOpen, sequence, defaultSecure, isPending, onCancel, onConfirm }: Props) {
  const [secure, setSecure] = useState(defaultSecure)
  const [dontAskAgain, setDontAskAgain] = useState(false)

  // Reset on (re)open/message/default change — else a prior selection survives
  // Cancel into the next message.
  useEffect(() => {
    if (isOpen) {
      setSecure(defaultSecure)
      setDontAskAgain(false)
    }
  }, [isOpen, sequence, defaultSecure])

  if (!isOpen) return null

  return (
    <Modal isOpen={isOpen} onClose={onCancel} title="Delete Message">
      <Modal.Body>
        <div className="space-y-4">
          <Alert variant="warning">
            This will permanently delete message <strong>#{sequence}</strong> from the stream. This
            action cannot be undone.
          </Alert>

          <fieldset className="space-y-2" data-testid="delete-mode">
            <legend className="text-sm font-medium text-gray-700 mb-1">Delete mode</legend>
            <label className="flex items-start gap-2 text-sm cursor-pointer">
              <input
                type="radio"
                name="delete-mode"
                className="mt-0.5"
                checked={!secure}
                onChange={() => setSecure(false)}
                data-testid="delete-mode-normal"
              />
              <span>
                <span className="font-medium text-gray-800">Normal delete</span>
                <span className="block text-xs text-content-tertiary">
                  Removes the message. The sequence number is left as an interior gap.
                </span>
              </span>
            </label>
            <label className="flex items-start gap-2 text-sm cursor-pointer">
              <input
                type="radio"
                name="delete-mode"
                className="mt-0.5"
                checked={secure}
                onChange={() => setSecure(true)}
                data-testid="delete-mode-secure"
              />
              <span>
                <span className="font-medium text-gray-800">Secure erase</span>
                <span className="block text-xs text-content-tertiary">
                  Overwrites the message payload before removal. Slower; use for accidentally
                  published secrets.
                </span>
              </span>
            </label>
          </fieldset>

          <DontAskAgainCheckbox checked={dontAskAgain} onChange={setDontAskAgain} />
        </div>
      </Modal.Body>
      <Modal.Footer>
        <Button variant="secondary" onClick={onCancel} disabled={isPending}>
          Cancel
        </Button>
        <Button
          variant="danger"
          onClick={() => onConfirm({ secure, dontAskAgain })}
          disabled={isPending}
          data-testid="confirm-delete-message-btn"
        >
          {isPending ? 'Deleting…' : secure ? 'Secure erase' : 'Delete'}
        </Button>
      </Modal.Footer>
    </Modal>
  )
}
