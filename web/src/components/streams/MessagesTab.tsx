import { useOutletContext } from 'react-router-dom'
import UnifiedMessageList from '../messages/UnifiedMessageList'
import type { StreamViewOutletContext } from './StreamView'

export default function MessagesTab() {
  const { scope, connectionId, streamName, selectedMessage, setSelectedMessage } = useOutletContext<StreamViewOutletContext>()

  return (
    <UnifiedMessageList
      scope={scope}
      streamName={streamName}
      connectionId={connectionId}
      onSelectMessage={setSelectedMessage}
      selectedMessageId={selectedMessage?.id}
    />
  )
}
