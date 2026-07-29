import { memo, useState, useMemo, useEffect, useRef } from 'react'
import { Tabs, tabPanelProps } from '@/components/ui'
import { copyText } from '@/utils/clipboard'
import { decodeBase64ToBytes } from '@/utils/base64'
import JsonTreeViewer from './JsonTreeViewer'

type ViewMode = 'decoded' | 'json' | 'raw' | 'hex'

interface PayloadViewerProps {
  rawData: string  // base64 encoded data
  decodedData?: unknown  // protobuf decoded data
  jsonData?: unknown
  headers?: Record<string, string>
  defaultMode?: ViewMode
  onModeChange?: (mode: ViewMode) => void
  isDecoding?: boolean  // show loading state for decoded tab
  jsonIndentSize?: number  // indent size for JSON display (default 2)
  // Truncate-CTA: render "load full payload" inline above the data view, next
  // to the partial payload.
  truncated?: boolean
  truncatedFullSize?: number
  onLoadFull?: () => void
  loadingFull?: boolean
  loadFullError?: string | null
}

// Helper to decode base64 to raw bytes (for hex view)
const decodeBase64 = (base64: string): Uint8Array | null => {
  try {
    return decodeBase64ToBytes(base64)
  } catch {
    return null
  }
}

// Format bytes as hex dump with ASCII column
const formatHexDump = (bytes: Uint8Array): string[] => {
  const lines: string[] = []
  const bytesPerLine = 16

  for (let i = 0; i < bytes.length; i += bytesPerLine) {
    const chunk = bytes.slice(i, i + bytesPerLine)

    // Offset column
    const offset = i.toString(16).padStart(8, '0')

    // Hex column
    const hexParts: string[] = []
    for (let j = 0; j < bytesPerLine; j++) {
      if (j < chunk.length) {
        hexParts.push(chunk[j].toString(16).padStart(2, '0'))
      } else {
        hexParts.push('  ')
      }
      if (j === 7) hexParts.push(' ') // Extra space in middle
    }
    const hex = hexParts.join(' ')

    // ASCII column
    let ascii = ''
    for (let j = 0; j < chunk.length; j++) {
      const byte = chunk[j]
      if (byte >= 32 && byte <= 126) {
        ascii += String.fromCharCode(byte)
      } else {
        ascii += '.'
      }
    }

    lines.push(`${offset}  ${hex}  |${ascii.padEnd(bytesPerLine, ' ')}|`)
  }

  return lines
}

const PayloadViewer = memo(function PayloadViewer({
  rawData,
  decodedData,
  jsonData,
  headers,
  defaultMode,
  onModeChange,
  isDecoding = false,
  jsonIndentSize = 2,
  truncated = false,
  truncatedFullSize,
  onLoadFull,
  loadingFull = false,
  loadFullError,
}: PayloadViewerProps) {
  // Determine initial mode
  const initialMode = defaultMode || (decodedData ? 'decoded' : (jsonData ? 'json' : 'raw'))
  const [viewMode, setViewMode] = useState<ViewMode>(initialMode)

  const userPickedRef = useRef(false)
  useEffect(() => {
    userPickedRef.current = false
  }, [rawData])
  useEffect(() => {
    if (defaultMode && !userPickedRef.current) {
      setViewMode(defaultMode)
    }
  }, [defaultMode, rawData])

  // Notify parent of mode changes
  const handleModeChange = (mode: ViewMode) => {
    userPickedRef.current = true
    setViewMode(mode)
    onModeChange?.(mode)
  }

  // Decode bytes for hex view
  const bytes = useMemo(() => decodeBase64(rawData), [rawData])

  // Hex dump lines
  const hexLines = useMemo(() => {
    if (!bytes) return []
    return formatHexDump(bytes)
  }, [bytes])

  const rawText = useMemo(
    () => (bytes ? new TextDecoder('utf-8', { fatal: false }).decode(bytes) : ''),
    [bytes],
  )

  // Available modes: JSON tab only for raw JSON; Decoded tab when decoding or
  // have decoded data.
  const allModes: { mode: ViewMode; label: string; available: boolean; loading?: boolean }[] = [
    { mode: 'decoded' as const, label: 'Decoded', available: !!decodedData || isDecoding, loading: isDecoding },
    { mode: 'json' as const, label: 'JSON', available: !!jsonData && !decodedData },
    { mode: 'raw' as const, label: 'Raw', available: true },
    { mode: 'hex' as const, label: 'Hex', available: !!bytes },
  ]
  const availableModes = allModes.filter(m => m.available)

  // Get current data for display
  const currentData = decodedData || jsonData

  return (
    <div className="flex flex-col h-full">
      {/* View Mode Tabs */}
      <Tabs
        variant="underline"
        label="Payload view"
        idPrefix="payload"
        className="bg-surface-primary sticky top-0 z-10"
        value={viewMode}
        onChange={(mode) => handleModeChange(mode as ViewMode)}
        tabs={availableModes.map(({ mode, label, loading }) => ({
          value: mode,
          label,
          trailing: loading ? (
            <span className="inline-block w-3 h-3 border-2 border-border-focus border-t-transparent rounded-full animate-spin" />
          ) : undefined,
        }))}
      />

      {/* Content */}
      <div {...tabPanelProps('payload', viewMode)} className="flex-1 overflow-auto p-4 bg-surface-secondary">
        {/* Decoded View — JsonTreeViewer owns the truncated-preview UX; PayloadViewer stays a thin tab router. */}
        {viewMode === 'decoded' && (
          isDecoding ? (
            <div className="flex items-center justify-center h-32">
              <div className="text-center">
                <div className="inline-block w-6 h-6 border-2 border-border-focus border-t-transparent rounded-full animate-spin" />
                <p className="mt-2 text-sm text-content-tertiary">Decoding...</p>
              </div>
            </div>
          ) : decodedData ? (
            <JsonTreeViewer
              data={decodedData}
              title={typeof decodedData === 'string' ? 'Decoded Protobuf (preview)' : 'Decoded Protobuf'}
              searchable
              jsonIndentSize={jsonIndentSize}
              truncated={truncated}
              truncatedFullSize={truncatedFullSize}
              onLoadFull={onLoadFull}
              loadingFull={loadingFull}
              loadFullError={loadFullError}
            />
          ) : null
        )}

        {/* JSON View */}
        {viewMode === 'json' && currentData ? (
          <JsonTreeViewer data={currentData} title="JSON Data" searchable jsonIndentSize={jsonIndentSize} />
        ) : null}

        {/* Raw View */}
        {viewMode === 'raw' && (
          <div className="space-y-4">
            <div>
              <div className="flex items-center justify-between mb-2">
                <h3 className="text-sm font-medium text-gray-700">Base64</h3>
                <button
                  onClick={() => copyText(rawData)}
                  className="text-xs text-accent hover:text-accent-text"
                >
                  Copy
                </button>
              </div>
              <div className="bg-surface-primary p-3 rounded border font-mono text-xs break-all max-h-64 overflow-auto">
                {rawData}
              </div>
            </div>

            {bytes && (
              <div>
                <div className="flex items-center justify-between mb-2">
                  <h3 className="text-sm font-medium text-gray-700">
                    Decoded UTF-8 ({bytes.length} bytes)
                  </h3>
                  <button
                    onClick={() => copyText(rawText)}
                    className="text-xs text-accent hover:text-accent-text"
                  >
                    Copy
                  </button>
                </div>
                <div className="bg-surface-primary p-3 rounded border font-mono text-xs break-all max-h-64 overflow-auto whitespace-pre-wrap">
                  {rawText}
                </div>
              </div>
            )}
          </div>
        )}

        {/* Hex View */}
        {viewMode === 'hex' && bytes && (
          <div>
            <div className="flex items-center justify-between mb-2">
              <h3 className="text-sm font-medium text-gray-700">
                Hex Dump ({bytes.length} bytes)
              </h3>
              <button
                onClick={() => copyText(hexLines.join('\n'))}
                className="text-xs text-accent hover:text-accent-text"
              >
                Copy
              </button>
            </div>
            <div className="bg-surface-primary p-3 rounded border font-mono text-xs overflow-auto max-h-96">
              <pre className="text-gray-700">
                {hexLines.map((line, i) => (
                  <div key={i} className="hover:bg-surface-secondary">
                    {line}
                  </div>
                ))}
              </pre>
            </div>
          </div>
        )}

        {/* Headers */}
        {headers && Object.keys(headers).length > 0 && (
          <div className="mt-4">
            <h3 className="text-sm font-medium text-gray-700 mb-2">Headers</h3>
            <JsonTreeViewer data={headers} defaultExpanded searchable={false} initialHeight={128} />
          </div>
        )}
      </div>
    </div>
  )
})

export default PayloadViewer
