import { useState, useEffect, useMemo, useRef, useCallback } from 'react'
import { Alert } from './Alert'

interface JsonEditorProps<T> {
  value: T
  onChange: (value: T) => void
  /** Top-level field names the parent considers "locked" (immutable on
   * update); with `originalValue`, splits into a read-only block + textarea. */
  immutableFields?: readonly string[]
  /** Marks "edit" mode; null/undefined means create mode (single textarea,
   * locked fields have no meaning). */
  originalValue?: T | null
  className?: string
  readOnly?: boolean
}

export function JsonEditor<T>({
  value,
  onChange,
  immutableFields = [],
  originalValue,
  className = '',
  readOnly = false,
}: JsonEditorProps<T>) {
  const isEditMode = !!originalValue

  const presentLocked = useMemo(() => {
    if (!isEditMode || immutableFields.length === 0) return [] as string[]
    const v = (value ?? {}) as Record<string, unknown>
    return immutableFields.filter((k) => k in v)
  }, [isEditMode, immutableFields, value])

  if (readOnly || !isEditMode || presentLocked.length === 0) {
    return <PlainEditor value={value} onChange={onChange} readOnly={readOnly} className={className} />
  }
  return <SplitEditor value={value} onChange={onChange} lockedKeys={presentLocked} className={className} />
}

// PlainEditor — single textarea (create mode or no locked fields).
interface PlainEditorProps<T> {
  value: T
  onChange: (value: T) => void
  readOnly: boolean
  className: string
}

function PlainEditor<T>({ value, onChange, readOnly, className }: PlainEditorProps<T>) {
  const textareaRef = useRef<HTMLTextAreaElement>(null)
  const lastExternalValue = useRef<string>(JSON.stringify(value))
  const [jsonError, setJsonError] = useState<string | null>(null)

  useEffect(() => {
    const newStr = JSON.stringify(value)
    if (newStr !== lastExternalValue.current && textareaRef.current) {
      lastExternalValue.current = newStr
      textareaRef.current.value = JSON.stringify(value, null, 2)
      setJsonError(null)
    }
  }, [value])

  const handleInput = useCallback(() => {
    const ta = textareaRef.current
    if (!ta) return
    try {
      const parsed = JSON.parse(ta.value)
      lastExternalValue.current = JSON.stringify(parsed)
      onChange(parsed as T)
      setJsonError(null)
    } catch (e) {
      setJsonError((e as Error).message)
    }
  }, [onChange])

  const handleFormat = useCallback(() => {
    const ta = textareaRef.current
    if (!ta) return
    try {
      const parsed = JSON.parse(ta.value)
      ta.value = JSON.stringify(parsed, null, 2)
      setJsonError(null)
    } catch (e) {
      setJsonError((e as Error).message)
    }
  }, [])

  return (
    <div className={`flex flex-col h-full ${className}`}>
      {!readOnly && (
        <div className="flex justify-end mb-2">
          <button onClick={handleFormat} className="text-sm text-accent hover:text-blue-800">
            Format JSON
          </button>
        </div>
      )}
      <textarea
        ref={textareaRef}
        defaultValue={JSON.stringify(value, null, 2)}
        onInput={readOnly ? undefined : handleInput}
        readOnly={readOnly}
        className={`flex-1 w-full p-3 font-mono text-sm border rounded resize-none focus:outline-none min-h-0 ${
          readOnly
            ? 'border-border bg-surface-secondary cursor-default'
            : jsonError
              ? 'border-red-300 bg-status-error-bg focus:ring-2 focus:ring-border-focus'
              : 'border-border-strong focus:ring-2 focus:ring-border-focus'
        }`}
        spellCheck={false}
      />
      {jsonError && (
        <Alert variant="error" className="mt-2 shrink-0">
          JSON Error: {jsonError}
        </Alert>
      )}
    </div>
  )
}

// SplitEditor — separates locked fields (read-only block) from the editable
// textarea so the user can plainly see what they can and cannot change.
interface SplitEditorProps<T> {
  value: T
  onChange: (value: T) => void
  lockedKeys: readonly string[]
  className: string
}

function splitObject(value: unknown, lockedKeys: readonly string[]) {
  const v = (value ?? {}) as Record<string, unknown>
  const lockedSet = new Set(lockedKeys)
  const locked: Record<string, unknown> = {}
  const editable: Record<string, unknown> = {}
  for (const k of Object.keys(v)) {
    if (lockedSet.has(k)) locked[k] = v[k]
    else editable[k] = v[k]
  }
  return { locked, editable }
}

function SplitEditor<T>({ value, onChange, lockedKeys, className }: SplitEditorProps<T>) {
  const textareaRef = useRef<HTMLTextAreaElement>(null)
  const lastExternalValue = useRef<string>(JSON.stringify(value))
  const [jsonError, setJsonError] = useState<string | null>(null)
  const [copied, setCopied] = useState(false)

  const { locked, editable } = useMemo(() => splitObject(value, lockedKeys), [value, lockedKeys])
  const lockedJson = useMemo(() => JSON.stringify(locked, null, 2), [locked])

  useEffect(() => {
    const externalStr = JSON.stringify(value)
    if (externalStr !== lastExternalValue.current && textareaRef.current) {
      lastExternalValue.current = externalStr
      const split = splitObject(value, lockedKeys)
      textareaRef.current.value = JSON.stringify(split.editable, null, 2)
      setJsonError(null)
    }
  }, [value, lockedKeys])

  const handleInput = useCallback(() => {
    const ta = textareaRef.current
    if (!ta) return
    try {
      const parsedEditable = JSON.parse(ta.value)
      // Locked fields always win on merge so the user can never inject an
      // immutable field by typing it into the editable section.
      const merged = { ...parsedEditable, ...locked } as T
      lastExternalValue.current = JSON.stringify(merged)
      onChange(merged)
      setJsonError(null)
    } catch (e) {
      setJsonError((e as Error).message)
    }
  }, [locked, onChange])

  const handleFormat = useCallback(() => {
    const ta = textareaRef.current
    if (!ta) return
    try {
      const parsed = JSON.parse(ta.value)
      ta.value = JSON.stringify(parsed, null, 2)
      setJsonError(null)
    } catch (e) {
      setJsonError((e as Error).message)
    }
  }, [])

  const handleCopyLocked = useCallback(async () => {
    try {
      await navigator.clipboard.writeText(lockedJson)
      setCopied(true)
      setTimeout(() => setCopied(false), 1500)
    } catch {
      // Clipboard API unavailable (e.g. insecure context) — silently ignore.
    }
  }, [lockedJson])

  return (
    <div className={`flex flex-col h-full gap-3 ${className}`}>
      {/* Locked section: visually distinct, copyable, never editable. */}
      <div className="border border-amber-200 rounded-lg overflow-hidden shrink-0">
        <div className="flex items-center justify-between px-3 py-2 bg-status-warning-bg border-b border-amber-200">
          <div className="flex items-center gap-2 text-sm">
            <LockIcon className="w-4 h-4 text-status-warning-text" />
            <span className="font-medium text-amber-800">Read-only fields</span>
            <span className="text-amber-700 hidden sm:inline">
              — locked on update, not sent on save
            </span>
          </div>
          <button
            type="button"
            onClick={handleCopyLocked}
            className="text-xs text-amber-700 hover:text-amber-900 underline"
          >
            {copied ? 'Copied!' : 'Copy'}
          </button>
        </div>
        <pre className="p-3 font-mono text-sm bg-status-warning-bg/40 text-gray-700 max-h-48 overflow-auto select-text whitespace-pre m-0">
          {lockedJson}
        </pre>
      </div>

      {/* Editable section. */}
      <div className="flex flex-col flex-1 min-h-0">
        <div className="flex items-center justify-between mb-2">
          <span className="text-sm font-medium text-gray-700">Editable fields</span>
          <button onClick={handleFormat} className="text-sm text-accent hover:text-blue-800">
            Format JSON
          </button>
        </div>
        <textarea
          ref={textareaRef}
          defaultValue={JSON.stringify(editable, null, 2)}
          onInput={handleInput}
          className={`flex-1 w-full p-3 font-mono text-sm border rounded resize-none focus:outline-none min-h-0 ${
            jsonError
              ? 'border-red-300 bg-status-error-bg focus:ring-2 focus:ring-border-focus'
              : 'border-border-strong focus:ring-2 focus:ring-border-focus'
          }`}
          spellCheck={false}
        />
        {jsonError && (
          <Alert variant="error" className="mt-2 shrink-0">
            JSON Error: {jsonError}
          </Alert>
        )}
      </div>
    </div>
  )
}

function LockIcon({ className = '' }: { className?: string }) {
  return (
    <svg className={className} fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth={2}
        d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z"
      />
    </svg>
  )
}
