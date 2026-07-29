import { cn } from '@/utils/cn'

export interface ToggleProps {
  checked: boolean
  onChange: (next: boolean) => void
  disabled?: boolean
  size?: 'sm' | 'xs'
  label?: string
  testId?: string
  className?: string
}

const SIZES = {
  sm: { track: 'w-9 h-5', thumb: 'after:h-3.5 after:w-3.5', on: 'peer-checked:after:translate-x-[16px]' },
  xs: { track: 'w-7 h-4', thumb: 'after:h-2.5 after:w-2.5', on: 'peer-checked:after:translate-x-[12px]' },
} as const

export function Toggle({ checked, onChange, disabled, size = 'sm', label, testId, className }: ToggleProps) {
  const s = SIZES[size]

  return (
    <label
      className={cn(
        'relative inline-flex items-center',
        disabled ? 'opacity-50 cursor-not-allowed' : 'cursor-pointer',
        className,
      )}
    >
      <input
        type="checkbox"
        role="switch"
        className="sr-only peer"
        checked={checked}
        disabled={disabled}
        aria-label={label}
        data-testid={testId}
        onChange={(e) => onChange(e.target.checked)}
      />
      <span
        className={cn(
          'block rounded-full bg-surface-hover transition-colors',
          'peer-checked:bg-accent peer-focus-visible:ring-2 peer-focus-visible:ring-blue-300',
          s.track,
          "after:content-[''] after:absolute after:left-[3px] after:top-1/2 after:-translate-y-1/2",
          'after:rounded-full after:bg-surface-primary after:transition-transform',
          s.thumb,
          s.on,
        )}
      />
    </label>
  )
}

Toggle.displayName = 'Toggle'
