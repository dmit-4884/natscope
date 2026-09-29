import type { StreamRelationEdge, StreamRelationKind, StreamRelationNode, StreamRelations } from '@/types/nats'
import { CHIP_HEIGHT, chipWidth, linkChip } from './relationStyles'

export const NODE_WIDTH = 248
export const NODE_HEIGHT = 100
const CURVE_GAP = 48
const MIN_LABEL_COLUMN = 48
const NODE_MARGIN = 16
const LABEL_MARGIN = 6
const LANE_MARGIN = 8
const LANE_HEIGHT = 2
const LANE_WEIGHT = 4
const PORT_INSET = 16
const PORT_GAP = 16
const ORDER_PASSES = 24
const PLACE_PASSES = 200
const SIFT_ROUNDS = 64
const RESTARTS = 32
const WORK_BUDGET = 3_000_000

export interface Neighborhood {
  nodes: StreamRelationNode[]
  edges: StreamRelationEdge[]
}

interface NodeBox {
  node: StreamRelationNode
  x: number
  y: number
}

export interface EdgePath {
  key: string
  edge: StreamRelationEdge
  path: string
  labelX: number
  labelY: number
  labelWidth?: number
  labelHeight?: number
}

export interface GraphLayout {
  nodes: NodeBox[]
  edges: EdgePath[]
  width: number
  height: number
}

interface LayoutOptions {
  orderPasses?: number
}

interface Point {
  x: number
  y: number
}

interface Item {
  col: number
  height: number
  margin: number
  node?: StreamRelationNode
  left: Item[]
  right: Item[]
  pos: number
  y: number
}

interface Route {
  key: string
  edge: StreamRelationEdge
  reversed: boolean
  chain: Item[]
  label: Item
  labelWidth: number
}

function push<K, V>(map: Map<K, V[]>, key: K, value: V) {
  const list = map.get(key)
  if (list) list.push(value)
  else map.set(key, [value])
}

export function neighborhood(
  relations: StreamRelations,
  rootId: string,
  depth: number,
  hidden: ReadonlySet<StreamRelationKind>,
): Neighborhood {
  const byId = new Map(relations.nodes.map((n) => [n.id, n]))
  const shown = relations.edges.filter((e) => !hidden.has(e.kind) && e.from !== e.to && byId.has(e.from) && byId.has(e.to))
  const adjacent = new Map<string, string[]>()
  for (const e of shown) {
    push(adjacent, e.from, e.to)
    push(adjacent, e.to, e.from)
  }

  const distance = new Map([[rootId, 0]])
  const queue = [rootId]
  for (let i = 0; i < queue.length; i++) {
    const id = queue[i]
    const d = distance.get(id) ?? 0
    if (d >= depth) continue
    for (const next of adjacent.get(id) ?? []) {
      if (distance.has(next)) continue
      distance.set(next, d + 1)
      queue.push(next)
    }
  }

  return {
    nodes: queue.flatMap((id) => byId.get(id) ?? []),
    edges: shown.filter((e) => distance.has(e.from) && distance.has(e.to)),
  }
}

function breakCycles(ids: string[], edges: StreamRelationEdge[]): Set<number> {
  const outgoing = new Map<string, number[]>()
  edges.forEach((e, i) => push(outgoing, e.from, i))
  const visiting = new Set<string>()
  const done = new Set<string>()
  const reversed = new Set<number>()

  const visit = (id: string) => {
    visiting.add(id)
    for (const i of outgoing.get(id) ?? []) {
      const next = edges[i].to
      if (visiting.has(next)) reversed.add(i)
      else if (!done.has(next)) visit(next)
    }
    visiting.delete(id)
    done.add(id)
  }
  for (const id of ids) if (!done.has(id)) visit(id)
  return reversed
}

function assignRanks(ids: string[], edges: StreamRelationEdge[]): { rank: Map<string, number>; reversed: Set<number> } {
  const reversed = breakCycles(ids, edges)
  const successors = new Map<string, string[]>()
  const predecessors = new Map<string, string[]>()
  const indegree = new Map(ids.map((id) => [id, 0]))
  edges.forEach((e, i) => {
    const [from, to] = reversed.has(i) ? [e.to, e.from] : [e.from, e.to]
    push(successors, from, to)
    push(predecessors, to, from)
    indegree.set(to, (indegree.get(to) ?? 0) + 1)
  })

  const rank = new Map(ids.map((id) => [id, 0]))
  const topo: string[] = []
  const ready = ids.filter((id) => indegree.get(id) === 0)
  for (let i = 0; i < ready.length; i++) {
    const id = ready[i]
    topo.push(id)
    for (const next of successors.get(id) ?? []) {
      rank.set(next, Math.max(rank.get(next) ?? 0, (rank.get(id) ?? 0) + 1))
      indegree.set(next, (indegree.get(next) ?? 0) - 1)
      if (indegree.get(next) === 0) ready.push(next)
    }
  }

  for (const id of topo.reverse()) {
    const next = successors.get(id)
    if (predecessors.has(id) || !next) continue
    rank.set(id, Math.min(...next.map((n) => rank.get(n) ?? 0)) - 1)
  }

  const dense = new Map([...new Set(rank.values())].sort((a, b) => a - b).map((r, i) => [r, i]))
  return { rank: new Map([...rank].map(([id, r]) => [id, dense.get(r) ?? 0])), reversed }
}

function link(a: Item, b: Item) {
  a.right.push(b)
  b.left.push(a)
}

function initialOrder(items: Item[], columnCount: number): Item[][] {
  const columns: Item[][] = Array.from({ length: columnCount }, () => [])
  const seen = new Set<Item>()
  for (const start of items) {
    const stack = [start]
    while (stack.length > 0) {
      const item = stack.pop()!
      if (seen.has(item)) continue
      seen.add(item)
      columns[item.col].push(item)
      for (const next of [...item.right].reverse()) stack.push(next)
      for (const next of [...item.left].reverse()) stack.push(next)
    }
  }
  return columns
}

function index(columns: Item[][]) {
  for (const column of columns) column.forEach((item, i) => (item.pos = i))
}

function gapCrossings(column: Item[], size: number): number {
  const pairs: [number, number][] = []
  for (const a of column) for (const b of a.right) pairs.push([a.pos, b.pos])
  pairs.sort((p, q) => p[0] - q[0] || p[1] - q[1])
  const tree = new Array<number>(size + 1).fill(0)
  let count = 0
  pairs.forEach(([, b], seen) => {
    let atMost = 0
    for (let i = b + 1; i > 0; i -= i & -i) atMost += tree[i]
    count += seen - atMost
    for (let i = b + 1; i <= size; i += i & -i) tree[i]++
  })
  return count
}

function crossings(columns: Item[][]): number {
  let count = 0
  for (let c = 0; c + 1 < columns.length; c++) count += gapCrossings(columns[c], columns[c + 1].length)
  return count
}

function sweep(columns: Item[][], c: number, side: 'left' | 'right', flip: boolean) {
  const column = columns[c]
  const center = new Map<Item, number>()
  for (const item of column) {
    const neighbors = item[side]
    if (neighbors.length > 0) center.set(item, neighbors.reduce((sum, n) => sum + n.pos, 0) / neighbors.length)
  }
  const movable = column
    .filter((item) => center.has(item))
    .sort((a, b) => center.get(a)! - center.get(b)! || (flip ? b.pos - a.pos : a.pos - b.pos))
  let next = 0
  columns[c] = column.map((item) => (center.has(item) ? movable[next++] : item))
  columns[c].forEach((item, i) => (item.pos = i))
}

function pairCrossings(upper: Item, lower: Item): number {
  let count = 0
  for (const a of upper.left) for (const b of lower.left) if (a.pos > b.pos) count++
  for (const a of upper.right) for (const b of lower.right) if (a.pos > b.pos) count++
  return count
}

interface Budget {
  left: number
}

function siftColumn(column: Item[], budget: Budget): boolean {
  let improved = false
  for (const item of [...column]) {
    if (budget.left <= 0) break
    budget.left -= 2 * column.length
    const from = item.pos
    let bestAt = from
    let bestGain = 0
    let gain = 0
    for (let p = from - 1; p >= 0; p--) {
      gain += pairCrossings(column[p], item) - pairCrossings(item, column[p])
      if (gain > bestGain) {
        bestGain = gain
        bestAt = p
      }
    }
    gain = 0
    for (let p = from + 1; p < column.length; p++) {
      gain += pairCrossings(item, column[p]) - pairCrossings(column[p], item)
      if (gain > bestGain) {
        bestGain = gain
        bestAt = p
      }
    }
    if (bestAt === from) continue
    column.splice(from, 1)
    column.splice(bestAt, 0, item)
    column.forEach((it, i) => (it.pos = i))
    improved = true
  }
  return improved
}

function sift(columns: Item[][], budget: Budget) {
  for (let round = 0; round < SIFT_ROUNDS && budget.left > 0; round++) {
    let improved = false
    for (const column of columns) if (siftColumn(column, budget)) improved = true
    if (!improved) return
  }
}

function reduceCrossings(columns: Item[][], passes: number, budget: Budget): { order: Item[][]; count: number } {
  index(columns)
  let order = columns.map((column) => [...column])
  let count = crossings(columns)
  for (let pass = 0; pass < passes && count > 0 && budget.left > 0; pass++) {
    const flip = pass % 4 >= 2
    if (pass % 2 === 0) for (let c = 1; c < columns.length; c++) sweep(columns, c, 'left', flip)
    else for (let c = columns.length - 2; c >= 0; c--) sweep(columns, c, 'right', flip)
    sift(columns, budget)
    const next = crossings(columns)
    if (next < count) {
      count = next
      order = columns.map((column) => [...column])
    }
  }
  return { order, count }
}

function shuffled(columns: Item[][], seed: number): Item[][] {
  let state = seed
  const random = () => {
    state = (Math.imul(state, 1103515245) + 12345) >>> 0
    return state / 2 ** 32
  }
  return columns.map((column) => {
    const copy = [...column]
    for (let i = copy.length - 1; i > 0; i--) {
      const j = Math.floor(random() * (i + 1))
      ;[copy[i], copy[j]] = [copy[j], copy[i]]
    }
    return copy
  })
}

function orderColumns(columns: Item[][], passes: number) {
  const starts = passes > 0 ? RESTARTS : 1
  const budget = { left: WORK_BUDGET }
  const initial = columns.map((column) => [...column])
  let best = reduceCrossings(columns, passes, budget)
  for (let start = 1; start < starts && best.count > 0 && budget.left > 0; start++) {
    const candidate = start === 1 ? initial.map((column) => [...column].reverse()) : shuffled(initial, start)
    const result = reduceCrossings(candidate, passes, budget)
    if (result.count < best.count) best = result
  }
  best.order.forEach((column, c) => (columns[c] = column))
  index(columns)
}

const separation = (a: Item, b: Item) => a.height / 2 + a.margin + b.margin + b.height / 2
const linkWeight = (a: Item, b: Item) => (a.node || b.node ? 1 : LANE_WEIGHT)

function settle(column: Item[]): number {
  const blocks: { start: number; weight: number; value: number }[] = []
  let offset = 0
  column.forEach((item, i) => {
    if (i > 0) offset += separation(column[i - 1], item)
    let sum = 0
    let weight = 0
    for (const n of [...item.left, ...item.right]) {
      const w = linkWeight(item, n)
      sum += w * n.y
      weight += w
    }
    const target = weight > 0 ? sum / weight : item.y
    let block = { start: i, weight: weight > 0 ? weight : 1e-3, value: target - offset }
    while (blocks.length > 0 && blocks[blocks.length - 1].value > block.value) {
      const last = blocks.pop()!
      const total = last.weight + block.weight
      block = { start: last.start, weight: total, value: (last.value * last.weight + block.value * block.weight) / total }
    }
    blocks.push(block)
  })

  let moved = 0
  offset = 0
  let b = 0
  column.forEach((item, i) => {
    if (i > 0) offset += separation(column[i - 1], item)
    while (b + 1 < blocks.length && blocks[b + 1].start <= i) b++
    const y = blocks[b].value + offset
    moved = Math.max(moved, Math.abs(y - item.y))
    item.y = y
  })
  return moved
}

function placeColumns(columns: Item[][]) {
  for (const column of columns) {
    let y = 0
    column.forEach((item, i) => {
      if (i > 0) y += separation(column[i - 1], item)
      item.y = y
    })
    for (const item of column) item.y -= y / 2
  }
  for (let pass = 0; pass < PLACE_PASSES; pass++) {
    const order = pass % 2 === 0 ? columns : [...columns].reverse()
    let moved = 0
    for (const column of order) moved = Math.max(moved, settle(column))
    if (moved < 0.1) break
  }
}

const round = (v: number) => Math.round(v * 100) / 100

function pathThrough(points: Point[], curved: boolean[]): string {
  let d = `M ${round(points[0].x)} ${round(points[0].y)}`
  for (let i = 1; i < points.length; i++) {
    const p = points[i - 1]
    const q = points[i]
    if (!curved[i - 1]) {
      d += ` L ${round(q.x)} ${round(q.y)}`
      continue
    }
    const dx = (q.x - p.x) / 2
    d += ` C ${round(p.x + dx)} ${round(p.y)}, ${round(q.x - dx)} ${round(q.y)}, ${round(q.x)} ${round(q.y)}`
  }
  return d
}

export function layoutGraph(graph: Neighborhood, { orderPasses = ORDER_PASSES }: LayoutOptions = {}): GraphLayout {
  const { nodes } = graph
  const ids = nodes.map((n) => n.id)
  const known = new Set(ids)
  const edges = graph.edges.filter((e) => e.from !== e.to && known.has(e.from) && known.has(e.to))
  const { rank, reversed } = assignRanks(ids, edges)
  const nodeItems = new Map<string, Item>(
    nodes.map((node) => [
      node.id,
      { col: 2 * (rank.get(node.id) ?? 0), height: NODE_HEIGHT, margin: NODE_MARGIN, node, left: [], right: [], pos: 0, y: 0 },
    ]),
  )
  const items = [...nodeItems.values()]

  const seen = new Map<string, number>()
  const routes = edges.map((edge, i): Route => {
    const id = `${edge.kind}\n${edge.from}\n${edge.to}`
    const nth = seen.get(id) ?? 0
    seen.set(id, nth + 1)
    const isReversed = reversed.has(i)
    const first = nodeItems.get(isReversed ? edge.to : edge.from)!
    const last = nodeItems.get(isReversed ? edge.from : edge.to)!
    const labelCol = first.col + 1 + 2 * Math.floor((last.col - first.col - 2) / 4)
    const labelWidth = chipWidth(linkChip(edge))
    const chain = [first]
    let label = first
    for (let col = first.col + 1; col < last.col; col++) {
      const isLabel = col === labelCol
      const dummy: Item = {
        col,
        height: isLabel ? CHIP_HEIGHT : LANE_HEIGHT,
        margin: isLabel ? LABEL_MARGIN : LANE_MARGIN,
        left: [],
        right: [],
        pos: 0,
        y: 0,
      }
      if (isLabel) label = dummy
      items.push(dummy)
      chain.push(dummy)
    }
    chain.push(last)
    for (let k = 1; k < chain.length; k++) link(chain[k - 1], chain[k])
    return { key: `${id}\n${nth}`, edge, reversed: isReversed, chain, label, labelWidth }
  })

  const columnCount = Math.max(0, ...items.map((item) => item.col)) + 1
  const columns = initialOrder(items, columnCount)
  orderColumns(columns, orderPasses)
  placeColumns(columns)

  const top = items.length > 0 ? Math.min(...items.map((item) => item.y - item.height / 2)) : 0
  for (const item of items) item.y -= top
  const height = Math.max(NODE_HEIGHT, ...items.map((item) => item.y + item.height / 2))

  const columnWidth = columns.map((_, c): number => (c % 2 === 0 ? NODE_WIDTH : MIN_LABEL_COLUMN))
  for (const route of routes) columnWidth[route.label.col] = Math.max(columnWidth[route.label.col], route.labelWidth)
  const columnX = [0]
  for (let c = 1; c < columnWidth.length; c++) columnX.push(columnX[c - 1] + columnWidth[c - 1] + CURVE_GAP)
  const width = columnX[columnX.length - 1] + columnWidth[columnWidth.length - 1]

  const starts = new Map<Item, Route[]>()
  const ends = new Map<Item, Route[]>()
  for (const route of routes) {
    push(starts, route.chain[0], route)
    push(ends, route.chain[route.chain.length - 1], route)
  }
  const portY = new Map<Route, { start: number; end: number }>(routes.map((r) => [r, { start: 0, end: 0 }]))
  const assignPorts = (sides: Map<Item, Route[]>, end: 'start' | 'end') => {
    for (const [item, attached] of sides) {
      const toward = (r: Route) => (end === 'start' ? r.chain[1] : r.chain[r.chain.length - 2]).y
      const ordered = attached
        .map((route, i) => ({ route, i }))
        .sort((a, b) => toward(a.route) - toward(b.route) || a.i - b.i)
      const gap = ordered.length > 1 ? Math.min(PORT_GAP, (NODE_HEIGHT - 2 * PORT_INSET) / (ordered.length - 1)) : 0
      ordered.forEach(({ route }, i) => (portY.get(route)![end] = item.y + (i - (ordered.length - 1) / 2) * gap))
    }
  }
  assignPorts(starts, 'start')
  assignPorts(ends, 'end')

  const paths = routes.map((route): EdgePath => {
    const { chain } = route
    const first = chain[0]
    const last = chain[chain.length - 1]
    const points: Point[] = [{ x: columnX[first.col] + NODE_WIDTH, y: portY.get(route)!.start }]
    const curved: boolean[] = []
    for (const dummy of chain.slice(1, -1)) {
      points.push({ x: columnX[dummy.col], y: dummy.y }, { x: columnX[dummy.col] + columnWidth[dummy.col], y: dummy.y })
      curved.push(true, false)
    }
    points.push({ x: columnX[last.col], y: portY.get(route)!.end })
    curved.push(true)
    if (route.reversed) {
      points.reverse()
      curved.reverse()
    }
    return {
      key: route.key,
      edge: route.edge,
      path: pathThrough(points, curved),
      labelX: round(columnX[route.label.col] + columnWidth[route.label.col] / 2),
      labelY: round(route.label.y),
      labelWidth: route.labelWidth,
      labelHeight: CHIP_HEIGHT,
    }
  })

  return {
    nodes: nodes.map((node) => {
      const item = nodeItems.get(node.id)!
      return { node, x: columnX[item.col], y: round(item.y - NODE_HEIGHT / 2) }
    }),
    edges: paths,
    width,
    height: round(height),
  }
}
