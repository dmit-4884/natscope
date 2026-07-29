import { useOutletContext } from 'react-router-dom'
import PublishContent from './PublishContent'
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
    handleOpenMappings,
  } = useOutletContext<StreamViewOutletContext>()

  return (
    <div className="flex flex-col h-full">
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
        />
      </div>
    </div>
  )
}
