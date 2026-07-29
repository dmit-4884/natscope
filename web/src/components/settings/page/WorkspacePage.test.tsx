import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@/test/utils'
import * as workspaceApi from '@/api/workspace'
import WorkspacePage from './WorkspacePage'

vi.mock('@/utils/toast', () => ({
  toast: { success: vi.fn(), error: vi.fn(), warning: vi.fn(), info: vi.fn() },
}))

vi.mock('@/api/workspace', () => ({
  listSections: vi.fn(),
  exportWorkspace: vi.fn(),
  validateWorkspace: vi.fn(),
  importWorkspace: vi.fn(),
}))

const api = vi.mocked(workspaceApi)

// A report where REPLACE would delete existing items (connections), MERGE not.
function reportFor(strategy: workspaceApi.WorkspaceStrategy): workspaceApi.SectionReport[] {
  return [
    {
      key: 'connections',
      created: 1,
      updated: 0,
      deleted: strategy === 'replace' ? 2 : 0,
      conflicts: [],
      warnings: [],
      unknown: false,
    },
  ]
}

function uploadFile() {
  const input = screen.getByTestId('workspace-file-input') as HTMLInputElement
  const file = new File([new Uint8Array([123, 125])], 'ws.json', { type: 'application/json' })
  // jsdom's File.arrayBuffer can be flaky; stub it deterministically.
  Object.defineProperty(file, 'arrayBuffer', {
    value: () => Promise.resolve(new Uint8Array([123, 125]).buffer),
  })
  fireEvent.change(input, { target: { files: [file] } })
}

describe('WorkspacePage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    api.listSections.mockResolvedValue([{ key: 'connections', title: 'Connections (no credentials)', count: 3 }])
    api.validateWorkspace.mockImplementation((_p, _k, strat) => Promise.resolve(reportFor(strat)))
    api.importWorkspace.mockResolvedValue([
      { key: 'connections', created: 1, updated: 0, deleted: 2, warnings: [] },
    ])
  })

  it('loads sections via React Query (m27)', async () => {
    render(<WorkspacePage />)
    expect(await screen.findByText(/Connections \(no credentials\)/)).toBeInTheDocument()
    expect(api.listSections).toHaveBeenCalled()
  })

  it('gates a destructive REPLACE import behind type-to-confirm and shows the deletion count (M13/M6)', async () => {
    render(<WorkspacePage />)
    await screen.findByText(/Connections \(no credentials\)/)

    uploadFile()
    // Dry-run renders (merge → no deletions shown).
    await screen.findByTestId('import-report-connections')
    expect(screen.queryByTestId('import-report-connections-deleted')).toBeNull()

    // Switch to REPLACE → re-validate surfaces the predicted deletions.
    fireEvent.click(screen.getByTestId('strategy-replace'))
    expect(await screen.findByTestId('import-report-connections-deleted')).toHaveTextContent('2 deleted')

    // Clicking Apply opens the type-to-confirm guard, does NOT import yet.
    fireEvent.click(screen.getByTestId('workspace-import-btn'))
    expect(await screen.findByTestId('replace-confirm')).toBeInTheDocument()
    expect(api.importWorkspace).not.toHaveBeenCalled()

    // Confirm button stays disabled until the exact word is typed.
    const confirmBtn = screen.getByTestId('replace-confirm-btn') as HTMLButtonElement
    expect(confirmBtn.disabled).toBe(true)
    fireEvent.change(screen.getByTestId('replace-confirm-input'), { target: { value: 'REPLACE' } })
    expect(confirmBtn.disabled).toBe(false)

    fireEvent.click(confirmBtn)
    await waitFor(() => expect(api.importWorkspace).toHaveBeenCalledTimes(1))
    expect(api.importWorkspace).toHaveBeenCalledWith(expect.anything(), ['connections'], 'replace')
  })

  it('applies a non-destructive MERGE import directly without the confirm guard', async () => {
    render(<WorkspacePage />)
    await screen.findByText(/Connections \(no credentials\)/)

    uploadFile()
    await screen.findByTestId('import-report-connections')

    // Merge has no deletions → Apply imports immediately, no confirm panel.
    fireEvent.click(screen.getByTestId('workspace-import-btn'))
    await waitFor(() => expect(api.importWorkspace).toHaveBeenCalledTimes(1))
    expect(screen.queryByTestId('replace-confirm')).toBeNull()
    expect(api.importWorkspace).toHaveBeenCalledWith(expect.anything(), ['connections'], 'merge')
  })
})
