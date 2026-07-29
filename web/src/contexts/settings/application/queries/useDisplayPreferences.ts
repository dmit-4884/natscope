import { DisplayPreferences } from '../../domain/value-objects/DisplayPreferences'
import { useSettings } from './useSettings'

/** Display preferences VO; defaults until settings load. */
export function useDisplayPreferences(): DisplayPreferences {
  const { data } = useSettings()
  return data?.display ?? DisplayPreferences.default()
}
