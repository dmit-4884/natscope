import {
  SubjectPattern,
  type PatternValidationError,
} from '@/contexts/mappings/domain/value-objects/SubjectPattern'
import { SourcePicker } from '../proto/SourcePicker'
import { ProtoTypePicker } from '../proto/ProtoTypePicker'

const PATTERN_ERROR_MESSAGE: Record<PatternValidationError, string> = {
  EMPTY_PATTERN: 'Subject pattern is required.',
  INVALID_CHARACTERS: 'Only letters, numbers, ".", "_", "-", "*", ">" are allowed.',
  CONSECUTIVE_DOTS: 'Consecutive dots are not allowed.',
  TRAILING_DOT: 'A trailing dot is not allowed.',
  GREATER_NOT_LAST: '">" must be the last token.',
  PARTIAL_WILDCARD_TOKEN: '"*" and ">" must occupy a whole token, not share one with other text.',
}

interface Props {
  pattern: string
  onPatternChange: (value: string) => void
  protoType: string
  onProtoTypeChange: (value: string) => void
  sourceId: string
  onSourceIdChange: (value: string) => void
  unmappedPatterns: string[]
  onSubmit: () => void
  onCancel: () => void
  mode?: 'create' | 'edit'
  isSubmitting?: boolean
}

/**
 * MappingAddForm — three-step picker matching the source-aware decode model.
 *
 * Order is intentional: Source → Type → Pattern. Type list filters by source
 * (resolves the "same FQN in two sources" ambiguity at create time).
 */
export function MappingAddForm({
  pattern,
  onPatternChange,
  protoType,
  onProtoTypeChange,
  sourceId,
  onSourceIdChange,
  unmappedPatterns,
  onSubmit,
  onCancel,
  mode = 'create',
  isSubmitting = false,
}: Props) {
  const patternResult = pattern ? SubjectPattern.create(pattern) : null
  const patternError = patternResult?.isErr() ? PATTERN_ERROR_MESSAGE[patternResult.error] : undefined
  const canSubmit = !!pattern && !patternError && !!protoType && !!sourceId && !isSubmitting

  return (
    <div className="mt-4 p-4 bg-surface-primary rounded-lg border border-border space-y-4">
      <div className="text-sm font-semibold text-gray-700">
        {mode === 'edit' ? 'Edit mapping' : 'New mapping'}
      </div>
      <div>
        <label className="block text-xs font-medium text-gray-700 mb-1">
          1. Source <span className="text-rose-500">*</span>
        </label>
        <SourcePicker value={sourceId} onChange={onSourceIdChange} />
        {!sourceId && (
          <p className="text-xs text-content-tertiary mt-1">
            Pick a source first — proto types are scoped to it.
          </p>
        )}
      </div>

      <div>
        <label className="block text-xs font-medium text-gray-700 mb-1">
          2. Proto message type <span className="text-rose-500">*</span>
        </label>
        <ProtoTypePicker sourceId={sourceId} value={protoType} onChange={onProtoTypeChange} />
      </div>

      <div>
        <label className="block text-xs font-medium text-gray-700 mb-1">
          3. Subject pattern <span className="text-rose-500">*</span>
        </label>
        <input
          type="text"
          value={pattern}
          onChange={(e) => onPatternChange(e.target.value)}
          placeholder="e.g., events.user.*"
          aria-invalid={!!patternError}
          className={`w-full px-3 py-2 text-sm font-mono border rounded-md focus:ring-border-focus focus:border-border-focus ${
            patternError ? 'border-status-error-border' : 'border-border-strong'
          }`}
          list="available-patterns"
        />
        {unmappedPatterns.length > 0 && (
          <datalist id="available-patterns">
            {unmappedPatterns.map((p) => (
              <option key={p} value={p} />
            ))}
          </datalist>
        )}
        {patternError && <p className="text-xs text-status-error-text mt-1">{patternError}</p>}
      </div>

      <div className="flex justify-end gap-2 pt-2 border-t border-gray-100">
        <button
          onClick={onCancel}
          className="px-3 py-1.5 text-sm text-gray-700 hover:bg-surface-tertiary rounded-md transition-colors"
        >
          Cancel
        </button>
        <button
          onClick={onSubmit}
          disabled={!canSubmit}
          className="px-3 py-1.5 text-sm font-medium text-content-inverse bg-accent hover:bg-accent-hover disabled:bg-gray-300 disabled:cursor-not-allowed rounded-md transition-colors"
        >
          {mode === 'edit' ? 'Save changes' : 'Add mapping'}
        </button>
      </div>
    </div>
  )
}
