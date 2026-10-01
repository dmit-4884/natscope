import { describe, it, expect } from 'vitest'
import { render, screen } from '@/test/utils'
import type { SchemaConflict } from '@/api/proto'
import type { ProtoSource } from '@/api/protoSources'
import SchemaConflicts from './SchemaConflicts'

const sources = [
  { id: 'a', name: 'Shop protos' },
  { id: 'b', name: 'Legacy protos' },
] as ProtoSource[]

const conflict = (c: Partial<SchemaConflict>): SchemaConflict => ({
  kind: 'different_shape',
  severity: 'error',
  symbol: 'shop.Order',
  first: { sourceId: 'a', revision: '0123456789abcdef', file: 'shop/order.proto' },
  second: { sourceId: 'b', revision: '', file: 'legacy/order.proto' },
  ...c,
})

describe('SchemaConflicts', () => {
  it('lists clashes before duplicates with both sides named', () => {
    render(
      <SchemaConflicts
        sources={sources}
        conflicts={[
          conflict({ kind: 'same_shape', severity: 'info', symbol: 'fin.Money' }),
          conflict({}),
          conflict({ kind: 'file_content', symbol: 'common.proto' }),
        ]}
      />,
    )

    const items = screen.getAllByTestId('schema-conflict')
    expect(items.map((i) => i.querySelector('.font-mono')?.textContent)).toEqual([
      'common.proto',
      'shop.Order',
      'fin.Money',
    ])
    expect(items[0]).toHaveTextContent('Clash')
    expect(items[0]).toHaveTextContent('Two sources ship different versions of this file.')
    expect(items[1]).toHaveTextContent('Shop protos · shop/order.proto @ 0123456789ab')
    expect(items[1]).toHaveTextContent('Legacy protos · legacy/order.proto')
    expect(items[2]).toHaveTextContent('Duplicate')
  })

  it('falls back to the source id for a source it does not know', () => {
    render(<SchemaConflicts sources={[]} conflicts={[conflict({})]} />)
    expect(screen.getByTestId('schema-conflict')).toHaveTextContent('a · shop/order.proto')
  })
})
