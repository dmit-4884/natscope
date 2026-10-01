import { useEffect, useRef, useState, type DragEvent } from 'react'
import { Button, Spinner, UploadIcon } from '@/components/ui'
import { cn } from '@/utils/cn'
import { pickedFromDrop, pickedFromList, prepareUpload, type PickedFile, type PreparedUpload } from './schemaUpload'

interface Props {
  onPick: (upload: PreparedUpload) => void
  busy?: boolean
  disabled?: boolean
  selected?: string
}

export function SchemaUploadPicker({ onPick, busy = false, disabled = false, selected }: Props) {
  const filesInput = useRef<HTMLInputElement>(null)
  const folderInput = useRef<HTMLInputElement>(null)
  const [dragging, setDragging] = useState(false)
  const [reading, setReading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const blocked = disabled || busy || reading

  useEffect(() => {
    folderInput.current?.setAttribute('webkitdirectory', '')
  }, [])

  const accept = async (picked: PickedFile[]) => {
    setError(null)
    setReading(true)
    try {
      const upload = await prepareUpload(picked)
      if (upload) onPick(upload)
      else setError('No .proto files or descriptor set in the selection.')
    } finally {
      setReading(false)
    }
  }

  const handleDrop = (e: DragEvent<HTMLDivElement>) => {
    e.preventDefault()
    setDragging(false)
    if (!blocked) void pickedFromDrop(e.dataTransfer).then(accept)
  }

  return (
    <div
      data-testid="schema-upload-dropzone"
      onDragOver={(e) => {
        e.preventDefault()
        if (!blocked) setDragging(true)
      }}
      onDragLeave={() => setDragging(false)}
      onDrop={handleDrop}
      className={cn(
        'rounded-md border-2 border-dashed px-4 py-5 text-center transition-colors',
        dragging ? 'border-border-focus bg-accent-light/40' : 'border-border bg-surface-secondary',
      )}
    >
      <UploadIcon className="w-6 h-6 mx-auto text-content-muted" />
      <p className="mt-2 text-sm text-content-secondary">Drop .proto files, a folder or a descriptor set</p>
      <p className="text-xs text-content-tertiary">
        Descriptor set: <code>buf build -o schema.binpb</code> or <code>protoc --include_imports --descriptor_set_out</code>
      </p>
      <div className="mt-3 flex justify-center gap-2">
        <Button size="sm" variant="secondary" onClick={() => filesInput.current?.click()} disabled={blocked}>
          Choose files
        </Button>
        <Button size="sm" variant="secondary" onClick={() => folderInput.current?.click()} disabled={blocked}>
          Choose folder
        </Button>
      </div>
      <input
        ref={filesInput}
        type="file"
        multiple
        hidden
        aria-label="Proto files or descriptor set"
        accept=".proto,.binpb,.pb,.desc,.protoset,.bin,.yaml,.lock"
        onChange={(e) => {
          if (e.target.files?.length) void accept(pickedFromList(e.target.files))
          e.target.value = ''
        }}
      />
      <input
        ref={folderInput}
        type="file"
        hidden
        aria-label="Proto folder"
        onChange={(e) => {
          if (e.target.files?.length) void accept(pickedFromList(e.target.files))
          e.target.value = ''
        }}
      />
      {(busy || reading) && (
        <p className="mt-3 flex items-center justify-center gap-2 text-xs text-content-tertiary">
          <Spinner size="sm" />
          {reading ? 'Reading files…' : 'Uploading and compiling…'}
        </p>
      )}
      {selected && !busy && !reading && (
        <p className="mt-3 text-xs text-content-secondary" data-testid="schema-upload-selected">
          {selected}
        </p>
      )}
      {error && (
        <p className="mt-3 text-xs text-red-700" role="alert">
          {error}
        </p>
      )}
    </div>
  )
}
