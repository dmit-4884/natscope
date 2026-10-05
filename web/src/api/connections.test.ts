import { describe, it, expect, vi, beforeEach } from 'vitest'
import { create } from '@bufbuild/protobuf'
import {
  CreateConnectionResponseSchema,
  ImportCliContextsResponseSchema,
  ListCliContextsResponseSchema,
  ListConnectionsResponseSchema,
  TestConnectionResponseSchema,
  UpdateConnectionResponseSchema,
  ConnectionCheckStatus,
  ConnectionCheckStep,
} from '../gen/services/grpc/nats/v1/connections/nats_connections_service_pb'
import { AuthMethod, LabelColor } from '../gen/types/nats/nats_connection_pb'

const client = vi.hoisted(() => ({
  listConnections: vi.fn(),
  createConnection: vi.fn(),
  updateConnection: vi.fn(),
  listCliContexts: vi.fn(),
  importCliContexts: vi.fn(),
}))

vi.mock('./grpc/clients', () => ({ connectionsClient: client }))

import { connectionChecks, createConnection, getConnections, importCliContexts, listCliContexts, updateConnection } from './connections'

describe('connections api', () => {
  beforeEach(() => {
    client.listConnections.mockReset()
    client.createConnection.mockReset()
    client.updateConnection.mockReset()
  })

  it('reads the read-only switch and the label', async () => {
    client.listConnections.mockResolvedValue(
      create(ListConnectionsResponseSchema, {
        connections: [
          { id: 'a', name: 'prod', urls: ['nats://p'], readOnly: true, label: { text: 'PROD', color: LabelColor.RED } },
          { id: 'b', name: 'dev', urls: ['nats://d'] },
        ],
      }),
    )

    const [prod, dev] = await getConnections()

    expect(prod.readOnly).toBe(true)
    expect(prod.label).toEqual({ text: 'PROD', color: 'red' })
    expect(dev.readOnly).toBe(false)
    expect(dev.label).toBeUndefined()
  })

  it('sends the read-only switch and the label on create and update', async () => {
    client.createConnection.mockResolvedValue(create(CreateConnectionResponseSchema, { connection: { id: 'a' } }))
    client.updateConnection.mockResolvedValue(create(UpdateConnectionResponseSchema, { connection: { id: 'a' } }))

    await createConnection({ name: 'prod', urls: ['nats://p'], readOnly: true, label: { text: 'PROD', color: 'amber' } })
    expect(client.createConnection).toHaveBeenCalledWith(
      expect.objectContaining({ readOnly: true, label: { text: 'PROD', color: LabelColor.AMBER } }),
    )

    await updateConnection('a', { readOnly: false, label: null })
    expect(client.updateConnection).toHaveBeenCalledWith(
      expect.objectContaining({ readOnly: false, label: { text: '', color: LabelColor.UNSPECIFIED } }),
    )
  })
})

describe('nats CLI contexts api', () => {
  it('lists contexts with their settings and warnings', async () => {
    client.listCliContexts.mockResolvedValue(
      create(ListCliContextsResponseSchema, {
        directory: '/home/me/.config/nats/context',
        contexts: [
          {
            name: 'prod',
            selected: true,
            importable: true,
            urls: ['tls://p:4222'],
            authMethod: AuthMethod.CREDENTIALS,
            tls: true,
            jetstreamDomain: 'hub',
            warnings: ['SOCKS proxies are not supported'],
          },
        ],
      }),
    )
    const upload = { name: 'prod.json', content: new Uint8Array([123, 125]) }

    const found = await listCliContexts([upload])

    expect(client.listCliContexts).toHaveBeenCalledWith({ files: [upload] })
    expect(found.directory).toBe('/home/me/.config/nats/context')
    expect(found.contexts[0]).toMatchObject({
      name: 'prod',
      selected: true,
      exists: false,
      importable: true,
      urls: ['tls://p:4222'],
      authMethod: 'credentials',
      tls: true,
      jetstreamDomain: 'hub',
      warnings: ['SOCKS proxies are not supported'],
    })
  })

  it('imports the chosen contexts and reports the skipped ones', async () => {
    client.importCliContexts.mockResolvedValue(
      create(ImportCliContextsResponseSchema, {
        connections: [{ id: 'a', name: 'prod' }],
        skipped: [{ name: 'dev', reason: 'a connection with this name already exists' }],
      }),
    )

    const res = await importCliContexts(['prod', 'dev'])

    expect(client.importCliContexts).toHaveBeenCalledWith({ names: ['prod', 'dev'], files: [] })
    expect(res.created.map((c) => c.name)).toEqual(['prod'])
    expect(res.skipped).toEqual([{ name: 'dev', reason: 'a connection with this name already exists' }])
  })
})

describe('connectionChecks', () => {
  it('names each step and its outcome', () => {
    const resp = create(TestConnectionResponseSchema, {
      checks: [
        { step: ConnectionCheckStep.DNS, status: ConnectionCheckStatus.OK, detail: '127.0.0.1 is an IP address', durationMs: 0n },
        { step: ConnectionCheckStep.TCP, status: ConnectionCheckStatus.FAILED, detail: 'refused', hint: 'Is the NATS server running?', durationMs: 4n },
        { step: ConnectionCheckStep.JETSTREAM, status: ConnectionCheckStatus.SKIPPED, detail: 'Not reached' },
      ],
    })

    expect(connectionChecks(resp)).toEqual([
      { step: 'dns', status: 'ok', detail: '127.0.0.1 is an IP address', hint: '', durationMs: 0 },
      { step: 'tcp', status: 'failed', detail: 'refused', hint: 'Is the NATS server running?', durationMs: 4 },
      { step: 'jetstream', status: 'skipped', detail: 'Not reached', hint: '', durationMs: 0 },
    ])
  })
})
