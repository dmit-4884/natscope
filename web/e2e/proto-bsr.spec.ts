import { test, expect } from './fixtures'
import { call } from './api'

const SOURCES = 'natscope.proto.sources.v1.SourcesService'
const MODULE = process.env.E2E_BSR_MODULE ?? 'buf.build/bufbuild/protovalidate'
const NAME = 'e2e-bsr'

async function deleteByName() {
  const list = await call<{ sources?: Array<{ id: string; name: string }> }>(SOURCES, 'ListSources', { pageSize: 500 })
  for (const s of list.sources ?? []) {
    if (s.name === NAME) await call(SOURCES, 'DeleteSource', { id: s.id })
  }
}

test.describe('Proto source: Buf Schema Registry', () => {
  test.beforeAll(deleteByName)
  test.afterAll(deleteByName)

  test('adds a module, loads its default label and browses it', async ({ page, env }) => {
    void env
    test.setTimeout(180_000)
    await page.goto('/settings/proto/new')
    await page.getByRole('button', { name: /Buf Schema Registry/ }).click()
    await page.getByLabel('Name *').fill(NAME)
    await page.getByLabel('Module *').fill(MODULE)
    await page.getByRole('button', { name: 'Validate' }).click()
    await expect(page.getByText('Module is accessible')).toBeVisible({ timeout: 60_000 })
    await page.getByRole('button', { name: 'Add Source' }).click()
    await expect(page).toHaveURL(/\/settings\/proto$/)

    const card = page.getByTestId('proto-source-card').filter({ hasText: NAME })
    await expect(card.getByText('BSR', { exact: true })).toBeVisible()
    await card.getByTestId('ref-change').click()
    await card.getByRole('button', { name: 'Label', exact: true }).click()
    const main = page.getByRole('option', { name: /^main · label · [0-9a-f]{7}$/ })
    await expect(main).toBeVisible({ timeout: 60_000 })
    await main.click()
    await expect(card.getByTestId('ref-compile-ok')).toBeVisible({ timeout: 120_000 })
    await expect(card.getByTestId('ref-badge')).toHaveText('main')
    await expect(card.getByText('label', { exact: true })).toBeVisible()

    const browser = page.getByTestId('schema-browser')
    await browser.getByRole('textbox', { name: 'Search types or comments…' }).fill('buf.validate.FieldRules')
    await browser.getByRole('button', { name: /^message FieldRules/ }).click()
    const detail = page.getByTestId('schema-type-detail')
    await expect(detail.getByRole('heading', { name: 'buf.validate.FieldRules' })).toBeVisible()
    await expect(detail.getByText('string', { exact: true }).first()).toBeVisible()

    await card.getByTestId('ref-refresh').click()
    await expect(card.getByTestId('ref-compile-ok')).toBeVisible({ timeout: 120_000 })
  })

  test('reports an unknown module', async ({ page, env }) => {
    void env
    await page.goto('/settings/proto/new')
    await page.getByRole('button', { name: /Buf Schema Registry/ }).click()
    await page.getByLabel('Name *').fill(`${NAME}-missing`)
    await page.getByLabel('Module *').fill('buf.build/bufbuild/no-such-module-natscope-e2e')
    await page.getByRole('button', { name: 'Validate' }).click()
    await expect(page.getByText(/^Module is not accessible: /)).toBeVisible({ timeout: 60_000 })
  })
})
