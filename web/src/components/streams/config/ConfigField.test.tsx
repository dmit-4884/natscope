import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { INT32_MAX, INT32_MIN } from '@/utils/numbers'
import { ConfigField } from './ConfigField'
import { STREAM_FIELDS } from './streamFieldDefinitions'

const maxMsgSizeDef = STREAM_FIELDS.find((f) => f.key === 'max_msg_size')!

describe('ConfigField — int32 fields', () => {
  it('clamps a value above the int32 range', () => {
    const onChange = vi.fn()
    render(<ConfigField def={maxMsgSizeDef} value={-1} onChange={onChange} mode="create" />)

    fireEvent.change(screen.getByLabelText('Max Message Size'), { target: { value: '3000000000' } })

    expect(onChange).toHaveBeenCalledWith(INT32_MAX)
  })

  it('clamps a value below the int32 range', () => {
    const onChange = vi.fn()
    render(<ConfigField def={maxMsgSizeDef} value={-1} onChange={onChange} mode="create" />)

    fireEvent.change(screen.getByLabelText('Max Message Size'), { target: { value: '-3000000000' } })

    expect(onChange).toHaveBeenCalledWith(INT32_MIN)
  })

  it('passes an in-range value through unchanged', () => {
    const onChange = vi.fn()
    render(<ConfigField def={maxMsgSizeDef} value={-1} onChange={onChange} mode="create" />)

    fireEvent.change(screen.getByLabelText('Max Message Size'), { target: { value: '65536' } })

    expect(onChange).toHaveBeenCalledWith(65536)
  })
})
