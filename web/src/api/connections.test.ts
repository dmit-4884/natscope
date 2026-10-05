import { describe, it, expect, vi, beforeEach } from 'vitest'
import { create } from '@bufbuild/protobuf'
import {
  CreateConnectionResponseSchema,
  ListConnectionsResponseSchema,
  UpdateConnectionResponseSchema,
} from '../gen/services/grpc/nats/v1/connections/nats_connections_service_pb'
import { LabelColor } from '../gen/types/nats/nats_connection_pb'

const client = vi.hoisted(() => ({
  listConnections: vi.fn(),
  createConnection: vi.fn(),
  updateConnection: vi.fn(),
}))

vi.mock('./grpc/clients', () => ({ connectionsClient: client }))

import { createConnection, getConnections, updateConnection } from './connections'

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
