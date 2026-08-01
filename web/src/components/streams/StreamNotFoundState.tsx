import { useNavigate } from 'react-router-dom'
import { Button, EmptyState, WarningIcon } from '@/components/ui'

interface StreamNotFoundStateProps {
  streamName: string
}

export default function StreamNotFoundState({ streamName }: StreamNotFoundStateProps) {
  const navigate = useNavigate()

  return (
    <EmptyState
      size="lg"
      icon={<WarningIcon className="w-full h-full" />}
      title={`Stream "${streamName}" not found`}
      description="It may have been deleted, or it lives on a different connection."
      action={
        <Button onClick={() => navigate('/streams')}>
          Back to streams
        </Button>
      }
    />
  )
}
