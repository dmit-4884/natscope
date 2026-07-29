import { ValueObject, Result, DomainError } from '@/shared'

export type FetchMethod = 'direct' | 'consumer'
export type FetchDirection = 'backward' | 'forward'
export type ExportFormat = 'json' | 'ndjson' | 'csv'

// Removed (not user-tunable): maxSearchRange / fetchConcurrency — never read
// by backend; concurrency now from runtime.NumCPU().
export interface MessageFetchPolicyProps {
  fetchMethod: FetchMethod
  defaultPageSize: number
  defaultDirection: FetchDirection
  // Per-message payload cap (BYTES) for list/live previews. 0 = unlimited;
  // detail panel always loads full payload. Default 65536 (64 KB).
  maxPayloadBytesInList: number
  /** Format pre-selected in the message export dialog. Default 'json'. */
  defaultExportFormat: ExportFormat
  // Hard cap on messages fetched by a full-range export. Always positive —
  // 0 coerced to default. Default 50000.
  exportRangeLimit: number
}

// Loose input DTO for the adapter — proto-shape may be missing fields.
export interface MessageFetchPolicyInput {
  fetchMethod?: string
  defaultPageSize?: number
  defaultDirection?: string
  maxPayloadBytesInList?: number
  defaultExportFormat?: string
  exportRangeLimit?: number
}

const VALID_METHODS: readonly FetchMethod[] = ['direct', 'consumer']
const VALID_DIRECTIONS: readonly FetchDirection[] = ['backward', 'forward']
const VALID_EXPORT_FORMATS: readonly ExportFormat[] = ['json', 'ndjson', 'csv']

/** Default ceiling for a full-range export. Mirrors the server-side default. */
const DEFAULT_EXPORT_RANGE_LIMIT = 50000

export const MESSAGE_FETCH_POLICY_DEFAULTS: MessageFetchPolicyProps = Object.freeze({
  fetchMethod: 'consumer',
  defaultPageSize: 50,
  defaultDirection: 'backward',
  maxPayloadBytesInList: 64 * 1024,
  defaultExportFormat: 'json',
  exportRangeLimit: DEFAULT_EXPORT_RANGE_LIMIT,
})

export class MessageFetchPolicy extends ValueObject<MessageFetchPolicyProps> {
  static readonly DEFAULTS = MESSAGE_FETCH_POLICY_DEFAULTS

  private constructor(props: MessageFetchPolicyProps) {
    super(props)
  }

  get fetchMethod(): FetchMethod {
    return this.props.fetchMethod
  }
  get defaultPageSize(): number {
    return this.props.defaultPageSize
  }
  get defaultDirection(): FetchDirection {
    return this.props.defaultDirection
  }
  get maxPayloadBytesInList(): number {
    return this.props.maxPayloadBytesInList
  }
  get defaultExportFormat(): ExportFormat {
    return this.props.defaultExportFormat
  }
  get exportRangeLimit(): number {
    return this.props.exportRangeLimit
  }

  toObject(): MessageFetchPolicyProps {
    return { ...this.props }
  }

  /** Immutable update with validation; loose form-boundary input. */
  merge(patch: MessageFetchPolicyInput): Result<MessageFetchPolicy, DomainError> {
    return MessageFetchPolicy.create({
      fetchMethod: (patch.fetchMethod ?? this.props.fetchMethod) as FetchMethod,
      defaultPageSize: patch.defaultPageSize ?? this.props.defaultPageSize,
      defaultDirection: (patch.defaultDirection ?? this.props.defaultDirection) as FetchDirection,
      maxPayloadBytesInList: patch.maxPayloadBytesInList ?? this.props.maxPayloadBytesInList,
      defaultExportFormat: (patch.defaultExportFormat ?? this.props.defaultExportFormat) as ExportFormat,
      exportRangeLimit: patch.exportRangeLimit ?? this.props.exportRangeLimit,
    })
  }

  /** Returns the VO with defaults. */
  static default(): MessageFetchPolicy {
    return new MessageFetchPolicy({ ...MESSAGE_FETCH_POLICY_DEFAULTS })
  }

  /** Validates and constructs a MessageFetchPolicy. */
  static create(
    input: Partial<MessageFetchPolicyProps>,
  ): Result<MessageFetchPolicy, DomainError> {
    const next: MessageFetchPolicyProps = {
      ...MESSAGE_FETCH_POLICY_DEFAULTS,
      ...input,
    }
    if (!VALID_METHODS.includes(next.fetchMethod)) {
      return Result.err(DomainError.validation('fetchMethod must be one of: direct, consumer', 'fetchMethod'))
    }
    if (!Number.isInteger(next.defaultPageSize) || next.defaultPageSize < 1 || next.defaultPageSize > 500) {
      return Result.err(DomainError.validation('defaultPageSize must be an integer in [1, 500]', 'defaultPageSize'))
    }
    if (!VALID_DIRECTIONS.includes(next.defaultDirection)) {
      return Result.err(DomainError.validation('defaultDirection must be one of: backward, forward', 'defaultDirection'))
    }
    if (
      !Number.isInteger(next.maxPayloadBytesInList) ||
      next.maxPayloadBytesInList < 0 ||
      next.maxPayloadBytesInList > 64 * 1024 * 1024
    ) {
      return Result.err(
        DomainError.validation('maxPayloadBytesInList must be a byte count in [0, 64 MiB]', 'maxPayloadBytesInList'),
      )
    }
    if (!VALID_EXPORT_FORMATS.includes(next.defaultExportFormat)) {
      return Result.err(
        DomainError.validation('defaultExportFormat must be one of: json, ndjson, csv', 'defaultExportFormat'),
      )
    }
    if (
      !Number.isInteger(next.exportRangeLimit) ||
      next.exportRangeLimit < 1 ||
      next.exportRangeLimit > 10_000_000
    ) {
      return Result.err(
        DomainError.validation('exportRangeLimit must be an integer in [1, 10000000]', 'exportRangeLimit'),
      )
    }
    return Result.ok(new MessageFetchPolicy(next))
  }

  /** Proto-shaped partial -> policy; coerces unknowns to defaults. */
  static fromPartial(input?: MessageFetchPolicyInput | null): MessageFetchPolicy {
    if (!input) return MessageFetchPolicy.default()
    const method = (input.fetchMethod ?? MESSAGE_FETCH_POLICY_DEFAULTS.fetchMethod) as FetchMethod
    const direction = (input.defaultDirection ?? MESSAGE_FETCH_POLICY_DEFAULTS.defaultDirection) as FetchDirection
    const exportFormat = (input.defaultExportFormat ?? MESSAGE_FETCH_POLICY_DEFAULTS.defaultExportFormat) as ExportFormat
    const rangeLimit = input.exportRangeLimit ?? MESSAGE_FETCH_POLICY_DEFAULTS.exportRangeLimit
    return new MessageFetchPolicy({
      fetchMethod: VALID_METHODS.includes(method) ? method : MESSAGE_FETCH_POLICY_DEFAULTS.fetchMethod,
      defaultPageSize: input.defaultPageSize ?? MESSAGE_FETCH_POLICY_DEFAULTS.defaultPageSize,
      defaultDirection: VALID_DIRECTIONS.includes(direction) ? direction : MESSAGE_FETCH_POLICY_DEFAULTS.defaultDirection,
      maxPayloadBytesInList:
        input.maxPayloadBytesInList ?? MESSAGE_FETCH_POLICY_DEFAULTS.maxPayloadBytesInList,
      defaultExportFormat: VALID_EXPORT_FORMATS.includes(exportFormat)
        ? exportFormat
        : MESSAGE_FETCH_POLICY_DEFAULTS.defaultExportFormat,
      // Same bounds as create() [1, 10_000_000] so fromPartial never yields a
      // value create() would reject.
      exportRangeLimit:
        Number.isInteger(rangeLimit) && rangeLimit >= 1 && rangeLimit <= 10_000_000
          ? rangeLimit
          : MESSAGE_FETCH_POLICY_DEFAULTS.exportRangeLimit,
    })
  }
}
