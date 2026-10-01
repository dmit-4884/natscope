import { useMemo, useState } from 'react'
import { Badge, Button, Input, Modal, Spinner } from '@/components/ui'
import ErrorAlert from '@/components/ui/ErrorAlert'
import type { TypeCandidate } from '@/api/decode'
import { getErrorMessage } from '@/api/errors'
import { getSubjectPattern } from '@/contexts/messages'
import { useProtoSources, useTypeCandidates } from '@/contexts/proto'

interface Props {
  dataBase64: string
  subject: string
  saving: boolean
  onClose: () => void
  onUse: (candidate: TypeCandidate) => void
  onSave: (candidate: TypeCandidate, pattern: string) => void
}

const FIT_VARIANT = (c: TypeCandidate) => (c.unknownBytes === 0 && c.score >= 90 ? 'success' : c.score >= 60 ? 'warning' : 'default')

export function DetectTypeDialog({ dataBase64, subject, saving, onClose, onUse, onSave }: Props) {
  const [pattern, setPattern] = useState(() => getSubjectPattern(subject))
  const [picked, setPicked] = useState(0)
  const { data: candidates, isLoading, error } = useTypeCandidates(dataBase64, true)
  const { data: sources = [] } = useProtoSources()
  const sourceNames = useMemo(() => new Map(sources.map((s) => [s.id, s.name])), [sources])
  const selected = candidates?.[picked]

  return (
    <Modal
      isOpen
      onClose={onClose}
      title="Detect message type"
      size="lg"
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>
            Cancel
          </Button>
          <Button
            variant="secondary"
            disabled={!selected}
            onClick={() => selected && onUse(selected)}
            data-testid="detect-use"
          >
            Decode as this type
          </Button>
          <Button
            disabled={!selected || pattern.trim() === ''}
            loading={saving}
            onClick={() => selected && onSave(selected, pattern.trim())}
            data-testid="detect-save"
          >
            Save as mapping
          </Button>
        </>
      }
    >
      <Modal.Body>
        <div className="space-y-4">
          <p className="text-sm text-content-secondary">
            Natscope decoded this payload as every message type of your enabled proto sources. The best fit comes first.
          </p>

          {isLoading && (
            <div className="flex items-center gap-2 text-sm text-content-tertiary">
              <Spinner size="sm" />
              Trying every message type…
            </div>
          )}
          {error && <ErrorAlert message={getErrorMessage(error)} />}
          {candidates?.length === 0 && (
            <div className="text-sm text-content-secondary" data-testid="detect-empty">
              <p className="font-medium text-content-primary">No message type fits this payload.</p>
              <p className="mt-1">
                Enable the proto source that defines it, or read the raw fields in the Wire tab.
              </p>
            </div>
          )}

          {candidates && candidates.length > 0 && (
            <>
              <fieldset className="space-y-1" data-testid="type-candidates">
                <legend className="sr-only">Message type</legend>
                {candidates.map((c, i) => (
                  <label
                    key={`${c.sourceId}/${c.messageType}`}
                    className={`flex items-center gap-3 px-3 py-2 rounded-md border cursor-pointer transition-colors ${
                      i === picked ? 'border-accent bg-accent-light' : 'border-border hover:bg-surface-hover'
                    }`}
                    data-testid="type-candidate"
                  >
                    <input
                      type="radio"
                      name="type-candidate"
                      checked={i === picked}
                      onChange={() => setPicked(i)}
                    />
                    <span className="min-w-0 flex-1">
                      <span className="block font-mono text-sm text-content-primary truncate">{c.messageType}</span>
                      <span className="block text-xs text-content-tertiary truncate">
                        {sourceNames.get(c.sourceId) ?? c.sourceId}
                        {c.unknownBytes > 0 && ` · ${c.unknownBytes} bytes the type does not declare`}
                      </span>
                    </span>
                    <Badge variant={FIT_VARIANT(c)} shape="pill">
                      {c.score}% fit
                    </Badge>
                  </label>
                ))}
              </fieldset>

              {selected && (
                <pre
                  className="max-h-48 overflow-auto rounded-md bg-surface-secondary border border-border p-3 text-xs font-mono text-content-primary"
                  data-testid="detect-preview"
                >
                  {JSON.stringify(selected.decoded, null, 2)}
                </pre>
              )}

              <div>
                <label htmlFor="detect-pattern" className="block text-sm font-medium text-content-primary mb-1">
                  Subject pattern
                </label>
                <Input
                  id="detect-pattern"
                  size="sm"
                  value={pattern}
                  onChange={(e) => setPattern(e.currentTarget.value)}
                  data-testid="detect-pattern"
                />
                <p className="mt-1 text-xs text-content-tertiary">
                  Saving maps every subject that matches this pattern to the selected type.
                </p>
              </div>
            </>
          )}
        </div>
      </Modal.Body>
    </Modal>
  )
}
