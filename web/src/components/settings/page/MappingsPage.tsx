import { useState } from 'react'
import { useOutletContext, useSearchParams } from 'react-router-dom'
import { Button } from '@/components/ui'
import { useMappingItems } from '@/contexts/mappings'
import { ConnectionsMappingsTab } from '@/components/connections/manager/ConnectionsMappingsTab'
import { ImportMappingsModal } from '@/components/mappings/ImportMappingsModal'
import { ExportMappingsModal } from '@/components/mappings/ExportMappingsModal'
import { useMappingImportExport } from '@/components/mappings/useMappingImportExport'
import { plural } from '@/utils/plural'
import type { ConnectionOutletContext } from '@/components/common/ConnectedLayout'
import { SettingsPage } from './SettingsPage'

export default function MappingsPage() {
  const ctx = useOutletContext<ConnectionOutletContext | undefined>()
  const connectionId = ctx?.connectionId || null

  const [searchParams] = useSearchParams()
  const initialSubjectPattern = searchParams.get('subject')

  const [showImportModal, setShowImportModal] = useState(false)
  const [showExportModal, setShowExportModal] = useState(false)
  const { data: mappings = [] } = useMappingItems()
  const mappingsCount = mappings.length

  const {
    importError,
    setImportError,
    exportToFile,
    exportToClipboard,
    parseText,
    parseFile,
    commit,
    isCommitting,
  } = useMappingImportExport({ onImportSuccess: () => setShowImportModal(false) })

  return (
    <SettingsPage
      title="Mappings"
      description="Bind NATS subject patterns to proto message types"
      meta={`${plural(mappingsCount, 'mapping')} configured`}
      scroll="fill"
      actions={
        <>
          <Button
            variant="secondary"
            onClick={() => {
              setImportError('')
              setShowImportModal(true)
            }}
          >
            Import
          </Button>
          <Button variant="secondary" onClick={() => setShowExportModal(true)}>
            Export
          </Button>
          <Button variant="secondary" onClick={() => exportToClipboard()}>
            Copy
          </Button>
        </>
      }
    >
      <ConnectionsMappingsTab
        connectionId={connectionId}
        initialSubjectPattern={initialSubjectPattern}
      />

      {showImportModal && (
        <ImportMappingsModal
          onClose={() => setShowImportModal(false)}
          parseText={parseText}
          parseFile={parseFile}
          commit={commit}
          isCommitting={isCommitting}
          error={importError}
        />
      )}

      {showExportModal && (
        <ExportMappingsModal
          onClose={() => setShowExportModal(false)}
          onExport={exportToFile}
          onCopy={exportToClipboard}
        />
      )}
    </SettingsPage>
  )
}
