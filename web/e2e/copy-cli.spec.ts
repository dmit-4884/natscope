import { test, expect, STREAM } from './fixtures'

test.describe('Copy as nats CLI', () => {
  test('stream config copies a valid `nats stream add` command', async ({ page, context, env }) => {
    void env
    await context.grantPermissions(['clipboard-read', 'clipboard-write'])
    await page.goto(`/streams/${STREAM}/config`)

    await page.getByRole('button', { name: 'Copy as nats CLI' }).click()

    const text = await page.evaluate(() => navigator.clipboard.readText())
    expect(text.startsWith(`nats stream add ${STREAM} `)).toBe(true)
    expect(text).toContain("--subjects='e2e.publish.*.*,e2e.plain,e2e.proto.*'")
    expect(text).toContain('--retention=limits')
    expect(text).toContain('--storage=file')
    expect(text).toContain('--max-msgs=10000')
    expect(text).toContain('--max-msg-size=1024')
  })
})
