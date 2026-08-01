import { test, expect } from './fixtures'
import { call } from './api'

// Covers Streams > Consumers: create a durable consumer with ack/delivery/
// replay/filter fields, pause, resume, delete — its own throwaway stream.
test.describe('Stream consumers', () => {
  test('create, pause, resume, delete a durable consumer', async ({ page, env }) => {
    const streamName = `E2E_CONS_${Date.now()}`
    await page.goto('/streams/new')
    await page.getByPlaceholder('my-stream').fill(streamName)
    await page.getByPlaceholder('orders.>').fill(`${streamName.toLowerCase()}.>`)
    await page.getByRole('button', { name: 'Create Stream' }).click()
    await expect(page).toHaveURL(new RegExp(`/streams/${streamName}/messages`))

    await page.goto(`/streams/${streamName}/consumers`)
    await page.getByRole('button', { name: 'Create New Consumer' }).click()

    const durableName = 'e2e-durable-consumer'
    await page.getByLabel('Name', { exact: true }).fill(durableName)
    await page.getByLabel('Filter Subject', { exact: true }).fill(`${streamName.toLowerCase()}.>`)

    await page.getByRole('button', { name: 'Acknowledgement' }).click()
    await page.getByLabel('Ack Policy').click()
    await page.getByRole('option', { name: 'Explicit' }).click()
    await page.getByLabel('Max Deliver').fill('5')

    await page.getByRole('button', { name: 'Delivery Policy' }).click()
    await page.getByLabel('Replay Policy').click()
    await page.getByRole('option', { name: 'Original' }).click()

    await page.getByRole('button', { name: 'Create Consumer' }).click()

    // Creating doesn't auto-select the new consumer — pick it from the list.
    await expect(page.getByText(durableName, { exact: true })).toBeVisible({ timeout: 10_000 })
    await page.getByText(durableName, { exact: true }).click()
    await expect(page.getByRole('heading', { name: durableName })).toBeVisible()

    // ConsumerInfo proto has no `paused` field (only Pause/ResumeConsumerResponse
    // do); verify pause state via the raw nats.go JSON blob (`.raw`) instead.
    const isPaused = async () => {
      const res = await call<{ consumers?: Array<{ name: string; raw?: string }> }>(
        'natscope.nats.management.v1.ManagementService',
        'ListConsumers',
        { connectionId: env.connectionId, streamName },
      )
      const raw = res.consumers?.find((c) => c.name === durableName)?.raw
      return raw ? (JSON.parse(raw).paused ?? false) : false
    }

    await page.getByRole('button', { name: 'Pause', exact: true }).click()
    await page.getByRole('button', { name: 'Pause', exact: true }).last().click()
    await expect.poll(isPaused, { timeout: 10_000 }).toBe(true)

    await page.getByRole('button', { name: 'Resume', exact: true }).click()
    await expect.poll(isPaused, { timeout: 10_000 }).toBe(false)

    await page.getByRole('button', { name: 'Delete', exact: true }).click()
    await page.getByRole('button', { name: 'Delete', exact: true }).last().click()
    await expect(page.getByText('Select a consumer to edit or create a new one')).toBeVisible({ timeout: 10_000 })

    // Cleanup the stream itself.
    await page.goto(`/streams/${streamName}/config`)
    await page.getByRole('button', { name: 'Delete', exact: true }).click()
    await page.getByPlaceholder(streamName).fill(streamName)
    await page.getByRole('button', { name: 'Delete', exact: true }).last().click()
  })
})
