import type { ReactNode } from 'react'
import type {
  SchemaEnum,
  SchemaEnumValue,
  SchemaField,
  SchemaMessage,
  SchemaMethod,
  SchemaService,
  SchemaType,
  SchemaTypeKind,
} from '@/api/proto'
import { fieldTypeLabel, useMessageExample, useTypeDescription } from '@/contexts/proto'
import { useMappingItems } from '@/contexts/mappings'
import { Badge, CopyButton, DataTable, QueryErrorState, SkeletonRows, type DataTableColumn } from '@/components/ui'

const KIND_LABELS: Record<SchemaTypeKind, string> = { message: 'message', enum: 'enum', service: 'service' }

export function KindBadge({ kind }: { kind: SchemaTypeKind }) {
  return (
    <Badge size="sm" variant={kind === 'message' ? 'primary' : kind === 'enum' ? 'warning' : 'success'}>
      {KIND_LABELS[kind]}
    </Badge>
  )
}

function Comment({ text }: { text: string }) {
  if (!text) return null
  return <p className="whitespace-pre-line text-xs text-content-secondary">{text}</p>
}

function TypeLink({ name, onOpen }: { name: string; onOpen: (fullName: string) => void }) {
  return (
    <button
      type="button"
      className="font-mono text-xs text-accent hover:underline text-left break-all"
      onClick={() => onOpen(name)}
    >
      {name}
    </button>
  )
}

function FieldNotes({ field }: { field: SchemaField }) {
  const notes: string[] = []
  if (field.oneof) notes.push(`oneof ${field.oneof}`)
  if (field.optional) notes.push('optional')
  if (field.required) notes.push('required')
  return (
    <span className="flex flex-wrap gap-1">
      {notes.map((n) => (
        <Badge key={n} size="sm">
          {n}
        </Badge>
      ))}
      {field.deprecated && (
        <Badge size="sm" variant="error">
          deprecated
        </Badge>
      )}
    </span>
  )
}

function MessageFields({ message, onOpen }: { message: SchemaMessage; onOpen: (fullName: string) => void }) {
  const columns: DataTableColumn<SchemaField>[] = [
    { key: 'number', header: '#', width: 'w-12', render: (f) => <span className="tabular-nums text-content-tertiary">{f.number}</span> },
    {
      key: 'name',
      header: 'Field',
      render: (f) => <span className={f.deprecated ? 'font-mono text-xs line-through' : 'font-mono text-xs'}>{f.name}</span>,
    },
    {
      key: 'type',
      header: 'Type',
      render: (f) =>
        f.typeName ? (
          <span className="font-mono text-xs">
            {f.mapKey ? `map<${f.mapKey}, ` : f.repeated ? 'repeated ' : ''}
            <TypeLink name={f.typeName} onOpen={onOpen} />
            {f.mapKey ? '>' : ''}
          </span>
        ) : (
          <span className="font-mono text-xs">{fieldTypeLabel(f)}</span>
        ),
    },
    { key: 'notes', header: '', render: (f) => <FieldNotes field={f} /> },
    { key: 'comment', header: 'Comment', render: (f) => <Comment text={f.comment} /> },
  ]
  return (
    <DataTable
      columns={columns}
      items={message.fields}
      rowKey={(f) => String(f.number)}
      emptyState={<p className="text-xs text-content-tertiary px-1">No fields.</p>}
    />
  )
}

function EnumValues({ value }: { value: SchemaEnum }) {
  const columns: DataTableColumn<SchemaEnumValue>[] = [
    { key: 'name', header: 'Value', render: (v) => <span className="font-mono text-xs">{v.name}</span> },
    { key: 'number', header: 'Number', width: 'w-20', render: (v) => <span className="tabular-nums">{v.number}</span> },
    { key: 'comment', header: 'Comment', render: (v) => <Comment text={v.comment} /> },
  ]
  return <DataTable columns={columns} items={value.values} rowKey={(v) => v.name} />
}

function ServiceMethods({ service, onOpen }: { service: SchemaService; onOpen: (fullName: string) => void }) {
  const columns: DataTableColumn<SchemaMethod>[] = [
    { key: 'name', header: 'Method', render: (m) => <span className="font-mono text-xs">{m.name}</span> },
    {
      key: 'io',
      header: 'Request → Response',
      render: (m) => (
        <span className="flex flex-wrap items-center gap-1 text-xs">
          {m.clientStreaming && <span className="text-content-tertiary">stream</span>}
          <TypeLink name={m.inputType} onOpen={onOpen} />
          <span className="text-content-muted">→</span>
          {m.serverStreaming && <span className="text-content-tertiary">stream</span>}
          <TypeLink name={m.outputType} onOpen={onOpen} />
        </span>
      ),
    },
    { key: 'comment', header: 'Comment', render: (m) => <Comment text={m.comment} /> },
  ]
  return <DataTable columns={columns} items={service.methods} rowKey={(m) => m.name} />
}

function MessageExample({ type }: { type: SchemaType }) {
  const { data, isLoading, error } = useMessageExample(type.sourceId, type.fullName)
  if (isLoading) return <SkeletonRows count={2} />
  if (error) return <QueryErrorState error={error} />
  const json = JSON.stringify(data ?? {}, null, 2)
  return (
    <div className="relative">
      <pre className="text-xs font-mono bg-surface-secondary border border-border rounded-md p-3 overflow-x-auto max-h-64" data-testid="schema-example">
        {json}
      </pre>
      <CopyButton value={json} className="absolute top-2 right-2" />
    </div>
  )
}

function UsedBy({ type }: { type: SchemaType }) {
  const { data: mappings = [] } = useMappingItems()
  const used = mappings.filter((m) => m.sourceId === type.sourceId && m.messageType === type.fullName)
  if (used.length === 0) return <p className="text-xs text-content-tertiary">No subject mappings use this type.</p>
  return (
    <div className="flex flex-wrap gap-1" data-testid="schema-used-by">
      {used.map((m) => (
        <code key={m.id} className="text-xs bg-surface-secondary border border-border rounded px-1.5 py-0.5">
          {m.pattern}
        </code>
      ))}
    </div>
  )
}

function Section({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section className="space-y-2">
      <h4 className="text-2xs font-semibold uppercase tracking-wide text-content-muted">{title}</h4>
      {children}
    </section>
  )
}

interface Props {
  type: SchemaType
  sourceName: string
  onOpenType: (fullName: string) => void
}

export function SchemaTypeDetail({ type, sourceName, onOpenType }: Props) {
  const { data, isLoading, error, refetch } = useTypeDescription(type.sourceId, type.fullName)
  const message = type.kind === 'message' ? data?.messages[0] : undefined
  const enumType = type.kind === 'enum' ? data?.enums[0] : undefined
  const service = type.kind === 'service' ? data?.services[0] : undefined
  const comment = message?.comment ?? enumType?.comment ?? service?.comment ?? ''
  const revision = /^[0-9a-f]{40}$/.test(type.sourceRevision) ? type.sourceRevision.slice(0, 7) : type.sourceRevision

  return (
    <div className="space-y-4 min-w-0" data-testid="schema-type-detail">
      <header className="space-y-1">
        <div className="flex items-center gap-2 min-w-0">
          <KindBadge kind={type.kind} />
          <h3 className="font-mono text-sm font-semibold text-content-primary break-all">{type.fullName}</h3>
          <CopyButton value={type.fullName} />
        </div>
        <p className="text-xs text-content-tertiary break-all">
          {type.file} · {sourceName} @ {revision}
          {type.dependency && ' · imported'}
        </p>
      </header>

      {isLoading ? (
        <SkeletonRows count={4} />
      ) : error ? (
        <QueryErrorState error={error} onRetry={() => void refetch()} />
      ) : (
        <>
          <Comment text={comment} />
          {message && (
            <Section title="Fields">
              <MessageFields message={message} onOpen={onOpenType} />
            </Section>
          )}
          {enumType && (
            <Section title="Values">
              <EnumValues value={enumType} />
            </Section>
          )}
          {service && (
            <Section title="Methods">
              <ServiceMethods service={service} onOpen={onOpenType} />
            </Section>
          )}
          {type.kind === 'message' && (
            <>
              <Section title="Example JSON">
                <MessageExample type={type} />
              </Section>
              <Section title="Used by mappings">
                <UsedBy type={type} />
              </Section>
            </>
          )}
        </>
      )}
    </div>
  )
}
