import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, within } from '@/test/utils'
import type { TypeCandidate } from '@/api/decode'
import { DetectTypeDialog } from './DetectTypeDialog'

const hoisted = vi.hoisted(() => ({ detectMessageType: vi.fn(), getProtoSources: vi.fn() }))

vi.mock('@/api/decode', () => ({ detectMessageType: hoisted.detectMessageType }))
vi.mock('@/api/protoSources', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/api/protoSources')>()),
  getProtoSources: hoisted.getProtoSources,
}))

const user: TypeCandidate = {
  sourceId: 'src-1',
  sourceRevision: 'local',
  messageType: 'shop.User',
  score: 93,
  unknownBytes: 0,
  decoded: { name: 'ann' },
}
const note: TypeCandidate = { ...user, messageType: 'shop.Note', score: 44, unknownBytes: 7, decoded: { text: 'ann' } }

function renderDialog() {
  const props = { onClose: vi.fn(), onUse: vi.fn(), onSave: vi.fn() }
  render(<DetectTypeDialog dataBase64="CgNhbm4=" subject="users.12345.created" saving={false} {...props} />)
  return props
}

describe('DetectTypeDialog', () => {
  beforeEach(() => {
    hoisted.getProtoSources.mockResolvedValue({ items: [{ id: 'src-1', name: 'Shop protos' }] })
  })

  it('ranks the candidates and previews the best one', async () => {
    hoisted.detectMessageType.mockResolvedValue([user, note])
    renderDialog()

    const rows = await screen.findAllByTestId('type-candidate')
    expect(rows).toHaveLength(2)
    expect(rows[0]).toHaveTextContent('shop.User')
    expect(rows[0]).toHaveTextContent('93% fit')
    expect(await within(rows[0]).findByText('Shop protos')).toBeInTheDocument()
    expect(rows[1]).toHaveTextContent('7 bytes the type does not declare')
    expect(within(rows[0]).getByRole('radio')).toBeChecked()
    expect(screen.getByTestId('detect-preview')).toHaveTextContent('"name": "ann"')
    expect(screen.getByTestId('detect-pattern')).toHaveValue('users.*.created')
  })

  it('decodes as the picked candidate', async () => {
    hoisted.detectMessageType.mockResolvedValue([user, note])
    const props = renderDialog()

    const rows = await screen.findAllByTestId('type-candidate')
    fireEvent.click(within(rows[1]).getByRole('radio'))
    expect(screen.getByTestId('detect-preview')).toHaveTextContent('"text": "ann"')
    fireEvent.click(screen.getByTestId('detect-use'))
    expect(props.onUse).toHaveBeenCalledWith(note)
  })

  it('saves the picked candidate under the edited pattern', async () => {
    hoisted.detectMessageType.mockResolvedValue([user])
    const props = renderDialog()

    await screen.findAllByTestId('type-candidate')
    fireEvent.change(screen.getByTestId('detect-pattern'), { target: { value: ' users.> ' } })
    fireEvent.click(screen.getByTestId('detect-save'))
    expect(props.onSave).toHaveBeenCalledWith(user, 'users.>')
  })

  it('blocks saving without a pattern', async () => {
    hoisted.detectMessageType.mockResolvedValue([user])
    renderDialog()

    await screen.findAllByTestId('type-candidate')
    fireEvent.change(screen.getByTestId('detect-pattern'), { target: { value: '  ' } })
    expect(screen.getByTestId('detect-save')).toBeDisabled()
  })

  it('explains when no type fits', async () => {
    hoisted.detectMessageType.mockResolvedValue([])
    renderDialog()

    expect(await screen.findByTestId('detect-empty')).toHaveTextContent('No message type fits this payload.')
    expect(screen.getByTestId('detect-use')).toBeDisabled()
    expect(screen.getByTestId('detect-save')).toBeDisabled()
  })
})
