import { test, expect, STREAM, PLAIN_SUBJECT } from './fixtures'
import { publishRaw, streamLastSeq, getMessageAt, listMessagesByTime } from './api'

const messagesUrl = `/streams/${STREAM}/messages`
const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms))

test.describe('Jump-to-time', () => {
  // Minute-granularity picker can't separate sub-second publishes: precise
  // resolution is tested via the API; the UI test only covers coarse wiring.
  test('resolves start_time to the first message at or after the target', async ({ env }) => {
    // Publish three messages spaced far enough apart that the server stamps
    // each with a distinct time.
    await publishRaw(env.connectionId, PLAIN_SUBJECT, PLAIN_SUBJECT, JSON.stringify({ n: 1 }))
    await sleep(80)
    await publishRaw(env.connectionId, PLAIN_SUBJECT, PLAIN_SUBJECT, JSON.stringify({ n: 2 }))
    await sleep(80)
    await publishRaw(env.connectionId, PLAIN_SUBJECT, PLAIN_SUBJECT, JSON.stringify({ n: 3 }))

    const last = await streamLastSeq(env.connectionId, STREAM)
    const m1 = await getMessageAt(env.connectionId, STREAM, last - 2)
    const m2 = await getMessageAt(env.connectionId, STREAM, last - 1)
    const m3 = await getMessageAt(env.connectionId, STREAM, last)

    // Jump to the exact time of the second message → page starts at it.
    const atM2 = await listMessagesByTime(env.connectionId, STREAM, m2.timestamp)
    expect(atM2[0]?.sequence).toBe(m2.sequence)

    // Jump to the first message's time → page starts at the first.
    const atM1 = await listMessagesByTime(env.connectionId, STREAM, m1.timestamp)
    expect(atM1[0]?.sequence).toBe(m1.sequence)

    // Jump to a time after the last message → nothing at/after it.
    const future = new Date(new Date(m3.timestamp).getTime() + 24 * 3600_000).toISOString()
    const atFuture = await listMessagesByTime(env.connectionId, STREAM, future)
    expect(atFuture.length).toBe(0)
  })

  test('UI: date picker jump shows the resolved sequence', async ({ page, env }) => {
    // Make sure the stream has at least one message to resolve to.
    await publishRaw(env.connectionId, PLAIN_SUBJECT, PLAIN_SUBJECT, JSON.stringify({ ui: 'jump' }))

    await page.goto(messagesUrl)

    // Open the filters panel and pick a coarse "7d ago" anchor, then apply.
    await page.getByRole('button', { name: 'Filters' }).click()
    await page.getByRole('button', { name: '7d ago' }).click()
    await page.getByRole('button', { name: 'Apply' }).click()

    // The resolved-sequence banner appears with a concrete sequence number.
    const banner = page.getByTestId('jump-resolved')
    await expect(banner).toBeVisible()
    await expect(page.getByTestId('jump-resolved-seq')).toContainText(/#\d+/)
  })
})
