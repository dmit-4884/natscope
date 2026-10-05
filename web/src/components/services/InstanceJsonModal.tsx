import { useState } from 'react'
import type { MicroInstance } from '@/api/discovery'
import { CopyButton, Modal, Tabs, tabPanelProps } from '@/components/ui'

type Tab = 'info' | 'stats'

function pretty(json: string | undefined): string {
  if (!json) return ''
  try {
    return JSON.stringify(JSON.parse(json), null, 2)
  } catch {
    return json
  }
}

export function InstanceJsonModal({ serviceName, instance, onClose }: { serviceName: string; instance: MicroInstance; onClose: () => void }) {
  const [tab, setTab] = useState<Tab>('info')
  const text = pretty(tab === 'info' ? instance.info_json : instance.stats_json)

  return (
    <Modal isOpen onClose={onClose} title={`${serviceName} · ${instance.id}`} size="lg">
      <Tabs
        idPrefix="instance-json"
        label="Reply"
        variant="underline"
        tabs={[
          { value: 'info', label: '$SRV.INFO' },
          { value: 'stats', label: '$SRV.STATS', disabled: !instance.stats_json },
        ]}
        value={tab}
        onChange={(value) => setTab(value as Tab)}
      />
      <div {...tabPanelProps('instance-json', tab)} className="mt-3">
        {text ? (
          <div className="relative">
            <div className="absolute right-2 top-2">
              <CopyButton value={text} label="Copy JSON" size="sm" />
            </div>
            <pre className="max-h-[60vh] overflow-auto rounded-md bg-surface-inverse p-3 text-xs font-mono text-content-inverse">{text}</pre>
          </div>
        ) : (
          <p className="text-sm text-content-tertiary">This instance did not report statistics.</p>
        )}
      </div>
    </Modal>
  )
}
