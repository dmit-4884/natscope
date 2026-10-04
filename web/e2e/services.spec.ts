import { test, expect } from './fixtures'

test.describe('services', () => {
  test('the page answers with the services it found or explains why there are none', async ({ page, env: _env }) => {
    await page.goto('/services')
    await expect(page.getByRole('heading', { name: 'Services', level: 2 })).toBeVisible()

    const list = page.getByTestId('service-detail')
    const empty = page.getByText('No services found')
    await expect(list.or(empty)).toBeVisible()
    await expect(page.getByTestId('services-updated')).toBeVisible()
  })

  test('refresh asks the services again', async ({ page, env: _env }) => {
    await page.goto('/services')
    await expect(page.getByTestId('services-updated')).toBeVisible()

    const request = page.waitForRequest((r) => r.url().includes('/ListServices'))
    await page.getByRole('button', { name: 'Refresh' }).last().click()
    await request
  })

  test('the sidebar leads to both pages', async ({ page, env: _env }) => {
    await page.goto('/request')
    await page.getByRole('link', { name: 'Services' }).click()
    await expect(page).toHaveURL(/\/services$/)
    await page.getByRole('link', { name: 'Subscribe' }).click()
    await expect(page).toHaveURL(/\/subscribe$/)
  })
})
