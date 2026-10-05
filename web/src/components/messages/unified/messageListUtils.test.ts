import { describe, expect, it } from 'vitest'
import { encodeBytesToBase64 } from '@/utils/base64'
import { livePayloadPreview } from './messageListUtils'

const b64 = (text: string) => encodeBytesToBase64(new TextEncoder().encode(text))

describe('livePayloadPreview', () => {
  it('shows a decoded payload as compact JSON', () => {
    expect(livePayloadPreview({ decoded: { id: 7, ok: true }, data_base64: b64('raw') })).toBe('{"id":7,"ok":true}')
  })

  it('shows JSON text compacted and plain text on one line', () => {
    expect(livePayloadPreview({ data_base64: b64('{ "a": 1,\n  "b": 2 }') })).toBe('{"a":1,"b":2}')
    expect(livePayloadPreview({ data_base64: b64('hello\n  world') })).toBe('hello world')
  })

  it('cuts long payloads', () => {
    expect(livePayloadPreview({ data_base64: b64('x'.repeat(300)) }, 10)).toBe('xxxxxxxxxx...')
  })

  it('is empty without a payload', () => {
    expect(livePayloadPreview({ data_base64: '' })).toBe('')
  })
})
