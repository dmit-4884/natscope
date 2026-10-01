import { mkdirSync, writeFileSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import type { Page } from '@playwright/test'
import { test, expect } from './fixtures'
import { call, createMapping, deleteMappingsByPrefix } from './api'

const SOURCES = 'natscope.proto.sources.v1.SourcesService'
const SOURCE_NAME = 'e2e-nested'
const NESTED_PROTO = `syntax = "proto3";
package e2enested;

enum Color {
  COLOR_UNSPECIFIED = 0;
  COLOR_RED = 1;
}

message Line {
  string sku = 1;
  Color color = 2;
}

message Order {
  Line primary = 5;
  repeated Line lines = 6;
  map<string, Line> by_key = 7;
  Color color = 8;
}
`

let sourceId = ''

async function deleteSource(): Promise<void> {
  const list = await call<{ sources?: Array<{ id: string; name: string }> }>(SOURCES, 'ListSources', { pageSize: 500 })
  for (const s of list.sources ?? []) {
    if (s.name === SOURCE_NAME) await call(SOURCES, 'DeleteSource', { id: s.id })
  }
}

test.beforeAll(async () => {
  await deleteSource()
  await deleteMappingsByPrefix('e2e.nested')
  const dir = join(tmpdir(), 'natscope-e2e-nested')
  mkdirSync(dir, { recursive: true })
  writeFileSync(join(dir, 'nested.proto'), NESTED_PROTO)
  const created = await call<{ source?: { id: string } }>(SOURCES, 'CreateSource', {
    name: SOURCE_NAME,
    sourceType: 'SOURCE_TYPE_LOCAL',
    localPath: dir,
    watcherEnabled: false,
  })
  sourceId = created.source?.id ?? ''
  await call(SOURCES, 'RefreshSource', { sourceId })
  await createMapping('e2e.nested.>', 'e2enested.Order', sourceId)
})

test.afterAll(async () => {
  await deleteMappingsByPrefix('e2e.nested')
  await deleteSource()
})

async function openEditor(page: Page) {
  await page.goto('/request')
  await page.getByLabel('Subject').fill('e2e.nested.one')
  await expect(page.getByRole('region', { name: 'Request' }).getByText('e2enested.Order', { exact: true })).toBeVisible()
  const editor = page.getByRole('region', { name: 'Request' }).locator('.cm-content')
  await editor.fill('')
  await editor.click()
  return editor
}

function suggestions(page: Page) {
  return page.locator('.cm-tooltip-autocomplete li')
}

test.describe('schema-aware JSON completion', () => {
  test('suggests the fields of nested messages inside repeated fields and maps', async ({ page, env: _env }) => {
    await openEditor(page)
    await page.keyboard.type('{"lines": [{"')
    await expect(suggestions(page)).toHaveText([/^sku/, /^color/])
    await page.keyboard.press('Escape')

    await page.keyboard.press('ControlOrMeta+a')
    await page.keyboard.press('Backspace')
    await page.keyboard.type('{"by_key": {"a": {"')
    await expect(suggestions(page)).toHaveText([/^sku/, /^color/])
  })

  test('suggests enum values and inserts the chosen key', async ({ page, env: _env }) => {
    const editor = await openEditor(page)
    await page.keyboard.type('{"color": "')
    await expect(suggestions(page)).toHaveText([/^COLOR_UNSPECIFIED/, /^COLOR_RED/])
    await page.keyboard.press('Escape')

    await page.keyboard.press('ControlOrMeta+a')
    await page.keyboard.press('Backspace')
    await page.keyboard.type('{"pri')
    await expect(suggestions(page).first()).toHaveText(/^primary/)
    await page.keyboard.press('Enter')
    await expect(editor).toHaveText('{"primary": }')
  })
})
