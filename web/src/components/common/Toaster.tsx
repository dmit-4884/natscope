import { Toaster as SonnerToaster } from 'sonner'

/**
 * Mounted once at the root layout. Pure UI — the `toast` API used by every
 * call site lives in `@/utils/toast` (cross-layer safe).
 */
export default function Toaster() {
  return (
    <SonnerToaster
      position="bottom-right"
      toastOptions={{
        duration: 3000,
        classNames: {
          toast: 'bg-surface-primary border border-border shadow-lg rounded-lg',
          title: 'text-content-primary font-medium',
          description: 'text-content-tertiary text-sm',
          success: 'border-green-200 bg-status-success-bg',
          error: 'border-red-200 bg-status-error-bg',
          warning: 'border-amber-200 bg-status-warning-bg',
          info: 'border-blue-200 bg-accent-light',
          closeButton: 'bg-surface-primary border border-border text-content-muted hover:text-content-secondary',
        },
      }}
      expand={false}
      richColors
      closeButton
    />
  )
}
