export function matchSubject(subject: string, pattern: string): boolean {
  const subjectParts = subject.split('.')
  const patternParts = pattern.split('.')

  let si = 0
  let pi = 0

  while (si < subjectParts.length && pi < patternParts.length) {
    const patternPart = patternParts[pi]

    if (patternPart === '>') return true

    if (patternPart === '*') {
      si++
      pi++
      continue
    }

    if (subjectParts[si] !== patternPart) return false

    si++
    pi++
  }

  return si === subjectParts.length && pi === patternParts.length
}

export function coversSubject(filter: string, pattern: string): boolean {
  const filterParts = filter.split('.')
  const patternParts = pattern.split('.')

  for (let i = 0; i < filterParts.length; i++) {
    const filterPart = filterParts[i]
    if (filterPart === '>') return i < patternParts.length
    if (i >= patternParts.length || patternParts[i] === '>') return false
    if (filterPart !== '*' && filterPart !== patternParts[i]) return false
  }

  return filterParts.length === patternParts.length
}
