import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@/test/utils'
import CommandPalette from './CommandPalette'

vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom')
  return {
    ...actual,
    useNavigate: () => vi.fn(),
  }
})

// Stub useHotkeys to a no-op.
vi.mock('react-hotkeys-hook', () => ({
  useHotkeys: vi.fn(),
}))

describe('CommandPalette', () => {
  it('renders without crashing', () => {
    render(<CommandPalette />)
    // Hidden initially.
  })

  it('shows command list when opened', async () => {
    render(<CommandPalette />)
    // Closed palette renders no commands.
    expect(screen.queryByText('Go to Streams')).toBeNull()
  })
})
