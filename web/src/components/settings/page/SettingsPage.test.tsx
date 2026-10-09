import { describe, expect, it } from 'vitest'
import { render, screen } from '@/test/utils'
import { SettingsPage } from './SettingsPage'

describe('SettingsPage', () => {
  it('keeps the line for a count that is still loading, without a number in it', () => {
    render(<SettingsPage title="Mappings" meta={null}><p>body</p></SettingsPage>)

    expect(screen.getByTestId('settings-meta')).toHaveTextContent(/^\s*$/)
  })
})
