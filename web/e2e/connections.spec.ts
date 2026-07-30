import { test, expect } from './fixtures'

test.describe('Connections management (Settings > Connections)', () => {
  test('list shows the seeded "local" connection with ping/edit/delete controls', async ({ page, env }) => {
    void env
    await page.goto('/settings/connections')
    const card = page.getByText('local', { exact: true }).first()
    await expect(card).toBeVisible()
    await expect(page.getByRole('button', { name: 'Ping local' })).toBeVisible()
    await expect(page.getByRole('button', { name: 'Edit local' })).toBeVisible()
  })

  test('create: empty name and empty URL shows a client-side validation error', async ({ page, env }) => {
    void env
    await page.goto('/settings/connections/new')
    await page.getByRole('button', { name: 'Create connection' }).click()
    await expect(page.getByText('Name is required')).toBeVisible()
    await expect(page.getByText('At least one server URL is required')).toBeVisible()
  })

  test('create: duplicate name is rejected by the backend unique constraint', async ({ page, env }) => {
    void env
    await page.goto('/settings/connections/new')
    await page.getByLabel('Name').fill('local')
    await page.getByLabel('Server URL 1').fill('nats://127.0.0.1:4300')
    await page.getByRole('button', { name: 'Create connection' }).click()
    // Backend ErrConnectionNameAlreadyInUse (or similar) surfaces as an inline error.
    await expect(page.locator('text=/already|in use|exists/i')).toBeVisible({ timeout: 10_000 })
  })

  test('create: valid connection is saved and listed, then cleaned up', async ({ page, env }) => {
    void env
    const name = `e2e-created-${Date.now()}`
    await page.goto('/settings/connections/new')
    await page.getByLabel('Name').fill(name)
    await page.getByLabel('Server URL 1').fill('nats://127.0.0.1:4300')
    await page.getByRole('button', { name: 'Create connection' }).click()
    await expect(page).toHaveURL(/\/settings\/connections$/)
    await expect(page.getByText(name, { exact: true })).toBeVisible()

    // Cleanup: delete it via the UI so repeated runs don't accumulate connections.
    await page.getByRole('button', { name: `Delete ${name}` }).click()
    await page.getByRole('button', { name: 'Delete', exact: true }).click()
    await expect(page.getByText(name, { exact: true })).toHaveCount(0)
  })

  test('test connection: success against the live dedicated NATS server', async ({ page, env }) => {
    void env
    await page.goto('/settings/connections/new')
    await page.getByLabel('Name').fill(`e2e-test-ok-${Date.now()}`)
    await page.getByLabel('Server URL 1').fill('nats://127.0.0.1:4300')
    await page.getByRole('button', { name: 'Test', exact: true }).click()
    await expect(page.getByText('Success')).toBeVisible({ timeout: 10_000 })
    await expect(page.getByText('enabled')).toBeVisible() // JetStream: enabled
  })

  test('test connection: failure against an unreachable port shows an error', async ({ page, env }) => {
    void env
    await page.goto('/settings/connections/new')
    await page.getByLabel('Name').fill(`e2e-test-fail-${Date.now()}`)
    await page.getByLabel('Server URL 1').fill('nats://127.0.0.1:1')
    await page.getByRole('button', { name: 'Test', exact: true }).click()
    await expect(page.getByText('Failed:')).toBeVisible({ timeout: 15_000 })
  })

  test('edit: change description and save, list reflects the change', async ({ page, env }) => {
    void env
    const name = `e2e-edit-${Date.now()}`
    // Seed via the UI so the edit target is guaranteed to exist and be cleaned up.
    await page.goto('/settings/connections/new')
    await page.getByLabel('Name').fill(name)
    await page.getByLabel('Server URL 1').fill('nats://127.0.0.1:4300')
    await page.getByRole('button', { name: 'Create connection' }).click()
    await expect(page.getByText(name, { exact: true })).toBeVisible()

    await page.getByRole('button', { name: `Edit ${name}` }).click()
    await expect(page).toHaveURL(/\/settings\/connections\/.+\/edit$/)
    await page.getByLabel('Description (optional)').fill('edited via e2e')
    await page.getByRole('button', { name: 'Save changes' }).click()
    await expect(page).toHaveURL(/\/settings\/connections$/)
    await expect(page.getByText('edited via e2e')).toBeVisible()

    // Cleanup.
    await page.getByRole('button', { name: `Delete ${name}` }).click()
    await page.getByRole('button', { name: 'Delete', exact: true }).click()
    await expect(page.getByText(name, { exact: true })).toHaveCount(0)
  })

  test('delete: confirm dialog cancel keeps the connection', async ({ page, env }) => {
    void env
    const name = `e2e-cancel-delete-${Date.now()}`
    await page.goto('/settings/connections/new')
    await page.getByLabel('Name').fill(name)
    await page.getByLabel('Server URL 1').fill('nats://127.0.0.1:4300')
    await page.getByRole('button', { name: 'Create connection' }).click()
    await expect(page.getByText(name, { exact: true })).toBeVisible()

    await page.getByRole('button', { name: `Delete ${name}` }).click()
    await expect(page.getByText('Delete connection')).toBeVisible()
    await page.getByRole('button', { name: 'Cancel', exact: true }).click()
    await expect(page.getByText(name, { exact: true })).toBeVisible()

    // Cleanup.
    await page.getByRole('button', { name: `Delete ${name}` }).click()
    await page.getByRole('button', { name: 'Delete', exact: true }).click()
  })
})
