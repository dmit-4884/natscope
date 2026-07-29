import { test, expect } from './fixtures'

/** Covers the KV Store: bucket create (every config field), key put/get/edit/reset/purge/delete, bucket delete. */
test.describe('KV store', () => {
  test('create bucket with full config, then CRUD a key, then delete the bucket', async ({ page, env }) => {
    void env
    const bucket = `e2e-kv-${Date.now()}`
    await page.goto('/kv/new')

    await page.getByLabel('Bucket Name').fill(bucket)
    await page.getByLabel('Description', { exact: true }).fill('e2e kv bucket')
    await page.getByLabel('History (revisions per key)').fill('3')

    await page.getByRole('button', { name: 'Limits' }).click()
    await page.getByLabel('Max Value Size').fill('65536')
    await page.getByLabel('Max Bytes', { exact: true }).fill('1048576')
    await page.getByLabel('TTL (nanoseconds)').fill('3600000000000')

    await page.getByRole('button', { name: 'Storage Options' }).click()
    await page.getByLabel('Storage Type').click()
    await page.getByRole('option', { name: 'Memory' }).click()

    await page.getByRole('button', { name: 'Create KV Store' }).click()
    await expect(page).toHaveURL(new RegExp(`/kv/${bucket}`))

    // Create a key.
    await page.getByRole('button', { name: 'Create New Key' }).click()
    await page.getByPlaceholder('my.key.name').fill('greeting')
    await page.getByPlaceholder('Enter value (text or JSON)').fill('hello-kv-e2e')
    await page.getByRole('button', { name: 'Create Key' }).click()

    await expect(page.getByText('greeting', { exact: true })).toBeVisible({ timeout: 10_000 })
    await page.getByText('greeting', { exact: true }).click()
    await expect(page.getByRole('heading', { name: 'greeting' })).toBeVisible()

    // Edit the value, save, verify Reset restores the last-saved value.
    const valueBox = page.locator('textarea').first()
    await valueBox.fill('updated-value')
    await page.getByRole('button', { name: 'Save Value' }).click()
    await expect(page.getByText('Rev 2', { exact: false })).toBeVisible({ timeout: 10_000 })

    await valueBox.fill('unsaved-scratch')
    await page.getByRole('button', { name: 'Reset' }).click()
    await expect(valueBox).toHaveValue('updated-value')

    // Purge (all revisions), then delete the key outright.
    await page.getByRole('button', { name: 'Purge', exact: true }).click()
    await page.getByRole('button', { name: 'Purge Key' }).click()
    await expect(page.getByText('Select a key to view/edit')).toBeVisible({ timeout: 10_000 })

    // Recreate + delete to cover the Delete-key path too.
    await page.getByRole('button', { name: 'Create New Key' }).click()
    await page.getByPlaceholder('my.key.name').fill('to-delete')
    await page.getByPlaceholder('Enter value (text or JSON)').fill('bye')
    await page.getByRole('button', { name: 'Create Key' }).click()
    await expect(page.getByText('to-delete', { exact: true })).toBeVisible({ timeout: 10_000 })
    await page.getByText('to-delete', { exact: true }).click()
    await page.getByRole('button', { name: 'Delete', exact: true }).click()
    await page.getByRole('dialog').getByRole('button', { name: 'Delete Key' }).click()
    await expect(page.getByText('to-delete', { exact: true })).toHaveCount(0)

    // Delete the bucket (type-to-confirm).
    await page.getByRole('button', { name: 'Delete Bucket' }).click()
    await page.getByPlaceholder(bucket).fill(bucket)
    await page.getByRole('button', { name: 'Delete KV Store' }).click()
    await expect(page).not.toHaveURL(new RegExp(`/kv/${bucket}$`))
  })
})
