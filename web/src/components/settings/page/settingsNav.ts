/**
 * Single source of truth for the "where to return to when Settings closes"
 * sessionStorage key. Both writers (any place that navigates *into* Settings)
 * and the reader (SettingsLayout's close button) use this module so the
 * round-trip stays in sync.
 *
 * Without this, deep-link navigations like "Manage all in Settings →" from
 * the publish template popover skipped writing the key, and the close button
 * fell back to /streams — losing the user's stream + tab context.
 */
const SETTINGS_RETURN_KEY = 'nats:settings:returnTo'

/**
 * Remember the current location so SettingsLayout's close button can return
 * the user there. No-op when the path is already inside /settings (we don't
 * want close to bounce between settings tabs).
 */
export function rememberSettingsReturn(pathWithSearch: string): void {
  if (pathWithSearch.startsWith('/settings')) return
  sessionStorage.setItem(SETTINGS_RETURN_KEY, pathWithSearch)
}

export function readSettingsReturn(): string | null {
  return sessionStorage.getItem(SETTINGS_RETURN_KEY)
}

export function clearSettingsReturn(): void {
  sessionStorage.removeItem(SETTINGS_RETURN_KEY)
}

export const SETTINGS_LAST_TAB_KEY = 'nats:settings:lastTab'

const SETTINGS_TABS = [
  'connections',
  'proto',
  'mappings',
  'templates',
  'preferences',
  'workspace',
] as const

export type SettingsTab = (typeof SETTINGS_TABS)[number]

export function isSettingsTab(value: unknown): value is SettingsTab {
  return SETTINGS_TABS.includes(value as SettingsTab)
}
