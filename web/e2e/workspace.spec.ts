import { readFileSync } from 'node:fs'
import { test, expect } from './fixtures'
import { deleteMappingsByPrefix, mappingExists } from './api'

const WORKSPACE_URL = '/settings/workspace'
const TEST_PREFIX = 'e2e.ws.'

test.describe('Workspace export/import', () => {
  test.afterEach(async () => {
    await deleteMappingsByPrefix(TEST_PREFIX)
  })

  test('export downloads selected sections and contains no secrets', async ({ page, env }) => {
    void env
    await page.goto(WORKSPACE_URL)
    await expect(page.getByTestId('export-sections')).toBeVisible()

    const [download] = await Promise.all([
      page.waitForEvent('download'),
      page.getByTestId('workspace-export-btn').click(),
    ])
    const content = readFileSync(await download.path(), 'utf-8')
    const file = JSON.parse(content)

    expect(file.version).toBe(1)
    // Every registered section is present (all checkboxes default-checked).
    expect(Object.keys(file.sections).sort()).toEqual([
      'connections',
      'mappings',
      'proto_sources',
      'settings',
      'templates',
    ])

    // Hard security assertion: no credential field keys or encrypted blobs.
    for (const secret of ['"password"', '"token":', '"clientKey"', '"clientCert"', '"nkeySeed"', '"jwt"', '"credentials"', 'enc:v1']) {
      expect(content, `export must not contain ${secret}`).not.toContain(secret)
    }
  })

  test('import dry-run reports, then merge applies', async ({ page, env }) => {
    const pattern = `${TEST_PREFIX}${Date.now()}`
    const workspaceFile = JSON.stringify({
      version: 1,
      sections: {
        mappings: {
          version: 1,
          items: [{ pattern, messageType: 'e2e.E2EMessage', sourceId: env.protoSourceId }],
        },
      },
    })

    await page.goto(WORKSPACE_URL)
    await page.getByTestId('workspace-file-input').setInputFiles({
      name: 'natscope-workspace.json',
      mimeType: 'application/json',
      buffer: Buffer.from(workspaceFile),
    })

    // Dry-run preview shows one new mapping.
    const report = page.getByTestId('import-report-mappings')
    await expect(report).toBeVisible()
    await expect(report).toContainText('1 new')

    // Apply the merge.
    await page.getByTestId('workspace-import-btn').click()
    await expect(page.locator('[data-sonner-toast]').getByText(/Import applied/)).toBeVisible()

    // Backend now has the mapping.
    expect(await mappingExists(pattern)).toBe(true)
  })

  test('a broken file shows a clear error', async ({ page, env }) => {
    void env
    await page.goto(WORKSPACE_URL)
    await page.getByTestId('workspace-file-input').setInputFiles({
      name: 'bad.json',
      mimeType: 'application/json',
      buffer: Buffer.from('this is not json {{{'),
    })
    await expect(page.locator('[data-sonner-toast]').getByText(/Invalid workspace file/)).toBeVisible()
    // No report rendered for an invalid file.
    await expect(page.getByTestId('import-reports')).toHaveCount(0)
  })
})
