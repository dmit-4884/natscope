import { ensureReadOnlyCopy, publishRaw } from './api'
import { test, expect, STREAM, PLAIN_SUBJECT } from './fixtures'

const NAME = 'e2e-readonly'

test.describe('read-only connection', () => {
  let roId = ''

  test.beforeEach(async ({ context, env }) => {
    roId = await ensureReadOnlyCopy(env.connectionId, NAME)
    await publishRaw(env.connectionId, PLAIN_SUBJECT, PLAIN_SUBJECT, '{"readonly":true}')
    await context.addInitScript(
      ([id, name, url]) => {
        localStorage.setItem('nats_active_connection_id', id)
        localStorage.setItem('nats_active_connection_info', JSON.stringify({ id, name, urls: [url] }))
      },
      [roId, NAME, env.connectionUrl] as const,
    )
  })

  test('the header names the environment and the stream offers no writes', async ({ page }) => {
    await page.goto(`/streams/${STREAM}/messages`)

    const header = page.getByRole('banner')
    await expect(header.getByText('PROD')).toBeVisible()
    await expect(header.getByText('Read-only')).toBeVisible()
    await expect(page.getByTestId('stream-tabs').getByRole('link', { name: 'Publish' })).toHaveCount(0)

    await page.locator('[role="row"]').filter({ hasText: PLAIN_SUBJECT }).first().click()
    await expect(page.getByRole('complementary').getByText(/^Message #\d+$/)).toBeVisible()
    await expect(page.getByTestId('delete-message')).toHaveCount(0)
    await expect(page.getByTestId('resend-message')).toHaveCount(0)

    await page.getByTestId('stream-tabs').getByRole('link', { name: 'Config' }).click()
    await expect(page.getByRole('heading', { name: 'Configuration', exact: true })).toBeVisible()
    await expect(page.getByRole('button', { name: 'Edit' })).toHaveCount(0)
    await expect(page.getByRole('button', { name: 'Delete' })).toHaveCount(0)
  })

  test('write pages explain the switch and link to the connection settings', async ({ page }) => {
    for (const path of [`/streams/${STREAM}/publish`, '/request', '/streams/new', '/kv/new']) {
      await page.goto(path)
      await expect(page.getByText('This connection is read-only')).toBeVisible()
    }
    await page.getByRole('link', { name: 'Connection settings' }).click()
    await expect(page).toHaveURL(new RegExp(`/settings/connections/${roId}/edit$`))
    await expect(page.getByRole('switch', { name: 'Read-only' })).toBeChecked()
    await expect(page.getByLabel('Label', { exact: true })).toHaveValue('PROD')
    await expect(page.getByRole('radio', { name: 'Red' })).toHaveAttribute('aria-checked', 'true')
  })
})
