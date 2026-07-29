import { describe, it, expect } from 'vitest'
import { encodeBytesToBase64 } from '@/utils/base64'
import { buildResendDraft } from './resend'

// resolvePattern stub: matches by token count (real one uses NATS wildcards).
const patterns = ['e2e.publish.*.*', 'e2e.plain']
const resolve = (subject: string): string => {
  const tokens = subject.split('.')
  return patterns.find((p) => p.split('.').length === tokens.length && p.split('.')[1] === tokens[1]) ?? subject
}

const b64 = (s: string) => encodeBytesToBase64(new TextEncoder().encode(s))

describe('buildResendDraft', () => {
  it('extracts wildcard values from the concrete subject', () => {
    const d = buildResendDraft({ subject: 'e2e.publish.eu.created', dataBase64: b64('{}') }, resolve)
    expect(d.pattern).toBe('e2e.publish.*.*')
    expect(d.wildcards).toEqual(['eu', 'created'])
    expect(d.binaryUndecodable).toBe(false)
  })

  it('returns no wildcards for a literal subject pattern', () => {
    const d = buildResendDraft({ subject: 'e2e.plain', dataBase64: b64('{}') }, resolve)
    expect(d.pattern).toBe('e2e.plain')
    expect(d.wildcards).toEqual([])
  })

  it('prefers a decoded object, pretty-printed', () => {
    const d = buildResendDraft({ subject: 'e2e.plain', decoded: { a: 1, b: 'x' } }, resolve)
    expect(d.messageJson).toBe('{\n  "a": 1,\n  "b": "x"\n}')
  })

  it('falls back to parsing raw base64 JSON', () => {
    const d = buildResendDraft({ subject: 'e2e.plain', dataBase64: b64('{"n":2}') }, resolve)
    expect(d.messageJson).toBe('{\n  "n": 2\n}')
  })

  it('keeps non-JSON raw payloads verbatim', () => {
    const d = buildResendDraft({ subject: 'e2e.plain', dataBase64: b64('hello world') }, resolve)
    expect(d.messageJson).toBe('hello world')
  })

  it('treats a decoded string (truncated) as non-object → raw fallback', () => {
    const d = buildResendDraft({ subject: 'e2e.plain', decoded: '{"trunc', dataBase64: b64('{"n":3}') }, resolve)
    expect(d.messageJson).toBe('{\n  "n": 3\n}')
  })

  it('drops NATS-reserved headers but keeps user headers', () => {
    const d = buildResendDraft(
      {
        subject: 'e2e.plain',
        dataBase64: b64('{}'),
        headers: { 'X-Trace': 'abc', 'Nats-Msg-Id': 'dedup-1', 'nats-sequence': '9' },
      },
      resolve,
    )
    expect(d.headers).toEqual([{ key: 'X-Trace', value: 'abc' }])
  })

  it('returns empty headers when the message has none', () => {
    const d = buildResendDraft({ subject: 'e2e.plain', dataBase64: b64('{}') }, resolve)
    expect(d.headers).toEqual([])
  })

  it('preserves valid UTF-8 multibyte payloads (no replacement chars)', () => {
    const d = buildResendDraft({ subject: 'e2e.plain', dataBase64: b64('héllo →') }, resolve)
    expect(d.binaryUndecodable).toBe(false)
    expect(d.messageJson).toBe('héllo →')
  })

  it('flags a binary (non-UTF-8) payload as undecodable and emits no text', () => {
    // Invalid UTF-8 must be rejected, not turned into U+FFFD chars that corrupt
    // resend.
    const binaryB64 = btoa(String.fromCharCode(0xff, 0xfe, 0x00, 0x80))
    const d = buildResendDraft({ subject: 'e2e.plain', dataBase64: binaryB64 }, resolve)
    expect(d.binaryUndecodable).toBe(true)
    expect(d.messageJson).toBe('')
    // wildcards/headers still resolve normally so the caller can warn cleanly.
    expect(d.pattern).toBe('e2e.plain')
  })

  it('prefers the decoded object even when raw bytes are binary', () => {
    const binaryB64 = btoa(String.fromCharCode(0xff, 0xfe))
    const d = buildResendDraft(
      { subject: 'e2e.plain', dataBase64: binaryB64, decoded: { ok: true } },
      resolve,
    )
    expect(d.binaryUndecodable).toBe(false)
    expect(d.messageJson).toBe('{\n  "ok": true\n}')
  })

  it('treats a missing payload as empty, not binary', () => {
    const d = buildResendDraft({ subject: 'e2e.plain' }, resolve)
    expect(d.binaryUndecodable).toBe(false)
    expect(d.messageJson).toBe('')
  })
})
