import { useState, useEffect, useCallback } from 'react'
import { toast } from '@/utils/toast'
import {
  useSettings,
  useUpdateSettings,
  useResetSettings,
  type MessageFetchPolicyInput,
  type LiveSubscriptionPolicyInput,
  type DisplayPreferencesInput,
  type PublishPolicyInput,
  type BehaviorPolicyInput,
} from '@/contexts/settings'
import { Spinner } from '@/components/ui'
import { HelpModal } from './HelpModal'
import { MessageFetchPolicySection } from './sections/MessageFetchPolicySection'
import { LiveSubscriptionSection } from './sections/LiveSubscriptionSection'
import { DisplayPreferencesSection } from './sections/DisplayPreferencesSection'
import { PublishPolicySection } from './sections/PublishPolicySection'
import { BehaviorSettingsSection } from './sections/BehaviorSettingsSection'
import { SettingsSaveBar } from './SettingsSaveBar'

type SectionId = 'messages' | 'live' | 'display' | 'publish' | 'behavior'

export default function SettingsTab() {
  const { data: settings, isLoading } = useSettings()
  const updateSettings = useUpdateSettings()
  const resetSettings = useResetSettings()

  const [messages, setMessages] = useState<MessageFetchPolicyInput>({})
  const [live, setLive] = useState<LiveSubscriptionPolicyInput>({})
  const [display, setDisplay] = useState<DisplayPreferencesInput>({})
  const [publish, setPublish] = useState<PublishPolicyInput>({})
  const [behavior, setBehavior] = useState<BehaviorPolicyInput>({})

  const [openSections, setOpenSections] = useState<Record<SectionId, boolean>>({
    messages: true,
    live: true,
    display: true,
    publish: true,
    behavior: true,
  })

  const [hasChanges, setHasChanges] = useState(false)
  const [activeHelp, setActiveHelp] = useState<string | null>(null)

  useEffect(() => {
    if (!settings || hasChanges) return
    setMessages(settings.messages.toObject())
    setLive(settings.live.toObject())
    setDisplay(settings.display.toObject())
    setPublish(settings.publish.toObject())
    setBehavior(settings.behavior.toObject())
  }, [settings, hasChanges])

  const toggleSection = useCallback((section: SectionId) => {
    setOpenSections((prev) => ({ ...prev, [section]: !prev[section] }))
  }, [])

  const updateMsg = useCallback((patch: Partial<MessageFetchPolicyInput>) => {
    setMessages((prev) => ({ ...prev, ...patch }))
    setHasChanges(true)
  }, [])

  const updateLv = useCallback((patch: Partial<LiveSubscriptionPolicyInput>) => {
    setLive((prev) => ({ ...prev, ...patch }))
    setHasChanges(true)
  }, [])

  const updateDisp = useCallback((patch: Partial<DisplayPreferencesInput>) => {
    setDisplay((prev) => ({ ...prev, ...patch }))
    setHasChanges(true)
  }, [])

  const updatePub = useCallback((patch: Partial<PublishPolicyInput>) => {
    setPublish((prev) => ({ ...prev, ...patch }))
    setHasChanges(true)
  }, [])

  const updateBehavior = useCallback((patch: Partial<BehaviorPolicyInput>) => {
    setBehavior((prev) => ({ ...prev, ...patch }))
    setHasChanges(true)
  }, [])

  const handleSave = () => {
    updateSettings.mutate(
      { messages, live, display, publish, behavior },
      {
        onSuccess: () => {
          toast.success('Settings saved')
          setHasChanges(false)
        },
        onError: (err) => {
          toast.error(`Failed to save settings: ${err.message}`)
        },
      },
    )
  }

  const handleReset = () => {
    resetSettings.mutate(undefined, {
      onSuccess: () => {
        toast.success('Settings reset to defaults')
        setHasChanges(false)
      },
      onError: (err) => {
        toast.error(`Failed to reset settings: ${err.message}`)
      },
    })
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Spinner size="lg" className="text-accent" />
        <span className="ml-2 text-sm text-content-tertiary">Loading settings...</span>
      </div>
    )
  }

  return (
    <div className="flex flex-col h-full min-h-0">
      {activeHelp && <HelpModal helpKey={activeHelp} onClose={() => setActiveHelp(null)} />}

      <div className="flex-1 space-y-6 overflow-y-auto">
        <MessageFetchPolicySection
          value={messages}
          onChange={updateMsg}
          isOpen={openSections.messages}
          onToggle={() => toggleSection('messages')}
          onHelp={setActiveHelp}
        />
        <LiveSubscriptionSection
          value={live}
          onChange={updateLv}
          isOpen={openSections.live}
          onToggle={() => toggleSection('live')}
          onHelp={setActiveHelp}
        />
        <DisplayPreferencesSection
          value={display}
          onChange={updateDisp}
          isOpen={openSections.display}
          onToggle={() => toggleSection('display')}
          onHelp={setActiveHelp}
        />
        <PublishPolicySection
          value={publish}
          onChange={updatePub}
          isOpen={openSections.publish}
          onToggle={() => toggleSection('publish')}
          onHelp={setActiveHelp}
        />
        <BehaviorSettingsSection
          value={behavior}
          onChange={updateBehavior}
          isOpen={openSections.behavior}
          onToggle={() => toggleSection('behavior')}
        />
      </div>

      <SettingsSaveBar
        canSave={hasChanges}
        isSaving={updateSettings.isPending}
        isResetting={resetSettings.isPending}
        onSave={handleSave}
        onReset={handleReset}
      />
    </div>
  )
}
