import { describe, it, expect, vi, beforeEach } from 'vitest'
import { Code, ConnectError } from '@connectrpc/connect'
import { create, toBinary } from '@bufbuild/protobuf'
import { ErrorInfoSchema } from '@/gen/google/rpc/error_details_pb'
import { render, screen, fireEvent, waitFor } from '@/test/utils'
import { importCliContexts, listCliContexts, type CliContextSummary } from '@/api/connections'
import { toast } from '@/utils/toast'
import { CliContextImportDialog } from './CliContextImportDialog'

vi.mock('@/api/connections', () => ({
  listCliContexts: vi.fn(),
  importCliContexts: vi.fn(),
  getConnections: vi.fn().mockResolvedValue([]),
}))

vi.mock('@/utils/toast', () => ({
  toast: { success: vi.fn(), error: vi.fn(), info: vi.fn(), warning: vi.fn() },
}))

const summary = (over: Partial<CliContextSummary>): CliContextSummary => ({
  name: 'x',
  selected: false,
  exists: false,
  importable: true,
  urls: ['nats://x:4222'],
  authMethod: 'none',
  tls: false,
  warnings: [],
  ...over,
})

describe('CliContextImportDialog', () => {
  beforeEach(() => {
    vi.mocked(listCliContexts).mockReset()
    vi.mocked(importCliContexts).mockReset()
  })

  it('lists the host contexts, preselects the new ones and imports them', async () => {
    vi.mocked(listCliContexts).mockResolvedValue({
      directory: '/home/me/.config/nats/context',
      contexts: [
        summary({ name: 'prod', selected: true, authMethod: 'credentials', tls: true, jetstreamDomain: 'hub', warnings: ['SOCKS proxies are not supported'] }),
        summary({ name: 'dev', exists: true }),
        summary({ name: 'broken', importable: false, urls: [], warnings: ['this is not a nats CLI context'] }),
      ],
    })
    vi.mocked(importCliContexts).mockResolvedValue({ created: [], skipped: [] })
    const onClose = vi.fn()
    render(<CliContextImportDialog isOpen onClose={onClose} />)

    expect(await screen.findByText('/home/me/.config/nats/context')).toBeInTheDocument()
    expect(screen.getByRole('checkbox', { name: 'Import prod' })).toBeChecked()
    expect(screen.getByRole('checkbox', { name: 'Import dev' })).toBeDisabled()
    expect(screen.getByRole('checkbox', { name: 'Import broken' })).toBeDisabled()
    expect(screen.getByText('current')).toBeInTheDocument()
    expect(screen.getByText('already saved')).toBeInTheDocument()
    expect(screen.getByText('SOCKS proxies are not supported')).toBeInTheDocument()
    expect(screen.getByText('domain hub')).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'Import 1 connection' }))

    await waitFor(() => expect(importCliContexts).toHaveBeenCalledWith(['prod'], []))
    await waitFor(() => expect(onClose).toHaveBeenCalled())
  })

  it('reports the skipped contexts', async () => {
    vi.mocked(listCliContexts).mockResolvedValue({ directory: '/d', contexts: [summary({ name: 'prod' })] })
    vi.mocked(importCliContexts).mockResolvedValue({ created: [], skipped: [{ name: 'prod', reason: 'a server URL is invalid' }] })
    render(<CliContextImportDialog isOpen onClose={vi.fn()} />)

    fireEvent.click(await screen.findByRole('button', { name: 'Import 1 connection' }))

    await waitFor(() => expect(toast.warning).toHaveBeenCalledWith('Skipped prod: a server URL is invalid'))
  })

  it('reads uploaded context files when the host has none', async () => {
    vi.mocked(listCliContexts)
      .mockResolvedValueOnce({ directory: '/d', contexts: [] })
      .mockResolvedValueOnce({ directory: '', contexts: [summary({ name: 'remote' })] })
    vi.mocked(importCliContexts).mockResolvedValue({ created: [], skipped: [] })
    render(<CliContextImportDialog isOpen onClose={vi.fn()} />)

    expect(await screen.findByText(/No nats CLI contexts/)).toBeInTheDocument()
    const file = new File(['{"url":"nats://remote:4222"}'], 'remote.json', { type: 'application/json' })
    fireEvent.change(screen.getByLabelText('Upload context files'), { target: { files: [file] } })

    expect(await screen.findByRole('checkbox', { name: 'Import remote' })).toBeChecked()
    const [files] = vi.mocked(listCliContexts).mock.calls[1]
    expect(files?.[0].name).toBe('remote.json')
    expect(new TextDecoder().decode(files?.[0].content)).toBe('{"url":"nats://remote:4222"}')

    fireEvent.click(screen.getByRole('button', { name: 'Import 1 connection' }))
    await waitFor(() => expect(importCliContexts).toHaveBeenCalledWith(['remote'], files))
  })

  it('asks for uploads when the server does not read its own contexts', async () => {
    const err = new ConnectError('off', Code.FailedPrecondition)
    err.details = [{ type: ErrorInfoSchema.typeName, value: toBinary(ErrorInfoSchema, create(ErrorInfoSchema, { reason: 'CLI_CONTEXTS_HOST_DISABLED' })) }]
    vi.mocked(listCliContexts).mockRejectedValue(err)
    render(<CliContextImportDialog isOpen onClose={vi.fn()} />)

    expect(await screen.findByText(/does not read this host's nats CLI contexts/)).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /try again|retry/i })).not.toBeInTheDocument()
  })
})
