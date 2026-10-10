import { describe, it, expect, vi } from 'vitest'
import { fireEvent, render, screen } from '@/test/utils'
import { DataTable, type DataTableColumn } from './DataTable'

interface Row {
  name: string
  count: number
}

const columns: DataTableColumn<Row>[] = [
  { key: 'name', header: 'Name', sortable: true, render: (r) => r.name },
  { key: 'count', header: 'Count', hint: 'How many there are', sortable: true, align: 'right', render: (r) => r.count },
  { key: 'note', header: 'Note', render: () => '' },
]

const items: Row[] = [
  { name: 'a', count: 1 },
  { name: 'b', count: 2 },
]

describe('DataTable sorting', () => {
  it('marks the sorted column and asks to sort by a clicked header', () => {
    const onSortChange = vi.fn()
    render(<DataTable columns={columns} items={items} rowKey={(r) => r.name} sort={{ key: 'count', direction: 'desc' }} onSortChange={onSortChange} />)

    expect(screen.getByRole('columnheader', { name: /count/i })).toHaveAttribute('aria-sort', 'descending')
    expect(screen.getByRole('columnheader', { name: /name/i })).not.toHaveAttribute('aria-sort')

    fireEvent.click(screen.getByRole('button', { name: /name/i }))

    expect(onSortChange).toHaveBeenCalledWith('name')
  })

  it('reports an ascending sort', () => {
    render(<DataTable columns={columns} items={items} rowKey={(r) => r.name} sort={{ key: 'name', direction: 'asc' }} onSortChange={vi.fn()} />)

    expect(screen.getByRole('columnheader', { name: /name/i })).toHaveAttribute('aria-sort', 'ascending')
  })

  it('keeps a column without sortable as plain text', () => {
    render(<DataTable columns={columns} items={items} rowKey={(r) => r.name} sort={{ key: 'name', direction: 'asc' }} onSortChange={vi.fn()} />)

    expect(screen.queryByRole('button', { name: /note/i })).not.toBeInTheDocument()
    expect(screen.getByRole('columnheader', { name: /note/i })).toBeInTheDocument()
  })
})
