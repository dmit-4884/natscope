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
  uploadSchema: vi.fn(),
}))

const api = vi.mocked(protoSourcesApi)

function renderPage() {
  render(
    <MemoryRouter>
      <ProtoSourceEditPage mode="create" />
    </MemoryRouter>,
  )
}

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
      created_at: 0,
    })

    renderPage()

    fireEvent.click(screen.getByRole('button', { name: /Local Directory/i }))
    fireEvent.change(screen.getByLabelText('Name *'), { target: { value: 'qa-165-ok' } })
    fireEvent.change(screen.getByLabelText('Directory Path *'), { target: { value: '/tmp/ok' } })

    fireEvent.click(screen.getByRole('button', { name: 'Add Source' }))

    await waitFor(() => expect(api.createProtoSource).toHaveBeenCalledTimes(1))
  })

  it('creates an upload source and uploads the picked files', async () => {
    api.createProtoSource.mockResolvedValue({
      id: 'src-up',
      name: 'uploads',
      sourceType: 'upload',
      repository: '',
      watcherEnabled: false,
      enabled: true,
      created_at: 0,
    })
    api.uploadSchema.mockResolvedValue({
      source: {} as protoSourcesApi.ProtoSource,
      outcome: { valid: false, messageTypes: 0, fileDescriptors: 0, diagnostics: [
        { severity: 'error', file: 'shop/order.proto', line: 3, column: 1, message: 'syntax error: unexpected identifier' },
      ] },
    })

    renderPage()

    fireEvent.click(screen.getByRole('button', { name: /Upload/ }))
    fireEvent.change(screen.getByLabelText('Name *'), { target: { value: 'uploads' } })
    fireEvent.change(screen.getByLabelText('Proto files or descriptor set'), {
      target: {
        files: [
          new File(['syntax = "proto3";'], 'order.proto'),
          new File(['version: v2'], 'buf.yaml'),
          new File(['# notes'], 'README.md'),
        ],
      },
    })
    expect(await screen.findByTestId('schema-upload-selected')).toHaveTextContent('1 .proto file · 1 buf config · 1 other skipped')

    fireEvent.click(screen.getByRole('button', { name: 'Add Source' }))

    await waitFor(() => expect(api.uploadSchema).toHaveBeenCalledTimes(1))
    expect(api.createProtoSource).toHaveBeenCalledWith(expect.objectContaining({ name: 'uploads', sourceType: 'upload' }))
    expect(api.uploadSchema).toHaveBeenCalledWith('src-up', {
      kind: 'files',
      files: [
        { path: 'order.proto', content: 'syntax = "proto3";' },
        { path: 'buf.yaml', content: 'version: v2' },
      ],
    })
    expect(await screen.findByText('syntax error: unexpected identifier')).toBeInTheDocument()
  })

  it('validates a BSR module before creating the source', async () => {
    api.validateRepository.mockResolvedValue({ valid: true })
    api.createProtoSource.mockResolvedValue({
      id: 'src-bsr',
      name: 'payments',
      sourceType: 'bsr',
      repository: 'buf.build/acme/payments',
      watcherEnabled: false,
      enabled: true,
      created_at: 0,
    })

    renderPage()

    fireEvent.click(screen.getByRole('button', { name: /Buf Schema Registry/ }))
    fireEvent.change(screen.getByLabelText('Name *'), { target: { value: 'payments' } })
    fireEvent.change(screen.getByLabelText('Module *'), { target: { value: 'buf.build/acme/payments' } })
    fireEvent.change(screen.getByLabelText(/Access Token/), { target: { value: 'bsr-token' } })
    expect(screen.queryByText('Advanced (optional)')).not.toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'Add Source' }))

    await waitFor(() => expect(api.createProtoSource).toHaveBeenCalledTimes(1))
    expect(api.validateRepository).toHaveBeenCalledWith('buf.build/acme/payments', 'bsr-token', 'bsr')
    expect(api.createProtoSource).toHaveBeenCalledWith(
      expect.objectContaining({ sourceType: 'bsr', repository: 'buf.build/acme/payments', token: 'bsr-token' }),
    )
  })

  it('blocks Save for a git repository the server cannot reach', async () => {
    api.validateRepository.mockResolvedValue({
      valid: false,
      error: 'unsupported repository URL scheme: file',
    })

    renderPage()

    fireEvent.change(screen.getByLabelText('Name *'), { target: { value: 'qa-165-git' } })
    fireEvent.change(screen.getByLabelText('Repository URL *'), {
      target: { value: 'file:///tmp/repo' },
    })

    fireEvent.click(screen.getByRole('button', { name: 'Add Source' }))

    await waitFor(() =>
      expect(api.validateRepository).toHaveBeenCalledWith('file:///tmp/repo', undefined, 'git'),
    )
    await waitFor(() =>
      expect(screen.getByText('unsupported repository URL scheme: file')).toBeInTheDocument(),
    )
    expect(api.createProtoSource).not.toHaveBeenCalled()
  })
})
