import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useProtoSources, useSchemaStatus } from '@/contexts/proto'
import { Spinner, Button, EmptyState, QueryErrorState } from '@/components/ui'
import { SettingsSection } from '@/components/settings/SettingsSection'
import type { ProtoSource } from '@/api/protoSources'
import ProtoSourceCard from './ProtoSourceCard'
import SchemaBrowser from './SchemaBrowser'
import SchemaConflicts from './SchemaConflicts'

function SourcesIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" stroke="currentColor" viewBox="0 0 24 24">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4" />
    </svg>
  )
}

function ConflictIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" stroke="currentColor" viewBox="0 0 24 24">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126zM12 15.75h.007v.008H12v-.008z" />
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

export default function ProtoManager() {
  const [browserOpen, setBrowserOpen] = useState(true)

  const { data: sources = [], isLoading: isLoadingSources, error: sourcesError, refetch: refetchSources } = useProtoSources()
  const { data: status } = useSchemaStatus()
  const conflicts = status?.conflicts ?? []

  // ProtoManager is only rendered inside the dedicated Settings page
  // (/settings/proto), so create/edit always route to full-width pages.
  const navigate = useNavigate()

  const handleAddSource = () => {
    navigate('/settings/proto/new')
  }

  const handleEditSource = (source: ProtoSource) => {
    navigate(`/settings/proto/${source.id}/edit`)
  }

  const hasSchemas = sources.some((source) => source.activeSchema)

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
        ) : sourcesError ? (
          <QueryErrorState error={sourcesError} onRetry={() => void refetchSources()} />
        ) : sources.length > 0 ? (
          <div className="space-y-3">
            {sources.map((source) => (
              <ProtoSourceCard key={source.id} source={source} onEdit={handleEditSource} />
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

      {conflicts.length > 0 && (
        <SettingsSection
          title="Schema conflicts"
          description="Files and types that more than one enabled source defines"
          icon={<ConflictIcon />}
          badge={conflicts.length}
        >
          <SchemaConflicts conflicts={conflicts} sources={sources} />
        </SettingsSection>
      )}

      {hasSchemas && (
        <SettingsSection
          title="Schema browser"
          description="Messages, enums and services of the active schemas, with their comments"
          icon={<PackageIcon />}
          collapsible
          isOpen={browserOpen}
          onToggle={() => setBrowserOpen((prev) => !prev)}
        >
          <SchemaBrowser sources={sources} />
        </SettingsSection>
      )}
    </>
  )
}
