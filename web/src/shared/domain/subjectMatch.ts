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
