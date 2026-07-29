import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  useProtoSources,
  useProtoSelections,
  useDeleteProtoSelection,
  useProtoMessageEntities,
  groupMessagesByPackage,
} from '@/contexts/proto'
import { Spinner, Button, EmptyState } from '@/components/ui'
import { SettingsSection } from '@/components/settings/SettingsSection'
import type { ProtoSource } from '@/api/protoSources'
import ProtoSourceCard from './ProtoSourceCard'

function SourcesIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" stroke="currentColor" viewBox="0 0 24 24">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4" />
    </svg>
  )
}

function PackageIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" stroke="currentColor" viewBox="0 0 24 24">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M20.25 7.5l-.625 10.632a2.25 2.25 0 01-2.247 2.118H6.622a2.25 2.25 0 01-2.247-2.118L3.75 7.5M10 11.25h4M3.375 7.5h17.25c.621 0 1.125-.504 1.125-1.125v-1.5c0-.621-.504-1.125-1.125-1.125H3.375c-.621 0-1.125.504-1.125 1.125v1.5c0 .621.504 1.125 1.125 1.125z" />
    </svg>
  )
}

function ChevronIcon({ className, expanded }: { className?: string; expanded: boolean }) {
  return (
    <svg
      className={`${className} transition-transform duration-200 ${expanded ? 'rotate-90' : ''}`}
      fill="none"
      stroke="currentColor"
      viewBox="0 0 24 24"
    >
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
    </svg>
  )
}

export default function ProtoManager() {
  const [expandedPackages, setExpandedPackages] = useState<Set<string>>(new Set())
  const [messagesExpanded, setMessagesExpanded] = useState(false)

  const { data: sources = [], isLoading: isLoadingSources } = useProtoSources()
  const { data: selections = [], isLoading: isLoadingSelections } = useProtoSelections()
  const { messages, packages, isLoading: isLoadingMessages } = useProtoMessageEntities()

  const deleteMutation = useDeleteProtoSelection()

  const messageCount = messages.length
  const packageCount = packages.length
  const groupedByPackage = groupMessagesByPackage(messages)

  // Map source_id to selection for easy lookup
  const selectionBySourceId = new Map(
    selections.map((sel) => [sel.source_id, sel])
  )

  const handleRemoveSelection = async (selectionId: string) => {
    try {
      await deleteMutation.mutateAsync(selectionId)
    } catch {
      /* toasted by the global mutation error handler */
    }
  }

  // ProtoManager is only rendered inside the dedicated Settings page
  // (/settings/proto), so create/edit always route to full-width pages.
  const navigate = useNavigate()

  const handleAddSource = () => {
    navigate('/settings/proto/new')
  }

  const handleEditSource = (source: ProtoSource) => {
    navigate(`/settings/proto/${source.id}/edit`)
  }

  const togglePackage = (pkg: string) => {
    setExpandedPackages(prev => {
      const next = new Set(prev)
      if (next.has(pkg)) {
        next.delete(pkg)
      } else {
        next.add(pkg)
      }
      return next
    })
  }

  const hasSelections = selections.length > 0
  const hasMessages = messageCount > 0

  return (
    <>
      <SettingsSection
        title="Proto sources"
        description="Git repositories, local directories and uploads used to compile descriptors"
        icon={<SourcesIcon />}
        badge={sources.length > 0 ? sources.length : undefined}
        actions={
          <Button size="sm" onClick={handleAddSource}>
            Add source
          </Button>
        }
      >
        {isLoadingSources ? (
          <div className="flex items-center justify-center gap-2 text-sm text-content-tertiary py-8">
            <Spinner size="sm" />
            <span>Loading repositories...</span>
          </div>
        ) : sources.length > 0 ? (
          <div className="space-y-3">
            {sources.map((source) => (
              <ProtoSourceCard
                key={source.id}
                source={source}
                selection={selectionBySourceId.get(source.id)}
                onEdit={handleEditSource}
                onRemoveSelection={handleRemoveSelection}
              />
            ))}
          </div>
        ) : (
          <EmptyState
            icon={<SourcesIcon />}
            title="No proto sources yet"
            description="Add a proto source (Git repo, local directory, or upload files) to enable automatic message decoding."
            action={
              <Button size="sm" onClick={handleAddSource}>
                Add source
              </Button>
            }
          />
        )}
      </SettingsSection>

      {(hasSelections || isLoadingSelections) && (
        <SettingsSection
          title="Available messages"
          description="Message types compiled from the active source versions"
          icon={<PackageIcon />}
          badge={hasMessages ? `${packageCount} pkg · ${messageCount} msg` : undefined}
          collapsible
          isOpen={messagesExpanded}
          onToggle={() => setMessagesExpanded((prev) => !prev)}
        >
          {isLoadingMessages || isLoadingSelections ? (
            <div className="flex items-center justify-center gap-2 text-sm text-content-tertiary py-6">
              <Spinner size="sm" />
              <span>Loading proto files...</span>
            </div>
          ) : hasMessages ? (
            <div className="space-y-1">
              {Object.entries(groupedByPackage).map(([pkg, msgNames]) => {
                const isExpanded = expandedPackages.has(pkg)
                return (
                  <div key={pkg} className="rounded-md overflow-hidden">
                    <button
                      onClick={() => togglePackage(pkg)}
                      className="w-full flex items-center gap-2 px-2 py-1.5 text-left hover:bg-surface-secondary rounded-md transition-colors"
                    >
                      <ChevronIcon className="w-3.5 h-3.5 text-content-muted" expanded={isExpanded} />
                      <PackageIcon className="w-3.5 h-3.5 text-blue-500" />
                      <span className="text-sm font-medium text-gray-700 truncate flex-1">{pkg}</span>
                      <span className="text-xs text-content-muted">{msgNames.length}</span>
                    </button>
                    {isExpanded && (
                      <div className="ml-5 pl-3 border-l border-border space-y-0.5 py-1">
                        {msgNames.map((msgName) => (
                          <div
                            key={msgName}
                            className="text-xs text-content-secondary py-1 px-2 rounded hover:bg-surface-secondary truncate"
                            title={msgName}
                          >
                            {msgName}
                          </div>
                        ))}
                      </div>
                    )}
                  </div>
                )
              })}
            </div>
          ) : (
            <EmptyState
              size="sm"
              icon={<PackageIcon />}
              title="No messages loaded"
              description="Select a version in a proto source above to load message types."
            />
          )}
        </SettingsSection>
      )}
    </>
  )
}
