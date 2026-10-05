import { deleteConnectionsByPrefix } from './api'
import { test, expect } from './fixtures'

const PREFIX = 'e2e-cli-'

test.describe('import from nats CLI', () => {
  test.afterEach(async () => {
    await deleteConnectionsByPrefix(PREFIX)
  })

  test('uploaded context files become connections', async ({ page, env }) => {
    const name = `${PREFIX}${Date.now()}`
    await page.goto('/settings/connections')
    await page.getByRole('button', { name: 'From nats CLI' }).click()
    const dialog = page.getByRole('dialog', { name: 'Import from nats CLI' })

    await dialog.getByLabel('Upload context files').setInputFiles({
      name: `${name}.json`,
      mimeType: 'application/json',
      buffer: Buffer.from(JSON.stringify({ url: env.connectionUrl, jetstream_domain: 'hub', creds: '/tmp/nobody.creds' })),
    })

    await expect(dialog.getByRole('checkbox', { name: `Import ${name}` })).toBeChecked()
    await expect(dialog.getByText('domain hub')).toBeVisible()
    await expect(dialog.getByText(/nobody\.creds stays on the machine/)).toBeVisible()
    await dialog.getByRole('button', { name: 'Import 1 connection' }).click()

    await expect(dialog).toBeHidden()
    await expect(page.getByText(name, { exact: true })).toBeVisible()
  })
})
