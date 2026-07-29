export * from './domain'
export * from './application'

// Back-compat alias; remove once callers migrate to UserSettingsUpdate.
export type { UserSettingsUpdate as UpdateSettingsRequest } from './domain/entities/UserSettings'

// Adapters intentionally not exported (internal).
