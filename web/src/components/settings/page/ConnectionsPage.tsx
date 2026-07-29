import { useState } from 'react'
import { useNavigate, useOutletContext } from 'react-router-dom'
import { toast } from '@/utils/toast'
import { mapWithLimit } from '@/utils/async'
import { plural } from '@/utils/plural'
import { getErrorMessage, stripErrorCodePrefix } from '@/api/errors'
import { Button, DestructiveConfirm, EmptyState, RefreshIcon, PlusIcon, UploadIcon, DownloadIcon, BoltIcon } from '@/components/ui'
import Tooltip from '@/components/common/Tooltip'
import type { SavedConnection } from '@/api/connections'
import {
  useConnections,
  useCreateConnection,
  useDeleteConnection,
  useDuplicateConnection,
  useTestConnection,
} from '@/contexts/connection'
import { useMappingItems, useBulkSaveMappings } from '@/contexts/mappings'
import { useConnectionImportExport } from '@/components/connections/manager/useConnectionImportExport'
import { ConnectionCard } from '@/components/connections/manager/ConnectionCard'
import type { ConnectionOutletContext } from '@/components/common/ConnectedLayout'
import { SettingsPage } from './SettingsPage'

const PING_CONCURRENCY = 5

export default function ConnectionsPage() {
  const navigate = useNavigate()
  const ctx = useOutletContext<ConnectionOutletContext | undefined>()
  const activeConnectionId = ctx?.currentConnection?.id

  const { data: connections = [] } = useConnections()
  const createConnectionMutation = useCreateConnection()
  const deleteConnectionMutation = useDeleteConnection()
  const duplicateConnectionMutation = useDuplicateConnection()
  const testConnectionMutation = useTestConnection()
  const { data: mappings = [] } = useMappingItems()
  const bulkSaveMappingsMutation = useBulkSaveMappings()

  // pingingIds drives per-card spinners; progress is null when idle or
  // {done, total} during a sweep so the header can show "Pinging X / N".
  const [pingingIds, setPingingIds] = useState<Set<string>>(new Set())
  const [progress, setProgress] = useState<{ done: number; total: number } | null>(null)
  const pingingAll = progress !== null
  const [pendingDelete, setPendingDelete] = useState<SavedConnection | null>(null)

  const { fileInputRef, triggerImport, handleExport, handleImport } = useConnectionImportExport({
    connections,
    mappings,
    createConnection: (conn) => createConnectionMutation.mutateAsync(conn),
    bulkSaveMappings: (next) => bulkSaveMappingsMutation.mutateAsync(next),
  })

  const handleDelete = (id: string) => {
    const target = connections.find((c) => c.id === id) ?? null
    setPendingDelete(target)
  }

  const confirmDelete = async () => {
    if (!pendingDelete) return
    try {
      await deleteConnectionMutation.mutateAsync({ id: pendingDelete.id })
    } catch {
      toast.error('Failed to delete connection')
    } finally {
      setPendingDelete(null)
    }
  }

  const handleDuplicate = async (connection: SavedConnection) => {
    // Connection names are unique on the backend; probe "(copy)", "(copy 2)", …
    const taken = new Set(connections.map((c) => c.name))
    let name = `${connection.name} (copy)`
    for (let i = 2; taken.has(name); i++) {
      name = `${connection.name} (copy ${i})`
    }
    try {
      const created = await duplicateConnectionMutation.mutateAsync({ id: connection.id, name })
      toast.success(`Duplicated as "${created.name}"`)
    } catch (err) {
      const msg = getErrorMessage(err) || 'Duplicate failed'
      toast.error(`${connection.name}: ${msg}`)
    }
  }

  const handleConnect = (connection: SavedConnection) => {
    if (ctx?.handleSwitchConnection) {
      ctx.handleSwitchConnection(connection)
    } else {
      toast.error('Switching not available outside Connected layout')
    }
  }

  const markPinging = (id: string, on: boolean) => {
    setPingingIds((prev) => {
      const next = new Set(prev)
      if (on) next.add(id)
      else next.delete(id)
      return next
    })
  }

  const handlePingOne = async (connection: SavedConnection) => {
    markPinging(connection.id, true)
    try {
      const result = await testConnectionMutation.mutateAsync({
        urls: connection.urls,
        connectionId: connection.id,
      })
      if (result.success) {
        toast.success(
          `${connection.name}: v${result.serverVersion} · ${result.rttMs}ms${result.jetstreamEnabled ? ' · JS' : ''}`,
        )
      } else {
        toast.error(`${connection.name}: ${result.error ? stripErrorCodePrefix(result.error) : 'failed'}`)
      }
    } catch (err) {
      const msg = getErrorMessage(err) || 'Test failed'
      toast.error(`${connection.name}: ${msg}`)
    } finally {
      markPinging(connection.id, false)
    }
  }

  const handlePingAll = async () => {
    if (connections.length === 0 || pingingAll) return
    setProgress({ done: 0, total: connections.length })
    let ok = 0
    let fail = 0
    let done = 0
    await mapWithLimit(connections, PING_CONCURRENCY, async (c) => {
      markPinging(c.id, true)
      try {
        const r = await testConnectionMutation.mutateAsync({ urls: c.urls, connectionId: c.id })
        if (r.success) ok++
        else fail++
      } catch {
        fail++
      } finally {
        markPinging(c.id, false)
        done++
        setProgress({ done, total: connections.length })
      }
    })
    setProgress(null)
    if (fail === 0) {
      toast.success(`All ${ok}/${connections.length} probes succeeded`)
    } else if (ok === 0) {
      toast.error(`All ${fail}/${connections.length} probes failed`)
    } else {
      toast.warning(`${ok}/${connections.length} ok, ${fail} failed`)
    }
  }

  return (
    <SettingsPage
      title="Connections"
      description="Manage saved NATS server connections"
      meta={plural(connections.length, 'saved connection')}
      actions={
        <>
          <input
            ref={fileInputRef}
            type="file"
            accept=".json"
            onChange={handleImport}
            className="hidden"
          />
          <Tooltip content="Test every saved connection in parallel and update their status">
            <Button
              variant="secondary"
              onClick={handlePingAll}
              disabled={pingingAll || connections.length === 0}
              icon={<RefreshIcon className={pingingAll ? 'animate-spin' : ''} />}
            >
              {pingingAll ? `Pinging ${progress?.done}/${progress?.total}…` : 'Ping all'}
            </Button>
          </Tooltip>
          <Button variant="secondary" onClick={triggerImport} icon={<UploadIcon />}>
            Import
          </Button>
          <Button variant="secondary" onClick={handleExport} icon={<DownloadIcon />}>
            Export
          </Button>
          <Button onClick={() => navigate('/settings/connections/new')} icon={<PlusIcon />}>
            New connection
          </Button>
        </>
      }
    >
      {connections.length === 0 ? (
        <EmptyState
          icon={<BoltIcon className="w-full h-full" />}
          title="No saved connections yet"
          description="Add a NATS server connection to start browsing streams."
          action={
            <Button onClick={() => navigate('/settings/connections/new')} icon={<PlusIcon />}>
              New connection
            </Button>
          }
        />
      ) : (
        <div className="space-y-3">
          {connections.map((conn) => (
            <ConnectionCard
              key={conn.id}
              connection={conn}
              isActive={conn.id === activeConnectionId}
              isPinging={pingingIds.has(conn.id)}
              onConnect={() => handleConnect(conn)}
              onPing={pingingAll ? undefined : () => handlePingOne(conn)}
              onDuplicate={pingingAll ? undefined : () => handleDuplicate(conn)}
              onEdit={pingingAll ? undefined : () => navigate(`/settings/connections/${conn.id}/edit`)}
              onDelete={pingingAll ? undefined : () => handleDelete(conn.id)}
            />
          ))}
        </div>
      )}
      <DestructiveConfirm
        isOpen={pendingDelete !== null}
        title="Delete connection"
        description={
          pendingDelete ? (
            <span>
              Permanently remove connection <strong>{pendingDelete.name}</strong>?
              Saved credentials will be lost.
            </span>
          ) : null
        }
        confirmLabel="Delete"
        isPending={deleteConnectionMutation.isPending}
        onCancel={() => setPendingDelete(null)}
        onConfirm={confirmDelete}
      />
    </SettingsPage>
  )
}
