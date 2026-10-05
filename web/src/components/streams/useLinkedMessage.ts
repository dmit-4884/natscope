import { useEffect } from 'react'
import { useStreamMessage } from '@/contexts/messages'
import type { SelectedMessage } from '@/types/messages'
import { toSelectedHistoryMessage } from '../messages/unified/selectedMessage'

const HISTORY_ID = /^history-(\d+)$/

export function linkedSequence(param: string | null): number | null {
  const match = HISTORY_ID.exec(param ?? '')
  return match ? Number(match[1]) : null
}

interface Options {
  connectionId: string | null
  streamName: string | null
  param: string | null
  selected: SelectedMessage | null
  onFound: (message: SelectedMessage) => void
  onMissing: (sequence: number, error: unknown) => void
}

export function useLinkedMessage({ connectionId, streamName, param, selected, onFound, onMissing }: Options) {
  const sequence = linkedSequence(param)
  const wanted = sequence != null && selected?.id !== `history-${sequence}` ? sequence : null
  const { data, error } = useStreamMessage(connectionId, streamName, wanted)

  useEffect(() => {
    if (wanted != null && data?.sequence === wanted) onFound(toSelectedHistoryMessage(data))
  }, [wanted, data, onFound])

  useEffect(() => {
    if (wanted != null && error) onMissing(wanted, error)
  }, [wanted, error, onMissing])
}
