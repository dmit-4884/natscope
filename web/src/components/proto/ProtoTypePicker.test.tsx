import { describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { ProtoTypePicker } from './ProtoTypePicker'

vi.mock('@/contexts/proto', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/contexts/proto')>()),
  useMessageTypes: () => ({ messages: [], isLoading: true }),
}))

describe('ProtoTypePicker', () => {
  it('keeps its loading note out of sight for a quick answer', () => {
    render(<ProtoTypePicker sourceId="src-1" value="" onChange={vi.fn()} />)
    expect(screen.getByText('Loading types…')).toHaveClass('reveal-after-delay')
  })
})
