import { cloneElement, isValidElement, useId, type ReactNode } from 'react'
import Tooltip from '@/components/common/Tooltip'
import { SettingsSection } from './SettingsSection'

export const numberClass =
  'w-full px-3 py-1.5 text-sm border border-border-strong rounded-md focus:ring-border-focus focus:border-border-focus'

export interface SectionProps {
  title: string
  description: string
  isOpen: boolean
  onToggle: () => void
  children: ReactNode
}

export function Section({ title, description, isOpen, onToggle, children }: SectionProps) {
  return (
    <SettingsSection
      title={title}
      description={description}
      collapsible
      isOpen={isOpen}
      onToggle={onToggle}
    >
      {children}
    </SettingsSection>
  )
}

export interface FieldProps {
  label: string
  description: string
  helpKey: string
  onHelp: (key: string) => void
  children: ReactNode
}

export function Field({ label, description, helpKey, onHelp, children }: FieldProps) {
  const id = useId()
  const canAssociate = isValidElement<{ id?: string }>(children) && children.props.id === undefined
  const control = canAssociate ? cloneElement(children, { id }) : children

  return (
    <div className="flex flex-col gap-2 sm:flex-row sm:items-start sm:justify-between sm:gap-4">
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-1.5">
          <label htmlFor={canAssociate ? id : undefined} className="text-sm font-medium text-gray-700">
            {label}
          </label>
          <Tooltip content="Learn more">
            <button
              onClick={() => onHelp(helpKey)}
              className="flex-shrink-0 w-4 h-4 rounded-full bg-surface-hover hover:bg-accent-muted text-content-tertiary hover:text-accent flex items-center justify-center transition-colors"
              aria-label={`Learn more about ${label}`}
            >
              <span className="text-2xs font-bold leading-none" aria-hidden="true">?</span>
            </button>
          </Tooltip>
        </div>
        <p className="text-xs text-content-tertiary mt-0.5">{description}</p>
      </div>
      <div className="w-full sm:w-48 shrink-0">{control}</div>
    </div>
  )
}
