import { DestructiveConfirm } from '@/components/ui'

export type StreamConfirmType = 'delete' | 'purge' | 'seal'

interface Props {
  type: StreamConfirmType
  streamName: string
  onCancel: () => void
  onConfirm: () => void
}

const MESSAGES: Record<StreamConfirmType, (name: string) => React.ReactNode> = {
  delete: (name) => (
    <span>
      This will permanently delete the stream <strong>{name}</strong> and all its messages. This action cannot be undone.
    </span>
  ),
  purge: (name) => (
    <span>
      This will delete all messages from the stream <strong>{name}</strong>. The stream configuration will be preserved.
    </span>
  ),
  seal: (name) => (
    <span>
      This will seal the stream <strong>{name}</strong>, making it read-only. This action cannot be undone.
    </span>
  ),
}

export function StreamConfirmDialog({ type, streamName, onCancel, onConfirm }: Props) {
  const verb = type.charAt(0).toUpperCase() + type.slice(1)

  return (
    <DestructiveConfirm
      isOpen
      title={`${verb} Stream`}
      description={MESSAGES[type](streamName)}
      confirmLabel={verb}
      requireTypedName={streamName}
      onCancel={onCancel}
      onConfirm={onConfirm}
    />
  )
}
