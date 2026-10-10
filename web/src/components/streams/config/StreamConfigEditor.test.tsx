import { describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'
import type { StreamCreateRequest } from '@/types/management'

vi.mock('@/contexts/connection', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/contexts/connection')>()),
  useActiveConnection: () => ({ connectionId: 'c1' }),
  useServerCapabilities: () => ({ unsupportedReason: () => undefined }),
}))

import { StreamConfigEditor } from './StreamConfigEditor'

const value: StreamCreateRequest = { name: 'ORDERS', subjects: ['orders.>'], max_age: 0 }

describe('StreamConfigEditor', () => {
  it('blocks saving, and says why, while a duration cannot be read', () => {
    render(
      <StreamConfigEditor
        editorMode="form"
        onModeChange={vi.fn()}
        value={value}
        onChange={vi.fn()}
        originalValue={value}
        isSaving={false}
        hasChanges
        onCancel={vi.fn()}
        onSave={vi.fn()}
      />,
    )
    expect(screen.getByRole('button', { name: 'Save Changes' })).toBeEnabled()

    fireEvent.click(screen.getByRole('button', { name: /^limits$/i }))
    fireEvent.change(screen.getByLabelText('Max Age'), { target: { value: 'soon' } })

    expect(screen.getByRole('button', { name: 'Save Changes' })).toBeDisabled()
    expect(screen.getByTestId('save-blocked-reason')).toHaveTextContent('Max Age')
  })
})
