import { create, toBinary } from '@bufbuild/protobuf'
import {
  FieldDescriptorProto_Label,
  FieldDescriptorProto_Type,
  FileDescriptorSetSchema,
} from '@bufbuild/protobuf/wkt'
import { test, expect } from './fixtures'
import { call } from './api'

const SOURCES = 'natscope.proto.sources.v1.SourcesService'
const NAME = 'e2e-upload'

const ITEM = `syntax = "proto3";
package e2eupload;
// One parcel line.
message Item { string sku = 1; }
`
const ORDER = `syntax = "proto3";
package e2eupload;
import "item.proto";
message Shipment { repeated Item items = 1; }
`

const parcelSet = toBinary(
  FileDescriptorSetSchema,
  create(FileDescriptorSetSchema, {
    file: [
      {
        name: 'parcel.proto',
        package: 'e2eupload',
        syntax: 'proto3',
        messageType: [
          {
            name: 'Parcel',
            field: [
              {
                name: 'id',
                jsonName: 'id',
                number: 1,
                type: FieldDescriptorProto_Type.STRING,
                label: FieldDescriptorProto_Label.OPTIONAL,
              },
            ],
          },
        ],
      },
    ],
  }),
)

async function deleteByName() {
  const list = await call<{ sources?: Array<{ id: string; name: string }> }>(SOURCES, 'ListSources', { pageSize: 500 })
  for (const s of list.sources ?? []) {
    if (s.name === NAME) await call(SOURCES, 'DeleteSource', { id: s.id })
  }
}

test.describe('Proto source: upload', () => {
  test.beforeAll(deleteByName)
  test.afterAll(deleteByName)

  test('creates a source from files, then swaps in a descriptor set', async ({ page, env }) => {
    void env
    await page.goto('/settings/proto/new')
    await page.getByRole('button', { name: /^Upload/ }).click()
    await page.getByLabel('Name *').fill(NAME)
    await page.getByLabel('Proto files or descriptor set').setInputFiles([
      { name: 'item.proto', mimeType: 'text/plain', buffer: Buffer.from(ITEM) },
      { name: 'order.proto', mimeType: 'text/plain', buffer: Buffer.from(ORDER) },
    ])
    await expect(page.getByTestId('schema-upload-selected')).toHaveText('2 .proto files')
    await page.getByRole('button', { name: 'Add Source' }).click()

    await expect(page).toHaveURL(/\/settings\/proto$/)
    const card = page.getByTestId('proto-source-card').filter({ hasText: NAME })
    await expect(card.getByTestId('upload-revision')).toHaveText(/^[0-9a-f]{12}$/)
    const firstRevision = await card.getByTestId('upload-revision').textContent()

    const browser = page.getByTestId('schema-browser')
    await browser.getByRole('textbox', { name: 'Search types or comments…' }).fill('e2eupload')
    await expect(browser.getByRole('button', { name: /^message Item One parcel line\./ })).toBeVisible()
    await expect(browser.getByRole('button', { name: /^message Shipment/ })).toBeVisible()

    await card.getByTestId('upload-new-version').click()
    await card.getByLabel('Proto files or descriptor set').setInputFiles({
      name: 'schema.binpb',
      mimeType: 'application/octet-stream',
      buffer: Buffer.from(parcelSet),
    })
    await expect(card.getByTestId('upload-ok')).toHaveText('Compiled: 1 file descriptor, 1 message type')
    await expect(card.getByTestId('upload-revision')).not.toHaveText(firstRevision ?? '')
    await expect(browser.getByRole('button', { name: /^message Parcel/ })).toBeVisible()
    await expect(browser.getByRole('button', { name: /^message Shipment/ })).toHaveCount(0)
  })

  test('shows compile errors and keeps the previous schema', async ({ page, env }) => {
    void env
    await page.goto('/settings/proto')
    const card = page.getByTestId('proto-source-card').filter({ hasText: NAME })
    const revision = await card.getByTestId('upload-revision').textContent()

    await card.getByTestId('upload-new-version').click()
    await card.getByLabel('Proto files or descriptor set').setInputFiles({
      name: 'broken.proto',
      mimeType: 'text/plain',
      buffer: Buffer.from('syntax = "proto3";\nmessage Broken { strin id = 1; }\n'),
    })
    await expect(card.getByText(/broken\.proto/).first()).toBeVisible()
    await expect(card.getByTestId('upload-ok')).toHaveCount(0)
    await expect(card.getByTestId('upload-revision')).toHaveText(revision ?? '')
  })
})
