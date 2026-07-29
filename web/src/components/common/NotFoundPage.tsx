import { Link, useNavigate } from 'react-router-dom'

interface NotFoundPageProps {
  fullScreen?: boolean
}

export default function NotFoundPage({ fullScreen = false }: NotFoundPageProps) {
  const navigate = useNavigate()

  return (
    <div className={`flex-1 bg-surface-primary flex items-center justify-center p-4 ${fullScreen ? 'min-h-screen' : ''}`}>
      <div className="text-center">
        <div className="w-16 h-16 mx-auto mb-4 rounded-full bg-surface-tertiary flex items-center justify-center">
          <svg className="w-8 h-8 text-content-muted" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M9.172 16.172a4 4 0 015.656 0M9 10h.01M15 10h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
        </div>

        <h1 className="text-5xl font-bold text-gray-200 mb-1">404</h1>
        <h2 className="text-lg font-semibold text-content-primary mb-1">Page Not Found</h2>
        <p className="text-sm text-content-tertiary mb-6 max-w-sm">
          The page you're looking for doesn't exist or has been moved.
        </p>

        <div className="flex items-center justify-center gap-3">
          <button
            onClick={() => navigate(-1)}
            className="px-4 py-2 text-sm font-medium text-gray-700 bg-surface-primary border border-border-strong rounded-md hover:bg-surface-secondary transition-colors"
          >
            Go Back
          </button>
          <Link
            to="/"
            className="px-4 py-2 text-sm font-medium text-content-inverse bg-accent rounded-md hover:bg-accent-hover transition-colors"
          >
            Go Home
          </Link>
        </div>
      </div>
    </div>
  )
}
