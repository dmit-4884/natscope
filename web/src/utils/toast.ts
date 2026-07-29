import { toast as sonnerToast } from 'sonner'

/**
 * Toast duration from message length: ~500ms/word + 1000ms, clamped 3-7s.
 * Errors get a longer ceiling but still auto-dismiss — permanent toasts
 * follow the user across route changes.
 */
function calculateDuration(message: string, opts?: { min?: number; max?: number }): number {
  const min = opts?.min ?? 3000
  const max = opts?.max ?? 7000
  const words = message.split(/\s+/).length
  const duration = words * 500 + 1000
  return Math.min(Math.max(duration, min), max)
}

// Toast facade importable from any layer; the <Toaster> component
// (components/common/Toaster.tsx) shares this same sonner instance.
/** Inline toast action (e.g. an "Undo" button rendered inside the toast). */
export interface ToastAction {
  label: string
  onClick: () => void
}

export const toast = {
  success: (message: string, options?: { duration?: number; action?: ToastAction }) => {
    return sonnerToast.success(message, {
      duration: options?.duration ?? calculateDuration(message),
      action: options?.action,
    })
  },
  error: (message: string, options?: { duration?: number }) => {
    // Auto-dismiss, but slow enough to read. Pass duration: Infinity to stick.
    return sonnerToast.error(message, {
      duration: options?.duration ?? calculateDuration(message, { min: 6000, max: 12000 }),
    })
  },
  warning: (message: string, options?: { duration?: number }) => {
    return sonnerToast.warning(message, {
      duration: options?.duration ?? calculateDuration(message) + 2000, // extra time for warnings
    })
  },
  info: (message: string, options?: { duration?: number }) => {
    return sonnerToast.info(message, {
      duration: options?.duration ?? calculateDuration(message),
    })
  },
  loading: (message: string) => {
    return sonnerToast.loading(message)
  },
  dismiss: (id?: string | number) => {
    return sonnerToast.dismiss(id)
  },
  promise: sonnerToast.promise,
  custom: sonnerToast.custom,
}
