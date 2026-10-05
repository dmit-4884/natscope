import { test, expect } from './fixtures'

test.describe('connection diagnosis', () => {
  test('a failed test says which step broke and what to do', async ({ page, env: _env }) => {
    await page.goto('/settings/connections/new')
    await page.getByRole('textbox', { name: 'Server URL 1' }).fill('nats://127.0.0.1:1')
    await page.getByRole('button', { name: 'Test', exact: true }).click()

    const checks = page.getByRole('list', { name: 'Connection checks' })
    const tcp = checks.getByRole('listitem').filter({ hasText: 'TCP' })
    await expect(tcp).toHaveAttribute('data-status', 'failed')
    await expect(tcp).toContainText('Nothing listens on port 1')
    await expect(checks.getByRole('listitem').filter({ hasText: 'DNS' })).toHaveAttribute('data-status', 'ok')
    await expect(checks.getByRole('listitem').filter({ hasText: 'JetStream' })).toHaveAttribute('data-status', 'skipped')
  })

  test('an unknown host fails at DNS', async ({ page, env: _env }) => {
    await page.goto('/settings/connections/new')
    await page.getByRole('textbox', { name: 'Server URL 1' }).fill('nats://no-such-host.invalid:4222')
    await page.getByRole('button', { name: 'Test', exact: true }).click()

    const dns = page.getByRole('list', { name: 'Connection checks' }).getByRole('listitem').filter({ hasText: 'DNS' })
    await expect(dns).toHaveAttribute('data-status', 'failed')
  })
})
