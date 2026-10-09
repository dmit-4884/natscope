import { describe, expect, it, vi } from 'vitest'
import { MemoryRouter } from 'react-router-dom'
import { render, screen } from '@/test/utils'
import KVOverviewPage from './KVOverviewPage'

vi.mock('react-router-dom', async (importOriginal) => ({
  ...(await importOriginal<typeof import('react-router-dom')>()),
  useOutletContext: () => ({ connectionId: 'conn-1' }),
}))

vi.mock('@/contexts/connection', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/contexts/connection')>()),
  useConnectionPolicy: () => ({ readOnly: false, label: null, known: true }),
}))

vi.mock('@/contexts/kv', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/contexts/kv')>()),
  useKVBuckets: () => ({ data: undefined, isLoading: true, error: null, refetch: vi.fn() }),
}))

describe('KVOverviewPage', () => {
  it('shows no bucket count before the buckets load', () => {
    render(<MemoryRouter><KVOverviewPage /></MemoryRouter>)

    expect(screen.queryByText('0 buckets')).not.toBeInTheDocument()
  })
})
