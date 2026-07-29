import { tsToMillis } from '@/utils/timestamp'
import type { ProtoSource as ProtoSourceProto , CompileDiagnostic as PbDiagnostic } from '../gen/types/proto/proto_source_pb'
import { SourceType } from '../gen/types/proto/proto_source_pb'
import type { ProtoSelection as ProtoSelectionProto } from '../gen/types/proto/proto_selection_pb'
import { sourcesClient, selectionsClient } from './grpc/clients'

export type ProtoSourceType = 'git' | 'local' | 'files'

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
  files: string[]
  includeDirs: string[]
  // Manual import roots; non-empty disables auto-detection, empty = auto.
  importRoots?: string[]
  // Slash-relative prefixes excluded from compilation (e.g. "pb", "gen").
  excludePrefixes?: string[]
  lastCompile?: ProtoCompileResult
  created_at: number
  updated_at?: number
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

export interface CompileFilesResult {
  valid: boolean
  messageTypes: number
  fileDescriptors: number
  diagnostics: CompileDiagnostic[]
}

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
  files?: string[]
  includeDirs?: string[]
  importRoots?: string[]
  excludePrefixes?: string[]
}

export interface UpdateProtoSourceRequest {
  name?: string
  repository?: string
  token?: string
  localPath?: string
  files?: string[]
  includeDirs?: string[]
  importRoots?: string[]
  excludePrefixes?: string[]
}

export interface ProtoSelection {
  id: string
  source_id: string
  tag: string
  created_at: number
  updated_at?: number
}

function sourceTypeFromProto(st: SourceType): ProtoSourceType {
  switch (st) {
    case SourceType.LOCAL:
      return 'local'
    case SourceType.FILES:
      return 'files'
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
    case 'files':
      return SourceType.FILES
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
    files: p.files ?? [],
    includeDirs: p.includeDirs ?? [],
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
    created_at: tsToMillis(p.createdAt),
    updated_at: tsToMillis(p.updatedAt) || undefined,
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

function toProtoSelection(s: ProtoSelectionProto): ProtoSelection {
  return {
    id: s.id,
    source_id: s.sourceId,
    tag: s.tag,
    created_at: tsToMillis(s.createdAt),
    updated_at: tsToMillis(s.updatedAt) || undefined,
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
    files: data.files ?? [],
    includeDirs: data.includeDirs ?? [],
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
    files: data.files ?? [],
    includeDirs: data.includeDirs ?? [],
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

// Local compile

export async function compileLocal(sourceId: string): Promise<{
  messageTypes: number
  fileDescriptors: number
  valid: boolean
  diagnostics: CompileDiagnostic[]
}> {
  const response = await sourcesClient.compileLocal({ sourceId })
  return {
    messageTypes: response.messageTypes,
    fileDescriptors: response.fileDescriptors,
    valid: response.valid,
    diagnostics: (response.diagnostics ?? []).map(fromPbDiagnostic),
  }
}

// Files validate / compile

export async function compileFiles(sourceId: string): Promise<CompileFilesResult> {
  const response = await sourcesClient.compileFiles({ sourceId })
  return {
    valid: response.valid,
    messageTypes: response.messageTypes,
    fileDescriptors: response.fileDescriptors,
    diagnostics: (response.diagnostics ?? []).map(fromPbDiagnostic),
  }
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

// Tags and Versions

export async function getProtoSourceTags(sourceId: string): Promise<string[]> {
  const response = await sourcesClient.listTags({ sourceId })
  return response.tags || []
}

// Selections

export async function getProtoSelections(): Promise<ProtoSelection[]> {
  const response = await selectionsClient.listSelections({ pageSize: 0, pageToken: '' })
  return (response.selections || []).map(toProtoSelection)
}

export async function selectProtoVersion(sourceId: string, tag: string): Promise<ProtoSelection> {
  const response = await selectionsClient.selectVersion({ sourceId, tag })
  return toProtoSelection(response.selection!)
}

export async function deleteProtoSelection(selectionId: string): Promise<void> {
  await selectionsClient.deleteSelection({ id: selectionId })
}

// Proto Loading (Lazy Loading V1)

export interface LoadProtoResponse {
  loaded: boolean
  selections: number
  messages_count: number
}

export async function loadProtoFiles(): Promise<LoadProtoResponse> {
  const response = await selectionsClient.loadSelections({})
  return {
    loaded: response.messageCount > 0,
    selections: response.fileCount,
    messages_count: response.messageCount,
  }
}
