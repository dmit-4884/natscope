import { describe, it, expect, vi } from 'vitest'
import type { ReactNode } from 'react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { renderHook, waitFor } from '@/test/utils'
import { NO_FRAMING } from '@/api/framing'
import { useMappingImportExport } from './useMappingImportExport'

const hoisted = vi.hoisted(() => ({ listMappings: vi.fn(), bulkSaveMappings: vi.fn() }))

vi.mock('@/api/mappings', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/api/mappings')>()),
  ...hoisted,
}))

function wrapper({ children }: { children: ReactNode }) {
  return <QueryClientProvider client={new QueryClient()}>{children}</QueryClientProvider>
}

describe('useMappingImportExport', () => {
  it('exports framing as hex and reads it back', async () => {
    hoisted.listMappings.mockResolvedValue({
      total: 2,
      items: [
        {
          id: 'm1', pattern: 'orders.>', messageType: 'shop.Order', sourceId: 's', createdAt: 0, updatedAt: 0,
          framing: { kind: 'custom', schemaId: 0, prefix: new Uint8Array([0xca, 0xfe]), suffix: new Uint8Array() },
        },
        { id: 'm2', pattern: 'refunds.>', messageType: 'shop.Refund', sourceId: 's', createdAt: 0, updatedAt: 0, framing: NO_FRAMING },
      ],
    })
    const writeText = vi.mocked(navigator.clipboard.writeText)
    const { result } = renderHook(() => useMappingImportExport(), { wrapper })
    await waitFor(() => expect(result.current.items).toHaveLength(2))

    await result.current.exportToClipboard()
    const exported = writeText.mock.calls.at(-1)?.[0] ?? ''
    expect(JSON.parse(exported).mappings).toEqual([
      { pattern: 'orders.>', messageType: 'shop.Order', framing: { kind: 'custom', prefixHex: 'cafe' } },
      { pattern: 'refunds.>', messageType: 'shop.Refund' },
    ])

    const parsed = result.current.parseText(exported)
    expect(parsed).toEqual({
      ok: true,
      rows: [
        {
          pattern: 'orders.>',
          messageType: 'shop.Order',
          framing: { kind: 'custom', schemaId: 0, prefix: new Uint8Array([0xca, 0xfe]), suffix: new Uint8Array() },
        },
        { pattern: 'refunds.>', messageType: 'shop.Refund', framing: undefined },
      ],
    })
  })

  it('rejects a broken framing', async () => {
    hoisted.listMappings.mockResolvedValue({ total: 0, items: [] })
    const { result } = renderHook(() => useMappingImportExport(), { wrapper })
    const parsed = result.current.parseText(
      JSON.stringify({ version: 3, mappings: [{ pattern: 'a.>', messageType: 'x.Y', framing: { kind: 'zstd' } }] }),
    )
    expect(parsed).toEqual({ ok: false, error: 'mappings[0].framing is not valid.' })
  })
})
