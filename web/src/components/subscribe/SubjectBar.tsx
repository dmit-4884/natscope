import { useState } from 'react'
import { SubjectAutocomplete } from '@/components/common/SubjectAutocomplete'
import Tooltip from '@/components/common/Tooltip'
import { Button, CloseIcon, LockClosedIcon, PlayIcon, StopIcon } from '@/components/ui'
import { SUBJECT_PRESETS, subscribeSubjectError } from './subscribeUtils'

const MAX_SUBJECTS = 100
const INPUT_ID = 'subscribe-subject'
const ERROR_ID = 'subscribe-subject-error'

interface Props {
  subjects: string[]
  onSubjectsChange: (subjects: string[]) => void
  deniedSubjects: string[]
  suggestions: string[]
  recentSubjects: string[]
  running: boolean
  onStart: (subjects: string[]) => void
  onStop: () => void
}

export function SubjectBar({
  subjects,
  onSubjectsChange,
  deniedSubjects,
  suggestions,
  recentSubjects,
  running,
  onStart,
  onStop,
}: Props) {
  const [input, setInput] = useState('')
  const [error, setError] = useState<string | null>(null)

  const withInput = (): string[] | null => {
    const subject = input.trim()
    if (!subject) return subjects
    const problem = subscribeSubjectError(subject)
      ?? (subjects.length >= MAX_SUBJECTS ? `Up to ${MAX_SUBJECTS} subjects at once` : null)
    if (problem) {
      setError(problem)
      return null
    }
    setError(null)
    setInput('')
    return subjects.includes(subject) ? subjects : [...subjects, subject]
  }

  const add = (subject?: string) => {
    if (subject !== undefined) {
      if (!subjects.includes(subject)) onSubjectsChange([...subjects, subject])
      return
    }
    const next = withInput()
    if (next && next !== subjects) onSubjectsChange(next)
  }

  const remove = (subject: string) => {
    const next = subjects.filter((s) => s !== subject)
    onSubjectsChange(next)
    if (next.length === 0 && running) onStop()
  }

  const start = () => {
    const next = withInput()
    if (!next || next.length === 0) {
      if (next) setError('Add a subject to subscribe to')
      return
    }
    if (next !== subjects) onSubjectsChange(next)
    onStart(next)
  }

  const quickAdd = [
    ...SUBJECT_PRESETS.map((p) => ({ subject: p.subject, label: p.label })),
    ...recentSubjects.filter((s) => !SUBJECT_PRESETS.some((p) => p.subject === s)).map((s) => ({ subject: s, label: s })),
  ].filter((o) => !subjects.includes(o.subject))

  return (
    <div className="mt-4">
      <label htmlFor={INPUT_ID} className="block text-sm font-medium text-gray-700 mb-1">
        Subjects
      </label>
      <div className="flex items-start gap-2">
        <div
          className={`flex-1 min-w-0 flex flex-wrap items-center gap-1.5 rounded-md border bg-surface-primary px-2 py-1.5 focus-within:ring-1 ${
            error ? 'border-status-error-border focus-within:ring-status-error-border' : 'border-border-strong focus-within:border-border-focus focus-within:ring-border-focus'
          }`}
          data-testid="subject-chips"
        >
          {subjects.map((subject) => {
            const denied = deniedSubjects.includes(subject)
            return (
              <span
                key={subject}
                data-testid="subject-chip"
                data-denied={denied || undefined}
                className={`inline-flex items-center gap-1 rounded-md border pl-2 pr-1 py-0.5 text-xs ${
                  denied
                    ? 'border-status-warning-border bg-status-warning-bg text-status-warning-text'
                    : 'border-border bg-surface-tertiary text-content-primary'
                }`}
              >
                {denied && <LockClosedIcon className="w-3 h-3 shrink-0" />}
                <span className="font-mono">{subject}</span>
                {denied && (
                  <Tooltip content="The server refused this subscription for your NATS user">
                    <span className="font-medium">· no permission</span>
                  </Tooltip>
                )}
                <button
                  type="button"
                  onClick={() => remove(subject)}
                  aria-label={`Remove ${subject}`}
                  className="ml-0.5 p-0.5 rounded hover:bg-surface-hover text-content-muted hover:text-content-secondary"
                >
                  <CloseIcon className="w-3 h-3" />
                </button>
              </span>
            )
          })}
          <div
            className="flex-1 min-w-[12rem]"
            onKeyDown={(e) => {
              if (e.key === 'Enter' && !e.defaultPrevented) {
                e.preventDefault()
                if (e.metaKey || e.ctrlKey) start()
                else add()
              } else if (e.key === 'Backspace' && !input && subjects.length > 0) {
                remove(subjects[subjects.length - 1])
              }
            }}
          >
            <SubjectAutocomplete
              inputId={INPUT_ID}
              value={input}
              onChange={(v) => {
                setInput(v)
                if (error) setError(null)
              }}
              options={suggestions.filter((s) => !subjects.includes(s))}
              placeholder={subjects.length === 0 ? 'orders.> — press Enter to add' : 'Add another subject'}
              invalid={!!error}
              describedBy={error ? ERROR_ID : undefined}
              className="w-full border-0 bg-transparent px-1 py-0.5 text-sm font-mono focus:outline-none focus:ring-0"
            />
          </div>
        </div>
        {running ? (
          <Button variant="secondary" icon={<StopIcon className="w-3.5 h-3.5" />} onClick={onStop}>
            Stop
          </Button>
        ) : (
          <Button icon={<PlayIcon className="w-3.5 h-3.5" />} onClick={start}>
            Start
          </Button>
        )}
      </div>
      {error && (
        <p id={ERROR_ID} className="mt-1 text-xs text-status-error-text" data-testid="subject-error">
          {error}
        </p>
      )}
      {quickAdd.length > 0 && (
        <div className="mt-2 flex flex-wrap items-center gap-1.5">
          <span className="text-xs text-content-tertiary">Quick add:</span>
          {quickAdd.map((o) => (
            <Tooltip key={o.subject} content={o.label === o.subject ? 'Recent subject' : o.subject}>
              <button
                type="button"
                onClick={() => add(o.subject)}
                className="rounded-full border border-border px-2 py-0.5 text-xs text-content-secondary hover:bg-surface-hover hover:text-content-primary transition-colors"
              >
                {o.label === o.subject ? <span className="font-mono">{o.subject}</span> : o.label}
              </button>
            </Tooltip>
          ))}
        </div>
      )}
    </div>
  )
}
