import { test, expect, STREAM } from './fixtures'

test('the stream tabs keep their place and their set when switching tabs', async ({ page, env: _env }) => {
  await page.goto(`/streams/${STREAM}/messages`)
  const tabs = page.getByTestId('stream-tabs')
  await expect(tabs.getByRole('link', { name: 'Messages' })).toBeVisible()
  const shown = async () => ({ box: await tabs.boundingBox(), links: await tabs.getByRole('link').allTextContents() })
  const onMessages = await shown()

  await tabs.getByRole('link', { name: 'Consumers' }).click()
  await expect(page).toHaveURL(/\/consumers$/)

  expect(await shown()).toEqual(onMessages)
})

test('the relations tab opens on what the stream already loaded, without a loading state', async ({ page, env: _env }) => {
  await page.goto(`/streams/${STREAM}/messages`)
  const tabs = page.getByTestId('stream-tabs')
  await expect(tabs.getByRole('link', { name: 'Messages' })).toBeVisible()
  await page.waitForTimeout(1_500)
  await page.evaluate(() => {
    const w = window as unknown as { sawLoading: boolean }
    w.sawLoading = false
    new MutationObserver(() => {
      if (document.querySelector('[aria-busy="true"]')) w.sawLoading = true
    }).observe(document.body, { subtree: true, childList: true, attributes: true })
  })

  await tabs.getByRole('link', { name: 'Relations' }).click()
  await expect(page).toHaveURL(/\/relations$/)
  await page.waitForTimeout(500)

  expect(await page.evaluate(() => (window as unknown as { sawLoading: boolean }).sawLoading)).toBe(false)
})

test('a selected message names its consumers without a checking state', async ({ page, env: _env }) => {
  await page.goto(`/streams/${STREAM}/messages`)
  await expect(page.getByRole('row').nth(1)).toBeVisible()
  await page.waitForTimeout(1_500)
  await page.evaluate(() => {
    const w = window as unknown as { sawChecking: boolean }
    w.sawChecking = false
    new MutationObserver(() => {
      if (document.body.textContent?.includes('checking…')) w.sawChecking = true
    }).observe(document.body, { subtree: true, childList: true, characterData: true })
  })

  await page.getByRole('row').nth(1).click()
  await expect(page.getByRole('complementary', { name: 'Details panel' })).toContainText('Consumers:')
  await page.waitForTimeout(500)

  expect(await page.evaluate(() => (window as unknown as { sawChecking: boolean }).sawChecking)).toBe(false)
})
