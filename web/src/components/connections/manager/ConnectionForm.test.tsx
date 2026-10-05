import { describe, it, expect, vi } from 'vitest'
import { useState } from 'react'
import { render, screen, fireEvent } from '@testing-library/react'
import { ConnectionForm } from './ConnectionForm'
import { emptyTls, noSecretsSet, type ConnectionFormData } from './connectionFormData'

const blank: ConnectionFormData = {
  name: 'prod',
  description: '',
  urls: ['nats://p:4222'],
  authMethod: 'none',
  username: '',
  password: '',
  token: '',
  nkeySeed: '',
  credentials: '',
  tls: { ...emptyTls },
  inboxPrefix: '',
  readOnly: false,
  labelText: '',
  labelColor: 'gray',
  secretsSet: { ...noSecretsSet },
}

function Harness({ onChange }: { onChange: (v: ConnectionFormData) => void }) {
  const [value, setValue] = useState(blank)
  return (
    <ConnectionForm
      value={value}
      onChange={(v) => {
        setValue(v)
        onChange(v)
      }}
      onCancel={vi.fn()}
      onTest={vi.fn()}
      isTesting={false}
    />
  )
}

describe('ConnectionForm environment', () => {
  it('sets a colored label and the read-only switch', () => {
    const onChange = vi.fn()
    render(<Harness onChange={onChange} />)

    fireEvent.change(screen.getByLabelText('Label'), { target: { value: 'PROD' } })
    fireEvent.click(screen.getByRole('radio', { name: 'Red' }))
    fireEvent.click(screen.getByRole('switch', { name: 'Read-only' }))

    expect(onChange).toHaveBeenLastCalledWith(expect.objectContaining({ labelText: 'PROD', labelColor: 'red', readOnly: true }))
    expect(screen.getByTestId('label-preview')).toHaveTextContent('PROD')
    expect(screen.getByLabelText('Label')).toHaveAttribute('maxLength', '16')
  })
})
