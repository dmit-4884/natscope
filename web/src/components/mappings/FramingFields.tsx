import { useState } from 'react'
import type { Framing, FramingKindName } from '@/api/framing'
import { Input, Select } from '@/components/ui'
import { bytesToHex, hexToBytes } from '@/utils/hex'

const KIND_OPTIONS: { value: FramingKindName; label: string }[] = [
  { value: 'none', label: 'None: the payload is the protobuf message' },
  { value: 'grpc', label: 'gRPC frame: 5-byte header, gzip when flagged' },
  { value: 'confluent', label: 'Confluent Schema Registry: magic byte, schema id, message indexes' },
  { value: 'varint_delimited', label: 'Varint length-delimited' },
  { value: 'custom', label: 'Custom prefix and suffix bytes' },
]

interface Props {
  value: Framing
  onChange: (framing: Framing) => void
  onValidityChange: (valid: boolean) => void
}

export function FramingFields({ value, onChange, onValidityChange }: Props) {
  const [prefixText, setPrefixText] = useState(() => bytesToHex(value.prefix))
  const [suffixText, setSuffixText] = useState(() => bytesToHex(value.suffix))
  const prefix = hexToBytes(prefixText)
  const suffix = hexToBytes(suffixText)

  const updateHex = (nextPrefix: string, nextSuffix: string) => {
    setPrefixText(nextPrefix)
    setSuffixText(nextSuffix)
    const p = hexToBytes(nextPrefix)
    const s = hexToBytes(nextSuffix)
    onValidityChange(p !== null && s !== null)
    if (p && s) onChange({ ...value, prefix: p, suffix: s })
  }

  return (
    <div className="space-y-2">
      <Select
        size="sm"
        aria-label="Framing"
        value={value.kind}
        onChange={(e) => {
          const kind = e.target.value as FramingKindName
          onValidityChange(kind !== 'custom' || (prefix !== null && suffix !== null))
          onChange({ ...value, kind })
        }}
        options={KIND_OPTIONS}
      />
      {value.kind === 'confluent' && (
        <div>
          <label htmlFor="framing-schema-id" className="block text-xs text-content-secondary mb-1">
            Schema id written when publishing
          </label>
          <Input
            id="framing-schema-id"
            size="sm"
            type="number"
            min={0}
            value={value.schemaId}
            onChange={(e) => onChange({ ...value, schemaId: Math.max(0, Number(e.target.value) || 0) })}
          />
        </div>
      )}
      {value.kind === 'custom' && (
        <div className="grid grid-cols-2 gap-2">
          <div>
            <label htmlFor="framing-prefix" className="block text-xs text-content-secondary mb-1">
              Prefix (hex)
            </label>
            <Input
              id="framing-prefix"
              size="sm"
              mono
              value={prefixText}
              placeholder="cafe"
              error={prefix === null}
              errorMessage={prefix === null ? 'Use pairs of hex digits.' : undefined}
              onChange={(e) => updateHex(e.target.value, suffixText)}
            />
          </div>
          <div>
            <label htmlFor="framing-suffix" className="block text-xs text-content-secondary mb-1">
              Suffix (hex)
            </label>
            <Input
              id="framing-suffix"
              size="sm"
              mono
              value={suffixText}
              placeholder="0d0a"
              error={suffix === null}
              errorMessage={suffix === null ? 'Use pairs of hex digits.' : undefined}
              onChange={(e) => updateHex(prefixText, e.target.value)}
            />
          </div>
        </div>
      )}
      <p className="text-xs text-content-tertiary">
        natscope strips the framing before decoding and adds it when publishing to this pattern.
      </p>
    </div>
  )
}
