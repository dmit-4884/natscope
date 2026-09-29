import { useState, type ComponentProps, type ReactElement } from 'react'
import { describe, it, expect, vi, beforeAll, afterAll } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { JetStreamOptions } from './JetStreamOptions'
import { EMPTY_PUBLISH_OPTIONS, type PublishOptions } from './publishOptions'

type Reasons = Omit<ComponentProps<typeof JetStreamOptions>, 'value' | 'onChange'>

function Harness({ initial = EMPTY_PUBLISH_OPTIONS, ...reasons }: { initial?: PublishOptions } & Reasons) {
  const [value, setValue] = useState(initial)
  return (
    <>
      <JetStreamOptions value={value} onChange={setValue} {...reasons} />
      <output data-testid="value">{JSON.stringify(value)}</output>
    </>
  )
}

function current(): PublishOptions {
  return JSON.parse(screen.getByTestId('value').textContent ?? '{}')
}

function toggleButton() {
  return screen.getByRole('button', { name: /JetStream options/ })
}

function renderExpanded(ui: ReactElement) {
  render(ui)
  fireEvent.click(toggleButton())
}

describe('JetStreamOptions', () => {
  const originalScrollIntoView = Element.prototype.scrollIntoView
  beforeAll(() => {
    Element.prototype.scrollIntoView = vi.fn()
  })
  afterAll(() => {
    Element.prototype.scrollIntoView = originalScrollIntoView
  })

  it('starts collapsed', () => {
    render(<Harness />)

    expect(toggleButton()).toHaveAttribute('aria-expanded', 'false')
    expect(screen.queryByLabelText('Message TTL')).not.toBeInTheDocument()
    expect(screen.queryByText('Active')).not.toBeInTheDocument()
  })

  it('flags options that are in use while collapsed', () => {
    render(<Harness initial={{ ...EMPTY_PUBLISH_OPTIONS, ttl: '5m' }} />)

    expect(toggleButton()).toHaveAttribute('aria-expanded', 'false')
    expect(screen.getByText('Active')).toBeInTheDocument()
  })

  it('edits the ttl and the counter increment', () => {
    renderExpanded(<Harness />)

    fireEvent.change(screen.getByLabelText('Message TTL'), { target: { value: '5m' } })
    fireEvent.change(screen.getByLabelText('Counter increment'), { target: { value: '+2' } })

    expect(current().ttl).toBe('5m')
    expect(current().increment).toBe('+2')
  })

  it('explains why an option is unavailable and locks it', () => {
    renderExpanded(
      <Harness
        ttlUnavailable='Enable "Allow Per-Message TTL" on this stream first'
        incrementUnavailable="Requires NATS 2.12+ (connected server is v2.11.3)"
      />,
    )

    expect(screen.getByLabelText('Message TTL')).toBeDisabled()
    expect(screen.getByText('Enable "Allow Per-Message TTL" on this stream first')).toBeInTheDocument()
    expect(screen.getByLabelText('Counter increment')).toBeDisabled()
    expect(screen.getByText('Requires NATS 2.12+ (connected server is v2.11.3)')).toBeInTheDocument()
  })

  it('reveals the schedule fields once scheduling is switched on', () => {
    renderExpanded(<Harness />)

    expect(screen.queryByLabelText('Target subject')).not.toBeInTheDocument()
    fireEvent.click(screen.getByRole('switch', { name: 'Schedule this message' }))

    expect(current().schedule.enabled).toBe(true)
    fireEvent.change(screen.getByLabelText('Target subject'), { target: { value: 'orders.run' } })
    expect(current().schedule.target).toBe('orders.run')
    expect(screen.getByLabelText('Publish at')).toBeInTheDocument()
  })

  it('shows the cron fields for a cron schedule', () => {
    renderExpanded(<Harness initial={{ ...EMPTY_PUBLISH_OPTIONS, schedule: { ...EMPTY_PUBLISH_OPTIONS.schedule, enabled: true, kind: 'cron' } }} />)

    fireEvent.change(screen.getByLabelText('Cron expression'), { target: { value: '0 0 9 * * *' } })
    fireEvent.change(screen.getByLabelText('Time zone'), { target: { value: 'Europe/Amsterdam' } })

    expect(current().schedule.cron).toBe('0 0 9 * * *')
    expect(current().schedule.timeZone).toBe('Europe/Amsterdam')
  })

  it('disables repeating schedules on servers before 2.14', () => {
    renderExpanded(
      <Harness
        initial={{ ...EMPTY_PUBLISH_OPTIONS, schedule: { ...EMPTY_PUBLISH_OPTIONS.schedule, enabled: true } }}
        cronUnavailable="Requires NATS 2.14+ (connected server is v2.12.3)"
      />,
    )

    fireEvent.click(screen.getByLabelText('Schedule type'))

    expect(screen.getByRole('option', { name: 'Every interval' })).toHaveAttribute('aria-disabled', 'true')
    expect(screen.getByRole('option', { name: 'Cron' })).toHaveAttribute('aria-disabled', 'true')
    expect(screen.getByRole('option', { name: 'Once at a time' })).not.toHaveAttribute('aria-disabled', 'true')
  })

  it('locks the schedule switch when schedules are unavailable', () => {
    renderExpanded(<Harness scheduleUnavailable='Enable "Allow Message Schedules" on this stream first' />)

    expect(screen.getByRole('switch', { name: 'Schedule this message' })).toBeDisabled()
    expect(screen.getByText('Enable "Allow Message Schedules" on this stream first')).toBeInTheDocument()
  })
})
