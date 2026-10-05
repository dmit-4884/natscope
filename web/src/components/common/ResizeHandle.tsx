import type { useResizablePanel } from '@/hooks/useResizablePanel'

type SeparatorProps = ReturnType<typeof useResizablePanel>['separatorProps']

export function ResizeHandle(props: SeparatorProps) {
  return (
    <div
      {...props}
      className="flex-shrink-0 px-0.5 cursor-col-resize group flex items-stretch focus:outline-none focus-visible:ring-2 focus-visible:ring-border-focus"
    >
      <div className="w-px bg-surface-hover group-hover:bg-accent group-active:bg-accent group-focus-visible:bg-accent transition-colors" />
    </div>
  )
}
