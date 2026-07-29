import { ValueObject, Result, DomainError } from '@/shared'

export type Density = 'comfortable' | 'compact'
export type ViewMode = 'history' | 'realtime'
export type TimestampFormat = 'relative' | 'absolute' | 'iso'
export type JsonIndentSize = 2 | 4

export interface DisplayPreferencesProps {
  density: Density
  defaultViewMode: ViewMode
  timestampFormat: TimestampFormat
  jsonIndentSize: JsonIndentSize
  autoScrollLive: boolean
}

export interface DisplayPreferencesInput {
  density?: string
  defaultViewMode?: string
  timestampFormat?: string
  jsonIndentSize?: number
  autoScrollLive?: boolean
}

const VALID_DENSITIES: readonly Density[] = ['comfortable', 'compact']
const VALID_VIEW_MODES: readonly ViewMode[] = ['history', 'realtime']
const VALID_TS_FORMATS: readonly TimestampFormat[] = ['relative', 'absolute', 'iso']
const VALID_INDENTS: readonly JsonIndentSize[] = [2, 4]

const DISPLAY_PREFERENCES_DEFAULTS: DisplayPreferencesProps = Object.freeze({
  density: 'comfortable',
  defaultViewMode: 'history',
  timestampFormat: 'relative',
  jsonIndentSize: 2,
  autoScrollLive: true,
})

export class DisplayPreferences extends ValueObject<DisplayPreferencesProps> {
  static readonly DEFAULTS = DISPLAY_PREFERENCES_DEFAULTS

  private constructor(props: DisplayPreferencesProps) {
    super(props)
  }

  get density(): Density {
    return this.props.density
  }
  get defaultViewMode(): ViewMode {
    return this.props.defaultViewMode
  }
  get timestampFormat(): TimestampFormat {
    return this.props.timestampFormat
  }
  get jsonIndentSize(): JsonIndentSize {
    return this.props.jsonIndentSize
  }
  get autoScrollLive(): boolean {
    return this.props.autoScrollLive
  }

  toObject(): DisplayPreferencesProps {
    return { ...this.props }
  }

  merge(patch: DisplayPreferencesInput): Result<DisplayPreferences, DomainError> {
    return DisplayPreferences.create({
      density: (patch.density ?? this.props.density) as Density,
      defaultViewMode: (patch.defaultViewMode ?? this.props.defaultViewMode) as ViewMode,
      timestampFormat: (patch.timestampFormat ?? this.props.timestampFormat) as TimestampFormat,
      jsonIndentSize: (patch.jsonIndentSize ?? this.props.jsonIndentSize) as JsonIndentSize,
      autoScrollLive: patch.autoScrollLive ?? this.props.autoScrollLive,
    })
  }

  static default(): DisplayPreferences {
    return new DisplayPreferences({ ...DISPLAY_PREFERENCES_DEFAULTS })
  }

  static create(
    input: Partial<DisplayPreferencesProps>,
  ): Result<DisplayPreferences, DomainError> {
    const next: DisplayPreferencesProps = {
      ...DISPLAY_PREFERENCES_DEFAULTS,
      ...input,
    }
    if (!VALID_DENSITIES.includes(next.density)) {
      return Result.err(DomainError.validation('density must be comfortable or compact', 'density'))
    }
    if (!VALID_VIEW_MODES.includes(next.defaultViewMode)) {
      return Result.err(DomainError.validation('defaultViewMode must be history or realtime', 'defaultViewMode'))
    }
    if (!VALID_TS_FORMATS.includes(next.timestampFormat)) {
      return Result.err(DomainError.validation('timestampFormat must be relative, absolute, or iso', 'timestampFormat'))
    }
    if (!VALID_INDENTS.includes(next.jsonIndentSize)) {
      return Result.err(DomainError.validation('jsonIndentSize must be 2 or 4', 'jsonIndentSize'))
    }
    if (typeof next.autoScrollLive !== 'boolean') {
      return Result.err(DomainError.validation('autoScrollLive must be a boolean', 'autoScrollLive'))
    }
    return Result.ok(new DisplayPreferences(next))
  }

  static fromPartial(input?: DisplayPreferencesInput | null): DisplayPreferences {
    if (!input) return DisplayPreferences.default()
    const density = (input.density ?? DISPLAY_PREFERENCES_DEFAULTS.density) as Density
    const viewMode = (input.defaultViewMode ?? DISPLAY_PREFERENCES_DEFAULTS.defaultViewMode) as ViewMode
    const tsFormat = (input.timestampFormat ?? DISPLAY_PREFERENCES_DEFAULTS.timestampFormat) as TimestampFormat
    const indent = (input.jsonIndentSize ?? DISPLAY_PREFERENCES_DEFAULTS.jsonIndentSize) as JsonIndentSize
    return new DisplayPreferences({
      density: VALID_DENSITIES.includes(density) ? density : DISPLAY_PREFERENCES_DEFAULTS.density,
      defaultViewMode: VALID_VIEW_MODES.includes(viewMode) ? viewMode : DISPLAY_PREFERENCES_DEFAULTS.defaultViewMode,
      timestampFormat: VALID_TS_FORMATS.includes(tsFormat) ? tsFormat : DISPLAY_PREFERENCES_DEFAULTS.timestampFormat,
      jsonIndentSize: VALID_INDENTS.includes(indent) ? indent : DISPLAY_PREFERENCES_DEFAULTS.jsonIndentSize,
      autoScrollLive: input.autoScrollLive ?? DISPLAY_PREFERENCES_DEFAULTS.autoScrollLive,
    })
  }
}
