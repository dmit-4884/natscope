import { describe, it, expect } from 'vitest'
import type { StreamRelationEdge, StreamRelationKind, StreamRelations } from '@/types/nats'
import { NODE_WIDTH, layoutGraph, neighborhood } from './graph'

function edge(kind: StreamRelationKind, from: string, to: string): StreamRelationEdge {
  return { kind, from, to }
}

function graph(...edges: StreamRelationEdge[]): StreamRelations {
  const ids = [...new Set(edges.flatMap((e) => [e.from, e.to]))]
  return { nodes: ids.map((id) => ({ id, name: id, kind: 'stream' })), edges }
}

const none = new Set<StreamRelationKind>()

describe('neighborhood', () => {
  const chain = graph(edge('source', 'A', 'B'), edge('source', 'B', 'C'), edge('mirror', 'C', 'D'), edge('republish', 'X', 'B'))

  it('walks links in both directions up to the depth', () => {
    const ids = (depth: number) => neighborhood(chain, 'B', depth, none).nodes.map((n) => n.id)
    expect(ids(1)).toEqual(['B', 'A', 'C', 'X'])
    expect(ids(2)).toEqual(['B', 'A', 'C', 'X', 'D'])
  })

  it('keeps only links between visible nodes', () => {
    const { edges } = neighborhood(chain, 'A', 1, none)
    expect(edges).toEqual([edge('source', 'A', 'B')])
  })

  it('drops hidden kinds before walking', () => {
    const { nodes } = neighborhood(chain, 'B', 5, new Set<StreamRelationKind>(['republish', 'mirror']))
    expect(nodes.map((n) => n.id)).toEqual(['B', 'A', 'C'])
  })

  it('returns nothing for a stream without relations', () => {
    expect(neighborhood(chain, 'NOPE', 3, none)).toEqual({ nodes: [], edges: [] })
  })

  it('ignores self links and links to nodes the graph does not have', () => {
    const g = graph(edge('source', 'A', 'B'))
    g.edges.push(edge('source', 'B', 'GHOST'), edge('mirror', 'B', 'B'))
    expect(neighborhood(g, 'A', 3, none)).toEqual({ nodes: g.nodes, edges: [edge('source', 'A', 'B')] })
    expect(layoutGraph({ nodes: g.nodes, edges: g.edges }).edges.map((e) => e.edge)).toEqual([edge('source', 'A', 'B')])
  })
})

describe('layoutGraph', () => {
  const xOf = (layout: ReturnType<typeof layoutGraph>) => new Map(layout.nodes.map((b) => [b.node.id, b.x]))

  it('places upstreams left of downstreams', () => {
    const g = graph(edge('source', 'EU', 'AGG'), edge('source', 'US', 'AGG'), edge('mirror', 'AGG', 'BACKUP'))
    const x = xOf(layoutGraph(neighborhood(g, 'AGG', 3, none)))
    expect(x.get('EU')).toBe(0)
    expect(x.get('US')).toBe(0)
    expect(x.get('AGG')!).toBeGreaterThan(x.get('EU')!)
    expect(x.get('BACKUP')!).toBeGreaterThan(x.get('AGG')!)
  })

  it('pulls a lone upstream next to its target', () => {
    const g = graph(edge('source', 'A', 'B'), edge('source', 'B', 'C'), edge('source', 'LATE', 'C'))
    const x = xOf(layoutGraph(neighborhood(g, 'C', 3, none)))
    expect(x.get('LATE')).toBe(x.get('B'))
  })

  it('lays out cycles without throwing', () => {
    const g = graph(edge('source', 'A', 'B'), edge('source', 'B', 'A'), edge('republish', 'B', 'C'), edge('source', 'C', 'A'))
    const layout = layoutGraph(neighborhood(g, 'A', 3, none))
    expect(layout.nodes).toHaveLength(3)
    expect(layout.edges).toHaveLength(4)
    for (const e of layout.edges) {
      expect(e.path).toMatch(/^M [\d.-]+ [\d.-]+ C /)
      expect(Number.isFinite(e.labelX) && Number.isFinite(e.labelY)).toBe(true)
    }
  })

  it('separates parallel links between the same streams', () => {
    const g = graph(edge('source', 'ORDERS', 'AGG'), edge('source', 'ORDERS', 'AGG'))
    const [first, second] = layoutGraph(neighborhood(g, 'AGG', 1, none)).edges
    expect(first.labelY).not.toBe(second.labelY)
    expect(first.labelX).toBe(second.labelX)
  })

  it('keeps link labels apart where links cross', () => {
    const g = graph(edge('source', 'A', 'C'), edge('source', 'B', 'C'), edge('source', 'A', 'D'), edge('republish', 'B', 'D'))
    const labels = layoutGraph(neighborhood(g, 'C', 3, none)).edges
    for (const [i, a] of labels.entries()) {
      for (const b of labels.slice(i + 1)) {
        expect(Math.abs(a.labelX - b.labelX) >= 104 || Math.abs(a.labelY - b.labelY) >= 24).toBe(true)
      }
    }
  })

  it('sizes the canvas to the columns', () => {
    const g = graph(edge('source', 'A', 'B'))
    expect(layoutGraph(neighborhood(g, 'A', 1, none)).width).toBeGreaterThan(2 * NODE_WIDTH)
  })
})
