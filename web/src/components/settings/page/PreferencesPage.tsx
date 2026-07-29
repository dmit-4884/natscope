import SettingsTab from '@/components/settings/SettingsTab'
import { SettingsPage } from './SettingsPage'

export default function PreferencesPage() {
  return (
    <SettingsPage
      title="Preferences"
      description="Display options, message fetching and live subscription behavior"
      meta="5 sections · saved per user"
      scroll="fill"
    >
      <SettingsTab />
    </SettingsPage>
  )
}
