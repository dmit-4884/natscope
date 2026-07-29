import { useState, useEffect, useMemo } from 'react'
import { Modal, Button } from '@/components/ui'
import TemplateJsonEditor from '@/components/common/TemplateJsonEditor'
import { SubjectAutocomplete } from '@/components/common/SubjectAutocomplete'
import { HeadersEditor, type HeaderEntry } from '@/components/streams/publish/HeadersEditor'
import { EditableSubject } from '@/components/streams/publish/EditableSubject'
import { hasWildcards, parsePattern, countWildcards } from '@/components/streams/publish/subjectPatternUtils'
import { useActiveConnection } from '@/contexts/connection'
import { useAllStreamSubjects } from '@/contexts/streams/application/queries/useAllStreamSubjects'
import type { MessageTemplate } from '@/contexts/templates'

type Mode = 'create' | 'edit' | 'duplicate'

export interface TemplateValues {
  name: string
  subject: string
  messageType: string
  data: string
  headers?: Record<string, string>
  wildcards?: string[]
}

interface Props {
  isOpen: boolean
  mode: Mode
  initial: Partial<Pick<MessageTemplate, 'name' | 'subject' | 'messageType' | 'data' | 'headers' | 'wildcards'>>
  onClose: () => void
  onSave: (values: TemplateValues) => void
  isSaving?: boolean
}

const titleByMode: Record<Mode, string> = {
  create: 'Save template',
  edit: 'Edit template',
  duplicate: 'Duplicate template',
}

const ctaByMode: Record<Mode, string> = {
  create: 'Save template',
  edit: 'Save changes',
  duplicate: 'Save copy',
}

function headersToEntries(headers?: Record<string, string>): HeaderEntry[] {
  return Object.entries(headers ?? {}).map(([key, value]) => ({ key, value }))
}

function entriesToHeaders(entries: HeaderEntry[]): Record<string, string> | undefined {
  const out: Record<string, string> = {}
  for (const e of entries) {
    const k = e.key.trim()
    if (!k) continue
    out[k] = e.value
  }
  return Object.keys(out).length > 0 ? out : undefined
}

export function TemplateEditModal({ isOpen, mode, initial, onClose, onSave, isSaving = false }: Props) {
  const [name, setName] = useState(initial.name ?? '')
  const [subject, setSubject] = useState(initial.subject ?? '')
  const [messageType, setMessageType] = useState(initial.messageType ?? '')
  const [data, setData] = useState(initial.data ?? '')
  const [headerEntries, setHeaderEntries] = useState<HeaderEntry[]>(headersToEntries(initial.headers))
  const [wildcardValues, setWildcardValues] = useState<string[]>(initial.wildcards ?? [])
  const [jsonError, setJsonError] = useState<string | null>(null)

  // Pull every stream subject across the active connection for autocomplete.
  // The user can still type any pattern — these are suggestions only.
  const { connectionId } = useActiveConnection()
  const subjectOptions = useAllStreamSubjects(connectionId)

  useEffect(() => {
    if (!isOpen) return
    setName(initial.name ?? '')
    setSubject(initial.subject ?? '')
    setMessageType(initial.messageType ?? '')
    setData(initial.data ?? '')
    setHeaderEntries(headersToEntries(initial.headers))
    setWildcardValues(initial.wildcards ?? [])
    setJsonError(null)
  }, [isOpen, initial.name, initial.subject, initial.messageType, initial.data, initial.headers, initial.wildcards])

  // Wildcard array resizes to match the pattern; values are preserved
  // positionally so editing the pattern doesn't wipe filled-in values.
  const patternHasWildcards = useMemo(() => hasWildcards(subject), [subject])
  const patternSegments = useMemo(() => parsePattern(subject), [subject])
  const wildcardCount = useMemo(() => countWildcards(subject), [subject])
  useEffect(() => {
    setWildcardValues((prev) => {
      if (prev.length === wildcardCount) return prev
      const next = new Array(wildcardCount).fill('')
      for (let i = 0; i < Math.min(prev.length, wildcardCount); i++) next[i] = prev[i]
      return next
    })
  }, [wildcardCount])

  const handleDataChange = (val: string) => {
    setData(val)
    if (!val.trim() || val.includes('{{')) {
      // Empty is fine; helpers are evaluated at publish time, can't validate here.
      setJsonError(null)
      return
    }
    try {
      JSON.parse(val)
      setJsonError(null)
    } catch (e) {
      setJsonError(e instanceof Error ? e.message : 'Invalid JSON')
    }
  }

  const canSave = name.trim().length > 0 && !jsonError

  const handleSave = () => {
    if (!canSave) return
    // Strip trailing empties so unused wildcard slots aren't persisted as
    // `""`; leading/middle empties stay since they're positional markers.
    const trimmedWildcards = [...wildcardValues]
    while (trimmedWildcards.length > 0 && trimmedWildcards[trimmedWildcards.length - 1] === '') {
      trimmedWildcards.pop()
    }
    onSave({
      name: name.trim(),
      subject: subject.trim(),
      messageType: messageType.trim(),
      data,
      headers: entriesToHeaders(headerEntries),
      wildcards: trimmedWildcards.length > 0 ? trimmedWildcards : undefined,
    })
  }

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title={titleByMode[mode]}
      size="xl"
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>Cancel</Button>
          <Button disabled={!canSave} loading={isSaving} onClick={handleSave}>{ctaByMode[mode]}</Button>
        </>
      }
    >
      <div className="px-6 py-4 space-y-4 overflow-y-auto">
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">
            Name <span className="text-rose-500">*</span>
          </label>
          <input
            autoFocus
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="e.g. orders.create — happy path"
            className="w-full rounded-md border border-border-strong px-3 py-2 text-sm focus:border-border-focus focus:ring-1 focus:ring-border-focus focus:outline-none"
          />
        </div>

        <div className="grid grid-cols-2 gap-3">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Subject pattern</label>
            <SubjectAutocomplete
              value={subject}
              onChange={setSubject}
              options={subjectOptions}
              placeholder="orders.*"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Message type</label>
            <input
              value={messageType}
              onChange={(e) => setMessageType(e.target.value)}
              placeholder="api.v1.OrderCreated"
              className="w-full rounded-md border border-border-strong px-3 py-2 text-sm font-mono focus:border-border-focus focus:ring-1 focus:ring-border-focus focus:outline-none"
            />
          </div>
        </div>

        {patternHasWildcards && (
          <EditableSubject
            segments={patternSegments}
            wildcardValues={wildcardValues}
            onWildcardChange={(i, v) =>
              setWildcardValues((prev) => {
                const next = [...prev]
                next[i] = v
                return next
              })
            }
          />
        )}

        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Payload (JSON)</label>
          <TemplateJsonEditor value={data} onChange={handleDataChange} error={jsonError} height="h-56" />
        </div>

        <HeadersEditor
          headers={headerEntries}
          onAdd={() => setHeaderEntries([...headerEntries, { key: '', value: '' }])}
          onRemove={(i) => setHeaderEntries(headerEntries.filter((_, idx) => idx !== i))}
          onUpdate={(i, field, val) =>
            setHeaderEntries(headerEntries.map((h, idx) => (idx === i ? { ...h, [field]: val } : h)))
          }
        />
      </div>
    </Modal>
  )
}
