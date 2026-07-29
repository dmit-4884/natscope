/**
 * Base64 -> UTF-8 string. Unlike atob() (Latin-1), handles multi-byte
 * sequences (Cyrillic, CJK, emoji).
 */
export function decodeBase64ToUtf8(base64: string): string {
  const binaryString = atob(base64)
  const bytes = new Uint8Array(binaryString.length)
  for (let i = 0; i < binaryString.length; i++) {
    bytes[i] = binaryString.charCodeAt(i)
  }
  return new TextDecoder('utf-8', { fatal: false }).decode(bytes)
}

/** Base64 -> raw bytes; for hex view / protobuf decoding. */
export function decodeBase64ToBytes(base64: string): Uint8Array<ArrayBuffer> {
  const binaryString = atob(base64)
  const bytes = new Uint8Array(binaryString.length)
  for (let i = 0; i < binaryString.length; i++) {
    bytes[i] = binaryString.charCodeAt(i)
  }
  return bytes
}

/** UTF-8/binary-safe base64 encode; chunked to avoid arg-count limits. */
export function encodeBytesToBase64(bytes: Uint8Array): string {
  let binary = ''
  const CHUNK = 0x8000
  for (let i = 0; i < bytes.length; i += CHUNK) {
    binary += String.fromCharCode(...bytes.subarray(i, i + CHUNK))
  }
  return btoa(binary)
}
