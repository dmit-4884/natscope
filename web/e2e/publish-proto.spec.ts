import type { Page } from '@playwright/test'
import { test, expect, publishUrl, PROTO_PATTERN, PROTO_MESSAGE_TYPE } from './fixtures'

/**
 * Proto-mode behaviors: the fixture maps `e2e.proto.*` to `e2e.E2EMessage`
 * (string name = 1; int32 count = 2; repeated string tags = 3) via a
 * compiled "files" source, so the publish tab auto-resolves to Protobuf.
 */

async function openProtoPublish(page: Page) {
  await page.goto(publishUrl)
  await expect(page.locator('#main-content').getByText('Subject Pattern', { exact: true })).toBeVisible()
  const trigger = page.locator('#main-content button', { hasText: /Select Subject Pattern|e2e\./ }).first()
  await trigger.click()
  await page.getByRole('button', { name: PROTO_PATTERN, exact: true }).click()
  // Mappings load async — wait until the mode resolves to Protobuf.
  await expect(page.locator('#main-content').getByText(PROTO_MESSAGE_TYPE)).toBeVisible()
}

function editor(page: Page) {
  return page.locator('#main-content .cm-content')
}

test.describe('publish tab — proto mode', () => {
  test('auto-validation badge: invalid payload → violations, valid payload → green', async ({ page, env: _env }) => {
    await openProtoPublish(page)

    // Wrong type for int32 `count` → schema-invalid.
    await editor(page).fill('{"name": "x", "count": "not-a-number"}')
    await expect(page.getByTestId('validation-badge')).toHaveAttribute('data-state', 'invalid', {
      timeout: 10_000,
    })

    await editor(page).fill('{"name": "x", "count": 2}')
    await expect(page.getByTestId('validation-badge')).toHaveAttribute('data-state', 'valid', {
      timeout: 10_000,
    })
  })

  test('editor suggests proto fields while typing a key', async ({ page, env: _env }) => {
    await openProtoPublish(page)

    await editor(page).fill('')
    await editor(page).click()
    await page.keyboard.type('{"na')

    const tooltip = page.locator('.cm-tooltip-autocomplete')
    await expect(tooltip).toBeVisible()
    await expect(tooltip.getByText('name', { exact: true })).toBeVisible()
    // Type detail comes from the schema.
    await expect(tooltip).toContainText('string')
    await page.keyboard.press('Escape')
  })

  test('Use Example Message generates a schema-shaped payload', async ({ page, env: _env }) => {
    await openProtoPublish(page)

    await page.getByRole('button', { name: 'Use Example Message' }).click()
    await expect(editor(page)).toContainText('name')
    await expect(editor(page)).toContainText('count')
  })
})
