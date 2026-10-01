import { test, expect } from './fixtures'
import { call } from './api'

const SOURCES = 'natscope.proto.sources.v1.SourcesService'
const REPO = process.env.E2E_GIT_REPO ?? 'https://github.com/bufbuild/protovalidate.git'
const NAME = 'e2e-git-refs'

async function deleteByName() {
  const list = await call<{ sources?: Array<{ id: string; name: string }> }>(SOURCES, 'ListSources', { pageSize: 500 })
  for (const s of list.sources ?? []) {
    if (s.name === NAME) await call(SOURCES, 'DeleteSource', { id: s.id })
  }
}

test.describe('Proto source: git refs', () => {
  test.beforeAll(async () => {
    await deleteByName()
    await call(SOURCES, 'CreateSource', {
      name: NAME,
      sourceType: 'SOURCE_TYPE_GIT',
      repository: REPO,
      excludePrefixes: ['proto/protovalidate-testing'],
    })
  })
  test.afterAll(deleteByName)

  test('picks a tag, switches to a branch and refreshes it', async ({ page, env }) => {
    void env
    test.setTimeout(240_000)
    await page.goto('/settings/proto')
    const card = page.getByTestId('proto-source-card').filter({ hasText: NAME })

    await card.getByTestId('git-ref-change').click()
    await card.getByRole('button', { name: 'Tag or branch' }).click()
    await page.getByRole('option', { name: /^v1\.2\.2 · tag · / }).click()
    await expect(card.getByTestId('git-compile-ok')).toBeVisible({ timeout: 120_000 })
    await expect(card.getByTestId('git-ref-badge')).toHaveText('v1.2.2')
    await expect(card.getByText('tag', { exact: true })).toBeVisible()

    await card.getByTestId('git-ref-change').click()
    await card.getByRole('button', { name: 'Tag or branch' }).click()
    await page.getByRole('option', { name: /^main · branch · / }).click()
    await expect(card.getByTestId('git-ref-badge')).toHaveText('main', { timeout: 120_000 })

    await card.getByTestId('git-ref-refresh').click()
    await expect(card.getByTestId('git-compile-ok')).toBeVisible({ timeout: 120_000 })
    await expect(card.getByText('branch', { exact: true })).toBeVisible()
  })
})
