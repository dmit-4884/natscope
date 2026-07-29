import type { Message } from '@/types/nats'

interface Props {
  messageA: Message | null
  messageB: Message | null
  onOpenDiff: () => void
  onCancel: () => void
}

export function CompareModeBar({ messageA, messageB, onOpenDiff, onCancel }: Props) {
  return (
    <div className="px-4 py-2 bg-indigo-50 border-b border-indigo-200 flex items-center justify-between">
      <div className="flex items-center gap-3 text-sm text-indigo-700">
        <span className="font-medium">Compare mode:</span>
        {messageA ? (
          <span className="bg-indigo-100 px-2 py-0.5 rounded font-mono text-xs">seq {messageA.sequence}</span>
        ) : (
          <span className="text-indigo-400">click 1st message</span>
        )}
        <span className="text-indigo-300">&rarr;</span>
        {messageB ? (
          <span className="bg-indigo-100 px-2 py-0.5 rounded font-mono text-xs">seq {messageB.sequence}</span>
        ) : (
          <span className="text-indigo-400">{messageA ? 'click 2nd message' : '...'}</span>
        )}
      </div>
      <div className="flex items-center gap-2">
        {messageA && messageB && (
          <button
            onClick={onOpenDiff}
            className="px-3 py-1 text-xs font-medium bg-indigo-600 text-content-inverse rounded-md hover:bg-indigo-700"
          >
            View Diff
          </button>
        )}
        <button onClick={onCancel} className="px-3 py-1 text-xs font-medium text-indigo-600 hover:bg-indigo-100 rounded-md">
          Cancel
        </button>
      </div>
    </div>
  )
}
