const HEADER_NAME_RE = /^[!#$%&'*+\-.^_`|~0-9A-Za-z]+$/

export function isValidHeaderName(key: string): boolean {
  return HEADER_NAME_RE.test(key)
}
