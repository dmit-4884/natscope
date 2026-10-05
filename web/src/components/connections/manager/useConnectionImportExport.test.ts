import { describe, it, expect, vi } from 'vitest'
import { renderHook, waitFor } from '@testing-library/react'
import type { MappingItem } from '@/contexts/mappings'
import { useConnectionImportExport } from './useConnectionImportExport'

const hoisted = vi.hoisted(() => ({ downloadBlob: vi.fn() }))
vi.mock('@/utils/download', () => hoisted)

const grpcMapping: MappingItem = {
  id: 'm1',
  pattern: 'orders.>',
  messageType: 'shop.Order',
  sourceId: 'src-1',
  framing: { kind: 'custom', schemaId: 0, prefix: new Uint8Array([0xca, 0xfe]), suffix: new Uint8Array() },
  createdAt: 0,
  updatedAt: 0,
}

describe('useConnectionImportExport', () => {
  it('round-trips mapping framing', async () => {
    const bulkSaveMappings = vi.fn().mockResolvedValue(undefined)
    const { result } = renderHook(() =>
      useConnectionImportExport({ connections: [], mappings: [grpcMapping], createConnection: vi.fn(), bulkSaveMappings }),
    )

    result.current.handleExport()
    const exported = hoisted.downloadBlob.mock.lastCall?.[0] as string
    expect(JSON.parse(exported).mappings[0].framing).toEqual({ kind: 'custom', prefixHex: 'cafe' })

    const config = JSON.parse(exported)
    config.mappings.push({ pattern: 'bad.>', messageType: 'x.Y', sourceId: 'src-1', framing: { kind: 'zip' } })
    const file = new File([JSON.stringify(config)], 'config.json')
    result.current.handleImport({ target: { files: [file] } } as unknown as React.ChangeEvent<HTMLInputElement>)

    await waitFor(() => expect(bulkSaveMappings).toHaveBeenCalled())
    expect(bulkSaveMappings.mock.lastCall?.[0]).toEqual([
      { pattern: 'orders.>', messageType: 'shop.Order', sourceId: 'src-1', framing: grpcMapping.framing },
    ])
  })

  it('keeps a connection read-only and labelled on import', async () => {
    const createConnection = vi.fn().mockResolvedValue(undefined)
    const { result } = renderHook(() =>
      useConnectionImportExport({ connections: [], mappings: [], createConnection, bulkSaveMappings: vi.fn() }),
    )
    const config = {
      version: 2,
      connections: [{ name: 'prod', urls: ['nats://prod:4222'], readOnly: true, label: { text: 'PROD', color: 'red' } }],
    }

    result.current.handleImport({
      target: { files: [new File([JSON.stringify(config)], 'config.json')] },
    } as unknown as React.ChangeEvent<HTMLInputElement>)

    await waitFor(() => expect(createConnection).toHaveBeenCalled())
    expect(createConnection.mock.lastCall?.[0]).toMatchObject({ readOnly: true, label: { text: 'PROD', color: 'red' } })
  })
})
