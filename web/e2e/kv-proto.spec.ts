import { test, expect, PROTO_MESSAGE_TYPE } from './fixtures'
import {
  createKVBucket,
  createMapping,
  deleteKVBucket,
  deleteMappingsByPrefix,
  putKVProto,
  resetSettings,
  restoreSettings,
  snapshotSettings,
} from './api'

const RUN = Date.now()
const MAPPED = `e2e_kvp_mapped_${RUN}`
const DETECTED = `e2e_kvp_auto_${RUN}`

let snapshot: Awaited<ReturnType<typeof snapshotSettings>>
test.beforeAll(async () => {
  snapshot = await snapshotSettings()
  await resetSettings()
})
test.afterAll(async () => {
  await deleteMappingsByPrefix('$KV.e2e_kvp_')
  await restoreSettings(snapshot)
})

test.describe('KV Protobuf values', () => {
  test('a mapped bucket stores and edits JSON as Protobuf', async ({ page, env }) => {
    await createKVBucket(env.connectionId, MAPPED)
    await createMapping(`$KV.${MAPPED}.>`, PROTO_MESSAGE_TYPE, env.protoSourceId)
    try {
      await page.goto(`/kv/${MAPPED}`)
      await page.getByRole('button', { name: 'Create New Key' }).click()
      await page.getByPlaceholder('my.key.name').fill('limits')
      await expect(page.getByTestId('kv-new-proto')).toContainText(PROTO_MESSAGE_TYPE)
      await page.getByPlaceholder(`JSON for ${PROTO_MESSAGE_TYPE}`).fill('{"name":"cfg","count":4}')
      await page.getByRole('button', { name: 'Create Key' }).click()

      await page.getByText('limits', { exact: true }).click()
      const bar = page.getByTestId('kv-proto')
      await expect(bar).toContainText(PROTO_MESSAGE_TYPE)
      await expect(bar).toContainText(`via $KV.${MAPPED}.>`)
      const value = page.getByLabel('Key value')
      await expect(value).toHaveValue(/"name": "cfg"/)

      await value.fill('{"name":"cfg","count":5}')
      await page.getByRole('button', { name: 'Save Value' }).click()
      await expect(page.getByText('Rev 2')).toBeVisible()
      await expect(value).toHaveValue(/"count": 5/)

      await page.getByTestId('kv-raw-toggle').click()
      await expect(page.getByTestId('wire-view')).toBeVisible()
      await expect(page.getByRole('button', { name: 'Save Value' })).toBeDisabled()
      await page.getByTestId('kv-raw-toggle').click()

      await page.getByRole('button', { name: 'History' }).click()
      const history = page.getByRole('dialog', { name: 'History: limits' })
      await expect(history).toContainText('"count": 4')
      await expect(history).toContainText('"count": 5')
    } finally {
      await deleteKVBucket(env.connectionId, MAPPED)
    }
  })

  test('an unmapped bucket shows the detected type and saves it as a mapping', async ({ page, env }) => {
    await createKVBucket(env.connectionId, DETECTED)
    await putKVProto(env.connectionId, DETECTED, 'profile', {
      messageType: PROTO_MESSAGE_TYPE,
      sourceId: env.protoSourceId,
      json: '{"name":"guess","count":2,"tags":["a"]}',
    })
    try {
      await page.goto(`/kv/${DETECTED}`)
      await page.getByText('profile', { exact: true }).click()
      await expect(page.getByTestId('kv-decoded-auto')).toBeVisible()
      await expect(page.getByLabel('Key value')).toHaveValue(/"name": "guess"/)

      await page.getByTestId('kv-save-mapping').click()
      await expect(
        page.locator('[data-sonner-toast]').getByText(`Mapped $KV.${DETECTED}.> to ${PROTO_MESSAGE_TYPE}`),
      ).toBeVisible()
      await expect(page.getByTestId('kv-proto')).toContainText(`via $KV.${DETECTED}.>`)
      await expect(page.getByTestId('kv-decoded-auto')).toBeHidden()
    } finally {
      await deleteKVBucket(env.connectionId, DETECTED)
    }
  })
})
