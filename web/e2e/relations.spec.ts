import type { Locator } from '@playwright/test'
import { test, expect } from './fixtures'
import { call, ConnectError } from './api'

const UPSTREAM = 'E2E_REL_UP'
const AGGREGATE = 'E2E_REL_AGG'
const COPY = 'E2E_REL_COPY'
const MANAGEMENT = 'natscope.nats.management.v1.ManagementService'

async function deleteStream(connectionId: string, streamName: string) {
  try {
    await call(MANAGEMENT, 'DeleteStream', { connectionId, streamName })
  } catch (e) {
    if (!(e instanceof ConnectError && /not found/i.test(e.message))) throw e
  }
}

test.describe('stream relations', () => {
  test.beforeEach(async ({ env }) => {
    for (const name of [COPY, AGGREGATE, UPSTREAM]) await deleteStream(env.connectionId, name)
    await call(MANAGEMENT, 'CreateStream', { connectionId: env.connectionId, name: UPSTREAM, subjects: ['e2e.rel.up.>'] })
    await call(MANAGEMENT, 'CreateStream', {
      connectionId: env.connectionId,
      name: AGGREGATE,
      sources: [{ name: UPSTREAM, filterSubject: 'e2e.rel.up.eu.>' }],
    })
    await call(MANAGEMENT, 'CreateStream', { connectionId: env.connectionId, name: COPY, mirror: { name: AGGREGATE } })
  })

  test.afterEach(async ({ env }) => {
    for (const name of [COPY, AGGREGATE, UPSTREAM]) await deleteStream(env.connectionId, name)
  })

  test('draws sources and mirrors and walks to a neighbour', async ({ page }) => {
    await page.goto(`/streams/${AGGREGATE}/relations`)

    const canvas = page.getByTestId('relations-canvas')
    await expect(canvas.getByTestId('relation-node')).toHaveCount(3)
    await expect(canvas.getByRole('button', { name: `Mirror from ${AGGREGATE} to ${COPY}. Show details` })).toBeVisible()

    await canvas.getByRole('button', { name: `Source from ${UPSTREAM} to ${AGGREGATE}. Show details` }).click()
    const details = page.getByTestId('relation-details')
    await expect(details.getByText('e2e.rel.up.eu.>')).toBeVisible()

    const area = (await canvas.boundingBox())!
    const before = (await details.boundingBox())!
    const grip = (await page.getByTestId('relation-details-handle').boundingBox())!
    await page.mouse.move(grip.x + 120, grip.y + grip.height / 2)
    await page.mouse.down()
    await page.mouse.move(grip.x - 2000, grip.y + grip.height / 2, { steps: 10 })
    await page.mouse.up()
    const moved = (await details.boundingBox())!
    expect(moved.x).toBeCloseTo(area.x, 0)
    expect(moved.y).toBeCloseTo(before.y, 0)

    await canvas.getByRole('button', { name: `Mirror from ${AGGREGATE} to ${COPY}. Show details` }).click()
    await expect(details.getByRole('heading', { name: 'Mirror' })).toBeVisible()
    expect(await details.boundingBox()).toEqual(moved)

    await details.getByRole('button', { name: 'Close link details' }).click()
    await expect(details).toHaveCount(0)

    await canvas.getByRole('link', { name: `${COPY}: show its relations` }).click()
    await expect(page).toHaveURL(new RegExp(`/streams/${COPY}/relations$`))
    await expect(page.getByTestId('relation-node').and(page.locator('[aria-current="page"]'))).toContainText(COPY)
  })

  test('keeps the graph centered when the window is resized', async ({ page }) => {
    await page.setViewportSize({ width: 1280, height: 720 })
    await page.goto(`/streams/${AGGREGATE}/relations`)
    const canvas = page.getByTestId('relations-canvas')
    await expect(canvas.getByTestId('relation-node')).toHaveCount(3)

    await page.setViewportSize({ width: 1700, height: 950 })
    await expect(async () => {
      const area = (await canvas.boundingBox())!
      const boxes = await canvas.getByTestId('relation-node').evaluateAll((nodes) =>
        nodes.map((n) => n.getBoundingClientRect()).map((r) => ({ left: r.left, right: r.right, top: r.top, bottom: r.bottom })),
      )
      const left = Math.min(...boxes.map((b) => b.left))
      const right = Math.max(...boxes.map((b) => b.right))
      const top = Math.min(...boxes.map((b) => b.top))
      const bottom = Math.max(...boxes.map((b) => b.bottom))
      expect(Math.abs((left + right) / 2 - (area.x + area.width / 2))).toBeLessThan(4)
      expect(Math.abs((top + bottom) / 2 - (area.y + area.height / 2))).toBeLessThan(4)
    }).toPass()
  })
})

const DENSE = {
  EU: 'E2E_RELD_EU',
  US: 'E2E_RELD_US',
  HUB: 'E2E_RELD_HUB',
  ARCHIVE: 'E2E_RELD_ARCHIVE',
  WORK: 'E2E_RELD_WORK',
  EVENTS: 'E2E_RELD_EVENTS',
  AUDIT: 'E2E_RELD_AUDIT',
  TIMELINE: 'E2E_RELD_TIMELINE',
  NOTIFY: 'E2E_RELD_NOTIFY',
}

function drawingProblems(canvas: Locator) {
  return canvas.evaluate((root) => {
    interface Box {
      left: number
      right: number
      top: number
      bottom: number
    }
    interface Point {
      x: number
      y: number
    }
    const rect = (el: Element): Box => el.getBoundingClientRect()
    const shrink = (r: Box, by: number): Box => ({ left: r.left + by, right: r.right - by, top: r.top + by, bottom: r.bottom - by })
    const overlap = (a: Box, b: Box) => a.left < b.right && b.left < a.right && a.top < b.bottom && b.top < a.bottom
    const inside = (p: Point, r: Box) => p.x > r.left && p.x < r.right && p.y > r.top && p.y < r.bottom
    const cards = [...root.querySelectorAll('[data-testid="relation-node"]')].map((el) => shrink(rect(el), 1))
    const chips = [...root.querySelectorAll('[data-testid="relation-edge"]')].map((el) => shrink(rect(el), 1))
    const lines = [...root.querySelectorAll<SVGPathElement>('svg > path[marker-end]')].map((path) => {
      const matrix = path.getScreenCTM()!
      const length = path.getTotalLength()
      const points: Point[] = []
      for (let s = 0; s <= length; s += 2) points.push(path.getPointAtLength(s).matrixTransform(matrix))
      points.push(path.getPointAtLength(length).matrixTransform(matrix))
      return points
    })

    const problems = new Set<string>()
    if (lines.length === 0 || lines.length !== chips.length) problems.add(`${lines.length} links for ${chips.length} labels`)
    chips.forEach((chip, i) => {
      chips.forEach((other, j) => j > i && overlap(chip, other) && problems.add(`label ${i} on label ${j}`))
      cards.forEach((card, k) => overlap(chip, card) && problems.add(`label ${i} on card ${k}`))
    })
    lines.forEach((points, i) => {
      for (const p of points) {
        cards.forEach((card, k) => inside(p, card) && problems.add(`link ${i} through card ${k}`))
        chips.forEach((chip, k) => k !== i && inside(p, chip) && problems.add(`link ${i} under label ${k}`))
      }
    })

    const side = (o: Point, a: Point, b: Point) => Math.sign((a.x - o.x) * (b.y - o.y) - (a.y - o.y) * (b.x - o.x))
    let crossings = 0
    lines.forEach((a, i) => {
      lines.slice(i + 1).forEach((b) => {
        for (let s = 1; s < a.length; s++) {
          for (let t = 1; t < b.length; t++) {
            const crosses =
              side(a[s - 1], a[s], b[t - 1]) * side(a[s - 1], a[s], b[t]) < 0 &&
              side(b[t - 1], b[t], a[s - 1]) * side(b[t - 1], b[t], a[s]) < 0
            if (crosses) crossings++
          }
        }
      })
    })
    if (crossings > 0) problems.add(`${crossings} link crossings`)
    return [...problems]
  })
}

test.describe('stream relations drawing', () => {
  const names = Object.values(DENSE)

  test.beforeEach(async ({ env }) => {
    const connectionId = env.connectionId
    for (const name of [...names].reverse()) await deleteStream(connectionId, name)
    const create = (body: Record<string, unknown>) => call(MANAGEMENT, 'CreateStream', { connectionId, ...body })
    await create({ name: DENSE.EU, subjects: ['e2e.reld.eu.>'] })
    await create({ name: DENSE.US, subjects: ['e2e.reld.us.>'] })
    await create({
      name: DENSE.HUB,
      subjects: ['e2e.reld.hub.>'],
      sources: [
        { name: DENSE.EU, filterSubject: 'e2e.reld.eu.orders.>' },
        { name: DENSE.EU, filterSubject: 'e2e.reld.eu.payments.>' },
        { name: DENSE.US },
        { name: 'E2E_RELD_GHOST' },
      ],
    })
    await create({ name: DENSE.ARCHIVE, mirror: { name: DENSE.HUB } })
    await create({ name: DENSE.WORK, sources: [{ name: DENSE.HUB, filterSubject: 'e2e.reld.eu.orders.>' }] })
    await create({
      name: DENSE.EVENTS,
      subjects: ['e2e.reld.events.>'],
      republish: { src: 'e2e.reld.events.>', dest: 'e2e.reld.audit.events.>' },
    })
    await create({ name: DENSE.AUDIT, subjects: ['e2e.reld.audit.>'], sources: [{ name: DENSE.HUB }] })
    await create({
      name: DENSE.TIMELINE,
      sources: [{ name: DENSE.AUDIT }, { name: DENSE.WORK }, { name: DENSE.HUB, filterSubject: 'e2e.reld.us.>' }],
    })
    await create({
      name: DENSE.NOTIFY,
      subjects: ['e2e.reld.notify.>'],
      republish: { src: 'e2e.reld.notify.>', dest: 'e2e.reld.nowhere.>' },
    })
  })

  test.afterEach(async ({ env }) => {
    for (const name of names) await deleteStream(env.connectionId, name)
  })

  test('draws a busy neighborhood without crossings or covered cards and labels', async ({ page }) => {
    await page.setViewportSize({ width: 1600, height: 1000 })
    await page.goto(`/streams/${DENSE.HUB}/relations`)
    const canvas = page.getByTestId('relations-canvas')
    await expect(canvas.getByTestId('relation-node')).toHaveCount(9)
    await expect(canvas.getByTestId('relation-edge')).toHaveCount(11)
    await expect(canvas.getByText('Not found on this connection')).toBeVisible()
    await expect(canvas.getByRole('button', { name: `Republish from ${DENSE.EVENTS} to ${DENSE.AUDIT}. Show details` })).toBeVisible()

    await expect.poll(() => drawingProblems(canvas)).toEqual([])

    await page.getByRole('button', { name: 'Zoom in' }).click()
    await page.getByRole('button', { name: 'Zoom in' }).click()
    expect(await drawingProblems(canvas)).toEqual([])
  })

  test('draws a republish to a subject no stream captures', async ({ page }) => {
    await page.goto(`/streams/${DENSE.NOTIFY}/relations`)
    const canvas = page.getByTestId('relations-canvas')
    await expect(canvas.getByTestId('relation-node')).toHaveCount(2)
    await expect(canvas.getByText('No stream captures it')).toBeVisible()
    expect(await drawingProblems(canvas)).toEqual([])
  })
})
