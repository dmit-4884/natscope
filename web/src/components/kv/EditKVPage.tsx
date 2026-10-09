import { useState } from 'react'
import { Link, useNavigate, useOutletContext, useParams } from 'react-router-dom'
import type { KVBucketConfig, KVBucketInfo, KVBucketSettings } from '@/types/management'
import { useKVBuckets, useUpdateKVBucket } from '@/contexts/kv'
import { useServerCapabilities } from '@/contexts/connection'
import { Button, EmptyState, QueryErrorState, SkeletonRows } from '@/components/ui'
import { KVBucketFormFields } from '@/components/management/kv/KVBucketFormFields'
import { isKVHistoryValid } from '@/components/management/kv/kvHistory'
import { kvMarkerTtlError } from '@/components/management/kv/kvMarkerTtl'
import ConfigDiffModal from '../common/ConfigDiffModal'
import type { ConnectionOutletContext } from '../common/ConnectedLayout'

function toForm(info: KVBucketInfo): KVBucketConfig {
  return {
    bucket: info.bucket,
    description: info.description,
    history: info.history,
    ttl: info.ttl,
    max_value_size: info.max_value_size,
    max_bytes: info.max_bytes,
    storage: info.storage === 'memory' ? 'memory' : 'file',
    num_replicas: info.num_replicas,
    compression: info.is_compressed ?? false,
    limit_marker_ttl: info.limit_marker_ttl,
    metadata: info.metadata,
  }
}

function toSettings(form: KVBucketConfig): KVBucketSettings {
  return {
    description: form.description ?? '',
    history: form.history ?? 1,
    ttl: form.ttl ?? 0,
    max_value_size: form.max_value_size ?? -1,
    max_bytes: form.max_bytes ?? -1,
    num_replicas: form.num_replicas ?? 1,
    compression: form.compression ?? false,
    limit_marker_ttl: form.limit_marker_ttl ?? 0,
    metadata: form.metadata ?? {},
  }
}

export default function EditKVPage() {
  const { bucketName } = useParams<{ bucketName: string }>()
  const navigate = useNavigate()
  const { connectionId, currentConnection } = useOutletContext<ConnectionOutletContext>()
  const { data: buckets, isLoading, error, refetch } = useKVBuckets(connectionId)
  const updateBucket = useUpdateKVBucket(connectionId)
  const { unsupportedReason } = useServerCapabilities(connectionId)
  const [draft, setDraft] = useState<KVBucketConfig | null>(null)
  const [reviewing, setReviewing] = useState(false)

  const info = buckets?.find((b) => b.bucket === bucketName)

  if (isLoading) return <SkeletonRows count={6} rowClassName="h-12" className="p-6" />
  if (error) return <QueryErrorState error={error} onRetry={() => refetch()} />
  if (!info || !bucketName) {
    return (
      <EmptyState
        title="Bucket not found"
        description={`There is no KV bucket "${bucketName ?? ''}" on this connection.`}
        action={<Link to="/kv" className="text-sm text-accent hover:text-accent-text">All KV buckets</Link>}
      />
    )
  }

  const original = toForm(info)
  const value = draft ?? original
  const keyTtlLocked = !!info.limit_marker_ttl
  const valid = isKVHistoryValid(value.history) && !kvMarkerTtlError(value.limit_marker_ttl, keyTtlLocked)
  const back = () => navigate(`/kv/${encodeURIComponent(bucketName)}`)

  const save = async () => {
    try {
      await updateBucket.mutateAsync({ bucket: bucketName, settings: toSettings(value) })
      setReviewing(false)
      back()
    } catch {
      /* toasted by useUpdateKVBucket */
    }
  }

  return (
    <div className="flex-1 bg-surface-primary flex flex-col overflow-hidden">
      <div className="px-4 pt-4 pb-3 border-b bg-surface-secondary">
        <h3 className="font-semibold text-content-primary">Edit KV Store {bucketName}</h3>
        <p className="text-sm text-content-tertiary mt-0.5">
          {currentConnection?.name || currentConnection?.urls[0] || 'Unknown connection'}
        </p>
      </div>

      <div className="flex-1 overflow-auto p-6">
        <div className="max-w-4xl">
          <KVBucketFormFields
            value={value}
            onChange={setDraft}
            isEditMode
            keyTtlLocked={keyTtlLocked}
            keyTtlUnsupportedReason={unsupportedReason('messageTtl')}
          />
        </div>
      </div>

      <div className="flex justify-end gap-3 px-6 py-4 border-t bg-surface-secondary">
        <Button variant="secondary" onClick={back}>
          Cancel
        </Button>
        <Button onClick={() => setReviewing(true)} disabled={!valid || updateBucket.isPending}>
          Save changes
        </Button>
      </div>

      {reviewing && (
        <ConfigDiffModal
          isOpen
          onClose={() => setReviewing(false)}
          onConfirm={() => void save()}
          title="Confirm KV Store Changes"
          description={`Review the changes to the KV store "${bucketName}" before applying them.`}
          originalConfig={toSettings(original)}
          newConfig={toSettings(value)}
          isLoading={updateBucket.isPending}
        />
      )}
    </div>
  )
}
