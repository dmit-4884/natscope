import { describe, it, expect } from 'vitest'
import type { StreamNodeKind, StreamRelationEdge, StreamRelationKind, StreamRelations } from '@/types/nats'
import { layoutGraph, neighborhood, type Neighborhood } from './graph'
import { checkLayout, parsePath } from './layoutChecks'

const none = new Set<StreamRelationKind>()

function link(kind: StreamRelationKind, from: string, to: string): StreamRelationEdge {
  return { kind, from, to }
}

const source = (from: string, to: string) => link('source', from, to)
const mirror = (from: string, to: string) => link('mirror', from, to)
const republish = (from: string, to: string) => link('republish', from, to)

function placeholderKind(id: string): StreamNodeKind {
  if (id.startsWith('external ')) return 'external'
  if (id.startsWith('subject ')) return 'subject'
  if (id.startsWith('missing:')) return 'missing'
  return 'stream'
}

function graphOf(...edges: StreamRelationEdge[]): StreamRelations {
  const ids = [...new Set(edges.flatMap((e) => [e.from, e.to]))].sort()
  return { nodes: ids.map((id) => ({ id, name: id.split(' ').pop() ?? id, kind: placeholderKind(id) })), edges }
}

function whole(relations: StreamRelations): Neighborhood {
  return { nodes: relations.nodes, edges: relations.edges.filter((e) => e.from !== e.to) }
}

const external = (name: string, domain: string) => `external $JS.${domain}.API ${name}`

const hub = graphOf(
  source('PAYMENTS', 'AUDIT'),
  source('CUSTOMERS', 'CUSTOMER_TIMELINE'),
  source('ORDERS', 'CUSTOMER_TIMELINE'),
  source('SHIPMENTS', 'CUSTOMER_TIMELINE'),
  source('ORDERS', 'FULFILLMENT'),
  source(external('KV_stock_eu', 'eu'), 'KV_stock'),
  source(external('KV_stock_us', 'us'), 'KV_stock'),
  source(external('KV_stock_apac', 'apac'), 'KV_stock'),
  source(external('ORDERS_EU', 'eu'), 'ORDERS'),
  source(external('ORDERS_US', 'us'), 'ORDERS'),
  source(external('ORDERS_APAC', 'apac'), 'ORDERS'),
  mirror('ORDERS', 'ORDERS_ARCHIVE'),
  source(external('PAYMENTS_EU', 'eu'), 'PAYMENTS'),
  source(external('PAYMENTS_US', 'us'), 'PAYMENTS'),
  source(external('PAYMENTS_APAC', 'apac'), 'PAYMENTS'),
  mirror('OBJ_product_images', 'PRODUCT_IMAGES_BACKUP'),
  republish('CATALOG', 'subject cache.invalidate.products.{{wildcard(1)}}'),
  republish('CUSTOMERS', 'AUDIT'),
  republish('SHIPMENTS', 'NOTIFICATIONS'),
)

const analytics = graphOf(
  source('external shop.API ORDERS', 'SALES'),
  source('external shop.API PAYMENTS', 'SALES'),
  source('external shop.API PAYMENTS', 'SALES'),
  mirror('SALES', 'SALES_LAST_24H'),
)

const apac = graphOf(
  source(external('CATALOG', 'hub'), 'CATALOG_APAC'),
  source(external('CATALOG', 'hub'), 'CATALOG_APAC'),
  source(external('KV_stock_apac', 'apac'), 'KV_stock_apac_copy'),
)

function chain(n: number): StreamRelationEdge[] {
  return Array.from({ length: n - 1 }, (_, i) => source(`N${i}`, `N${i + 1}`))
}

function tree(depth: number, fanout: number, inward: boolean): StreamRelationEdge[] {
  const edges: StreamRelationEdge[] = []
  const grow = (id: string, level: number) => {
    if (level === depth) return
    for (let i = 0; i < fanout; i++) {
      const child = `${id}.${i}`
      edges.push(inward ? source(child, id) : source(id, child))
      grow(child, level + 1)
    }
  }
  grow('R', 0)
  return edges
}

const planar: Record<string, StreamRelations> = {
  'single link': graphOf(source('A', 'B')),
  chain: graphOf(...chain(6)),
  'fan-in': graphOf(...['EU', 'US', 'APAC', 'LATAM', 'MEA', 'ANZ'].map((r) => source(r, 'HUB'))),
  'fan-out': graphOf(...['ARCHIVE', 'WORK', 'AUDIT', 'TIMELINE', 'BI', 'SEARCH'].map((d) => source('HUB', d))),
  'out-tree': graphOf(...tree(3, 2, false)),
  'in-tree': graphOf(...tree(3, 2, true)),
  diamond: graphOf(source('A', 'B'), source('A', 'C'), source('B', 'D'), mirror('C', 'D')),
  'parallel links': graphOf(source('A', 'B'), source('A', 'B'), republish('A', 'B'), source('B', 'C')),
  'long links': graphOf(...chain(4), source('N0', 'N3'), republish('N0', 'N2'), mirror('N1', 'N3')),
  'sibling link': graphOf(source('P', 'A'), source('P', 'B'), republish('A', 'B')),
  'two-cycle': graphOf(source('A', 'B'), source('B', 'A')),
  'three-cycle': graphOf(source('A', 'B'), source('B', 'C'), republish('C', 'A')),
  'cycle with tail': graphOf(source('A', 'B'), source('B', 'C'), source('C', 'D'), republish('D', 'B'), mirror('D', 'E')),
  placeholders: graphOf(
    source(external('UP', 'eu'), 'ROOT'),
    source('missing:GONE', 'ROOT'),
    republish('ROOT', 'subject cache.>'),
    mirror('ROOT', 'COPY'),
  ),
  'hub topology': hub,
  'analytics account': analytics,
  'apac domain': apac,
}

function expectClean(layout: ReturnType<typeof layoutGraph>) {
  const report = checkLayout(layout)
  expect(report.nonFinite).toEqual([])
  expect(report.nodeOverlaps).toEqual([])
  expect(report.piercedNodes).toEqual([])
  expect(report.labelOverlaps).toEqual([])
  expect(report.labelsOnNodes).toEqual([])
  expect(report.linksUnderLabels).toEqual([])
  expect(report.outOfBounds).toEqual([])
  return report
}

describe('layoutChecks', () => {
  it('samples lines and curves of a path', () => {
    const points = parsePath('M 0 0 C 10 0, 10 10, 20 10 L 30 10')
    expect(points[0]).toEqual({ x: 0, y: 0 })
    expect(points[points.length - 2]).toEqual({ x: 20, y: 10 })
    expect(points[points.length - 1]).toEqual({ x: 30, y: 10 })
  })

  it('finds a crossing and a link through a card', () => {
    const report = checkLayout({
      nodes: [{ node: { id: 'X', name: 'X', kind: 'stream' }, x: 400, y: 0 }],
      edges: [
        { key: 'a', edge: source('A', 'B'), path: 'M 0 0 L 1000 60', labelX: 100, labelY: 200 },
        { key: 'b', edge: source('C', 'D'), path: 'M 0 60 L 200 0', labelX: 300, labelY: 200 },
      ],
      width: 1000,
      height: 300,
    })
    expect(report.crossings).toBe(1)
    expect(report.piercedNodes).toEqual(['a through X'])
  })

  it('finds labels on cards, on each other and on other links', () => {
    const report = checkLayout({
      nodes: [{ node: { id: 'X', name: 'X', kind: 'stream' }, x: 0, y: 0 }],
      edges: [
        { key: 'a', edge: source('A', 'B'), path: 'M 300 10 L 600 10', labelX: 250, labelY: 50, labelWidth: 80, labelHeight: 20 },
        { key: 'b', edge: source('C', 'D'), path: 'M 300 200 L 600 200', labelX: 290, labelY: 60, labelWidth: 80, labelHeight: 20 },
        { key: 'c', edge: source('E', 'F'), path: 'M 450 0 L 450 300', labelX: 450, labelY: 200, labelWidth: 80, labelHeight: 20 },
      ],
      width: 500,
      height: 300,
    })
    expect(report.labelsOnNodes).toEqual(['a on X'])
    expect(report.labelOverlaps).toEqual(['a / b'])
    expect(report.linksUnderLabels).toEqual(['b under c'])
    expect(report.outOfBounds).toEqual(['path a', 'path b'])
    expect(report.crossings).toBe(2)
  })
})

describe('layoutGraph geometry', () => {
  for (const [name, relations] of Object.entries(planar)) {
    it(`draws ${name} without overlaps or crossings`, () => {
      expect(expectClean(layoutGraph(whole(relations))).crossings).toBe(0)
    })
  }

  it('draws every neighborhood of the hub topology without crossings', () => {
    for (const root of hub.nodes.map((n) => n.id)) {
      for (let depth = 1; depth <= 10; depth++) {
        const report = expectClean(layoutGraph(neighborhood(hub, root, depth, none)))
        expect({ root, depth, crossings: report.crossings }).toEqual({ root, depth, crossings: 0 })
      }
    }
  })

  it('keeps the unavoidable crossings of a complete bipartite graph at the minimum', () => {
    const edges = ['A', 'B', 'C'].flatMap((a) => ['X', 'Y', 'Z'].map((b) => source(a, b)))
    expect(expectClean(layoutGraph(whole(graphOf(...edges)))).crossings).toBe(9)
  })
})

describe('layoutGraph structure', () => {
  const columnOf = (layout: ReturnType<typeof layoutGraph>) => new Map(layout.nodes.map((b) => [b.node.id, b.x]))

  it('keeps data flowing left to right outside cycles', () => {
    const layout = layoutGraph(whole(hub))
    const x = columnOf(layout)
    for (const e of hub.edges) expect(x.get(e.from)!).toBeLessThan(x.get(e.to)!)
  })

  it('points every link at its downstream card', () => {
    const layout = layoutGraph(whole(planar['cycle with tail']))
    const boxes = new Map(layout.nodes.map((b) => [b.node.id, b]))
    for (const e of layout.edges) {
      const points = parsePath(e.path)
      const start = points[0]
      const end = points[points.length - 1]
      const from = boxes.get(e.edge.from)!
      const to = boxes.get(e.edge.to)!
      expect(start.y).toBeGreaterThanOrEqual(from.y)
      expect(start.y).toBeLessThanOrEqual(from.y + 100)
      expect([from.x, from.x + 248]).toContain(start.x)
      expect(end.y).toBeGreaterThanOrEqual(to.y)
      expect(end.y).toBeLessThanOrEqual(to.y + 100)
      expect([to.x, to.x + 248]).toContain(end.x)
    }
  })

  it('lays out the same graph the same way every time', () => {
    const first = layoutGraph(neighborhood(hub, 'AUDIT', 10, none))
    const second = layoutGraph(neighborhood(hub, 'AUDIT', 10, none))
    expect(second).toEqual(first)
  })

  it('lays out a lone stream', () => {
    const layout = layoutGraph({ nodes: [{ id: 'A', name: 'A', kind: 'stream' }], edges: [] })
    expect(layout).toMatchObject({ nodes: [{ x: 0, y: 0 }], edges: [], width: 248, height: 100 })
  })

  it('reserves room for the widest link label in its column', () => {
    const lagging: StreamRelationEdge = { ...source('A', 'B'), state: { lag: 1_234_567, active_ns: -1, error: 'stream not found' } }
    const layout = layoutGraph(whole(graphOf(lagging, source('A', 'B'))))
    const [wide, narrow] = layout.edges
    expect(wide.labelWidth!).toBeGreaterThan(narrow.labelWidth!)
    expect(wide.labelX).toBe(narrow.labelX)
    expect(layout.nodes[1].x - layout.nodes[0].x - 248).toBeGreaterThan(wide.labelWidth!)
    expectClean(layout)
  })
})

function mulberry32(seed: number) {
  let state = seed
  return () => {
    state = (state + 0x6d2b79f5) | 0
    let t = Math.imul(state ^ (state >>> 15), 1 | state)
    t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296
  }
}

const KINDS: StreamRelationKind[] = ['source', 'mirror', 'republish']

function randomGraph(random: () => number): StreamRelations {
  const n = 2 + Math.floor(random() * 8)
  const m = 1 + Math.floor(random() * n * 2)
  const edges: StreamRelationEdge[] = []
  for (let i = 0; i < m; i++) {
    const a = Math.floor(random() * n)
    let b = Math.floor(random() * (n - 1))
    if (b >= a) b++
    const e = link(KINDS[Math.floor(random() * 3)], `N${a}`, `N${b}`)
    edges.push(e)
    if (random() < 0.15) edges.push({ ...e })
  }
  return graphOf(...edges)
}

function randomTree(random: () => number, inward: boolean): StreamRelations {
  const n = 2 + Math.floor(random() * 13)
  const edges: StreamRelationEdge[] = []
  for (let i = 1; i < n; i++) {
    const parent = `T${Math.floor(random() * i)}`
    edges.push(inward ? source(`T${i}`, parent) : source(parent, `T${i}`))
  }
  return graphOf(...edges)
}

describe('layoutGraph on random graphs', () => {
  it('never overlaps, pierces a card or loses to the unordered baseline', () => {
    const random = mulberry32(20260929)
    let crossings = 0
    let baseline = 0
    for (let i = 0; i < 300; i++) {
      const g = whole(randomGraph(random))
      const ours = expectClean(layoutGraph(g)).crossings
      const unordered = checkLayout(layoutGraph(g, { orderPasses: 0 })).crossings
      expect(ours).toBeLessThanOrEqual(unordered)
      expect(ours).toBeLessThanOrEqual(6)
      crossings += ours
      baseline += unordered
    }
    expect(crossings).toBeLessThanOrEqual(30)
    expect(crossings * 20).toBeLessThan(baseline)
  })

  it('draws random source trees without crossings', () => {
    const random = mulberry32(7)
    for (let i = 0; i < 100; i++) {
      for (const inward of [false, true]) {
        const g = whole(randomTree(random, inward))
        expect({ i, inward, crossings: expectClean(layoutGraph(g)).crossings }).toEqual({ i, inward, crossings: 0 })
      }
    }
  })
})
