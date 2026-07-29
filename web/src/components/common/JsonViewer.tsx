import { useCopyToClipboard } from '@/hooks/useCopyToClipboard'

interface JsonViewerProps {
  data: unknown
  title?: string
}

export default function JsonViewer({ data, title }: JsonViewerProps) {
  const { copy, isCopied } = useCopyToClipboard(2000)
  const jsonString = typeof data === 'string' ? data : JSON.stringify(data, null, 2)

  const handleCopy = () => {
    void copy(jsonString)
  }

  return (
    <div className="bg-surface-inverse rounded-lg overflow-hidden">
      {title && (
        <div className="px-4 py-2 bg-gray-800 border-b border-gray-700 flex items-center justify-between">
          <h3 className="text-sm font-medium text-gray-200">{title}</h3>
          <button
            onClick={handleCopy}
            className="px-3 py-1 text-xs bg-gray-700 hover:bg-gray-600 text-gray-200 rounded transition-colors"
          >
            {isCopied ? 'Copied!' : 'Copy'}
          </button>
        </div>
      )}
      <pre className="p-4 overflow-auto min-h-96 max-h-[800px] resize-y text-sm">
        <code className="text-gray-100 font-mono">{jsonString}</code>
      </pre>
    </div>
  )
}
