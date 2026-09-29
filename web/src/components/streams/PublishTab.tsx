import { useOutletContext } from 'react-router-dom'
import { useServerCapabilities } from '@/contexts/connection'
import PublishContent from './PublishContent'
import { optionAvailability } from './publish/publishOptions'
import type { StreamViewOutletContext } from './StreamView'

export default function PublishTab() {
  const {
    connectionId,
    streamName,
    subjects,
    publishPattern,
    setPublishPattern,
    publishWildcards,
    setPublishWildcards,
    publishMessageJson,
    setPublishMessageJson,
    publishHeaders,
    setPublishHeaders,
    streamMaxMsgSize,
    streamPublishFeatures,
    handleOpenMappings,
  } = useOutletContext<StreamViewOutletContext>()
  const { unsupportedReason } = useServerCapabilities(connectionId)

  return (
    <div className="flex-1 flex flex-col min-h-0">
      <div className="flex-1 overflow-auto">
        <PublishContent
          subjects={subjects}
          connectionId={connectionId}
          streamName={streamName}
          subjectPattern={publishPattern}
          onSubjectPatternChange={setPublishPattern}
          wildcardValues={publishWildcards}
          onWildcardValuesChange={setPublishWildcards}
          messageJson={publishMessageJson}
          onMessageJsonChange={setPublishMessageJson}
          headers={publishHeaders}
          onHeadersChange={setPublishHeaders}
          maxMsgSize={streamMaxMsgSize}
          onOpenMappings={handleOpenMappings}
          jetStreamOptions={optionAvailability(unsupportedReason, streamPublishFeatures)}
        />
      </div>
    </div>
  )
}
