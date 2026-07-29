import { useEffect } from 'react'
import { Outlet, useLocation } from 'react-router-dom'
import { useDisplayPreferences } from '@/contexts/settings'
import { useMigrateLegacyTemplates } from '@/contexts/templates'
import { toast } from '@/utils/toast'
import Toaster from './Toaster'
import CommandPalette from './CommandPalette'

/**
 * Dismisses any open toasts when the route changes. Without this, an error
 * toast triggered on /streams/X/publish stays visible after the user
 * navigates to /streams or switches connection — confusing because its
 * context no longer applies.
 */
function useDismissToastsOnRouteChange() {
  const { pathname } = useLocation()
  useEffect(() => {
    toast.dismiss()
  }, [pathname])
}

const capitalize = (s: string) => s.charAt(0).toUpperCase() + s.slice(1)

function titleForPath(pathname: string): string {
  const seg = pathname.split('/').filter(Boolean).map(decodeURIComponent)
  const [root, name, sub] = seg
  switch (root) {
    case 'streams':
      if (!name) return 'Streams — Natscope'
      if (name === 'new') return 'New stream — Natscope'
      return `${name}${sub ? ` · ${capitalize(sub)}` : ''} — Natscope`
    case 'kv':
      if (!name) return 'KV buckets — Natscope'
      if (name === 'new') return 'New KV bucket — Natscope'
      return `${name} · KV — Natscope`
    case 'objects':
      if (!name) return 'Object buckets — Natscope'
      if (name === 'new') return 'New object bucket — Natscope'
      return `${name} · Objects — Natscope`
    case 'settings':
      return `${capitalize(name ?? 'Settings')} · Settings — Natscope`
    default:
      return 'Natscope'
  }
}

function useDocumentTitle() {
  const { pathname } = useLocation()
  useEffect(() => {
    document.title = titleForPath(pathname)
  }, [pathname])
}

export default function RootLayout() {
  const density = useDisplayPreferences().density
  useDismissToastsOnRouteChange()
  useDocumentTitle()
  // One-time upload of legacy localStorage templates → backend (idempotent).
  useMigrateLegacyTemplates()

  return (
    <div className="h-screen bg-surface-tertiary flex flex-col overflow-hidden" data-density={density}>
      {/* Skip link for keyboard users */}
      <a href="#main-content" className="skip-link">
        Skip to main content
      </a>
      <Outlet />
      <Toaster />
      <CommandPalette />
    </div>
  )
}
