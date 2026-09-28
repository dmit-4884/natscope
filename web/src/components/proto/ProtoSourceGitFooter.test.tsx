import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@/test/utils'
import { ProtoSourceGitFooter } from './ProtoSourceGitFooter'

// QA-100: SourcesService/ListTags errors must surface to the user instead of
// rendering an empty "No versions found" dropdown.
describe('ProtoSourceGitFooter — tagsError', () => {
  it('shows the error and a Retry action instead of the dropdown', () => {
    const onRetryTags = vi.fn()

    render(
      <ProtoSourceGitFooter
        showTagSelector
        onToggleTagSelector={() => {}}
        onRemoveSelection={() => {}}
        tags={[]}
        isLoadingTags={false}
        tagsError={new Error('repository not accessible or not a git server')}
        onRetryTags={onRetryTags}
        isSelecting={false}
        onSelectTag={() => {}}
      />,
    )

    const alert = screen.getByRole('alert')
    expect(alert).toHaveTextContent('Failed to load versions')
    expect(alert).toHaveTextContent('repository not accessible or not a git server')
    expect(screen.queryByText('No versions found')).not.toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'Retry' }))
    expect(onRetryTags).toHaveBeenCalledTimes(1)
  })

  it('falls back to the version dropdown when there is no error', () => {
    render(
      <ProtoSourceGitFooter
        showTagSelector
        onToggleTagSelector={() => {}}
        onRemoveSelection={() => {}}
        tags={[]}
        isLoadingTags={false}
        tagsError={null}
        isSelecting={false}
        onSelectTag={() => {}}
      />,
    )

    expect(screen.queryByRole('alert')).not.toBeInTheDocument()
    expect(screen.getByText('No versions found')).toBeInTheDocument()
  })
})
