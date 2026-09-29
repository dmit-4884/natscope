import { useState } from 'react'
import { describe, it, expect, vi, beforeAll, afterAll } from 'vitest'
import { render, screen, fireEvent } from '@/test/utils'
import type { ConsumerCreateRequest } from '@/types/management'
import { ConsumerEditor } from './ConsumerEditor'
import { defaultConsumerConfig } from './consumerUtils'

function Harness({ initial }: { initial: ConsumerCreateRequest }) {
  const [value, setValue] = useState(initial)
  const [mode, setMode] = useState<'form' | 'json'>('form')
  return (
    <ConsumerEditor
      title="Create New Consumer"
      subtitle="Create a consumer"
      mode={mode}
      onModeChange={setMode}
      value={value}
      onChange={setValue}
      isEditMode={false}
      isSaving={false}
      onCancel={vi.fn()}
      onCreate={vi.fn()}
    />
  )
}

function renderEditor(initial: ConsumerCreateRequest = defaultConsumerConfig) {
  const view = render(<Harness initial={initial} />)
  const jsonEditor = view.container.querySelector('textarea')
  if (!jsonEditor) throw new Error('JSON editor textarea not rendered')
  return { ...view, jsonEditor }
}

describe('ConsumerEditor ephemeral toggle', () => {
  it('starts durable so existing behaviour is preserved', () => {
    const { jsonEditor } = renderEditor()
    expect(screen.getByTestId('consumer-ephemeral-toggle')).not.toBeChecked()
    expect(JSON.parse(jsonEditor.value).ephemeral).toBe(false)
  })

  it('mirrors the toggle into the raw JSON view', () => {
    const { jsonEditor } = renderEditor()
    fireEvent.click(screen.getByTestId('consumer-ephemeral-toggle'))
    expect(JSON.parse(jsonEditor.value).ephemeral).toBe(true)
  })

  it('mirrors an ephemeral key typed into the raw JSON view back into the toggle', () => {
    const { jsonEditor } = renderEditor()
    fireEvent.input(jsonEditor, {
      target: { value: JSON.stringify({ ...defaultConsumerConfig, ephemeral: true }) },
    })
    expect(screen.getByTestId('consumer-ephemeral-toggle')).toBeChecked()
  })

  it('explains what an ephemeral consumer costs the user', () => {
    renderEditor()
    fireEvent.click(screen.getByTestId('consumer-ephemeral-toggle'))
    expect(screen.getByText(/removes this consumer/i)).toBeInTheDocument()
  })
})

describe('ConsumerEditor filter subjects', () => {
  it('lets the single filter be edited when no list is in use', () => {
    const { jsonEditor } = renderEditor({ ...defaultConsumerConfig, filter_subject: 'orders.>' })
    const input = screen.getByLabelText('Filter Subject')
    expect(input).not.toBeDisabled()
    fireEvent.change(input, { target: { value: 'orders.paid' } })
    expect(JSON.parse(jsonEditor.value).filter_subject).toBe('orders.paid')
  })

  it('disables the single filter while the multi-subject list is in use', () => {
    renderEditor({ ...defaultConsumerConfig, filter_subjects: ['orders.created'] })
    expect(screen.getByLabelText('Filter Subject')).toBeDisabled()
  })

  it('clears the single filter as soon as a list entry is added', () => {
    const { jsonEditor } = renderEditor({ ...defaultConsumerConfig, filter_subject: 'orders.>' })
    fireEvent.click(screen.getByRole('button', { name: 'Add' }))
    const next = JSON.parse(jsonEditor.value)
    expect(next.filter_subject).toBe('')
    expect(next.filter_subjects).toEqual([''])
    expect(screen.getByLabelText('Filter Subject')).toBeDisabled()
  })

  it('re-enables the single filter once the list is emptied', () => {
    renderEditor({ ...defaultConsumerConfig, filter_subjects: ['orders.created'] })
    fireEvent.click(screen.getByRole('button', { name: 'Remove' }))
    expect(screen.getByLabelText('Filter Subject')).not.toBeDisabled()
  })

  it('allows explicitly clearing the single filter', () => {
    const { jsonEditor } = renderEditor({ ...defaultConsumerConfig, filter_subject: 'orders.>' })
    fireEvent.change(screen.getByLabelText('Filter Subject'), { target: { value: '' } })
    expect(JSON.parse(jsonEditor.value).filter_subject).toBe('')
  })
})

function PriorityHarness({
  initial = defaultConsumerConfig,
  priorityUnsupportedReason,
  prioritizedUnsupportedReason,
}: {
  initial?: ConsumerCreateRequest
  priorityUnsupportedReason?: string
  prioritizedUnsupportedReason?: string
}) {
  const [value, setValue] = useState(initial)
  return (
    <>
      <ConsumerEditor
        title="Create New Consumer"
        subtitle="Create a consumer"
        mode="form"
        onModeChange={vi.fn()}
        value={value}
        onChange={setValue}
        isEditMode={false}
        isSaving={false}
        onCancel={vi.fn()}
        onCreate={vi.fn()}
        priorityUnsupportedReason={priorityUnsupportedReason}
        prioritizedUnsupportedReason={prioritizedUnsupportedReason}
      />
      <output data-testid="value">{JSON.stringify(value)}</output>
    </>
  )
}

function currentValue(): ConsumerCreateRequest {
  return JSON.parse(screen.getByTestId('value').textContent ?? '{}')
}

function openPrioritySection() {
  fireEvent.click(screen.getByRole('button', { name: /Priority Groups/ }))
}

function choosePolicy(label: string) {
  fireEvent.click(screen.getByLabelText('Priority Policy'))
  fireEvent.click(screen.getByRole('option', { name: label }))
}

describe('ConsumerEditor priority groups', () => {
  const originalScrollIntoView = Element.prototype.scrollIntoView
  beforeAll(() => {
    Element.prototype.scrollIntoView = vi.fn()
  })
  afterAll(() => {
    Element.prototype.scrollIntoView = originalScrollIntoView
  })

  it('reveals groups and pinned ttl for a pinned client policy', () => {
    render(<PriorityHarness />)
    openPrioritySection()

    choosePolicy('Pinned client')

    expect(currentValue().priority_policy).toBe('pinned_client')
    expect(screen.getByText('Groups')).toBeInTheDocument()
    fireEvent.change(screen.getByLabelText('Pinned TTL (seconds)'), { target: { value: '90' } })
    expect(currentValue().priority_timeout).toBe(90_000_000_000)
  })

  it('clears groups and ttl when the policy is switched off', () => {
    render(
      <PriorityHarness
        initial={{ ...defaultConsumerConfig, priority_policy: 'pinned_client', priority_groups: ['jobs'], priority_timeout: 60_000_000_000 }}
      />,
    )
    openPrioritySection()

    choosePolicy('None')

    expect(currentValue().priority_policy).toBe('none')
    expect(currentValue().priority_groups).toBeUndefined()
    expect(currentValue().priority_timeout).toBeUndefined()
  })

  it('disables the prioritized policy on servers before 2.12', () => {
    render(<PriorityHarness prioritizedUnsupportedReason="Requires NATS 2.12+" />)
    openPrioritySection()

    fireEvent.click(screen.getByLabelText('Priority Policy'))

    expect(screen.getByRole('option', { name: 'Prioritized' })).toHaveAttribute('aria-disabled', 'true')
  })

  it('locks the section on servers without priority groups', () => {
    render(<PriorityHarness priorityUnsupportedReason="Requires NATS 2.11+ (connected server is v2.10.24)" />)
    openPrioritySection()

    expect(screen.getByText('Requires NATS 2.11+ (connected server is v2.10.24)')).toBeInTheDocument()
    expect(screen.getByLabelText('Priority Policy')).toBeDisabled()
  })

  it('locks the section for push consumers', () => {
    render(<PriorityHarness initial={{ ...defaultConsumerConfig, deliver_subject: 'push.here' }} />)
    openPrioritySection()

    expect(screen.getByText('Priority groups apply to pull consumers only.')).toBeInTheDocument()
    expect(screen.getByLabelText('Priority Policy')).toBeDisabled()
  })
})
