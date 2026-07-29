import { Alert } from './Alert'
import { WarningIcon } from './icons'

interface ErrorAlertProps {
  message: string
  className?: string
  compact?: boolean
}

export default function ErrorAlert({ message, className = '', compact = false }: ErrorAlertProps) {
  if (compact) {
    return (
      <div role="alert" className={`flex items-center gap-1.5 text-xs text-status-error-text ${className}`}>
        <WarningIcon className="w-3.5 h-3.5 shrink-0" />
        <span>{message}</span>
      </div>
    )
  }

  return (
    <Alert variant="error" className={className}>
      {message}
    </Alert>
  )
}
