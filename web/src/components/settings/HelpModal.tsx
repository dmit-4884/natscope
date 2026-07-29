import type { ReactNode } from 'react'
import { CloseIcon } from '@/components/ui'
import { useDialogA11y } from '@/hooks/useDialogA11y'
import { HELP } from './helpContent'
import { HELP_PREVIEWS } from './helpPreviews'

function renderInlineMarkdown(text: string): ReactNode[] {
  const parts = text.split(/(\*\*.*?\*\*)/g)
  return parts.map((part, i) => {
    if (part.startsWith('**') && part.endsWith('**')) {
      return <strong key={i} className="font-semibold text-content-primary">{part.slice(2, -2)}</strong>
    }
    return <span key={i}>{part}</span>
  })
}

function renderHelpBody(body: string): ReactNode[] {
  const blocks = body.split(/(```[\s\S]*?```)/g)
  return blocks.map((block, i) => {
    if (block.startsWith('```') && block.endsWith('```')) {
      const code = block.slice(3, -3).replace(/^\w*\n/, '')
      return (
        <pre key={i} className="my-3 px-4 py-3 bg-surface-inverse text-gray-100 rounded-lg text-xs font-mono overflow-x-auto whitespace-pre leading-5">
          {code}
        </pre>
      )
    }
    return block.split('\n\n').map((paragraph, j) => {
      const trimmed = paragraph.trim()
      if (trimmed === '---') {
        return <hr key={`${i}-${j}`} className="my-4 border-border" />
      }
      const lines = trimmed.split('\n')
      return (
        <p key={`${i}-${j}`} className={j > 0 || i > 0 ? 'mt-3' : ''}>
          {lines.map((line, k) => (
            <span key={k}>
              {k > 0 && <br />}
              {renderInlineMarkdown(line)}
            </span>
          ))}
        </p>
      )
    })
  })
}

interface HelpModalProps {
  helpKey: string
  onClose: () => void
}

export function HelpModal({ helpKey, onClose }: HelpModalProps) {
  const help = HELP[helpKey]
  const { dialogRef, onKeyDown } = useDialogA11y<HTMLDivElement>(!!help, onClose)

  if (!help) return null

  const renderPreview = HELP_PREVIEWS[helpKey]

  return (
    <div className="fixed inset-0 z-modal-nested flex items-center justify-center bg-black/40" onClick={onClose}>
      <div
        ref={dialogRef}
        role="dialog"
        aria-modal="true"
        aria-label={help.title}
        tabIndex={-1}
        onKeyDown={onKeyDown}
        className="bg-surface-primary rounded-xl shadow-2xl max-w-2xl w-full mx-4 max-h-[80vh] flex flex-col focus:outline-none"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-center justify-between px-5 py-4 border-b border-border">
          <h3 className="text-base font-semibold text-content-primary">{help.title}</h3>
          <button onClick={onClose} aria-label="Close help" className="p-1 rounded-md hover:bg-surface-tertiary transition-colors">
            <CloseIcon className="w-5 h-5 text-content-muted" />
          </button>
        </div>
        <div className="px-5 py-4 overflow-y-auto text-sm text-gray-700 leading-relaxed">
          {renderPreview && renderPreview()}
          {renderHelpBody(help.body)}
        </div>
      </div>
    </div>
  )
}
