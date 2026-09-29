import { NODE_HEIGHT, NODE_WIDTH, type GraphLayout } from './graph'

interface Point {
  x: number
  y: number
}

interface Rect {
  x: number
  y: number
  w: number
  h: number
}

export interface LayoutReport {
  nonFinite: string[]
  nodeOverlaps: string[]
  piercedNodes: string[]
  labelOverlaps: string[]
  labelsOnNodes: string[]
  linksUnderLabels: string[]
  outOfBounds: string[]
  crossings: number
}

const CURVE_SAMPLES = 24
const EPSILON = 0.5
const DEFAULT_LABEL_WIDTH = 104
const DEFAULT_LABEL_HEIGHT = 24

const TOKEN = /[MLC]|-?(?:\d+\.?\d*|\.\d+)(?:e[-+]?\d+)?/gi

export function parsePath(d: string): Point[] {
  const tokens = d.match(TOKEN) ?? []
  const points: Point[] = []
  let command = ''
  let i = 0
  const read = () => {
    const value = Number(tokens[i++])
    if (i > tokens.length) throw new Error(`path ended early: ${d}`)
    return value
  }
  const readPoint = () => ({ x: read(), y: read() })
  while (i < tokens.length) {
    if (/^[MLC]$/i.test(tokens[i])) command = tokens[i++].toUpperCase()
    if (command === 'M') {
      if (points.length > 0) throw new Error(`path has more than one subpath: ${d}`)
      points.push(readPoint())
      command = 'L'
    } else if (command === 'L') {
      points.push(readPoint())
    } else if (command === 'C') {
      const p0 = points[points.length - 1]
      const p1 = readPoint()
      const p2 = readPoint()
      const p3 = readPoint()
      for (let s = 1; s <= CURVE_SAMPLES; s++) {
        const t = s / CURVE_SAMPLES
        const u = 1 - t
        const a = u * u * u
        const b = 3 * u * u * t
        const c = 3 * u * t * t
        const e = t * t * t
        points.push({ x: a * p0.x + b * p1.x + c * p2.x + e * p3.x, y: a * p0.y + b * p1.y + c * p2.y + e * p3.y })
      }
    } else {
      throw new Error(`unsupported path command in ${d}`)
    }
  }
  return points
}

function inside(p: Point, r: Rect): boolean {
  return p.x > r.x && p.x < r.x + r.w && p.y > r.y && p.y < r.y + r.h
}

function cross(o: Point, a: Point, b: Point): number {
  return (a.x - o.x) * (b.y - o.y) - (a.y - o.y) * (b.x - o.x)
}

function intersection(p: Point, q: Point, r: Point, s: Point): Point | null {
  const d1 = cross(r, s, p)
  const d2 = cross(r, s, q)
  const d3 = cross(p, q, r)
  const d4 = cross(p, q, s)
  if (((d1 > 0 && d2 < 0) || (d1 < 0 && d2 > 0)) && ((d3 > 0 && d4 < 0) || (d3 < 0 && d4 > 0))) {
    const t = d1 / (d1 - d2)
    return { x: p.x + (q.x - p.x) * t, y: p.y + (q.y - p.y) * t }
  }
  return null
}

function segmentHitsRect(p: Point, q: Point, r: Rect): boolean {
  if (inside(p, r) || inside(q, r)) return true
  if (Math.max(p.x, q.x) <= r.x || Math.min(p.x, q.x) >= r.x + r.w) return false
  if (Math.max(p.y, q.y) <= r.y || Math.min(p.y, q.y) >= r.y + r.h) return false
  const corners = [
    { x: r.x, y: r.y },
    { x: r.x + r.w, y: r.y },
    { x: r.x + r.w, y: r.y + r.h },
    { x: r.x, y: r.y + r.h },
  ]
  return corners.some((c, i) => intersection(p, q, c, corners[(i + 1) % 4]) !== null)
}

function overlaps(a: Rect, b: Rect): boolean {
  return a.x < b.x + b.w - EPSILON && b.x < a.x + a.w - EPSILON && a.y < b.y + b.h - EPSILON && b.y < a.y + a.h - EPSILON
}

function shrink(r: Rect, by: number): Rect {
  return { x: r.x + by, y: r.y + by, w: r.w - 2 * by, h: r.h - 2 * by }
}

function bounds(points: Point[]): Rect {
  const xs = points.map((p) => p.x)
  const ys = points.map((p) => p.y)
  const x = Math.min(...xs)
  const y = Math.min(...ys)
  return { x, y, w: Math.max(...xs) - x, h: Math.max(...ys) - y }
}

function touching(a: Rect, b: Rect): boolean {
  return a.x <= b.x + b.w && b.x <= a.x + a.w && a.y <= b.y + b.h && b.y <= a.y + a.h
}

function countCrossings(a: Point[], b: Point[]): number {
  if (!touching(bounds(a), bounds(b))) return 0
  const found: Point[] = []
  for (let i = 1; i < a.length; i++) {
    const segment = bounds([a[i - 1], a[i]])
    for (let j = 1; j < b.length; j++) {
      if (!touching(segment, bounds([b[j - 1], b[j]]))) continue
      const hit = intersection(a[i - 1], a[i], b[j - 1], b[j])
      if (hit && !found.some((f) => Math.hypot(f.x - hit.x, f.y - hit.y) < 3)) found.push(hit)
    }
  }
  return found.length
}

export function checkLayout(layout: GraphLayout): LayoutReport {
  const report: LayoutReport = {
    nonFinite: [],
    nodeOverlaps: [],
    piercedNodes: [],
    labelOverlaps: [],
    labelsOnNodes: [],
    linksUnderLabels: [],
    outOfBounds: [],
    crossings: 0,
  }
  const finite = (what: string, ...values: number[]) => {
    if (!values.every(Number.isFinite)) report.nonFinite.push(what)
  }
  finite('size', layout.width, layout.height)

  const boxes = layout.nodes.map((b) => {
    finite(`node ${b.node.id}`, b.x, b.y)
    return { id: b.node.id, rect: { x: b.x, y: b.y, w: NODE_WIDTH, h: NODE_HEIGHT } }
  })
  const canvas = { x: -EPSILON, y: -EPSILON, w: layout.width + 2 * EPSILON, h: layout.height + 2 * EPSILON }
  const within = (r: Rect) => r.x >= canvas.x && r.y >= canvas.y && r.x + r.w <= canvas.x + canvas.w && r.y + r.h <= canvas.y + canvas.h

  boxes.forEach((a, i) => {
    if (!within(a.rect)) report.outOfBounds.push(`node ${a.id}`)
    for (const b of boxes.slice(i + 1)) {
      if (overlaps(a.rect, b.rect)) report.nodeOverlaps.push(`${a.id} / ${b.id}`)
    }
  })

  const labels = layout.edges.map((e) => {
    finite(`label ${e.key}`, e.labelX, e.labelY)
    const w = e.labelWidth ?? DEFAULT_LABEL_WIDTH
    const h = e.labelHeight ?? DEFAULT_LABEL_HEIGHT
    return { key: e.key, rect: { x: e.labelX - w / 2, y: e.labelY - h / 2, w, h } }
  })
  labels.forEach((a, i) => {
    if (!within(a.rect)) report.outOfBounds.push(`label ${a.key}`)
    for (const b of labels.slice(i + 1)) {
      if (overlaps(a.rect, b.rect)) report.labelOverlaps.push(`${a.key} / ${b.key}`)
    }
    for (const box of boxes) {
      if (overlaps(a.rect, box.rect)) report.labelsOnNodes.push(`${a.key} on ${box.id}`)
    }
  })

  const lines = layout.edges.map((e) => {
    const points = parsePath(e.path)
    finite(`path ${e.key}`, ...points.flatMap((p) => [p.x, p.y]))
    if (!within(bounds(points))) report.outOfBounds.push(`path ${e.key}`)
    const hits = (r: Rect) => points.some((p, i) => i > 0 && segmentHitsRect(points[i - 1], p, shrink(r, 1)))
    for (const box of boxes) {
      if (hits(box.rect)) report.piercedNodes.push(`${e.key} through ${box.id}`)
    }
    for (const label of labels) {
      if (label.key !== e.key && hits(label.rect)) report.linksUnderLabels.push(`${e.key} under ${label.key}`)
    }
    return points
  })
  lines.forEach((a, i) => {
    for (const b of lines.slice(i + 1)) report.crossings += countCrossings(a, b)
  })
  return report
}
