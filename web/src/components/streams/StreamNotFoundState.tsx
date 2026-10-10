import { useNavigate } from 'react-router-dom'
import { Button, EmptyState, WarningIcon } from '@/components/ui'

interface StreamNotFoundStateProps {
  streamName: string
  deleted?: boolean
}

export default function StreamNotFoundState({ streamName, deleted = false }: StreamNotFoundStateProps) {
  const navigate = useNavigate()

  return (
    <EmptyState
      size="lg"
      icon={<WarningIcon className="w-full h-full" />}
      title={deleted ? 'This stream was deleted' : `Stream "${streamName}" not found`}
      description={
        deleted
          ? `Stream "${streamName}" no longer exists on this connection.`
          : 'It may have been deleted, or it lives on a different connection.'
      }
      action={
        <Button onClick={() => navigate('/streams')}>
          Back to streams
        </Button>
      }
    />
  )
}
