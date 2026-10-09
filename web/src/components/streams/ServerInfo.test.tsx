import { describe, expect, it, vi } from 'vitest'
import { render, screen } from '@/test/utils'
import ServerInfo from './ServerInfo'

vi.mock('@/api/stats', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/api/stats')>()),
  getServerInfo: () => new Promise(() => {}),
}))

vi.mock('@/contexts/connection', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/contexts/connection')>()),
  useConnectionHealth: () => ({ status: 'connected', rtt: undefined }),
}))

describe('ServerInfo while loading', () => {
  it('can already be closed', () => {
    render(<ServerInfo connectionId="conn-1" onClose={vi.fn()} />)

    expect(screen.getByRole('button', { name: 'Close server info' })).toBeInTheDocument()
  })
})
