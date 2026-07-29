const SUBJECT_COLORS: Record<string, { bg: string; border: string; text: string }> = {
  error: { bg: 'bg-status-error-bg', border: 'border-l-red-500', text: 'text-red-700' },
  warn: { bg: 'bg-yellow-50', border: 'border-l-yellow-500', text: 'text-yellow-700' },
  info: { bg: 'bg-accent-light', border: 'border-l-blue-500', text: 'text-accent-text' },
  debug: { bg: 'bg-surface-secondary', border: 'border-l-gray-400', text: 'text-content-secondary' },
  success: { bg: 'bg-green-50', border: 'border-l-green-500', text: 'text-green-700' },
  event: { bg: 'bg-purple-50', border: 'border-l-purple-500', text: 'text-purple-700' },
  command: { bg: 'bg-indigo-50', border: 'border-l-indigo-500', text: 'text-indigo-700' },
}

export function getSubjectColor(subject: string): { bg: string; border: string; text: string } | null {
  const lowerSubject = subject.toLowerCase()
  for (const [pattern, colors] of Object.entries(SUBJECT_COLORS)) {
    if (lowerSubject.includes(pattern)) {
      return colors
    }
  }
  return null
}
