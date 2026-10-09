import { useSchemaTypes } from '@/contexts/proto'
import ProtoManager from '@/components/proto/ProtoManager'
import { plural } from '@/utils/plural'
import { SettingsPage } from './SettingsPage'

export default function ProtoPage() {
  const { data: types } = useSchemaTypes()
  const messages = (types ?? []).filter((t) => t.kind === 'message' && !t.dependency)
  const packageCount = new Set(messages.map((t) => t.packageName)).size

  const meta = !types
    ? null
    : messages.length > 0
      ? `${plural(packageCount, 'package')} · ${plural(messages.length, 'message')}`
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
