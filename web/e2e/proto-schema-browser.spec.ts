import { mkdtempSync, writeFileSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { test, expect } from './fixtures'
import { call } from './api'

const SOURCES = 'natscope.proto.sources.v1.SourcesService'
const NAME = 'e2e-schema-browser'
const PROTO = `syntax = "proto3";
package e2e.browser;

import "google/protobuf/timestamp.proto";

// A parcel on its way to a customer.
message Parcel {
  // Tracking number printed on the label.
  string tracking = 1;
  Stage stage = 2;
  google.protobuf.Timestamp shipped_at = 3;
  map<string, string> labels = 4;
}

// Where the parcel is.
enum Stage {
  STAGE_UNSPECIFIED = 0;
  // Left the warehouse.
  STAGE_SHIPPED = 1;
}

// Parcel tracking API.
service Tracking {
  // Streams parcel updates.
  rpc Watch(Parcel) returns (stream Parcel);
}
`

async function deleteByName() {
  const list = await call<{ sources?: Array<{ id: string; name: string }> }>(SOURCES, 'ListSources', { pageSize: 500 })
  for (const s of list.sources ?? []) {
    if (s.name === NAME) await call(SOURCES, 'DeleteSource', { id: s.id })
  }
}

test.describe('Proto schema browser', () => {
  test.beforeAll(async () => {
    await deleteByName()
    const dir = mkdtempSync(join(tmpdir(), 'e2e-browser-'))
    writeFileSync(join(dir, 'parcel.proto'), PROTO)
    const created = await call<{ source?: { id: string } }>(SOURCES, 'CreateSource', {
      name: NAME,
      sourceType: 'SOURCE_TYPE_LOCAL',
      localPath: dir,
      watcherEnabled: false,
    })
    await call(SOURCES, 'RefreshSource', { sourceId: created.source?.id })
  })
  test.afterAll(deleteByName)

  test('browses messages, enums and services with their comments', async ({ page, env }) => {
    void env
    await page.goto('/settings/proto')
    const browser = page.getByTestId('schema-browser')
    await browser.getByRole('textbox', { name: 'Search types or comments…' }).fill('e2e.browser')

    const parcel = browser.getByRole('button', { name: /^message Parcel/ })
    await expect(parcel).toContainText('A parcel on its way to a customer.')
    await parcel.click()

    const detail = page.getByTestId('schema-type-detail')
    await expect(detail.getByRole('heading', { name: 'e2e.browser.Parcel' })).toBeVisible()
    await expect(detail.getByText('Tracking number printed on the label.')).toBeVisible()
    await expect(detail.getByText('map<string, string>')).toBeVisible()
    await expect(detail.getByTestId('schema-example')).toContainText('"tracking"')

    await detail.getByRole('button', { name: 'e2e.browser.Stage' }).click()
    await expect(detail.getByRole('heading', { name: 'e2e.browser.Stage' })).toBeVisible()
    await expect(detail.getByText('Left the warehouse.')).toBeVisible()

    await browser.getByRole('button', { name: /^service Tracking/ }).click()
    await expect(detail.getByText('Streams parcel updates.')).toBeVisible()
    await expect(detail.getByText('stream', { exact: true })).toBeVisible()
  })

  test('opens an imported type from a field', async ({ page, env }) => {
    void env
    await page.goto('/settings/proto')
    const browser = page.getByTestId('schema-browser')
    await browser.getByRole('textbox', { name: 'Search types or comments…' }).fill('e2e.browser.Parcel')
    await browser.getByRole('button', { name: /^message Parcel/ }).click()

    const detail = page.getByTestId('schema-type-detail')
    await detail.getByRole('button', { name: 'google.protobuf.Timestamp' }).click()
    await expect(detail.getByRole('heading', { name: 'google.protobuf.Timestamp' })).toBeVisible()
    await expect(detail).toContainText('imported')
    await expect(page.getByTestId('schema-show-imported')).toBeChecked()
  })
})
