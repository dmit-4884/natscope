import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@/test/utils'
import { CorePublishDialog } from './CorePublishDialog'

const publishCoreMessage = vi.hoisted(() => vi.fn())

vi.mock('@/api/publish', () => ({ publishCoreMessage }))

vi.mock('@/utils/toast', () => ({
  toast: { success: vi.fn(), error: vi.fn(), info: vi.fn(), warning: vi.fn() },
}))

vi.mock('@/contexts/mappings', () => ({
  useSubjectMappingEntity: () => ({ messageType: null, sourceId: null, framing: undefined, pinnedFingerprint: undefined }),
}))

vi.mock('@/components/common/TemplateJsonEditor', () => ({
  default: ({ value, onChange, title }: { value: string; onChange: (v: string) => void; title?: string }) => (
    <textarea aria-label={title} value={value} onChange={(e) => onChange(e.target.value)} />
  ),
}))

describe('CorePublishDialog', () => {
  beforeEach(() => {
    publishCoreMessage.mockReset()
  })

  it('resends the message over core NATS with its headers', async () => {
    publishCoreMessage.mockResolvedValue(undefined)
    const onClose = vi.fn()
    render(
      <CorePublishDialog
        connectionId="conn-1"
        mode="resend"
        initial={{ subject: 'orders.new', payload: '{"id":1}', headers: [{ key: 'X-Trace', value: 't' }] }}
        onClose={onClose}
      />,
    )

    expect(screen.getByRole('dialog', { name: 'Resend message' })).toBeInTheDocument()
    fireEvent.change(screen.getByLabelText('Payload'), { target: { value: '{"id":2}' } })
    fireEvent.click(screen.getByRole('button', { name: 'Send' }))

    await waitFor(() => expect(onClose).toHaveBeenCalled())
    expect(publishCoreMessage.mock.calls[0][0]).toEqual(
      expect.objectContaining({ connection_id: 'conn-1', subject: 'orders.new', data: '{"id":2}', headers: { 'X-Trace': 't' } }),
    )
  })

  it('keeps the reply subject fixed and shows a failure inline', async () => {
    publishCoreMessage.mockRejectedValue(new Error('Failed to publish message: no permission to publish to "_INBOX.x"'))
    const onClose = vi.fn()
    render(
      <CorePublishDialog connectionId="conn-1" mode="reply" initial={{ subject: '_INBOX.x', payload: '', headers: [] }} onClose={onClose} />,
    )

    expect(screen.getByRole('dialog', { name: 'Reply to request' })).toBeInTheDocument()
    expect(screen.getByLabelText('Reply subject')).toHaveAttribute('readonly')
    fireEvent.click(screen.getByRole('button', { name: 'Send reply' }))

    expect(await screen.findByText(/no permission to publish to "_INBOX.x"/)).toBeInTheDocument()
    expect(onClose).not.toHaveBeenCalled()
  })

  it('blocks a wildcard subject with a reason', () => {
    render(
      <CorePublishDialog connectionId="conn-1" mode="resend" initial={{ subject: 'orders.*', payload: '', headers: [] }} onClose={vi.fn()} />,
    )

    expect(screen.getByRole('button', { name: 'Send' })).toBeDisabled()
    expect(screen.getByText('Publish to a literal subject, without * or > wildcards')).toBeInTheDocument()
  })
})
