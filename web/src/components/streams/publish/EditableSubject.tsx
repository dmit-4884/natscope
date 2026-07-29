import type { PatternSegment } from './subjectPatternUtils'

interface Props {
  segments: PatternSegment[]
  wildcardValues: string[]
  onWildcardChange: (index: number, value: string) => void
}

export function EditableSubject({ segments, wildcardValues, onWildcardChange }: Props) {
  const hasEmpty = wildcardValues.some((v) => !v.trim())

  return (
    <div>
      <label className="block text-sm font-medium text-gray-700 mb-2">Subject</label>
      <div className="flex items-center flex-wrap gap-0 px-3 py-2 border border-border-strong rounded-lg bg-surface-primary font-mono text-sm">
        {segments.map((segment, i) => {
          if (segment.type === 'static') {
            return (
              <span key={i} className="text-content-secondary whitespace-nowrap">
                {segment.value}
              </span>
            )
          }
          const wildcardIdx = segment.index!
          const value = wildcardValues[wildcardIdx] || ''
          const isEmpty = !value.trim()
          return (
            <input
              key={i}
              type="text"
              value={value}
              onChange={(e) => onWildcardChange(wildcardIdx, e.target.value)}
              placeholder="*"
              className={`min-w-24 max-w-64 px-1.5 py-0.5 text-sm font-mono border rounded focus:ring-2 focus:ring-border-focus focus:border-border-focus ${
                isEmpty ? 'border-orange-300 bg-orange-50' : 'border-border-strong bg-surface-secondary'
              }`}
            />
          )
        })}
      </div>
      {hasEmpty && (
        <p className="mt-1 text-xs text-orange-600">Fill in all values (e.g., user ID, transaction ID)</p>
      )}
    </div>
  )
}
