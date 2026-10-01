import { describe, it, expect, vi } from 'vitest'
import { useState } from 'react'
import { render, screen, fireEvent } from '@/test/utils'
import { NO_FRAMING, type Framing } from '@/api/framing'
import { FramingFields } from './FramingFields'

function Harness({ onChange, onValidityChange }: { onChange: (f: Framing) => void; onValidityChange: (v: boolean) => void }) {
  const [framing, setFraming] = useState<Framing>(NO_FRAMING)
  return (
    <FramingFields
      value={framing}
      onChange={(f) => {
        setFraming(f)
        onChange(f)
      }}
      onValidityChange={onValidityChange}
    />
  )
}

describe('FramingFields', () => {
  it('edits a Confluent schema id', () => {
    const onChange = vi.fn()
    render(<Harness onChange={onChange} onValidityChange={vi.fn()} />)

    fireEvent.change(screen.getByLabelText('Framing'), { target: { value: 'confluent' } })
    fireEvent.change(screen.getByLabelText('Schema id written when publishing'), { target: { value: '42' } })
    expect(onChange).toHaveBeenLastCalledWith(expect.objectContaining({ kind: 'confluent', schemaId: 42 }))
  })

  it('takes custom bytes as hex and flags bad input', () => {
    const onChange = vi.fn()
    const onValidityChange = vi.fn()
    render(<Harness onChange={onChange} onValidityChange={onValidityChange} />)

    fireEvent.change(screen.getByLabelText('Framing'), { target: { value: 'custom' } })
    fireEvent.change(screen.getByLabelText('Prefix (hex)'), { target: { value: 'caf' } })
    expect(onValidityChange).toHaveBeenLastCalledWith(false)
    expect(screen.getByText('Use pairs of hex digits.')).toBeInTheDocument()

    fireEvent.change(screen.getByLabelText('Prefix (hex)'), { target: { value: 'cafe' } })
    fireEvent.change(screen.getByLabelText('Suffix (hex)'), { target: { value: '0d0a' } })
    expect(onValidityChange).toHaveBeenLastCalledWith(true)
    expect(onChange).toHaveBeenLastCalledWith(
      expect.objectContaining({ kind: 'custom', prefix: new Uint8Array([0xca, 0xfe]), suffix: new Uint8Array([0x0d, 0x0a]) }),
    )
  })
})
