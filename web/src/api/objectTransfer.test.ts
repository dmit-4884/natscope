import { afterEach, describe, expect, it, vi } from 'vitest'
import { objectDownloadUrl, uploadObject } from './objectTransfer'

describe('object transfer', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('points a download at the object by connection, bucket and name', () => {
    const url = new URL(objectDownloadUrl('conn-1', 'BIG', 'reports/q3 é.bin'))

    expect(url.pathname).toBe('/api/objects')
    expect(Object.fromEntries(url.searchParams)).toEqual({ connection: 'conn-1', bucket: 'BIG', name: 'reports/q3 é.bin' })
  })

  it('puts the file itself as the body, without reading it first', async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(null, { status: 204 }))
    vi.stubGlobal('fetch', fetchMock)
    const file = new File(['x'], 'dump.bin')

    await uploadObject('conn-1', 'BIG', file, 'nightly')

    const [url, init] = fetchMock.mock.calls[0]
    expect(Object.fromEntries(new URL(url).searchParams)).toEqual({
      connection: 'conn-1',
      bucket: 'BIG',
      name: 'dump.bin',
      description: 'nightly',
    })
    expect(init).toMatchObject({ method: 'PUT', body: file })
  })

  it('fails with the message the server gave', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(Response.json({ message: 'connection is read-only' }, { status: 412 })))

    await expect(uploadObject('conn-1', 'BIG', new File(['x'], 'a'))).rejects.toThrow('connection is read-only')
  })
})
