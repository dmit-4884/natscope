import { useMemo, useRef } from 'react'
import CodeMirror from '@uiw/react-codemirror'
import { json } from '@codemirror/lang-json'
import { EditorView, keymap, placeholder as cmPlaceholder } from '@codemirror/view'
import { Prec } from '@codemirror/state'
import { autocompletion } from '@codemirror/autocomplete'
import { HighlightStyle, syntaxHighlighting } from '@codemirror/language'
import { tags } from '@lezer/highlight'
import type { ProtoSchema } from './protoSchema'
import { protoCompletionSource } from './protoCompletion'

interface Props {
  value: string
  onChange: (value: string) => void
  heightPx: number
  placeholder?: string
  /** Called on Cmd/Ctrl+Enter. */
  onSubmit?: () => void
  /** Called on Cmd/Ctrl+S. */
  onFormat?: () => void
  /** When set, completes field names and enum values of this message type. */
  schema?: ProtoSchema
  ariaLabel?: string
}

// Dark theme matching the app's gray-900 editor chrome.
const darkTheme = EditorView.theme(
  {
    '&': { backgroundColor: '#111827', color: '#d1d5db', fontSize: '13px' },
    '.cm-content': {
      fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Consolas, monospace',
      caretColor: '#e5e7eb',
      padding: '10px 0',
    },
    '.cm-gutters': {
      backgroundColor: '#111827',
      color: '#4b5563',
      border: 'none',
      borderRight: '1px solid #1f2937',
    },
    '.cm-activeLine': { backgroundColor: 'rgba(255,255,255,0.04)' },
    '.cm-activeLineGutter': { backgroundColor: 'rgba(255,255,255,0.06)' },
    '&.cm-focused': { outline: 'none' },
    '.cm-selectionBackground, &.cm-focused .cm-selectionBackground': {
      backgroundColor: 'rgba(59,130,246,0.30) !important',
    },
    '.cm-cursor': { borderLeftColor: '#e5e7eb' },
    '.cm-placeholder': { color: '#4b5563' },
    '.cm-tooltip': {
      backgroundColor: '#1f2937',
      border: '1px solid #374151',
      color: '#d1d5db',
    },
    '.cm-tooltip-autocomplete ul li[aria-selected]': {
      backgroundColor: '#2563eb',
      color: '#ffffff',
    },
    '.cm-completionDetail': { color: '#9ca3af', fontStyle: 'normal' },
    '.cm-panels': { backgroundColor: '#1f2937', color: '#d1d5db' },
    '.cm-searchMatch': { backgroundColor: 'rgba(251,191,36,0.25)' },
    '.cm-searchMatch-selected': { backgroundColor: 'rgba(251,191,36,0.45)' },
  },
  { dark: true },
)

const jsonHighlight = HighlightStyle.define([
  { tag: tags.propertyName, color: '#67e8f9' },
  { tag: tags.string, color: '#86efac' },
  { tag: tags.number, color: '#fcd34d' },
  { tag: tags.bool, color: '#c4b5fd' },
  { tag: tags.null, color: '#c4b5fd' },
  { tag: tags.punctuation, color: '#9ca3af' },
  { tag: tags.invalid, color: '#fca5a5' },
])

export default function JsonCodeMirror({
  value,
  onChange,
  heightPx,
  placeholder,
  onSubmit,
  onFormat,
  schema,
  ariaLabel,
}: Props) {
  // Keymap handlers go through refs so the extensions array stays stable and
  // CodeMirror isn't reconfigured on every parent render.
  const submitRef = useRef(onSubmit)
  submitRef.current = onSubmit
  const formatRef = useRef(onFormat)
  formatRef.current = onFormat

  const extensions = useMemo(() => {
    const ext = [
      json(),
      syntaxHighlighting(jsonHighlight),
      darkTheme,
      EditorView.lineWrapping,
      Prec.highest(
        keymap.of([
          {
            key: 'Mod-Enter',
            run: () => {
              // Return false when unwired so CodeMirror's default behaviour
              // runs instead of silently swallowing the key.
              if (!submitRef.current) return false
              submitRef.current()
              return true
            },
          },
          {
            key: 'Mod-s',
            run: () => {
              // Same reasoning: only claim the key when there is a real handler.
              if (!formatRef.current) return false
              formatRef.current()
              return true
            },
          },
        ]),
      ),
    ]
    if (placeholder) ext.push(cmPlaceholder(placeholder))
    if (ariaLabel) ext.push(EditorView.contentAttributes.of({ 'aria-label': ariaLabel }))
    if (schema) {
      ext.push(autocompletion({ override: [protoCompletionSource(schema.description, schema.messageType)] }))
    }
    return ext
  }, [schema, placeholder, ariaLabel])

  return (
    <CodeMirror
      value={value}
      onChange={onChange}
      height={`${heightPx}px`}
      theme="none"
      basicSetup={{ autocompletion: false }}
      extensions={extensions}
    />
  )
}
