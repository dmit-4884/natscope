import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@/test/utils'
import { ProtoSourceUploadFooter } from './ProtoSourceUploadFooter'

const pick = (name: string, content: string) =>
  fireEvent.change(screen.getByLabelText('Proto files or descriptor set'), {
    target: { files: [new File([content], name)] },
  })

describe('ProtoSourceUploadFooter', () => {
  it('opens the picker until something is uploaded and reports the compile', async () => {
    const onUpload = vi.fn().mockResolvedValue({ valid: true, messageTypes: 3, fileDescriptors: 2, diagnostics: [] })
    render(<ProtoSourceUploadFooter onUpload={onUpload} isUploading={false} />)

    expect(screen.getByText('Nothing uploaded yet')).toBeInTheDocument()
    pick('order.proto', 'syntax = "proto3";')

    await waitFor(() => expect(onUpload).toHaveBeenCalledTimes(1))
    expect(onUpload.mock.calls[0][0].content).toEqual({
      kind: 'files',
      files: [{ path: 'order.proto', content: 'syntax = "proto3";' }],
    })
    expect(await screen.findByTestId('upload-ok')).toHaveTextContent('Compiled: 2 file descriptors, 3 message types')
    expect(screen.queryByTestId('schema-upload-dropzone')).not.toBeInTheDocument()
  })

  it('keeps the picker open and lists diagnostics when the upload does not compile', async () => {
    const onUpload = vi.fn().mockResolvedValue({
      valid: false,
      messageTypes: 0,
      fileDescriptors: 0,
      diagnostics: [{ severity: 'error', file: 'order.proto', line: 1, column: 1, message: 'syntax error' }],
    })
    render(
      <ProtoSourceUploadFooter
        activeSchema={{ revision: 'abc123def456', fingerprint: 'fp', compiledAt: 1, messageCount: 4, active: true }}
        onUpload={onUpload}
        isUploading={false}
      />,
    )

    expect(screen.getByTestId('upload-revision')).toHaveTextContent('abc123def456')
    fireEvent.click(screen.getByTestId('upload-new-version'))
    pick('order.proto', 'broken')

    expect(await screen.findByText('syntax error')).toBeInTheDocument()
    expect(screen.getByTestId('schema-upload-dropzone')).toBeInTheDocument()
    expect(screen.queryByTestId('upload-ok')).not.toBeInTheDocument()
  })

  it('says when the selection has nothing to upload', async () => {
    const onUpload = vi.fn()
    render(<ProtoSourceUploadFooter onUpload={onUpload} isUploading={false} />)
    pick('notes.txt', 'x')
    expect(await screen.findByRole('alert')).toHaveTextContent('No .proto files or descriptor set in the selection.')
    expect(onUpload).not.toHaveBeenCalled()
  })
})
