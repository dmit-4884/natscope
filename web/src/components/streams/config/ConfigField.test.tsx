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

describe('ConfigField — durations', () => {
  const maxAgeDef = STREAM_FIELDS.find((f) => f.key === 'max_age')!

  it('shows the stored nanoseconds as a readable duration', () => {
    render(<ConfigField def={maxAgeDef} value={3_600_000_000_000} onChange={vi.fn()} mode="create" />)

    expect(screen.getByLabelText('Max Age')).toHaveValue('1h')
  })

  it('turns typed durations into nanoseconds on the wire', () => {
    const onChange = vi.fn()
    render(<ConfigField def={maxAgeDef} value={0} onChange={onChange} mode="create" />)

    fireEvent.change(screen.getByLabelText('Max Age'), { target: { value: '7d' } })

    expect(onChange).toHaveBeenLastCalledWith(604_800_000_000_000)
  })

  it('flags text it cannot read, keeps the last good value and reports the field as invalid', () => {
    const onChange = vi.fn()
    const onInvalid = vi.fn()
    render(<ConfigField def={maxAgeDef} value={0} onChange={onChange} mode="create" onInvalid={onInvalid} />)

    fireEvent.change(screen.getByLabelText('Max Age'), { target: { value: '3600000000000' } })

    expect(screen.getByText(/like 30s, 5m, 12h or 7d/)).toBeInTheDocument()
    expect(onChange).not.toHaveBeenCalled()
    expect(onInvalid).toHaveBeenLastCalledWith(true)
  })
})

describe('ConfigField — name and subjects', () => {
  const nameDef = STREAM_FIELDS.find((f) => f.key === 'name')!
  const subjectsDef = STREAM_FIELDS.find((f) => f.key === 'subjects')!

  it('shows an inline error for a stream name NATS would refuse', () => {
    render(<ConfigField def={nameDef} value="A3 bad name" onChange={vi.fn()} mode="create" />)

    expect(screen.getByText(/cannot contain spaces/)).toBeInTheDocument()
  })

  it('shows no error for a good name', () => {
    render(<ConfigField def={nameDef} value="ORDERS" onChange={vi.fn()} mode="create" />)

    expect(screen.queryByText(/cannot contain spaces/)).toBeNull()
  })

  it('shows an inline error under the subject with an empty part', () => {
    render(<ConfigField def={subjectsDef} value={['ok.>', 'a3..x']} onChange={vi.fn()} mode="create" />)

    expect(screen.getByText(/empty parts/)).toBeInTheDocument()
  })
})
