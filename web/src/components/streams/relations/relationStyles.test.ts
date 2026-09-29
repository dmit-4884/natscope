import { describe, it, expect } from 'vitest'
import type { StreamRelationEdge } from '@/types/nats'
import { chipWidth, linkChip } from './relationStyles'

const source: StreamRelationEdge = { kind: 'source', from: 'A', to: 'B' }

describe('linkChip', () => {
  it('adds lag and a never-seen note to the kind', () => {
    expect(linkChip({ ...source, state: { lag: 1234, active_ns: -1 } })).toEqual({
      label: 'Source',
      details: ['lag 1.2K', 'never seen'],
      problem: 'inactive',
    })
    expect(linkChip({ ...source, kind: 'republish' })).toEqual({ label: 'Republish', details: [], problem: null })
    expect(linkChip({ ...source, state: { lag: 0, active_ns: -1, error: 'stream not found' } }).problem).toBe('error')
  })
})

describe('chipWidth', () => {
  it('never reserves less than the rendered chip takes', () => {
    const rendered: [StreamRelationEdge, number][] = [
      [source, 56],
      [{ ...source, kind: 'mirror' }, 51],
      [{ ...source, kind: 'republish' }, 71],
      [{ ...source, state: { lag: 0, active_ns: -1 } }, 123],
      [{ ...source, state: { lag: 0, active_ns: -1, error: 'stream not found' } }, 72],
      [{ ...source, state: { lag: 12, active_ns: 5 } }, 97],
    ]
    for (const [edge, width] of rendered) expect(chipWidth(linkChip(edge))).toBeGreaterThanOrEqual(width)
  })
})
