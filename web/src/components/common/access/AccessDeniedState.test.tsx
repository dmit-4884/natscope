import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { AccessDeniedState } from './AccessDeniedState'

describe('AccessDeniedState', () => {
  it('names the missing permission and offers a retry', () => {
    const onRetry = vi.fn()
    render(
      <AccessDeniedState
        check={{ status: 'denied', operation: 'publish', subject: '$SRV.INFO' }}
        title="No access to services"
        description="This NATS user may not ask services about themselves."
        onRetry={onRetry}
      />,
    )

    expect(screen.getByRole('heading', { name: 'No access to services' })).toBeInTheDocument()
    expect(screen.getByText('publish to $SRV.INFO')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Check again' }))
    expect(onRetry).toHaveBeenCalled()
  })
})
