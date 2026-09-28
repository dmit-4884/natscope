// RFC 7230 "token" characters — the header-name grammar NATS headers use
// (serialized as HTTP-style headers over the wire). A key outside this set
// is silently dropped on encode instead of reaching the peer.
const HEADER_NAME_RE = /^[!#$%&'*+\-.^_`|~0-9A-Za-z]+$/

export function isValidHeaderName(key: string): boolean {
  return HEADER_NAME_RE.test(key)
}
