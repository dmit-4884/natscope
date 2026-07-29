import { lazy, Suspense, useCallback, useEffect, useRef, useState } from 'react'
import { toast } from '@/utils/toast'
import { copyText } from '@/utils/clipboard'
import type { CompletionField } from '@/components/common/editor/JsonCodeMirror'
import { CopyIcon, RefreshIcon } from '@/components/ui'
import HelpersDropdown from './HelpersDropdown'
import Tooltip from './Tooltip'

// Lazy-load the heavy CodeMirror chunk.
const JsonCodeMirror = lazy(() => import('@/components/common/editor/JsonCodeMirror'))

interface JsonEditorProps {
  value: string
  onChange: (value: string) => void
  error?: string | null
  placeholder?: string
  title?: string
  onUseExample?: () => void
  exampleLoading?: boolean
  exampleDisabled?: boolean
  /** Loads the latest real message from the stream into the editor. */
  onPrefillFromLast?: () => void
  prefillLoading?: boolean
  /** Current payload size; rendered in the toolbar when provided. */
  sizeBytes?: number
  /** Stream max_msg_size; counter turns red above this. 0/undefined = unlimited. */
  sizeLimitBytes?: number
  /** Proto message fields for schema-aware key autocomplete. */
  completionFields?: CompletionField[]
  height?: string
  /** Called on Cmd/Ctrl+Enter */
  onSubmit?: () => void
}

function formatByteSize(n: number): string {
  if (n < 1024) return `${n} B`
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`
  return `${(n / (1024 * 1024)).toFixed(1)} MB`
}

/** Tailwind height classes mapped to pixel defaults. */
const HEIGHT_BY_CLASS: Record<string, number> = {
  'h-64': 256,
  'h-96': 384,
}

export default function TemplateJsonEditor({
  value,
  onChange,
  error,
  placeholder = '{"field": "value"}',
  title = 'Message (JSON)',
  onUseExample,
  exampleLoading,
  exampleDisabled,
  onPrefillFromLast,
  prefillLoading,
  sizeBytes,
  sizeLimitBytes,
  completionFields,
  height = 'h-64',
  onSubmit,
}: JsonEditorProps) {
  const [editorHeight, setEditorHeight] = useState<number>(() => HEIGHT_BY_CLASS[height] ?? 384)
  const [isResizing, setIsResizing] = useState(false)
  // Track active resize listeners so they're removed if the editor unmounts
  // mid-drag (handleMouseUp only fires on a completed drag).
  const activeResizeListeners = useRef<{ move: (e: MouseEvent) => void; up: () => void } | null>(null)
  useEffect(
    () => () => {
      if (activeResizeListeners.current) {
        document.removeEventListener('mousemove', activeResizeListeners.current.move)
        document.removeEventListener('mouseup', activeResizeListeners.current.up)
      }
    },
    [],
  )

  // Temporarily swap {{helpers}} for valid JSON placeholders so the text parses;
  // restore() puts the originals back.
  const withPlaceholders = useCallback((text: string): { processed: string, restore: (s: string) => string } => {
    const placeholders: { placeholder: string, original: string }[] = []
    let idx = 0

    const processed = text.replace(/\{\{[^}]+\}\}/g, (match) => {
      const placeholder = `"___HELPER_${idx}___"`
      placeholders.push({ placeholder: `___HELPER_${idx}___`, original: match })
      idx++
      return placeholder
    })

    const restore = (s: string): string => {
      let result = s
      placeholders.forEach(({ placeholder, original }) => {
        result = result.replace(`"${placeholder}"`, original)
        result = result.replace(placeholder, original)
      })
      return result
    }

    return { processed, restore }
  }, [])

  // Handle resize drag
  const handleResizeStart = useCallback((e: React.MouseEvent) => {
    e.preventDefault()
    setIsResizing(true)
    const startY = e.clientY
    const startHeight = editorHeight

    const handleMouseMove = (e: MouseEvent) => {
      const delta = e.clientY - startY
      const newHeight = Math.max(150, Math.min(800, startHeight + delta))
      setEditorHeight(newHeight)
    }

    const handleMouseUp = () => {
      setIsResizing(false)
      document.removeEventListener('mousemove', handleMouseMove)
      document.removeEventListener('mouseup', handleMouseUp)
      activeResizeListeners.current = null
    }

    activeResizeListeners.current = { move: handleMouseMove, up: handleMouseUp }
    document.addEventListener('mousemove', handleMouseMove)
    document.addEventListener('mouseup', handleMouseUp)
  }, [editorHeight])

  // Format JSON (preserves {{helpers}})
  const handleFormat = useCallback(() => {
    try {
      const { processed, restore } = withPlaceholders(value)
      const parsed = JSON.parse(processed)
      const formatted = JSON.stringify(parsed, null, 2)
      onChange(restore(formatted))
      toast.success('JSON formatted')
    } catch {
      toast.error('Cannot format: invalid JSON')
    }
  }, [value, onChange, withPlaceholders])

  // Minify JSON (preserves {{helpers}})
  const handleMinify = useCallback(() => {
    try {
      const { processed, restore } = withPlaceholders(value)
      const parsed = JSON.parse(processed)
      const minified = JSON.stringify(parsed)
      onChange(restore(minified))
      toast.success('JSON minified')
    } catch {
      toast.error('Cannot minify: invalid JSON')
    }
  }, [value, onChange, withPlaceholders])

  const handleCopy = useCallback(async () => {
    await copyText(value)
  }, [value])

  return (
    <div className="bg-surface-inverse rounded-lg border border-gray-800 overflow-hidden">
      {/* Header */}
      <div className="flex items-center justify-between px-3 py-2 bg-gray-800/50 border-b border-gray-800">
        <div className="flex items-center gap-2 min-w-0">
          <span className="font-medium text-sm text-content-muted">{title}</span>
          {sizeBytes !== undefined && sizeBytes > 0 && (
            <span
              className={`text-2xs font-mono px-1.5 py-0.5 rounded ${
                sizeLimitBytes && sizeLimitBytes > 0 && sizeBytes > sizeLimitBytes
                  ? 'text-red-400 bg-red-900/30'
                  : 'text-content-tertiary bg-gray-800'
              }`}
              title={
                sizeLimitBytes && sizeLimitBytes > 0
                  ? `Payload size / stream max message size (${formatByteSize(sizeLimitBytes)})`
                  : 'Payload size'
              }
              data-testid="payload-size"
            >
              {formatByteSize(sizeBytes)}
              {sizeLimitBytes && sizeLimitBytes > 0 ? ` / ${formatByteSize(sizeLimitBytes)}` : ''}
            </span>
          )}
        </div>

        <div className="flex items-center gap-1">
          {onUseExample && (
            <Tooltip content="Use Example Message">
              <button
                onClick={onUseExample}
                disabled={exampleLoading || exampleDisabled}
                className="px-2 py-1 text-xs text-content-muted hover:text-gray-200 hover:bg-gray-700 rounded transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-1"
              >
                {exampleLoading ? (
                  <svg className="w-3.5 h-3.5 animate-spin" fill="none" viewBox="0 0 24 24">
                    <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                    <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
                  </svg>
                ) : (
                  <RefreshIcon className="w-3.5 h-3.5" />
                )}
                Use Example Message
              </button>
            </Tooltip>
          )}

          {onPrefillFromLast && (
            <Tooltip content="Prefill from the latest message on this subject">
              <button
                onClick={onPrefillFromLast}
                disabled={prefillLoading}
                className="px-2 py-1 text-xs text-content-muted hover:text-gray-200 hover:bg-gray-700 rounded transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-1"
                data-testid="prefill-from-last"
              >
                {prefillLoading ? (
                  <svg className="w-3.5 h-3.5 animate-spin" fill="none" viewBox="0 0 24 24">
                    <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                    <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
                  </svg>
                ) : (
                  <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
                  </svg>
                )}
                Last message
              </button>
            </Tooltip>
          )}

          <div className="w-px h-4 bg-gray-700 mx-1" />

          <Tooltip content="Format JSON (Cmd/Ctrl+S)">
            <button
              onClick={handleFormat}
              className="p-1 text-content-tertiary hover:text-gray-300 hover:bg-gray-700 rounded transition-colors"
              aria-label="Format JSON (Cmd/Ctrl+S)"
            >
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h16m-7 6h7" />
              </svg>
            </button>
          </Tooltip>

          <Tooltip content="Minify JSON">
            <button
              onClick={handleMinify}
              className="p-1 text-content-tertiary hover:text-gray-300 hover:bg-gray-700 rounded transition-colors"
              aria-label="Minify JSON"
            >
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M20 12H4" />
              </svg>
            </button>
          </Tooltip>

          <Tooltip content="Copy JSON">
            <button
              onClick={handleCopy}
              className="p-1 text-content-tertiary hover:text-gray-300 hover:bg-gray-700 rounded transition-colors"
              aria-label="Copy JSON"
            >
              <CopyIcon className="w-4 h-4" />
            </button>
          </Tooltip>

          <div className="w-px h-4 bg-gray-700 mx-1" />

          <HelpersDropdown />
        </div>
      </div>

      {/* CodeMirror editor (lazy chunk) */}
      <div style={{ height: editorHeight }} className="relative">
        <Suspense
          fallback={
            <div className="h-full flex items-center justify-center text-xs text-content-secondary font-mono">
              Loading editor…
            </div>
          }
        >
          <JsonCodeMirror
            value={value}
            onChange={onChange}
            heightPx={editorHeight}
            placeholder={placeholder}
            onSubmit={onSubmit}
            onFormat={handleFormat}
            completionFields={completionFields}
          />
        </Suspense>
      </div>

      {/* Resize handle */}
      <div
        onMouseDown={handleResizeStart}
        className={`h-2 cursor-ns-resize flex items-center justify-center hover:bg-gray-700/50 transition-colors ${isResizing ? 'bg-gray-700/50' : ''}`}
      >
        <div className="w-8 h-1 bg-gray-600 rounded-full" />
      </div>

      {/* Error */}
      {error && (
        <div className="px-3 py-2 bg-red-900/30 border-t border-red-800/50 text-xs text-red-400">
          {error}
        </div>
      )}
    </div>
  )
}
