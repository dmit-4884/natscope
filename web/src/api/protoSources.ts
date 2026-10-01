import { tsToMillis } from '@/utils/timestamp'
import type {
  ProtoSource as ProtoSourceProto,
  CompileDiagnostic as PbDiagnostic,
  ProtoRef as PbRef,
  SchemaRevision as PbRevision,
} from '../gen/types/proto/proto_source_pb'
import { SourceType, RefKind } from '../gen/types/proto/proto_source_pb'
import type { CompileOutcome as PbOutcome } from '../gen/services/grpc/proto/v1/sources/proto_sources_service_pb'
import { sourcesClient } from './grpc/clients'

export type ProtoSourceType = 'git' | 'local' | 'upload'

/** Snapshot of the most recent compile attempt — server-populated only. */
interface ProtoCompileResult {
  at: number
  ok: boolean
  error?: string
  messageCount: number
  fileCount: number
  diagnostics: CompileDiagnostic[]
  roots: string[]
  rootsOrigin: 'manual' | 'buf' | 'inferred' | ''
}

export interface ProtoSource {
  id: string
  name: string
  repository: string
  sourceType: ProtoSourceType
  localPath?: string
  watcherEnabled: boolean
  enabled: boolean
  // Manual import roots; non-empty disables auto-detection, empty = auto.
  importRoots?: string[]
  // Slash-relative prefixes excluded from compilation (e.g. "pb", "gen").
  excludePrefixes?: string[]
  lastCompile?: ProtoCompileResult
  selectedRef?: ProtoRef
  activeSchema?: SchemaRevision
  created_at: number
  updated_at?: number
}

type RefKindName = 'tag' | 'branch' | 'commit'

export interface ProtoRef {
  name: string
  kind: RefKindName
  revision: string
}

export interface SchemaRevision {
  revision: string
  fingerprint: string
  compiledAt: number
  messageCount: number
  active: boolean
}

export interface CompileDiagnostic {
  severity: 'error' | 'warning' | 'info'
  file: string
  line: number
  column: number
  message: string
  missingImport?: string
  hint?: string
}

export interface CompileOutcome {
  valid: boolean
  messageTypes: number
  fileDescriptors: number
  diagnostics: CompileDiagnostic[]
}

export interface SourceUpdateResult {
  source: ProtoSource
  outcome: CompileOutcome
}

export type SchemaUploadContent =
  | { kind: 'files'; files: Array<{ path: string; content: string }> }
  | { kind: 'descriptorSet'; data: Uint8Array }

export interface ProtoSourcesList {
  items: ProtoSource[]
  next_cursor?: string
  total_count?: number
}

export interface CreateProtoSourceRequest {
  name: string
  sourceType: ProtoSourceType
  repository?: string
  token?: string
  localPath?: string
  watcherEnabled?: boolean
  importRoots?: string[]
  excludePrefixes?: string[]
}

export interface UpdateProtoSourceRequest {
  name?: string
  repository?: string
  token?: string
  localPath?: string
  importRoots?: string[]
  excludePrefixes?: string[]
}

function sourceTypeFromProto(st: SourceType): ProtoSourceType {
  switch (st) {
    case SourceType.LOCAL:
      return 'local'
    case SourceType.UPLOAD:
      return 'upload'
    case SourceType.GIT:
    case SourceType.UNSPECIFIED:
    default:
      return 'git'
  }
}

function sourceTypeToProto(st: ProtoSourceType): SourceType {
  switch (st) {
    case 'local':
      return SourceType.LOCAL
    case 'upload':
      return SourceType.UPLOAD
    case 'git':
    default:
      return SourceType.GIT
  }
}

function toProtoSource(p: ProtoSourceProto): ProtoSource {
  return {
    id: p.id,
    name: p.name,
    repository: p.repository,
    sourceType: sourceTypeFromProto(p.sourceType),
    localPath: p.localPath,
    watcherEnabled: p.watcherEnabled,
    enabled: p.enabled,
    importRoots: p.importRoots ?? [],
    excludePrefixes: p.excludePrefixes ?? [],
    lastCompile: p.lastCompile
      ? {
          at: Number(p.lastCompile.at),
          ok: p.lastCompile.ok,
          error: p.lastCompile.error || undefined,
          messageCount: p.lastCompile.messageCount,
          fileCount: p.lastCompile.fileCount,
          diagnostics: (p.lastCompile.diagnostics ?? []).map(fromPbDiagnostic),
          roots: p.lastCompile.roots ?? [],
          rootsOrigin: (p.lastCompile.rootsOrigin ?? '') as ProtoCompileResult['rootsOrigin'],
        }
      : undefined,
    selectedRef: p.selectedRef ? fromPbRef(p.selectedRef) : undefined,
    activeSchema: p.activeSchema ? fromPbRevision(p.activeSchema) : undefined,
    created_at: tsToMillis(p.createdAt),
    updated_at: tsToMillis(p.updatedAt) || undefined,
  }
}

function fromPbRef(r: PbRef): ProtoRef {
  const kind: RefKindName = r.kind === RefKind.BRANCH ? 'branch' : r.kind === RefKind.COMMIT ? 'commit' : 'tag'
  return { name: r.name, kind, revision: r.revision }
}

function fromPbRevision(r: PbRevision): SchemaRevision {
  return {
    revision: r.revision,
    fingerprint: r.fingerprint,
    compiledAt: Number(r.compiledAt),
    messageCount: r.messageCount,
    active: r.active,
  }
}

function fromPbOutcome(o: PbOutcome | undefined): CompileOutcome {
  return {
    valid: o?.valid ?? false,
    messageTypes: o?.messageTypes ?? 0,
    fileDescriptors: o?.fileDescriptors ?? 0,
    diagnostics: (o?.diagnostics ?? []).map(fromPbDiagnostic),
  }
}


function fromPbDiagnostic(d: PbDiagnostic): CompileDiagnostic {
  return {
    severity: d.severity === 'warning' ? 'warning' : d.severity === 'info' ? 'info' : 'error',
    file: d.file,
    line: d.line,
    column: d.column,
    message: d.message,
    missingImport: d.missingImport || undefined,
    hint: d.hint || undefined,
  }
}

// Proto Sources CRUD

export async function getProtoSources(): Promise<ProtoSourcesList> {
  const items: ProtoSource[] = []
  let pageToken = ''
  for (let page = 0; page < 100; page++) {
    const response = await sourcesClient.listSources({
      pageSize: 500,
      pageToken,
    })
    for (const s of response.sources) {
      items.push(toProtoSource(s))
    }
    if (!response.nextPageToken) break
    pageToken = response.nextPageToken
  }
  return { items }
}

export async function getProtoSource(id: string): Promise<ProtoSource> {
  const response = await sourcesClient.getSource({ id })
  return toProtoSource(response.source!)
}

export async function createProtoSource(data: CreateProtoSourceRequest): Promise<ProtoSource> {
  const response = await sourcesClient.createSource({
    name: data.name,
    sourceType: sourceTypeToProto(data.sourceType),
    repository: data.repository,
    token: data.token,
    localPath: data.localPath,
    watcherEnabled: data.watcherEnabled,
    importRoots: data.importRoots ?? [],
    excludePrefixes: data.excludePrefixes ?? [],
  })
  return toProtoSource(response.source!)
}

export async function updateProtoSource(
  id: string,
  data: UpdateProtoSourceRequest,
): Promise<ProtoSource> {
  const response = await sourcesClient.updateSource({
    id,
    name: data.name,
    repository: data.repository,
    token: data.token,
    localPath: data.localPath,
    importRoots: data.importRoots ?? [],
    excludePrefixes: data.excludePrefixes ?? [],
  })
  return toProtoSource(response.source!)
}

export async function deleteProtoSource(id: string): Promise<void> {
  await sourcesClient.deleteSource({
    id,
  })
}

// Enabled / Watcher toggles

export async function setSourceEnabled(sourceId: string, enabled: boolean): Promise<ProtoSource> {
  const response = await sourcesClient.setEnabled({ sourceId, enabled })
  return toProtoSource(response.source!)
}

export async function setWatcher(sourceId: string, enabled: boolean): Promise<ProtoSource> {
  const response = await sourcesClient.setWatcher({ sourceId, enabled })
  return toProtoSource(response.source!)
}

export async function refreshSource(sourceId: string): Promise<SourceUpdateResult> {
  const response = await sourcesClient.refreshSource({ sourceId })
  return { source: toProtoSource(response.source!), outcome: fromPbOutcome(response.outcome) }
}

// Validate local path

export async function validateLocalPath(path: string): Promise<{ valid: boolean; protoFileCount: number; error?: string }> {
  const response = await sourcesClient.validateLocalPath({ path })
  return {
    valid: response.valid,
    protoFileCount: response.protoFileCount,
    error: response.error,
  }
}

// Validate repository

export async function validateRepository(
  repository: string,
  token?: string,
): Promise<{ valid: boolean; error?: string }> {
  const response = await sourcesClient.validateRepository({ repository, token })
  return {
    valid: response.valid,
    error: response.error,
  }
}

export async function listSourceRefs(sourceId: string): Promise<ProtoRef[]> {
  const response = await sourcesClient.listRefs({ sourceId })
  return response.refs.map(fromPbRef)
}

export async function selectSourceRef(sourceId: string, ref: string): Promise<SourceUpdateResult> {
  const response = await sourcesClient.selectRef({ sourceId, ref })
  return { source: toProtoSource(response.source!), outcome: fromPbOutcome(response.outcome) }
}

export async function uploadSchema(sourceId: string, content: SchemaUploadContent): Promise<SourceUpdateResult> {
  const response = await sourcesClient.uploadSchema({
    sourceId,
    content:
      content.kind === 'files'
        ? { case: 'files', value: { files: content.files } }
        : { case: 'descriptorSet', value: content.data },
  })
  return { source: toProtoSource(response.source!), outcome: fromPbOutcome(response.outcome) }
}

export async function listSourceRevisions(sourceId: string): Promise<SchemaRevision[]> {
  const response = await sourcesClient.listRevisions({ sourceId })
  return response.revisions.map(fromPbRevision)
}
