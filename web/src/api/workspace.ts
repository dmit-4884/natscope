import { Strategy } from '@/gen/services/grpc/workspace/v1/workspace/workspace_service_pb'
import { workspaceClient } from './grpc/clients'

export type WorkspaceStrategy = 'merge' | 'replace'

export interface SectionInfo {
  key: string
  title: string
  count: number
}

export interface SectionReport {
  key: string
  created: number
  updated: number
  deleted: number
  conflicts: string[]
  warnings: string[]
  unknown: boolean
}

export interface SectionResult {
  key: string
  created: number
  updated: number
  deleted: number
  warnings: string[]
}

function toStrategy(s: WorkspaceStrategy): Strategy {
  return s === 'replace' ? Strategy.REPLACE : Strategy.MERGE
}

/** List the exportable sections with their current item counts. */
export async function listSections(): Promise<SectionInfo[]> {
  const res = await workspaceClient.listSections({})
  return res.sections.map((s) => ({ key: s.key, title: s.title, count: s.count }))
}

/** Export the selected sections; returns the workspace file bytes. */
export async function exportWorkspace(keys: string[]): Promise<Uint8Array<ArrayBuffer>> {
  const res = await workspaceClient.exportWorkspace({ sectionKeys: keys })
  // Fresh ArrayBuffer-backed view to satisfy Blob/Uint8Array typings.
  return new Uint8Array(res.payload)
}

/** Dry-run an import, returning per-section reports. */
export async function validateWorkspace(
  payload: Uint8Array<ArrayBuffer>,
  keys: string[],
  strategy: WorkspaceStrategy,
): Promise<SectionReport[]> {
  const res = await workspaceClient.validateWorkspace({
    payload,
    sectionKeys: keys,
    strategy: toStrategy(strategy),
  })
  return res.reports.map(reportFromProto)
}

/** Apply an import, returning per-section results. */
export async function importWorkspace(
  payload: Uint8Array<ArrayBuffer>,
  keys: string[],
  strategy: WorkspaceStrategy,
): Promise<SectionResult[]> {
  const res = await workspaceClient.importWorkspace({
    payload,
    sectionKeys: keys,
    strategy: toStrategy(strategy),
  })
  return res.results.map((r) => ({
    key: r.key,
    created: r.created,
    updated: r.updated,
    deleted: r.deleted,
    warnings: r.warnings,
  }))
}

function reportFromProto(r: {
  key: string
  created: number
  updated: number
  deleted: number
  conflicts: string[]
  warnings: string[]
  unknown: boolean
}): SectionReport {
  return {
    key: r.key,
    created: r.created,
    updated: r.updated,
    deleted: r.deleted,
    conflicts: r.conflicts,
    warnings: r.warnings,
    unknown: r.unknown,
  }
}
