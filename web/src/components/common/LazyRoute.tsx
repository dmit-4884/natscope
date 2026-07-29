import { Suspense, type ReactNode } from 'react'

/**
 * Suspense wrapper used by every code-split route in the router. Renders a
 * small spinner while the chunk loads. Lives in its own file so the router
 * config (`router.tsx`) only exports the `router` constant — keeps fast
 * refresh working in dev for the router file.
 */
export function LazyRoute({ children }: { children: ReactNode }) {
  return (
    <Suspense
      fallback={
        <div className="p-4 text-center">
          <div className="w-8 h-8 mx-auto mb-2 rounded-full bg-surface-tertiary flex items-center justify-center">
            <svg
              className="w-4 h-4 text-content-muted animate-spin"
              fill="none"
              viewBox="0 0 24 24"
              aria-hidden="true"
            >
              <circle
                className="opacity-25"
                cx="12"
                cy="12"
                r="10"
                stroke="currentColor"
                strokeWidth="4"
              />
              <path
                className="opacity-75"
                fill="currentColor"
                d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"
              />
            </svg>
          </div>
          <p className="text-sm text-content-muted">Loading...</p>
        </div>
      }
    >
      {children}
    </Suspense>
  )
}
