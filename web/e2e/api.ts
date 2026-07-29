/**
 * Thin Connect-JSON helpers for test setup/teardown. Talk straight to the
 * backend (:4280) over the Connect protocol's JSON encoding:
 * POST {base}/{service.typeName}/{Method} with a JSON body.
 */
const BACKEND = process.env.NATSCOPE_BACKEND_URL ?? 'http://localhost:4280'

export async function call<T>(service: string, method: string, body: unknown): Promise<T> {
  const res = await fetch(`${BACKEND}/${service}/${method}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  if (!res.ok) {
    let code = String(res.status)
    let message = res.statusText || 'request failed'
    try {
      const json = (await res.json()) as { code?: string; message?: string }
      code = json.code ?? code
      message = json.message ?? message
    } catch {
      // body was not JSON (e.g. HTML error page from a proxy) — use the
      // raw text so the error message is still informative.
      try {
        const text = await res.text()
        if (text) message = text.slice(0, 200)
      } catch {
        // ignore secondary read failures
      }
    }
    throw new ConnectError(code, message)
  }
  return (await res.json()) as T
}

export class ConnectError extends Error {
  constructor(
    public code: string,
    message: string,
  ) {
    super(`[${code}] ${message}`)
  }
}

interface SavedConnection {
  id: string
  name: string
  urls: string[]
}

/** Resolve the saved NATS connection by name (the user's local server). */
export async function findConnectionByName(name: string): Promise<SavedConnection> {
  const res = await call<{ connections?: SavedConnection[] }>(
    'natscope.nats.connections.v1.ConnectionsService',
    'ListConnections',
    { pageSize: 500 },
  )
  const conn = (res.connections ?? []).find((c) => c.name === name)
  if (!conn) {
    throw new Error(
      `Saved connection "${name}" not found on the backend — e2e tests need a local NATS connection`,
    )
  }
  return conn
}

/**
 * Create the e2e stream if it doesn't exist yet. If it exists with a
 * different subject set (older fixture version), recreate it — UpdateStream
 * isn't used on purpose: omitted fields would reset limits like maxMsgSize.
 */
export async function ensureStream(
  connectionId: string,
  name: string,
  subjects: string[],
  maxMsgSize: number,
): Promise<void> {
  const create = () =>
    call('natscope.nats.management.v1.ManagementService', 'CreateStream', {
      connectionId,
      name,
      subjects,
      maxMsgSize,
      maxMsgs: '10000',
    })
  try {
    await create()
    return
  } catch (e) {
    if (!(e instanceof ConnectError && /exist|in use|already/i.test(e.message))) throw e
  }
  const res = await call<{
    stream?: { config?: { subjects?: string[]; maxMsgSize?: number; maxMsgs?: string } }
  }>(
    'natscope.nats.streams.v1.StreamsService',
    'GetStream',
    { connectionId, streamName: name },
  )
  const cfg = res.stream?.config ?? {}
  const current = cfg.subjects ?? []
  const subjectsMissing = subjects.some((s) => !current.includes(s))
  const maxMsgSizeMismatch = cfg.maxMsgSize !== undefined && cfg.maxMsgSize !== maxMsgSize
  const maxMsgsMismatch =
    cfg.maxMsgs !== undefined && Number(cfg.maxMsgs) !== 10000
  if (subjectsMissing || maxMsgSizeMismatch || maxMsgsMismatch) {
    await call('natscope.nats.management.v1.ManagementService', 'DeleteStream', {
      connectionId,
      streamName: name,
    })
    await create()
  }
}

/**
 * Ensure a compiled "files" proto source + subject mapping exist, so the
 * publish tab resolves the given pattern to a proto message type.
 * Returns the source id.
 */
export async function ensureProtoMapping(
  sourceName: string,
  protoFilePath: string,
  pattern: string,
  messageType: string,
): Promise<string> {
  let sourceId: string | undefined
  const list = await call<{ sources?: Array<{ id: string; name: string }> }>(
    'natscope.proto.sources.v1.SourcesService',
    'ListSources',
    { pageSize: 500 },
  )
  sourceId = (list.sources ?? []).find((s) => s.name === sourceName)?.id
  if (!sourceId) {
    const created = await call<{ source?: { id: string } }>(
      'natscope.proto.sources.v1.SourcesService',
      'CreateSource',
      { name: sourceName, sourceType: 'SOURCE_TYPE_FILES', files: [protoFilePath], includeDirs: [] },
    )
    sourceId = created.source?.id
    if (!sourceId) throw new Error('CreateSource returned no source id')
  }
  // Recompile every run: cheap, and revalidates after backend restarts.
  await call('natscope.proto.sources.v1.SourcesService', 'CompileFiles', { sourceId })

  const mappings = await call<{ mappings?: Array<{ id: string; pattern: string; sourceId?: string }> }>(
    'natscope.mappings.v1.MappingsService',
    'ListMappings',
    { pageSize: 500 },
  )
  const existing = (mappings.mappings ?? []).find((m) => m.pattern === pattern)
  if (!existing) {
    await call('natscope.mappings.v1.MappingsService', 'CreateMapping', {
      pattern,
      messageType,
      sourceId,
    })
  } else if (existing.sourceId && existing.sourceId !== sourceId) {
    // The source was recreated with a new id — repair the stale mapping.
    await call('natscope.mappings.v1.MappingsService', 'UpdateMapping', {
      id: existing.id,
      pattern,
      messageType,
      sourceId,
    })
  }
  return sourceId
}

/** Publish a message via the backend (used to seed "prefill from last" data). */
export async function publishMessage(
  connectionId: string,
  subject: string,
  subjectPattern: string,
  data: unknown,
): Promise<void> {
  // The proto carries the payload as a JSON string, not a Struct.
  const res = await call<{ error?: string }>('natscope.nats.publish.v1.PublishService', 'PublishMessage', {
    connectionId,
    subject,
    subjectPattern,
    data: JSON.stringify(data),
  })
  if (res.error) throw new Error(res.error)
}

interface TemplateItem {
  id: string
  name: string
}

const SETTINGS_SERVICE = 'natscope.settings.settings.v1.SettingsService'

interface BehaviorSettings {
  confirmDeleteConsumer?: boolean
  confirmDeleteMessage?: boolean
  confirmDeleteKvKey?: boolean
  confirmDeleteObject?: boolean
  confirmPurgeKvHistory?: boolean
  secureDeleteDefault?: boolean
}

/** Reset all user settings to defaults (test isolation for behavior toggles). */
export async function resetSettings(): Promise<void> {
  await call(SETTINGS_SERVICE, 'ResetSettings', {})
}

/** Snapshot the current settings so they can be restored after tests. */
export async function snapshotSettings(): Promise<{
  settings?: { behavior?: BehaviorSettings; messages?: Record<string, unknown> }
}> {
  return getSettings()
}

/** Restore settings from a previous snapshot (see snapshotSettings). */
export async function restoreSettings(snap: {
  settings?: { behavior?: BehaviorSettings; messages?: Record<string, unknown> }
}): Promise<void> {
  const s = snap.settings ?? {}
  // Use UpdateSettings with all captured sections so the user's real config
  // is put back exactly as it was before the test suite ran.
  await call(SETTINGS_SERVICE, 'UpdateSettings', {
    behavior: s.behavior ?? {},
    ...(s.messages !== undefined ? { messages: s.messages } : {}),
  })
}

/** Partial-update the behavior settings section. */
export async function updateBehavior(behavior: BehaviorSettings): Promise<void> {
  await call(SETTINGS_SERVICE, 'UpdateSettings', { behavior })
}

/** Read the current user settings (behavior section included). */
export async function getSettings(): Promise<{
  settings?: { behavior?: BehaviorSettings; messages?: Record<string, unknown> }
}> {
  return call(SETTINGS_SERVICE, 'GetSettings', {})
}

/**
 * Publish a raw (non-proto) message. `data` is sent as-is (string). Useful for
 * seeding stream messages that the UI then lists/deletes/exports.
 */
export async function publishRaw(
  connectionId: string,
  subject: string,
  subjectPattern: string,
  data: string,
): Promise<void> {
  const res = await call<{ error?: string }>('natscope.nats.publish.v1.PublishService', 'PublishMessage', {
    connectionId,
    subject,
    subjectPattern,
    data,
  })
  if (res.error) throw new Error(res.error)
}

/** Read a single message's sequence, subject and publish timestamp (RFC3339). */
export async function getMessageAt(
  connectionId: string,
  streamName: string,
  sequence: number,
): Promise<{ sequence: number; subject: string; timestamp: string }> {
  const res = await call<{ message?: { sequence?: string; subject?: string; timestamp?: string } }>(
    'natscope.nats.messages.v1.MessagesService',
    'GetMessage',
    { connectionId, streamName, sequence: String(sequence) },
  )
  const m = res.message ?? {}
  return { sequence: Number(m.sequence ?? '0'), subject: m.subject ?? '', timestamp: m.timestamp ?? '' }
}

/**
 * List messages anchored at a publish time (jump-to-time). `startTime` is an
 * RFC3339 string. Returns the resolved page (sequences ascending in forward).
 */
export async function listMessagesByTime(
  connectionId: string,
  streamName: string,
  startTime: string,
  direction: 'DIRECTION_FORWARD' | 'DIRECTION_BACKWARD' = 'DIRECTION_FORWARD',
  limit = 10,
): Promise<Array<{ sequence: number; subject: string }>> {
  const res = await call<{ messages?: Array<{ sequence?: string; subject?: string }> }>(
    'natscope.nats.messages.v1.MessagesService',
    'ListMessages',
    { connectionId, streamName, startTime, direction, limit: String(limit) },
  )
  return (res.messages ?? []).map((m) => ({ sequence: Number(m.sequence ?? '0'), subject: m.subject ?? '' }))
}

/** Return the current last sequence number of a stream (0 if empty). */
export async function streamLastSeq(connectionId: string, streamName: string): Promise<number> {
  const res = await call<{ stream?: { state?: { lastSeq?: string } } }>(
    'natscope.nats.streams.v1.StreamsService',
    'GetStream',
    { connectionId, streamName },
  )
  return Number(res.stream?.state?.lastSeq ?? '0')
}

/** Return the number of present (non-deleted) messages in a stream. */
export async function streamMessageCount(connectionId: string, streamName: string): Promise<number> {
  const res = await call<{ stream?: { state?: { msgs?: string } } }>(
    'natscope.nats.streams.v1.StreamsService',
    'GetStream',
    { connectionId, streamName },
  )
  return Number(res.stream?.state?.msgs ?? '0')
}

/** True if a message at the given sequence still exists in the stream. */
export async function messageExists(
  connectionId: string,
  streamName: string,
  sequence: number,
): Promise<boolean> {
  try {
    await call('natscope.nats.messages.v1.MessagesService', 'GetMessage', {
      connectionId,
      streamName,
      sequence: String(sequence),
    })
    return true
  } catch (e) {
    if (e instanceof ConnectError && /not.?found/i.test(e.message)) return false
    throw e
  }
}

/** Delete all subject mappings whose pattern starts with the given prefix. */
export async function deleteMappingsByPrefix(prefix: string): Promise<void> {
  const res = await call<{ mappings?: Array<{ id: string; pattern: string }> }>(
    'natscope.mappings.v1.MappingsService',
    'ListMappings',
    { pageSize: 1000 },
  )
  for (const m of res.mappings ?? []) {
    if (m.pattern.startsWith(prefix)) {
      await call('natscope.mappings.v1.MappingsService', 'DeleteMapping', { id: m.id })
    }
  }
}

/** True if any subject mapping has the exact pattern. */
export async function mappingExists(pattern: string): Promise<boolean> {
  const res = await call<{ mappings?: Array<{ pattern: string }> }>(
    'natscope.mappings.v1.MappingsService',
    'ListMappings',
    { pageSize: 1000 },
  )
  return (res.mappings ?? []).some((m) => m.pattern === pattern)
}

/** Delete all templates whose name starts with the given prefix (test cleanup). */
export async function deleteTemplatesByPrefix(prefix: string): Promise<void> {
  const res = await call<{ templates?: TemplateItem[] }>(
    'natscope.templates.v1.TemplatesService',
    'ListTemplates',
    { pageSize: 500 },
  )
  for (const t of res.templates ?? []) {
    if (t.name.startsWith(prefix)) {
      await call('natscope.templates.v1.TemplatesService', 'DeleteTemplate', { id: t.id })
    }
  }
}
