import { beforeEach, describe, expect, it } from 'vitest'
import { fireEvent, render, screen } from '@/test/utils'
import { resetSidebarUi } from '@/stores/sidebarUiStore'
import CollapsibleSection from './CollapsibleSection'

const section = () => (
  <CollapsibleSection title="KV Stores" defaultOpen={false} storageKey="kv">
    <p>buckets</p>
  </CollapsibleSection>
)

describe('CollapsibleSection', () => {
  beforeEach(() => {
    resetSidebarUi()
  })

  it('opens as the user left it when it is shown again', () => {
    const first = render(section())
    fireEvent.click(screen.getByRole('button', { name: 'Expand KV Stores' }))
    first.unmount()

    render(section())

    expect(screen.getByText('buckets')).toBeInTheDocument()
  })

  it('starts from its default the first time', () => {
    render(section())

    expect(screen.queryByText('buckets')).not.toBeInTheDocument()
  })
})
