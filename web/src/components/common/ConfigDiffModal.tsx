import { useMemo, type ReactNode } from 'react'
import * as Diff from 'diff'
import { Modal, Button } from '@/components/ui'
import { stableJson } from '@/utils/stableJson'

interface ConfigDiffModalProps {
  isOpen: boolean
  onClose: () => void
  onConfirm: () => void
  title: string
  description?: string
  originalConfig: unknown
  newConfig: unknown
  notice?: ReactNode
  isLoading?: boolean
}

interface DiffLine {
  type: 'unchanged' | 'added' | 'removed'
  content: string
  oldLineNum?: number
  newLineNum?: number
}

export default function ConfigDiffModal({
  isOpen,
  onClose,
  onConfirm,
  title,
  description,
  originalConfig,
  newConfig,
  notice,
  isLoading = false,
}: ConfigDiffModalProps) {
  const { leftLines, rightLines, hasChanges } = useMemo(() => {
    const originalJson = stableJson(originalConfig, 2)
    const newJson = stableJson(newConfig, 2)

    const changes = Diff.diffLines(originalJson, newJson)

    const left: DiffLine[] = []
    const right: DiffLine[] = []
    let oldLineNum = 1
    let newLineNum = 1

    changes.forEach((change) => {
      // Handle trailing newline
      const actualLines = change.value.endsWith('\n')
        ? change.value.slice(0, -1).split('\n')
        : change.value.split('\n')

      if (change.removed) {
        actualLines.forEach((line) => {
          left.push({ type: 'removed', content: line, oldLineNum: oldLineNum++ })
          right.push({ type: 'removed', content: '', newLineNum: undefined })
        })
      } else if (change.added) {
        actualLines.forEach((line) => {
          left.push({ type: 'added', content: '', oldLineNum: undefined })
          right.push({ type: 'added', content: line, newLineNum: newLineNum++ })
        })
      } else {
        actualLines.forEach((line) => {
          left.push({ type: 'unchanged', content: line, oldLineNum: oldLineNum++ })
          right.push({ type: 'unchanged', content: line, newLineNum: newLineNum++ })
        })
      }
    })

    const hasChanges = changes.some(c => c.added || c.removed)

    return { leftLines: left, rightLines: right, hasChanges }
  }, [originalConfig, newConfig])

  const getLineClass = (type: DiffLine['type'], side: 'left' | 'right') => {
    if (type === 'removed') {
      return side === 'left' ? 'bg-status-error-bg text-red-800' : 'bg-status-error-bg/30'
    }
    if (type === 'added') {
      return side === 'right' ? 'bg-status-success-bg text-green-800' : 'bg-status-success-bg/30'
    }
    return ''
  }

  const getGutterClass = (type: DiffLine['type'], side: 'left' | 'right') => {
    if (type === 'removed') {
      return side === 'left' ? 'bg-status-error-light text-status-error-text' : 'bg-status-error-bg/50 text-red-300'
    }
    if (type === 'added') {
      return side === 'right' ? 'bg-status-success-light text-status-success-text' : 'bg-status-success-bg/50 text-green-300'
    }
    return 'bg-surface-secondary text-content-muted'
  }

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title={title}
      size="xl"
    >
      <Modal.Body className="p-0">
        <div className="px-6 pt-4 pb-3">
          {description && (
            <p className="text-sm text-content-secondary mb-4">{description}</p>
          )}

          {notice && <div className="mb-4">{notice}</div>}

          {!hasChanges ? (
            <div className="text-center py-8 text-content-tertiary">
              No changes detected
            </div>
          ) : (
            <>
              <div className="text-xs font-medium text-content-tertiary mb-2">
                Configuration (Original → New)
              </div>

              <div className="border rounded-lg overflow-hidden">
                <div className="flex max-h-[60vh] overflow-auto">
                  {/* Left side (Original) */}
                  <div className="flex-1 border-r overflow-x-auto">
                    <div className="min-w-fit">
                      {leftLines.map((line, idx) => (
                        <div key={idx} className={`flex text-xs font-mono ${getLineClass(line.type, 'left')}`}>
                          <div className={`w-10 flex-shrink-0 px-2 py-0.5 text-right select-none ${getGutterClass(line.type, 'left')}`}>
                            {line.oldLineNum ?? ''}
                          </div>
                          <div className={`w-5 flex-shrink-0 text-center py-0.5 select-none ${getGutterClass(line.type, 'left')}`}>
                            {line.type === 'removed' ? '−' : ''}
                          </div>
                          <div className="px-2 py-0.5 whitespace-pre flex-1">
                            {line.content}
                          </div>
                        </div>
                      ))}
                    </div>
                  </div>

                  {/* Right side (New) */}
                  <div className="flex-1 overflow-x-auto">
                    <div className="min-w-fit">
                      {rightLines.map((line, idx) => (
                        <div key={idx} className={`flex text-xs font-mono ${getLineClass(line.type, 'right')}`}>
                          <div className={`w-10 flex-shrink-0 px-2 py-0.5 text-right select-none ${getGutterClass(line.type, 'right')}`}>
                            {line.newLineNum ?? ''}
                          </div>
                          <div className={`w-5 flex-shrink-0 text-center py-0.5 select-none ${getGutterClass(line.type, 'right')}`}>
                            {line.type === 'added' ? '+' : ''}
                          </div>
                          <div className="px-2 py-0.5 whitespace-pre flex-1">
                            {line.content}
                          </div>
                        </div>
                      ))}
                    </div>
                  </div>
                </div>
              </div>
            </>
          )}
        </div>
      </Modal.Body>

      <Modal.Footer>
        <Button variant="secondary" onClick={onClose}>
          Cancel
        </Button>
        <Button onClick={onConfirm} disabled={isLoading || !hasChanges}>
          {isLoading ? 'Saving...' : 'Confirm Changes'}
        </Button>
      </Modal.Footer>
    </Modal>
  )
}
