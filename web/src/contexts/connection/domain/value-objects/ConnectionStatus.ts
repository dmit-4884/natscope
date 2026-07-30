export type ReportedHealthStatus = 'connected' | 'reconnecting' | 'disconnected'

export type ConnectionStatus = 'connecting' | ReportedHealthStatus

export interface ConnectionStatusInput {
  reported?: ReportedHealthStatus
  isPending: boolean
  hasError: boolean
}

export function resolveConnectionStatus({
  reported,
  isPending,
  hasError,
}: ConnectionStatusInput): ConnectionStatus {
  if (hasError) return 'disconnected'
  if (isPending) return 'connecting'
  return reported ?? 'disconnected'
}
