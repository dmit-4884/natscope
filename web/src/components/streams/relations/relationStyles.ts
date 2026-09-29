import type { ExternalStreamRef, StreamRelationEdge, StreamRelationKind } from '@/types/nats'
import { formatDuration, formatNumber } from '@/utils/formatters'

export const RELATION_KINDS: StreamRelationKind[] = ['source', 'mirror', 'republish']

export const RELATION_STYLES: Record<
  StreamRelationKind,
  { label: string; stroke: string; fill: string; chip: string; swatch: string }
> = {
  source: {
    label: 'Source',
    stroke: 'stroke-orange-500',
    fill: 'fill-orange-500',
    chip: 'border-orange-300 text-orange-700 bg-orange-50 hover:bg-orange-100',
    swatch: 'bg-orange-500',
  },
  mirror: {
    label: 'Mirror',
    stroke: 'stroke-sky-500',
    fill: 'fill-sky-500',
    chip: 'border-sky-300 text-sky-700 bg-sky-50 hover:bg-sky-100',
    swatch: 'bg-sky-500',
  },
  republish: {
    label: 'Republish',
    stroke: 'stroke-violet-500',
    fill: 'fill-violet-500',
    chip: 'border-violet-300 text-violet-700 bg-violet-50 hover:bg-violet-100',
    swatch: 'bg-violet-500',
  },
}

export function lastSeen(activeNs: number): string {
  if (activeNs < 0) return 'never'
  if (activeNs < 1_000_000_000) return 'just now'
  return `${formatDuration(Math.floor(activeNs / 1_000_000_000))} ago`
}

export function linkProblem(edge: StreamRelationEdge): 'error' | 'inactive' | null {
  if (edge.state?.error) return 'error'
  if (edge.state && edge.state.active_ns < 0) return 'inactive'
  return null
}

export interface LinkChip {
  label: string
  details: string[]
  problem: 'error' | 'inactive' | null
}

export function linkChip(edge: StreamRelationEdge): LinkChip {
  const problem = linkProblem(edge)
  const lag = edge.state?.lag ?? 0
  const details: string[] = []
  if (lag > 0) details.push(`lag ${formatNumber(lag)}`)
  if (problem === 'inactive') details.push('never seen')
  return { label: RELATION_STYLES[edge.kind].label, details, problem }
}

export const CHIP_HEIGHT = 20
const CHIP_PADDING = 18
const CHIP_GAP = 4
const CHIP_ICON = 12
const NARROW_CHARS = new Set(" .,·:;'!|()[]fijlrtI")
const WIDE_CHARS = new Set('mwMW')

function textWidth(text: string): number {
  let width = 0
  for (const c of text) {
    if (NARROW_CHARS.has(c)) width += 4
    else if (WIDE_CHARS.has(c)) width += 11
    else if (c !== c.toLowerCase()) width += 8.5
    else width += 7
  }
  return width
}

export function chipWidth({ label, details, problem }: LinkChip): number {
  const parts = [label, ...details.map((d) => `· ${d}`)].map(textWidth)
  if (problem === 'error') parts.push(CHIP_ICON)
  return Math.ceil(CHIP_PADDING + parts.reduce((sum, w) => sum + w, 0) + CHIP_GAP * (parts.length - 1))
}

export function externalLabel(external: ExternalStreamRef | undefined): string {
  if (!external) return ''
  const domain = /^\$JS\.([^.]+)\.API$/.exec(external.api_prefix)?.[1]
  return domain ? `Domain ${domain}` : `API ${external.api_prefix}`
}
