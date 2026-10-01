import type { UnknownField } from '@/api/decode'
import { Alert } from '@/components/ui'
import { formatBytes } from '@/utils/formatters'
import { plural } from '@/utils/plural'

const MAX_LISTED = 5

export interface DecodeNotes {
  unknownCount: number
  unknownFields?: UnknownField[]
  validBytes?: number
}

interface Props {
  notes: DecodeNotes | null
  messageType: string
  totalBytes: number
  error?: string | null
}

function describe(f: UnknownField): string {
  return `#${f.number} (${f.wireType})${f.path ? ` in ${f.path}` : ''}`
}

export function DecodeNotices({ notes, messageType, totalBytes, error }: Props) {
  if (!notes) return null
  const listed = notes.unknownFields?.slice(0, MAX_LISTED) ?? []
  const more = (notes.unknownFields?.length ?? 0) - listed.length

  return (
    <div className="mt-2 space-y-2">
      {notes.validBytes ? (
        <Alert
          variant="warning"
          title={`Decoded the first ${formatBytes(notes.validBytes)} of ${formatBytes(totalBytes)}`}
          data-testid="decode-partial"
        >
          The rest does not decode as <code>{messageType}</code>; the Wire tab shows where it breaks.
          {error && <span className="block mt-1 font-mono text-xs break-all">{error}</span>}
        </Alert>
      ) : null}
      {notes.unknownCount > 0 && (
        <Alert variant="info" title={`${plural(notes.unknownCount, 'field')} not in the schema`} data-testid="decode-unknown">
          The payload carries fields <code>{messageType}</code> does not declare; the producer may use a newer schema. The Wire
          tab shows their values.
          {listed.length > 0 && (
            <span className="block mt-1 font-mono text-xs">
              {listed.map(describe).join(', ')}
              {more > 0 && ` and ${more} more`}
            </span>
          )}
        </Alert>
      )}
    </div>
  )
}
