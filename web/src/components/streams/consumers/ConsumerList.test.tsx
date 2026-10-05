import { describe, it, expect, vi } from 'vitest'
import { MemoryRouter } from 'react-router-dom'
import { render, screen, fireEvent } from '@/test/utils'
import type { ConsumerInfo } from '@/types/nats'
import { ConsumerList } from './ConsumerList'

const billing: ConsumerInfo = {
  name: 'billing',
  stream_name: 'ORDERS',
  num_pending: 3,
  num_ack_pending: 0,
  config: { ack_policy: 'explicit', durable_name: 'billing' },
}

function renderList(overrides: Partial<Parameters<typeof ConsumerList>[0]> = {}) {
  const onSelect = vi.fn()
  render(
    <MemoryRouter>
      <ConsumerList
        streamName="ORDERS"
        consumers={[billing]}
        selectedName={null}
        searchQuery=""
        onSearchChange={vi.fn()}
        isLoading={false}
        isFetching={false}
        onRefetch={vi.fn()}
        onSelect={onSelect}
        {...overrides}
      />
    </MemoryRouter>,
  )
  return onSelect
}

describe('ConsumerList', () => {
  it('links every consumer so it can open in a new tab', () => {
    const onSelect = renderList()

    const link = screen.getByRole('link', { name: /billing/ })
    expect(link).toHaveAttribute('href', '/streams/ORDERS/consumers?consumer=billing')

    expect(fireEvent.click(link, { ctrlKey: true })).toBe(true)
    expect(onSelect).not.toHaveBeenCalled()

    expect(fireEvent.click(link)).toBe(false)
    expect(onSelect).toHaveBeenCalledWith(billing)
  })

  it('does not count consumers before they load', () => {
    renderList({ consumers: [], isLoading: true })

    expect(screen.queryByText('0 consumers')).not.toBeInTheDocument()
  })
})
