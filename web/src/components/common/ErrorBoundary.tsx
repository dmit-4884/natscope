import { Component } from 'react'
import type { ErrorInfo, ReactNode } from 'react'
import { WarningIcon } from '@/components/ui'

interface Props {
  children: ReactNode
  fallback?: ReactNode
}

interface State {
  hasError: boolean
  error: Error | null
}

/**
 * Error Boundary component to catch rendering errors and display a fallback UI.
 * Prevents the entire app from crashing due to component errors.
 */
export class ErrorBoundary extends Component<Props, State> {
  state: State = { hasError: false, error: null }

  static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error }
  }

  componentDidCatch(error: Error, errorInfo: ErrorInfo) {
    console.error('ErrorBoundary caught an error:', error, errorInfo)
  }

  handleReset = () => {
    this.setState({ hasError: false, error: null })
  }

  render() {
    if (this.state.hasError) {
      if (this.props.fallback) {
        return this.props.fallback
      }

      return (
        <div className="min-h-screen flex items-center justify-center bg-surface-secondary">
          <div className="max-w-md w-full p-8 bg-surface-primary rounded-lg shadow-md">
            <div className="w-12 h-12 mx-auto mb-4 rounded-full bg-status-error-light flex items-center justify-center">
              <WarningIcon className="w-6 h-6 text-status-error-text" />
            </div>
            <h1 className="text-xl font-bold text-content-primary text-center mb-2">
              Something went wrong
            </h1>
            <p className="text-content-secondary text-center mb-4">
              An unexpected error occurred. Please try again.
            </p>
            {this.state.error?.message && (
              <div className="mb-4 p-3 bg-surface-tertiary rounded text-sm text-gray-700 font-mono overflow-auto max-h-32">
                {this.state.error.message}
              </div>
            )}
            <div className="flex gap-3 justify-center">
              <button
                onClick={this.handleReset}
                className="px-4 py-2 bg-blue-500 text-content-inverse rounded hover:bg-accent transition-colors"
              >
                Try again
              </button>
              <button
                onClick={() => window.location.reload()}
                className="px-4 py-2 bg-surface-hover text-gray-700 rounded hover:bg-gray-300 transition-colors"
              >
                Reload page
              </button>
            </div>
          </div>
        </div>
      )
    }

    return this.props.children
  }
}
