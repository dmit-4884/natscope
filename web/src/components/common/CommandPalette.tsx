import { useState, useCallback, useMemo } from 'react'
import { Command } from 'cmdk'
import { useHotkeys } from 'react-hotkeys-hook'
import { useNavigate } from 'react-router-dom'
import { useQueryClient } from '@tanstack/react-query'
import { clearActiveConnection, useActiveConnection } from '@/contexts/connection'
import { kvKeys, type KVBucketInfo } from '@/contexts/kv'
import { objectKeys, type ObjectBucketInfo } from '@/contexts/objects'
import { useDisplayPreferences, useUpdateSettings } from '@/contexts/settings'
import { CONNECTION_QUERY_PREFIX } from '@/hooks/useConnectionQuery'
import { useDialogA11y } from '@/hooks/useDialogA11y'
import { resetAllStores } from '@/stores/resetAllStores'
import { usePreferencesStore } from '@/stores/preferencesStore'
import type { StreamInfo } from '@/types/nats'

interface CommandItem {
  id: string
  label: string
  group: string
  action: () => void
  shortcut?: string
}

export default function CommandPalette() {
  const [open, setOpen] = useState(false)
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const density = useDisplayPreferences().density
  const { mutate: updateSettings } = useUpdateSettings()
  const recentCommands = usePreferencesStore((s) => s.recentCommands)
  const addRecentCommand = usePreferencesStore((s) => s.addRecentCommand)

  useHotkeys('mod+k', (e) => {
    e.preventDefault()
    setOpen((prev) => !prev)
  })

  const handleClose = useCallback(() => setOpen(false), [])
  const { dialogRef, onKeyDown } = useDialogA11y<HTMLDivElement>(open, handleClose)

  const runCommand = useCallback(
    (id: string, action: () => void) => {
      addRecentCommand(id)
      action()
      setOpen(false)
    },
    [addRecentCommand]
  )

  const { connectionId } = useActiveConnection()
  const resourceCommands = useMemo<CommandItem[]>(() => {
    if (!open || !connectionId) return []
    const out: CommandItem[] = []
    const streamsData = queryClient.getQueryData<{ streams: StreamInfo[] }>([
      CONNECTION_QUERY_PREFIX,
      connectionId,
      'streams',
    ])
    for (const s of streamsData?.streams ?? []) {
      if (s.name.startsWith('KV_') || s.name.startsWith('OBJ_')) continue
      out.push({
        id: `stream-${s.name}`,
        label: `Stream: ${s.name}`,
        group: 'Resources',
        action: () => navigate(`/streams/${encodeURIComponent(s.name)}`),
      })
    }
    const kvBuckets = queryClient.getQueryData<KVBucketInfo[]>(kvKeys.buckets(connectionId))
    const kvBucketNames = new Set<string>((kvBuckets ?? []).map((b) => b.bucket))
    for (const s of streamsData?.streams ?? []) {
      if (s.name.startsWith('KV_')) kvBucketNames.add(s.name.slice('KV_'.length))
    }
    for (const name of kvBucketNames) {
      out.push({
        id: `kv-${name}`,
        label: `KV bucket: ${name}`,
        group: 'Resources',
        action: () => navigate(`/kv/${encodeURIComponent(name)}`),
      })
    }
    const objBuckets = queryClient.getQueryData<ObjectBucketInfo[]>(objectKeys.buckets(connectionId))
    const objBucketNames = new Set<string>((objBuckets ?? []).map((b) => b.bucket))
    for (const s of streamsData?.streams ?? []) {
      if (s.name.startsWith('OBJ_')) objBucketNames.add(s.name.slice('OBJ_'.length))
    }
    for (const name of objBucketNames) {
      out.push({
        id: `obj-${name}`,
        label: `Object bucket: ${name}`,
        group: 'Resources',
        action: () => navigate(`/objects/${encodeURIComponent(name)}`),
      })
    }
    return out
  }, [open, connectionId, queryClient, navigate])

  const commands: CommandItem[] = [
    {
      id: 'go-streams',
      label: 'Go to Streams',
      group: 'Navigation',
      action: () => navigate('/streams'),
    },
    {
      id: 'go-kv',
      label: 'Go to KV Stores',
      group: 'Navigation',
      action: () => navigate('/kv'),
    },
    {
      id: 'go-objects',
      label: 'Go to Object Stores',
      group: 'Navigation',
      action: () => navigate('/objects'),
    },
    {
      id: 'toggle-density',
      label: 'Toggle Density',
      group: 'Preferences',
      action: () =>
        updateSettings({ display: { density: density === 'compact' ? 'comfortable' : 'compact' } }),
    },
    {
      id: 'disconnect',
      label: 'Disconnect',
      group: 'Connection',
      action: () => {
        clearActiveConnection()
        resetAllStores()
        const filter = { queryKey: [CONNECTION_QUERY_PREFIX] }
        void queryClient.cancelQueries(filter).then(() => {
          queryClient.removeQueries(filter)
        })
        navigate('/', { replace: true })
      },
    },
  ]

  const recentItems = recentCommands
    .map((id) => commands.find((c) => c.id === id))
    .filter(Boolean) as CommandItem[]

  if (!open) return null

  return (
    <div className="fixed inset-0 z-50">
      {/* Overlay */}
      <div
        className="fixed inset-0 bg-black/50"
        onClick={() => setOpen(false)}
      />

      {/* Dialog */}
      <div
        ref={dialogRef}
        role="dialog"
        aria-modal="true"
        aria-label="Command palette"
        tabIndex={-1}
        onKeyDown={onKeyDown}
        className="fixed top-[20%] left-1/2 -translate-x-1/2 w-full max-w-lg focus:outline-none"
      >
        <Command
          className="bg-surface-primary rounded-xl shadow-xl border border-border overflow-hidden"
          label="Command Palette"
        >
          <Command.Input
            placeholder="Type a command or search..."
            className="w-full px-4 py-3 text-sm border-b border-border outline-none placeholder-content-muted"
          />

          <Command.List className="max-h-80 overflow-y-auto p-2">
            <Command.Empty className="px-4 py-8 text-sm text-content-tertiary text-center">
              No results found.
            </Command.Empty>

            {recentItems.length > 0 && (
              <Command.Group
                heading="Recent"
                className="[&_[cmdk-group-heading]]:px-2 [&_[cmdk-group-heading]]:py-1.5 [&_[cmdk-group-heading]]:text-xs [&_[cmdk-group-heading]]:font-medium [&_[cmdk-group-heading]]:text-content-muted"
              >
                {recentItems.map((cmd) => (
                  <Command.Item
                    key={`recent-${cmd.id}`}
                    value={`recent ${cmd.label}`}
                    onSelect={() => runCommand(cmd.id, cmd.action)}
                    className="flex items-center gap-2 px-3 py-2 text-sm text-gray-700 rounded-lg cursor-pointer data-[selected=true]:bg-surface-tertiary"
                  >
                    <span className="text-content-muted text-xs">&#8635;</span>
                    {cmd.label}
                  </Command.Item>
                ))}
              </Command.Group>
            )}

            <Command.Group
              heading="Navigation"
              className="[&_[cmdk-group-heading]]:px-2 [&_[cmdk-group-heading]]:py-1.5 [&_[cmdk-group-heading]]:text-xs [&_[cmdk-group-heading]]:font-medium [&_[cmdk-group-heading]]:text-content-muted"
            >
              {commands
                .filter((c) => c.group === 'Navigation')
                .map((cmd) => (
                  <Command.Item
                    key={cmd.id}
                    value={cmd.label}
                    onSelect={() => runCommand(cmd.id, cmd.action)}
                    className="flex items-center gap-2 px-3 py-2 text-sm text-gray-700 rounded-lg cursor-pointer data-[selected=true]:bg-surface-tertiary"
                  >
                    {cmd.label}
                  </Command.Item>
                ))}
            </Command.Group>

            {resourceCommands.length > 0 && (
              <Command.Group
                heading="Resources"
                className="[&_[cmdk-group-heading]]:px-2 [&_[cmdk-group-heading]]:py-1.5 [&_[cmdk-group-heading]]:text-xs [&_[cmdk-group-heading]]:font-medium [&_[cmdk-group-heading]]:text-content-muted"
              >
                {resourceCommands.map((cmd) => (
                  <Command.Item
                    key={cmd.id}
                    value={cmd.label}
                    onSelect={() => runCommand(cmd.id, cmd.action)}
                    className="flex items-center gap-2 px-3 py-2 text-sm text-gray-700 rounded-lg cursor-pointer data-[selected=true]:bg-surface-tertiary"
                  >
                    <span className="font-mono truncate">{cmd.label}</span>
                  </Command.Item>
                ))}
              </Command.Group>
            )}

            <Command.Group
              heading="Preferences"
              className="[&_[cmdk-group-heading]]:px-2 [&_[cmdk-group-heading]]:py-1.5 [&_[cmdk-group-heading]]:text-xs [&_[cmdk-group-heading]]:font-medium [&_[cmdk-group-heading]]:text-content-muted"
            >
              {commands
                .filter((c) => c.group === 'Preferences')
                .map((cmd) => (
                  <Command.Item
                    key={cmd.id}
                    value={cmd.label}
                    onSelect={() => runCommand(cmd.id, cmd.action)}
                    className="flex items-center gap-2 px-3 py-2 text-sm text-gray-700 rounded-lg cursor-pointer data-[selected=true]:bg-surface-tertiary"
                  >
                    {cmd.label}
                  </Command.Item>
                ))}
            </Command.Group>

            <Command.Group
              heading="Connection"
              className="[&_[cmdk-group-heading]]:px-2 [&_[cmdk-group-heading]]:py-1.5 [&_[cmdk-group-heading]]:text-xs [&_[cmdk-group-heading]]:font-medium [&_[cmdk-group-heading]]:text-content-muted"
            >
              {commands
                .filter((c) => c.group === 'Connection')
                .map((cmd) => (
                  <Command.Item
                    key={cmd.id}
                    value={cmd.label}
                    onSelect={() => runCommand(cmd.id, cmd.action)}
                    className="flex items-center gap-2 px-3 py-2 text-sm text-gray-700 rounded-lg cursor-pointer data-[selected=true]:bg-surface-tertiary text-status-error-text"
                  >
                    {cmd.label}
                  </Command.Item>
                ))}
            </Command.Group>
          </Command.List>

          <div className="border-t border-border px-4 py-2 flex items-center justify-between text-xs text-content-muted">
            <div className="flex items-center gap-2">
              <span>Navigate</span>
              <kbd className="px-1.5 py-0.5 bg-surface-tertiary rounded text-2xs font-mono">&#8593;&#8595;</kbd>
            </div>
            <div className="flex items-center gap-2">
              <span>Select</span>
              <kbd className="px-1.5 py-0.5 bg-surface-tertiary rounded text-2xs font-mono">&#9166;</kbd>
            </div>
            <div className="flex items-center gap-2">
              <span>Close</span>
              <kbd className="px-1.5 py-0.5 bg-surface-tertiary rounded text-2xs font-mono">Esc</kbd>
            </div>
          </div>
        </Command>
      </div>
    </div>
  )
}
