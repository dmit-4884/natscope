import { describe, it, expect } from 'vitest'
import userEvent from '@testing-library/user-event'
import { render, screen, waitFor } from '@/test/utils'
import JsonViewer from './JsonViewer'

describe('JsonViewer', () => {

  it('should render JSON data', () => {
    const data = { name: 'Test', value: 123 }

    render(<JsonViewer data={data} />)

    expect(screen.getByText(/"name":/)).toBeInTheDocument()
    expect(screen.getByText(/"value":/)).toBeInTheDocument()
  })

  it('should render with title', () => {
    const data = { test: 'data' }

    render(<JsonViewer data={data} title="Test Title" />)

    expect(screen.getByText('Test Title')).toBeInTheDocument()
  })

  it('should render string data', () => {
    const jsonString = '{"key": "value"}'

    render(<JsonViewer data={jsonString} />)

    expect(screen.getByText(/key/)).toBeInTheDocument()
  })

  it('should show copied state when copy button is clicked', async () => {
    const user = userEvent.setup()
    const data = { test: 'data' }

    render(<JsonViewer data={data} title="Test" />)

    // Verify Copy button is shown
    expect(screen.getByText('Copy')).toBeInTheDocument()

    // Click the copy button
    const copyButton = screen.getByText('Copy')
    await user.click(copyButton)

    // Verify button changes to "Copied!" after click
    await waitFor(() => {
      expect(screen.getByText('Copied!')).toBeInTheDocument()
    })
  })

  it('should format nested objects correctly', () => {
    const data = {
      user: {
        name: 'John',
        age: 30,
      },
    }

    render(<JsonViewer data={data} />)

    // Check for parts of the formatted JSON
    expect(screen.getByText(/"user":/)).toBeInTheDocument()
    expect(screen.getByText(/"name":/)).toBeInTheDocument()
    expect(screen.getByText(/"John"/)).toBeInTheDocument()
  })
})
