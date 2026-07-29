import { useRef, useState } from 'react'
import { Modal, Button } from '@/components/ui'
import { parseImport, parseImportFile } from './templateImportExport'

interface Props {
  isOpen: boolean
  onClose: () => void
  onImport: (templates: Array<{
    name: string
    subject: string
    messageType: string
    data: string
    headers?: Record<string, string>
  }>) => void
}

export function ImportTemplatesModal({ isOpen, onClose, onImport }: Props) {
  const [text, setText] = useState('')
  const [error, setError] = useState('')
  const fileInputRef = useRef<HTMLInputElement>(null)

  const reset = () => {
    setText('')
    setError('')
  }

  const close = () => {
    reset()
    onClose()
  }

  const handleFile = async (file: File) => {
    setError('')
    const result = await parseImportFile(file)
    if (!result.ok) {
      setError(result.error)
      return
    }
    onImport(result.templates)
    setText('')
    onClose()
  }

  const handleSubmit = () => {
    setError('')
    const result = parseImport(text)
    if (!result.ok) {
      setError(result.error)
      return
    }
    onImport(result.templates)
    setText('')
    onClose()
  }

  return (
    <Modal
      isOpen={isOpen}
      onClose={close}
      title="Import templates"
      size="lg"
      footer={
        <>
          <Button variant="secondary" onClick={close}>Cancel</Button>
          <Button disabled={!text.trim()} onClick={handleSubmit}>Import</Button>
        </>
      }
    >
      <div className="px-6 py-4 space-y-4">
        <p className="text-sm text-content-secondary">
          Paste JSON exported from Natscope, or pick a file. Existing templates are kept — imported ones are appended with new IDs.
        </p>
        <div>
          <Button variant="secondary" size="sm" onClick={() => fileInputRef.current?.click()}>
            Choose file…
          </Button>
          <input
            ref={fileInputRef}
            type="file"
            accept="application/json,.json"
            className="hidden"
            onChange={async (e) => {
              const file = e.target.files?.[0]
              e.target.value = ''
              if (file) await handleFile(file)
            }}
          />
        </div>
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Or paste JSON</label>
          <textarea
            value={text}
            onChange={(e) => setText(e.target.value)}
            placeholder='{"version": 1, "templates": [...]}'
            spellCheck={false}
            className="w-full h-40 rounded-md border border-border-strong px-3 py-2 text-xs font-mono focus:border-border-focus focus:ring-1 focus:ring-border-focus focus:outline-none"
          />
        </div>
        {error && (
          <div className="rounded-md border border-rose-200 bg-rose-50 px-3 py-2 text-sm text-rose-700">
            {error}
          </div>
        )}
      </div>
    </Modal>
  )
}
