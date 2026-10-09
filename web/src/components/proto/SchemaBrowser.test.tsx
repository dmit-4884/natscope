import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, within } from '@/test/utils'
import type { SchemaType, TypeDescription } from '@/api/proto'
import type { ProtoSource } from '@/api/protoSources'
import { resetSchemaBrowser } from '@/stores/schemaBrowserStore'
import SchemaBrowser from './SchemaBrowser'

const hoisted = vi.hoisted(() => ({
  listSchemaTypes: vi.fn(),
  describeSchemaType: vi.fn(),
  getProtoMessageExample: vi.fn(),
}))

vi.mock('@/api/proto', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/api/proto')>()),
  ...hoisted,
}))

vi.mock('@/api/mappings', () => ({
  listMappings: vi.fn().mockResolvedValue({
    items: [{ id: 'm1', pattern: 'orders.>', messageType: 'shop.Order', sourceId: 'src', createdAt: 0, updatedAt: 0 }],
    total: 1,
  }),
  getMappingHealth: vi.fn().mockResolvedValue([]),
}))

const schemaType = (fullName: string, overrides: Partial<SchemaType> = {}): SchemaType => ({
  id: `src|${fullName}`,
  fullName,
  kind: 'message',
  file: 'shop.proto',
  packageName: fullName.split('.').slice(0, -1).join('.'),
  comment: '',
  memberCount: 1,
  dependency: false,
  sourceId: 'src',
  sourceRevision: 'local',
  ...overrides,
})

const orderDescription: TypeDescription = {
  messages: [
    {
      fullName: 'shop.Order',
      file: 'shop.proto',
      comment: 'A placed order.',
      deprecated: false,
      fields: [
        {
          name: 'created_at',
          jsonName: 'createdAt',
          number: 1,
          kind: 'message',
          typeName: 'google.protobuf.Timestamp',
          repeated: false,
          mapKey: '',
          optional: false,
          required: false,
          oneof: '',
          deprecated: false,
          comment: 'When it was placed.',
        },
      ],
    },
  ],
  enums: [],
  services: [],
}

const sources = [{ id: 'src', name: 'shop-protos' }] as ProtoSource[]

describe('SchemaBrowser', () => {
  beforeEach(() => {
    resetSchemaBrowser()
    hoisted.listSchemaTypes.mockResolvedValue([
      schemaType('shop.Order', { comment: 'A placed order.' }),
      schemaType('shop.Status', { kind: 'enum', comment: 'Order lifecycle.' }),
      schemaType('google.protobuf.Timestamp', { dependency: true, file: 'google/protobuf/timestamp.proto' }),
    ])
    hoisted.describeSchemaType.mockImplementation(async (_source: string, fullName: string) =>
      fullName === 'shop.Order'
        ? orderDescription
        : {
            messages: [{ fullName, file: 'google/protobuf/timestamp.proto', comment: 'A point in time.', deprecated: false, fields: [] }],
            enums: [],
            services: [],
          },
    )
    hoisted.getProtoMessageExample.mockResolvedValue({ message_type: 'shop.Order', example: { createdAt: '1970-01-01T00:00:00Z' } })
  })

  it('hides imported types until asked', async () => {
    render(<SchemaBrowser sources={sources} />)
    expect(await screen.findByText('Order')).toBeInTheDocument()
    expect(screen.getByText('Order lifecycle.')).toBeInTheDocument()
    expect(screen.queryByText('Timestamp')).not.toBeInTheDocument()

    fireEvent.click(screen.getByTestId('schema-show-imported'))
    expect(screen.getByText('Timestamp')).toBeInTheDocument()
  })

  it('opens on the type and filters left there after a visit elsewhere', async () => {
    const first = render(<SchemaBrowser sources={sources} />)
    fireEvent.click(await screen.findByText('Order'))
    await screen.findByTestId('schema-type-detail')
    first.unmount()

    render(<SchemaBrowser sources={sources} />)

    const detail = await screen.findByTestId('schema-type-detail')
    expect(within(detail).getByText('shop.Order')).toBeInTheDocument()
  })

  it('filters by name or comment', async () => {
    render(<SchemaBrowser sources={sources} />)
    await screen.findByText('Order')
    fireEvent.change(screen.getByRole('textbox', { name: 'Search types or comments…' }), { target: { value: 'lifecycle' } })
    expect(await screen.findByText('Status')).toBeInTheDocument()
    expect(screen.queryByText('Order')).not.toBeInTheDocument()
  })

  it('shows fields, comments, the example and mappings of a message', async () => {
    render(<SchemaBrowser sources={sources} />)
    fireEvent.click(await screen.findByText('Order'))

    const detail = await screen.findByTestId('schema-type-detail')
    expect(within(detail).getByText('shop.Order')).toBeInTheDocument()
    expect(within(detail).getByText('shop.proto · shop-protos @ local')).toBeInTheDocument()
    expect(await within(detail).findByText('When it was placed.')).toBeInTheDocument()
    expect(within(detail).getByText('A placed order.')).toBeInTheDocument()
    expect(await within(detail).findByTestId('schema-example')).toHaveTextContent('1970-01-01T00:00:00Z')
    expect(await within(detail).findByTestId('schema-used-by')).toHaveTextContent('orders.>')
  })

  it('opens a referenced imported type from a field', async () => {
    render(<SchemaBrowser sources={sources} />)
    fireEvent.click(await screen.findByText('Order'))
    fireEvent.click(await screen.findByRole('button', { name: 'google.protobuf.Timestamp' }))

    const detail = await screen.findByTestId('schema-type-detail')
    expect(within(detail).getByText('google.protobuf.Timestamp')).toBeInTheDocument()
    expect(await within(detail).findByText('A point in time.')).toBeInTheDocument()
    expect(screen.getByTestId('schema-show-imported')).toBeChecked()
  })
})
