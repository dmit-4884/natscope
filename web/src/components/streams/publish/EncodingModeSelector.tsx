import Tooltip from '@/components/common/Tooltip'

export type EncodingMode = 'auto' | 'json' | 'proto'

interface Props {
  encodingMode: EncodingMode
  isJsonMode: boolean
  messageType?: string | null
  mappedMessageType: string | null
  protoFieldCount?: number
  onModeChange: (mode: EncodingMode) => void
  onAddMapping: () => void
}

export function EncodingModeSelector({
  encodingMode,
  isJsonMode,
  messageType,
  mappedMessageType,
  protoFieldCount,
  onModeChange,
  onAddMapping,
}: Props) {
  return (
    <div className="flex items-center justify-between gap-3">
      <div className="flex items-center gap-2">
        <div className="inline-flex rounded-lg border border-border overflow-hidden text-xs">
          <button
            onClick={() => onModeChange('json')}
            className={`px-3 py-1.5 font-medium transition-colors ${
              isJsonMode ? 'bg-green-600 text-content-inverse' : 'bg-surface-primary text-content-secondary hover:bg-surface-secondary'
            }`}
          >
            JSON
          </button>
          <Tooltip content={!mappedMessageType ? 'No proto mapping configured for this subject' : ''}>
            <button
              onClick={() => onModeChange(mappedMessageType ? 'proto' : 'auto')}
              disabled={!mappedMessageType}
              className={`px-3 py-1.5 font-medium transition-colors border-l border-border ${
                !isJsonMode && messageType
                  ? 'bg-accent text-content-inverse'
                  : !mappedMessageType
                    ? 'bg-surface-secondary text-gray-300 cursor-not-allowed'
                    : 'bg-surface-primary text-content-secondary hover:bg-surface-secondary'
              }`}
            >
              Protobuf
            </button>
          </Tooltip>
        </div>
        {encodingMode === 'auto' && (
          <span
            className="px-1.5 py-0.5 text-2xs font-medium uppercase tracking-wide text-content-tertiary bg-surface-tertiary rounded"
            title={`Mode resolved automatically: ${mappedMessageType ? 'a proto mapping exists for this subject, so Protobuf is used' : 'no proto mapping for this subject, so raw JSON is used'}. Click JSON or Protobuf to pin it.`}
            data-testid="encoding-auto-chip"
          >
            auto
          </span>
        )}
        {isJsonMode ? (
          <span className="text-xs text-content-tertiary">Raw JSON bytes</span>
        ) : messageType ? (
          <span className="text-xs text-content-tertiary font-mono">{messageType}</span>
        ) : null}
      </div>

      <div className="flex items-center gap-2">
        {!isJsonMode && protoFieldCount !== undefined && (
          <span className="text-xs text-blue-500">{protoFieldCount} fields</span>
        )}
        {!mappedMessageType && (
          <button onClick={onAddMapping} className="text-xs text-accent hover:text-blue-800 underline">
            Add proto mapping
          </button>
        )}
      </div>
    </div>
  )
}
