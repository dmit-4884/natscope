import { useState } from 'react'
import { describe, expect, it } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'
import { Link, MemoryRouter, Route, Routes } from 'react-router-dom'
import { KeyedByParam } from './KeyedByParam'

function Draft() {
  const [text, setText] = useState('')
  return (
    <>
      <input aria-label="Draft" value={text} onChange={(e) => setText(e.target.value)} />
      <Link to="/items/b">Next item</Link>
      <Link to="/items/a?tab=2">Same item</Link>
    </>
  )
}

function renderAt(path: string) {
  render(
    <MemoryRouter initialEntries={[path]}>
      <Routes>
        <Route
          path="/items/:id"
          element={
            <KeyedByParam param="id">
              <Draft />
            </KeyedByParam>
          }
        />
      </Routes>
    </MemoryRouter>,
  )
}

describe('KeyedByParam', () => {
  it('starts the page afresh for another item, so nothing of the previous one shows', () => {
    renderAt('/items/a')
    fireEvent.change(screen.getByLabelText('Draft'), { target: { value: 'typed for a' } })

    fireEvent.click(screen.getByText('Next item'))

    expect(screen.getByLabelText('Draft')).toHaveValue('')
  })

  it('keeps the page while the item stays the same', () => {
    renderAt('/items/a')
    fireEvent.change(screen.getByLabelText('Draft'), { target: { value: 'typed for a' } })

    fireEvent.click(screen.getByText('Same item'))

    expect(screen.getByLabelText('Draft')).toHaveValue('typed for a')
  })
})
