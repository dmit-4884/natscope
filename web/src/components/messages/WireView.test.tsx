import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@/test/utils'
import type { WireField } from '@/api/decode'
import { WireView } from './WireView'
import { DecodeNotices } from './DecodeNotices'

const hoisted = vi.hoisted(() => ({ decodeWire: vi.fn() }))

vi.mock('@/api/decode', () => hoisted)

const field = (overrides: Partial<WireField>): WireField => ({
  number: 1,
  wireType: 'varint',
  offset: 0,
  length: 2,
  varint: 0n,
  fixed: 0n,
  bytes: new Uint8Array(),
  text: '',
  message: [],
  ...overrides,
})

describe('WireView', () => {
  it('shows every reading of a field and nests parsed messages', async () => {
    hoisted.decodeWire.mockResolvedValue({
      validBytes: 30,
      fields: [
        field({ number: 1, varint: 3n }),
        field({ number: 2, wireType: 'bytes', offset: 2, bytes: new TextEncoder().encode('hello'), text: 'hello' }),
        field({ number: 3, wireType: 'fixed32', offset: 9, fixed: 0x3f800000n }),
        field({
          number: 4,
          wireType: 'bytes',
          offset: 14,
          bytes: new Uint8Array([8, 7]),
          message: [field({ number: 1, offset: 16, varint: 7n })],
        }),
        field({ number: 5, wireType: 'bytes', offset: 18, bytes: new Uint8Array([0xff, 0x00]) }),
      ],
    })

    render(<WireView dataBase64="AAAA" totalBytes={30} />)

    const rows = await screen.findAllByTestId('wire-field')
    expect(rows).toHaveLength(6)
    expect(rows[0]).toHaveTextContent('#1varint3 · sint -2@0')
    expect(rows[1]).toHaveTextContent('"hello" (5 B)')
    expect(rows[2]).toHaveTextContent('1065353216 · int32 1065353216 · float 1')
    expect(rows[3]).toHaveTextContent('(message, 1 field, 2 B)')
    expect(rows[5]).toHaveTextContent('ff 00 (2 B)')

    fireEvent.click(screen.getByRole('button', { name: 'Collapse field 4' }))
    expect(screen.getAllByTestId('wire-field')).toHaveLength(5)
  })

  it('says where reading stopped', async () => {
    hoisted.decodeWire.mockResolvedValue({ validBytes: 3, error: 'field 3 at byte 3: unexpected EOF', fields: [field({ varint: 150n })] })
    render(<WireView dataBase64="AAAA" totalBytes={5} />)
    expect(await screen.findByText('Stopped after 3 B of 5 B')).toBeInTheDocument()
    expect(screen.getByText('field 3 at byte 3: unexpected EOF')).toBeInTheDocument()
  })
})

describe('DecodeNotices', () => {
  it('explains a partial decode and fields missing from the schema', () => {
    render(
      <>
        <DecodeNotices notes={{ unknownCount: 0, validBytes: 7 }} messageType="shop.Order" totalBytes={10} error="unexpected EOF" />
        <DecodeNotices
          notes={{
            unknownCount: 2,
            unknownFields: [
              { path: '', number: 15, wireType: 'varint', size: 2 },
              { path: 'items[0]', number: 4, wireType: 'bytes', size: 6 },
            ],
          }}
          messageType="shop.Order"
          totalBytes={10}
        />
      </>,
    )
    expect(screen.getByTestId('decode-partial')).toHaveTextContent('Decoded the first 7 B of 10 B')
    expect(screen.getByTestId('decode-partial')).toHaveTextContent('unexpected EOF')
    expect(screen.getByTestId('decode-unknown')).toHaveTextContent('2 fields not in the schema')
    expect(screen.getByTestId('decode-unknown')).toHaveTextContent('#15 (varint), #4 (bytes) in items[0]')
  })

  it('stays quiet for a clean decode', () => {
    const { container } = render(<DecodeNotices notes={{ unknownCount: 0 }} messageType="shop.Order" totalBytes={10} />)
    expect(container.querySelector('[data-testid]')).toBeNull()
  })
})
