import { describe, it, expect, vi, beforeAll, afterAll } from 'vitest'
import type { ComponentProps } from 'react'
import { render, screen, fireEvent } from '@/test/utils'
import { ProtoSourceRefFooter } from './ProtoSourceRefFooter'

type Props = ComponentProps<typeof ProtoSourceRefFooter>

const SHA_A = 'a'.repeat(40)
const SHA_B = 'b'.repeat(40)

function renderFooter(overrides: Partial<Props> = {}) {
  const props: Props = {
    registry: 'git',
    showPicker: false,
    onTogglePicker: vi.fn(),
    refs: [],
    isLoadingRefs: false,
    refsError: null,
    onRetryRefs: vi.fn(),
    onSelectRef: vi.fn(),
    onRefresh: vi.fn(),
    isBusy: false,
    actionError: null,
    outcome: null,
    ...overrides,
  }
  render(<ProtoSourceRefFooter {...props} />)
  return props
}

describe('ProtoSourceRefFooter', () => {
  const originalScrollIntoView = Element.prototype.scrollIntoView
  beforeAll(() => {
    Element.prototype.scrollIntoView = vi.fn()
  })
  afterAll(() => {
    Element.prototype.scrollIntoView = originalScrollIntoView
  })

  it('asks for a ref when none is selected', () => {
    const props = renderFooter()
    fireEvent.click(screen.getByText('Pick a tag, branch or commit to compile'))
    expect(props.onTogglePicker).toHaveBeenCalledTimes(1)
    expect(screen.queryByTestId('ref-refresh')).not.toBeInTheDocument()
  })

  it('shows the selected ref and refreshes it', () => {
    const props = renderFooter({
      selectedRef: { name: 'main', kind: 'branch', revision: SHA_A },
      activeSchema: { revision: SHA_A, fingerprint: 'fp', compiledAt: 1, messageCount: 12, active: true },
    })
    expect(screen.getByTestId('ref-badge')).toHaveTextContent('main')
    expect(screen.getByText('branch')).toBeInTheDocument()
    expect(screen.getByText('aaaaaaa')).toHaveAttribute('title', SHA_A)
    expect(screen.getByText('· 12 message types')).toBeInTheDocument()

    fireEvent.click(screen.getByTestId('ref-refresh'))
    expect(props.onRefresh).toHaveBeenCalledTimes(1)
  })

  it('speaks in labels and commit ids for a BSR source', () => {
    const props = renderFooter({ registry: 'bsr', showPicker: true })
    expect(screen.getByText('Pick a label or commit to load')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Label' })).toBeInTheDocument()
    fireEvent.change(screen.getByLabelText('Commit ID'), { target: { value: '0123456789abcdef0123456789abcdef' } })
    fireEvent.click(screen.getByTestId('ref-commit-use'))
    expect(props.onSelectRef).toHaveBeenCalledWith('0123456789abcdef0123456789abcdef')
  })

  it('lists tags and branches and selects one', () => {
    const props = renderFooter({
      showPicker: true,
      refs: [
        { name: 'v1.0.0', kind: 'tag', revision: SHA_A },
        { name: 'main', kind: 'branch', revision: SHA_B },
      ],
    })
    fireEvent.click(screen.getByRole('button', { name: 'Tag or branch' }))
    fireEvent.click(screen.getByRole('option', { name: 'main · branch · bbbbbbb' }))
    expect(props.onSelectRef).toHaveBeenCalledWith('main')
  })

  it('selects a commit by SHA', () => {
    const props = renderFooter({ showPicker: true })
    expect(screen.getByTestId('ref-commit-use')).toBeDisabled()
    fireEvent.change(screen.getByTestId('ref-commit-input'), { target: { value: ` ${SHA_B} ` } })
    fireEvent.click(screen.getByTestId('ref-commit-use'))
    expect(props.onSelectRef).toHaveBeenCalledWith(SHA_B)
  })

  it('shows a ref listing error with a retry', () => {
    const props = renderFooter({ showPicker: true, refsError: new Error('repository not accessible or not a git server') })
    const alert = screen.getByRole('alert')
    expect(alert).toHaveTextContent('Failed to list refs')
    expect(alert).toHaveTextContent('repository not accessible or not a git server')
    fireEvent.click(screen.getByRole('button', { name: 'Retry' }))
    expect(props.onRetryRefs).toHaveBeenCalledTimes(1)
  })

  it('reports the compile outcome', () => {
    renderFooter({ outcome: { valid: true, messageTypes: 3, fileDescriptors: 2, diagnostics: [] } })
    expect(screen.getByTestId('ref-compile-ok')).toHaveTextContent('Compiled 3 message types')
  })

  it('shows diagnostics when the selected ref does not compile', () => {
    renderFooter({
      outcome: {
        valid: false,
        messageTypes: 0,
        fileDescriptors: 0,
        diagnostics: [{ severity: 'error', file: 'shop.proto', line: 3, column: 17, message: 'syntax error' }],
      },
    })
    expect(screen.queryByTestId('ref-compile-ok')).not.toBeInTheDocument()
    expect(screen.getByText(/syntax error/)).toBeInTheDocument()
  })

  it('shows progress while fetching', () => {
    renderFooter({ isBusy: true, selectedRef: { name: 'v1', kind: 'tag', revision: SHA_A } })
    expect(screen.getByText('Fetching and compiling…')).toBeInTheDocument()
  })
})
