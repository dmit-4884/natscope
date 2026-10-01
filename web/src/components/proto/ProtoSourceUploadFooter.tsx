import { useState } from 'react'
import { Button } from '@/components/ui'
import ErrorAlert from '@/components/ui/ErrorAlert'
import type { CompileOutcome, SchemaRevision } from '@/api/protoSources'
import { getErrorMessage } from '@/api/errors'
import { plural } from '@/utils/plural'
import { CompileDiagnosticsList } from './CompileDiagnosticsList'
import { SchemaUploadPicker } from './SchemaUploadPicker'
import type { PreparedUpload } from './schemaUpload'

interface Props {
  activeSchema?: SchemaRevision
  onUpload: (upload: PreparedUpload) => Promise<CompileOutcome | undefined>
  isUploading: boolean
  uploadError?: Error | null
}

export function ProtoSourceUploadFooter({ activeSchema, onUpload, isUploading, uploadError }: Props) {
  const [showPicker, setShowPicker] = useState(!activeSchema)
  const [outcome, setOutcome] = useState<CompileOutcome | null>(null)
  const [picked, setPicked] = useState<string | undefined>()

  const handlePick = async (upload: PreparedUpload) => {
    setOutcome(null)
    setPicked(upload.summary)
    const result = await onUpload(upload)
    if (!result) return
    setOutcome(result)
    if (result.valid) setShowPicker(false)
  }

  return (
    <div className="space-y-2">
      <div className="flex items-center justify-between gap-3">
        <div className="text-xs text-content-tertiary min-w-0 truncate">
          {activeSchema ? (
            <>
              Uploaded schema <code data-testid="upload-revision">{activeSchema.revision}</code> ·{' '}
              {plural(activeSchema.messageCount, 'message type')}
            </>
          ) : (
            'Nothing uploaded yet'
          )}
        </div>
        {activeSchema && (
          <Button size="sm" variant="secondary" onClick={() => setShowPicker((v) => !v)} data-testid="upload-new-version">
            {showPicker ? 'Cancel' : 'Upload new version'}
          </Button>
        )}
      </div>

      {showPicker && (
        <SchemaUploadPicker onPick={(upload) => void handlePick(upload)} busy={isUploading} selected={picked} />
      )}

      {uploadError && <ErrorAlert compact message={getErrorMessage(uploadError)} />}
      {outcome?.valid && (
        <div className="text-xs text-green-700 p-2 bg-status-success-bg rounded" data-testid="upload-ok">
          Compiled: {plural(outcome.fileDescriptors, 'file descriptor')}, {plural(outcome.messageTypes, 'message type')}
        </div>
      )}
      {outcome && outcome.diagnostics.length > 0 && <CompileDiagnosticsList diagnostics={outcome.diagnostics} compact />}
    </div>
  )
}
