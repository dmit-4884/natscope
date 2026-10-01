import { Badge, Button } from '@/components/ui'
import Tooltip from '@/components/common/Tooltip'
import { useCreateMapping } from '@/contexts/mappings'
import { getErrorMessage } from '@/api/errors'
import { toast } from '@/utils/toast'
import { plural } from '@/utils/plural'
import type { KVDecodedValue } from '@/types/management'
import type { KVProtoTarget } from './useKVProtoTarget'

interface Props {
  bucket: string
  target: KVProtoTarget
  decoded?: KVDecodedValue
  showRaw: boolean
  onToggleRaw: () => void
}

export function KVProtoBar({ bucket, target, decoded, showRaw, onToggleRaw }: Props) {
  const { mutate: createMapping, isPending } = useCreateMapping()
  const pattern = `$KV.${bucket}.>`

  const saveMapping = () =>
    createMapping(
      { pattern, messageType: target.messageType, sourceId: target.sourceId },
      {
        onSuccess: () => toast.success(`Mapped ${pattern} to ${target.messageType}`),
        onError: (err) => toast.error(`Failed to save the mapping: ${getErrorMessage(err)}`),
      },
    )

  return (
    <div className="flex items-center justify-between gap-2 px-4 py-2 border-b bg-surface-secondary" data-testid="kv-proto">
      <div className="flex items-center gap-2 min-w-0 text-xs text-content-tertiary">
        <span className="font-medium">Protobuf</span>
        <span className="font-mono text-sm text-accent bg-accent-light px-2 py-0.5 rounded truncate">
          {target.messageType}
        </span>
        {target.pattern ? (
          <span className="truncate">via {target.pattern}</span>
        ) : (
          <Tooltip content="No mapping matches this key. This type decodes every byte and scores higher than any other.">
            <Badge variant="primary" shape="pill" data-testid="kv-decoded-auto">
              Auto-detected
            </Badge>
          </Tooltip>
        )}
        {decoded && decoded.unknownFields > 0 && (
          <span className="whitespace-nowrap">{plural(decoded.unknownFields, 'field')} not in the schema</span>
        )}
      </div>
      <div className="flex items-center gap-2 flex-none">
        {!target.pattern && (
          <Button size="sm" variant="secondary" loading={isPending} onClick={saveMapping} data-testid="kv-save-mapping">
            Save as mapping
          </Button>
        )}
        <Button size="sm" variant="ghost" aria-pressed={showRaw} onClick={onToggleRaw} data-testid="kv-raw-toggle">
          {showRaw ? 'Edit JSON' : 'Raw bytes'}
        </Button>
      </div>
    </div>
  )
}
