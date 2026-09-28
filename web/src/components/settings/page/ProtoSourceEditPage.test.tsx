import { describe, it, expect, vi, beforeEach } from 'vitest'
import { MemoryRouter } from 'react-router-dom'
import { render, screen, fireEvent, waitFor } from '@/test/utils'
import * as protoSourcesApi from '@/api/protoSources'
import ProtoSourceEditPage from './ProtoSourceEditPage'

vi.mock('@/api/protoSources', async (importOriginal) => ({
  ...(await importOriginal<typeof protoSourcesApi>()),
  validateLocalPath: vi.fn(),
  validateRepository: vi.fn(),
  createProtoSource: vi.fn(),
}))

const api = vi.mocked(protoSourcesApi)

function renderPage() {
  render(
    <MemoryRouter>
      <ProtoSourceEditPage mode="create" />
    </MemoryRouter>,
  )
}

// the form must validate the path/URL before saving instead of
// persisting a source that can never compile.
describe('ProtoSourceEditPage — validate before save', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('blocks Save and surfaces the error for a local path that does not validate', async () => {
    api.validateLocalPath.mockResolvedValue({
      valid: false,
      protoFileCount: 0,
      error: 'no .proto files found at this path',
    })

    renderPage()

    fireEvent.click(screen.getByRole('button', { name: /Local Directory/i }))
    fireEvent.change(screen.getByLabelText('Name *'), { target: { value: 'qa-165' } })
    fireEvent.change(screen.getByLabelText('Directory Path *'), {
      target: { value: '/nonexistent/qa-165' },
    })

    fireEvent.click(screen.getByRole('button', { name: 'Add Source' }))

    await waitFor(() => expect(api.validateLocalPath).toHaveBeenCalledWith('/nonexistent/qa-165'))
    await waitFor(() =>
      expect(screen.getByText('no .proto files found at this path')).toBeInTheDocument(),
    )
    expect(api.createProtoSource).not.toHaveBeenCalled()
  })

  it('saves once the local path validates successfully', async () => {
    api.validateLocalPath.mockResolvedValue({ valid: true, protoFileCount: 2 })
    api.createProtoSource.mockResolvedValue({
      id: 'src-1',
      name: 'qa-165-ok',
      sourceType: 'local',
      repository: '',
      localPath: '/tmp/ok',
      watcherEnabled: true,
      enabled: true,
      files: [],
      includeDirs: [],
      created_at: 0,
    })

    renderPage()

    fireEvent.click(screen.getByRole('button', { name: /Local Directory/i }))
    fireEvent.change(screen.getByLabelText('Name *'), { target: { value: 'qa-165-ok' } })
    fireEvent.change(screen.getByLabelText('Directory Path *'), { target: { value: '/tmp/ok' } })

    fireEvent.click(screen.getByRole('button', { name: 'Add Source' }))

    await waitFor(() => expect(api.createProtoSource).toHaveBeenCalledTimes(1))
  })

  it('blocks Save for a git repository the server cannot reach', async () => {
    api.validateRepository.mockResolvedValue({
      valid: false,
      error: 'unsupported repository URL scheme: file',
    })

    renderPage()

    // Git is the default source type — no need to click a type button.
    fireEvent.change(screen.getByLabelText('Name *'), { target: { value: 'qa-165-git' } })
    fireEvent.change(screen.getByLabelText('Repository URL *'), {
      target: { value: 'file:///tmp/repo' },
    })

    fireEvent.click(screen.getByRole('button', { name: 'Add Source' }))

    await waitFor(() =>
      expect(api.validateRepository).toHaveBeenCalledWith('file:///tmp/repo', undefined),
    )
    await waitFor(() =>
      expect(screen.getByText('unsupported repository URL scheme: file')).toBeInTheDocument(),
    )
    expect(api.createProtoSource).not.toHaveBeenCalled()
  })
})
