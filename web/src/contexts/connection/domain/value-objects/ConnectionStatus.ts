export type ReportedHealthStatus = 'connected' | 'reconnecting' | 'disconnected'

export type ConnectionStatus = 'connecting' | ReportedHealthStatus

export interface ConnectionStatusInput {
  reported?: ReportedHealthStatus
  isPending: boolean
  hasError: boolean
  isTransientError?: boolean
}

export function resolveConnectionStatus({
  reported,
  isPending,
  hasError,
  isTransientError = false,
}: ConnectionStatusInput): ConnectionStatus {
  if (isTransientError) return 'reconnecting'
  if (hasError) return 'disconnected'
  if (isPending) return 'connecting'
  return reported ?? 'disconnected'
}
