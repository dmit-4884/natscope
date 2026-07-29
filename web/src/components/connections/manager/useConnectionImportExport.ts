import { useRef } from 'react'
import { downloadBlob } from '@/utils/download'
import { toast } from '@/utils/toast'
import type { SavedConnection, CreateConnectionRequest } from '@/api/connections'
import { AuthConfig, toApiAuthConfig } from '@/contexts/connection'
import type { MappingItem } from '@/contexts/mappings'

interface Options {
  connections: SavedConnection[]
  mappings: MappingItem[]
  createConnection: (req: CreateConnectionRequest) => Promise<unknown>
  bulkSaveMappings: (
    next: Array<{ pattern: string; messageType: string; sourceId: string }>,
  ) => Promise<unknown>
}

/**
 * v2 export format; import is strict about `sourceId` — items missing it are
 * dropped. Pre-redesign exports (keyed by pattern) need manual re-export.
 */
export function useConnectionImportExport({
  connections,
  mappings,
  createConnection,
  bulkSaveMappings,
}: Options) {
  const fileInputRef = useRef<HTMLInputElement>(null)

  const triggerImport = () => fileInputRef.current?.click()

  const handleExport = () => {
    // SECURITY: never write credentials to a shareable file — strip auth
    // secrets and TLS private material; keep only method, username, CA cert.
    const redactedConnections = connections.map((c) => ({
      ...c,
      auth: c.auth ? { method: c.auth.method, username: c.auth.username } : undefined,
      tls: c.tls
        ? { caCert: c.tls.caCert, skipVerify: c.tls.skipVerify, tlsFirst: c.tls.tlsFirst }
        : undefined,
    }))
    const config = {
      version: 2,
      connections: redactedConnections,
      mappings: mappings.map((m) => ({
        pattern: m.pattern,
        messageType: m.messageType,
        sourceId: m.sourceId,
      })),
      exported_at: new Date().toISOString(),
    }
    downloadBlob(JSON.stringify(config, null, 2), `natscope-config-${new Date().toISOString().split('T')[0]}.json`)
    toast.warning(
      'Credentials excluded from export. Passwords, tokens, NKeys, and TLS private keys are never written to the export file — re-enter them after importing.',
    )
  }

  const handleImport = (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0]
    if (!file) return
    const reader = new FileReader()
    reader.onload = async (e) => {
      try {
        const content = e.target?.result as string
        const config = JSON.parse(content)

        if (!config.connections || !Array.isArray(config.connections)) {
          toast.error('Invalid config file format')
          return
        }

        const existingNames = new Set(connections.map((c) => c.name))
        let imported = 0
        let skipped = 0
        let failed = 0
        for (const conn of config.connections as Partial<SavedConnection>[]) {
          if (!conn || typeof conn.name !== 'string' || !conn.name.trim() ||
              !Array.isArray(conn.urls) || conn.urls.length === 0 ||
              conn.urls.some((u) => typeof u !== 'string' || !u.trim())) {
            failed++
            toast.error(`Skipped malformed connection entry${conn?.name ? `: ${conn.name}` : ''}`)
            continue
          }
          if (existingNames.has(conn.name)) {
            skipped++
            continue
          }
          try {
            const authProto = conn.auth
              ? toApiAuthConfig(AuthConfig.fromTrusted(conn.auth))
              : undefined
            await createConnection({
              name: conn.name,
              description: conn.description,
              urls: conn.urls,
              auth: authProto,
              // TLS / connection / reconnect / ping configs are not preserved
              // by the export at this time.
            })
            imported++
          } catch {
            failed++
            toast.error(`Failed to import connection: ${conn.name}`)
          }
        }

        // Mappings (v2: array of {pattern, messageType, sourceId}).
        if (Array.isArray(config.mappings)) {
          const incoming = config.mappings as Array<{
            pattern?: string
            messageType?: string
            sourceId?: string
          }>
          const valid: Array<{ pattern: string; messageType: string; sourceId: string }> = []
          for (const m of incoming) {
            if (!m.pattern || !m.messageType || !m.sourceId) continue
            valid.push({ pattern: m.pattern, messageType: m.messageType, sourceId: m.sourceId })
          }
          if (valid.length > 0) {
            await bulkSaveMappings(valid)
          }
        } else if (config.subject_mappings) {
          toast.error(
            'Legacy mapping format detected (pre-redesign). Re-export against the current backend.',
          )
        }

        const summary = [
          `${imported} imported`,
          skipped > 0 ? `${skipped} already present` : null,
          failed > 0 ? `${failed} failed` : null,
        ].filter(Boolean).join(', ')
        if (failed > 0 && imported === 0) {
          toast.error(`Import finished: ${summary}`)
        } else {
          toast.success(`Import finished: ${summary}`)
        }
        if (imported > 0) {
          toast.warning(
            'Imported connections have no credentials — passwords, tokens, NKeys, and TLS keys were stripped during export. Open each connection and re-enter its credentials before connecting.',
          )
        }
      } catch {
        toast.error('Failed to import configuration. Please check the file format.')
      }
    }
    reader.readAsText(file)

    if (fileInputRef.current) fileInputRef.current.value = ''
  }

  return { fileInputRef, triggerImport, handleExport, handleImport }
}
