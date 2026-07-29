import { useProtoMessageEntities, groupMessagesByPackage } from '@/contexts/proto'
import ProtoManager from '@/components/proto/ProtoManager'
import { plural } from '@/utils/plural'
import { SettingsPage } from './SettingsPage'

export default function ProtoPage() {
  const { messages } = useProtoMessageEntities()
  const groupedByPackage = groupMessagesByPackage(messages)
  const packageCount = Object.keys(groupedByPackage).length
  const messageCount = messages.length

  const meta =
    messageCount > 0
      ? `${plural(packageCount, 'package')} · ${plural(messageCount, 'message')}`
      : 'No proto files loaded'

  return (
    <SettingsPage
      title="Proto Files"
      description="Manage proto sources, versions and compiled descriptors"
      meta={meta}
    >
      <ProtoManager />
    </SettingsPage>
  )
}
