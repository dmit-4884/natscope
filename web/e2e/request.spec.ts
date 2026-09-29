import type { Page } from '@playwright/test'
import { test, expect } from './fixtures'

async function openRequest(page: Page) {
  await page.goto('/request')
  await expect(page.getByRole('heading', { name: 'Request / Reply' })).toBeVisible()
}

async function send(page: Page, subject: string) {
  await page.getByLabel('Subject').fill(subject)
  await page.getByRole('button', { name: 'Send Request' }).click()
}

function reply(page: Page) {
  return page.getByRole('region', { name: 'Reply' })
}

test.describe('request / reply', () => {
  test('the server answers a JetStream API request', async ({ page, env: _env }) => {
    await openRequest(page)
    await send(page, '$JS.API.INFO')

    await expect(reply(page).getByTestId('reply-success')).toBeVisible()
    await expect(reply(page).getByText('Received')).toBeVisible()
    await expect(reply(page)).toContainText('account_info_response')
  })

  test('a subject nobody listens on fails fast with no responders', async ({ page, env: _env }) => {
    await openRequest(page)
    await send(page, `e2e.request.nobody.${Date.now()}`)

    await expect(reply(page).getByTestId('reply-no-responders')).toBeVisible()
  })

  test('a wildcard subject cannot be sent', async ({ page, env: _env }) => {
    await openRequest(page)
    await page.getByLabel('Subject').fill('e2e.request.*')

    await expect(page.getByTestId('request-subject-error')).toBeVisible()
    await expect(page.getByRole('button', { name: 'Send Request' })).toBeDisabled()
  })

  test('the draft survives a reload', async ({ page, env: _env }) => {
    await openRequest(page)
    await page.getByLabel('Subject').fill('e2e.request.draft')
    await page.getByLabel('Timeout').selectOption('2000')

    await page.reload()
    await expect(page.getByLabel('Subject')).toHaveValue('e2e.request.draft')
    await expect(page.getByLabel('Timeout')).toHaveValue('2000')
  })
})
