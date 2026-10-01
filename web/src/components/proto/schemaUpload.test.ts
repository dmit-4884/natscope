import { describe, expect, it } from 'vitest'
import { pickedFromList, prepareUpload } from './schemaUpload'

const file = (content: string, name: string, relative = '') => {
  const f = new File([content], name)
  if (relative) Object.defineProperty(f, 'webkitRelativePath', { value: relative })
  return f
}

describe('prepareUpload', () => {
  it('sends a single descriptor set as bytes', async () => {
    const upload = await prepareUpload(pickedFromList([file('\u0001\u0002', 'schema.binpb')]))
    expect(upload?.content).toEqual({ kind: 'descriptorSet', data: new Uint8Array([1, 2]) })
    expect(upload?.summary).toMatch(/^Descriptor set schema\.binpb · /)
  })

  it('keeps folder paths, buf configs and skips everything else', async () => {
    const upload = await prepareUpload(
      pickedFromList([
        file('a', 'order.proto', 'protos/shop/order.proto'),
        file('v2', 'buf.yaml', 'protos/buf.yaml'),
        file('x', 'HEAD', 'protos/.git/HEAD'),
        file('y', 'dep.proto', 'protos/node_modules/dep.proto'),
        file('z', 'README.md', 'protos/README.md'),
      ]),
    )
    expect(upload?.content).toEqual({
      kind: 'files',
      files: [
        { path: 'protos/shop/order.proto', content: 'a' },
        { path: 'protos/buf.yaml', content: 'v2' },
      ],
    })
    expect(upload?.summary).toBe('1 .proto file · 1 buf config · 3 other skipped')
  })

  it('treats a descriptor set among other files as nothing to compile', async () => {
    expect(await prepareUpload(pickedFromList([file('x', 'schema.binpb'), file('y', 'notes.txt')]))).toBeNull()
  })

  it('returns null without .proto files', async () => {
    expect(await prepareUpload(pickedFromList([file('version: v2', 'buf.yaml')]))).toBeNull()
  })
})
