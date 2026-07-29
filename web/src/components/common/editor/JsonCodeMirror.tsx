import { useMemo, useRef } from 'react'
import CodeMirror from '@uiw/react-codemirror'
import { json } from '@codemirror/lang-json'
import { EditorView, keymap, placeholder as cmPlaceholder } from '@codemirror/view'
import { Prec } from '@codemirror/state'
import {
  autocompletion,
  type Completion,
  type CompletionContext,
  type CompletionResult,
} from '@codemirror/autocomplete'
import { HighlightStyle, syntaxHighlighting } from '@codemirror/language'
import { tags } from '@lezer/highlight'

/** Field metadata for schema-aware key completion (from the proto registry). */
export interface CompletionField {
  name: string
  type: string
  repeated?: boolean
  isMessage?: boolean
}

interface Props {
  value: string
  onChange: (value: string) => void
  heightPx: number
  placeholder?: string
  /** Called on Cmd/Ctrl+Enter. */
  onSubmit?: () => void
  /** Called on Cmd/Ctrl+S. */
  onFormat?: () => void
  /** When set, typing a key inside an object suggests these fields. */
  completionFields?: CompletionField[]
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

/**
 * Suggests proto message field names while typing an object key.
 * Root-message only — doesn't resolve nested object context, so a cursor
 * inside a nested value still gets top-level field suggestions.
 */
function protoFieldSource(fields: CompletionField[]) {
  return (context: CompletionContext): CompletionResult | null => {
    const word = context.matchBefore(/"[\w]*$|[A-Za-z_][\w]*$/)
    if (!word && !context.explicit) return null
    // `from` must exclude the opening quote: CodeMirror filters options
    // against doc text from `from` to cursor, and a leading `"` never matches.
    const hasQuote = word?.text.startsWith('"') ?? false
    const wordStart = word ? word.from : context.pos
    const from = wordStart + (hasQuote ? 1 : 0)
    // Keys are only valid right after `{` or `,` — stay quiet in value positions.
    const beforeText = context.state.sliceDoc(0, wordStart).replace(/\s+$/, '')
    const prevChar = beforeText.slice(-1)
    if (prevChar !== '' && prevChar !== '{' && prevChar !== ',') return null

    const applyField =
      (name: string): Completion['apply'] =>
      (view, _completion, applyFrom, applyTo) => {
        // closeBrackets pairs the opening quote — consume the auto-inserted
        // closing quote so the result is `"name": `, not `"name": "`.
        const nextChar = view.state.sliceDoc(applyTo, applyTo + 1)
        const to = hasQuote && nextChar === '"' ? applyTo + 1 : applyTo
        const insert = hasQuote ? `${name}": ` : `"${name}": `
        view.dispatch({
          changes: { from: applyFrom, to, insert },
          selection: { anchor: applyFrom + insert.length },
        })
      }

    return {
      from,
      options: fields.map((f) => ({
        label: f.name,
        type: f.isMessage ? 'class' : 'property',
        detail: f.repeated ? `${f.type}[]` : f.type,
        apply: applyField(f.name),
      })),
      validFor: /^[\w]*$/,
    }
  }
}

export default function JsonCodeMirror({
  value,
  onChange,
  heightPx,
  placeholder,
  onSubmit,
  onFormat,
  completionFields,
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
    if (completionFields && completionFields.length > 0) {
      ext.push(autocompletion({ override: [protoFieldSource(completionFields)] }))
    }
    return ext
  }, [completionFields, placeholder])

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
