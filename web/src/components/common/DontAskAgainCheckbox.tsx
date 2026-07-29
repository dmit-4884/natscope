interface Props {
  checked: boolean
  onChange: (next: boolean) => void
  /** Optional override label. Defaults to "Don't ask again". */
  label?: string
  /**
   * Override the data-testid attribute. Defaults to "dont-ask-again".
   * Provide a unique value when two dialogs with this checkbox can appear on
   * the same page so test selectors don't collide.
   */
  testId?: string
}

/**
 * Reusable "Don't ask again" opt-out for destructive confirm dialogs.
 *
 * When checked and the operation is confirmed, the caller persists the
 * matching behavior toggle via {@link useConfirmation}().disable() so the
 * prompt is skipped next time. Only used on recoverable-scope operations —
 * irreversible-scale ones keep type-to-confirm with no opt-out.
 */
export function DontAskAgainCheckbox({ checked, onChange, label = "Don't ask again", testId = 'dont-ask-again' }: Props) {
  return (
    <label className="flex items-center gap-2 text-xs text-content-tertiary select-none cursor-pointer">
      <input
        type="checkbox"
        className="h-3.5 w-3.5 rounded border-border-strong text-accent focus:ring-border-focus"
        checked={checked}
        onChange={(e) => onChange(e.target.checked)}
        data-testid={testId}
      />
      {label}
    </label>
  )
}
