import { ChevronDownIcon } from '@/components/ui'

export interface SectionPanelProps {
  label: string
  isOpen: boolean
  onToggle: () => void
  children: React.ReactNode
  badge?: React.ReactNode
}

export function SectionPanel({ label, isOpen, onToggle, children, badge }: SectionPanelProps) {
  return (
    <div className="border rounded-lg">
      <button
        type="button"
        onClick={onToggle}
        aria-expanded={isOpen}
        className="w-full px-4 py-3 flex items-center justify-between bg-surface-secondary hover:bg-surface-tertiary rounded-t-lg"
      >
        <div className="flex items-center gap-2">
          <span className="font-medium text-sm">{label}</span>
          {badge}
        </div>
        <ChevronDownIcon className={`w-5 h-5 transition-transform ${isOpen ? 'rotate-180' : ''}`} />
      </button>
      {isOpen && <div className="p-4">{children}</div>}
    </div>
  )
}
