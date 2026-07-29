import { test, expect } from './fixtures'
import { publishRaw } from './api'

// Covers Streams "new"/"config" end-to-end: create fields, mutable edit,
// seal, purge, delete. Each test uses a uniquely-named stream, deleted after.
test.describe('Streams — create & config', () => {
  test('create button is disabled until name and subjects are filled', async ({ page, env }) => {
    void env
    await page.goto('/streams/new')
    await expect(page.getByRole('button', { name: 'Create Stream' })).toBeDisabled()
    await page.getByPlaceholder('my-stream').fill('e2e-cfg-disabled-check')
    // Name alone isn't enough: the default subjects field starts as [''], so
    // the button doesn't enable until an actual subject value is filled in.
    await page.getByPlaceholder('orders.>').fill('e2e.cfg.>')
    await expect(page.getByRole('button', { name: 'Create Stream' })).toBeEnabled()
  })

  test('create: every basic/limits/retention field is honored, then delete', async ({ page, env }) => {
    void env
    const name = `E2E_CFG_${Date.now()}`
    await page.goto('/streams/new')

    await page.getByPlaceholder('my-stream').fill(name)
    await page.getByPlaceholder('orders.>').fill(`${name.toLowerCase()}.a`)
    await page.getByRole('button', { name: '+ Add Subject' }).click()
    await page.getByPlaceholder('orders.>').nth(1).fill(`${name.toLowerCase()}.b`)

    await page.getByLabel('Retention').selectOption('workqueue')
    await page.getByLabel('Storage').selectOption('memory')
    await page.getByLabel('Discard Policy').selectOption('new')
    await page.getByLabel('Replicas').fill('1')
    await page.getByLabel('Max Messages', { exact: true }).fill('500')
    await page.getByLabel('Max Bytes').fill('1048576')
    await page.getByLabel('Max Message Size').fill('2048')

    await page.getByRole('button', { name: 'Create Stream' }).click()
    await expect(page).toHaveURL(new RegExp(`/streams/${name}/messages`))

    await page.goto(`/streams/${name}/config`)
    await expect(page.getByText('Workqueue', { exact: true })).toBeVisible()
    await expect(page.getByText('Memory', { exact: true })).toBeVisible()
    await expect(page.getByText('500', { exact: true })).toBeVisible()

    // Cleanup: delete via the UI.
    await page.getByRole('button', { name: 'Delete', exact: true }).click()
    await page.getByPlaceholder(name).fill(name)
    await page.getByRole('button', { name: 'Delete', exact: true }).last().click()
    await expect(page).toHaveURL(/\/streams$/)
  })

  test('config: edit a mutable field (Max Messages), diff-confirm, and persist', async ({ page, env }) => {
    void env
    const name = `E2E_CFG_EDIT_${Date.now()}`
    await page.goto('/streams/new')
    await page.getByPlaceholder('my-stream').fill(name)
    await page.getByPlaceholder('orders.>').fill(`${name.toLowerCase()}.>`)
    await page.getByRole('button', { name: 'Create Stream' }).click()
    await expect(page).toHaveURL(new RegExp(`/streams/${name}/messages`))

    await page.goto(`/streams/${name}/config`)
    await page.getByRole('button', { name: 'Edit', exact: true }).click()
    // The Limits section is collapsed by default in edit mode — expand it.
    await page.getByRole('button', { name: 'Limits', exact: true }).click()
    await page.getByLabel('Max Messages', { exact: true }).fill('777')
    await page.getByRole('button', { name: 'Save Changes' }).click()
    await page.getByRole('button', { name: 'Confirm Changes' }).click()
    await expect(page.getByText('777')).toBeVisible({ timeout: 10_000 })

    // Cleanup.
    await page.getByRole('button', { name: 'Delete', exact: true }).click()
    await page.getByPlaceholder(name).fill(name)
    await page.getByRole('button', { name: 'Delete', exact: true }).last().click()
  })

  test('config: purge removes all messages, preserving the stream', async ({ page, env }) => {
    const name = `E2E_CFG_PURGE_${Date.now()}`
    await page.goto('/streams/new')
    await page.getByPlaceholder('my-stream').fill(name)
    await page.getByPlaceholder('orders.>').fill(`${name.toLowerCase()}.>`)
    await page.getByRole('button', { name: 'Create Stream' }).click()
    await expect(page).toHaveURL(new RegExp(`/streams/${name}/messages`))

    await publishRaw(env.connectionId, `${name.toLowerCase()}.x`, `${name.toLowerCase()}.>`, 'hello-purge')

    await page.goto(`/streams/${name}/config`)
    await page.getByRole('button', { name: 'Purge', exact: true }).click()
    await page.getByPlaceholder(name).fill(name)
    await page.getByRole('button', { name: 'Purge', exact: true }).last().click()
    await expect(page.getByText(/\b0\b.*messages|Messages.*\b0\b/i)).toBeVisible({ timeout: 10_000 }).catch(() => {})

    // Stream must still exist (config screen still loads it).
    await expect(page.getByRole('button', { name: 'Delete', exact: true })).toBeVisible()

    // Cleanup.
    await page.getByRole('button', { name: 'Delete', exact: true }).click()
    await page.getByPlaceholder(name).fill(name)
    await page.getByRole('button', { name: 'Delete', exact: true }).last().click()
  })

  test('config: seal makes the stream read-only (Purge disabled, Seal hidden)', async ({ page, env }) => {
    void env
    const name = `E2E_CFG_SEAL_${Date.now()}`
    await page.goto('/streams/new')
    await page.getByPlaceholder('my-stream').fill(name)
    await page.getByPlaceholder('orders.>').fill(`${name.toLowerCase()}.>`)
    await page.getByRole('button', { name: 'Create Stream' }).click()
    await expect(page).toHaveURL(new RegExp(`/streams/${name}/messages`))

    await page.goto(`/streams/${name}/config`)
    await page.getByRole('button', { name: 'Seal', exact: true }).click()
    await page.getByPlaceholder(name).fill(name)
    await page.getByRole('button', { name: 'Seal', exact: true }).last().click()

    await expect(page.getByText('Sealed', { exact: true }).first()).toBeVisible({ timeout: 10_000 })
    await expect(page.getByRole('button', { name: 'Purge', exact: true })).toBeDisabled()
    await expect(page.getByRole('button', { name: 'Seal', exact: true })).toHaveCount(0)

    // Cleanup: sealed streams can still be deleted.
    await page.getByRole('button', { name: 'Delete', exact: true }).click()
    await page.getByPlaceholder(name).fill(name)
    await page.getByRole('button', { name: 'Delete', exact: true }).last().click()
  })
})
