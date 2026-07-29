import { useState, useEffect, useMemo, useRef, useCallback } from 'react'
import { getMessages } from '@/api/messages'
import type { MessagesResponse } from '@/api/messages'
import type { Message } from '@/types/nats'
import { getErrorMessage } from '@/api/errors'
import { toast } from '@/utils/toast'

interface UseLoadMoreMessagesOptions {
  streamName: string | null
  connectionId: string | null
  /** First page from the history query; extra pages chain off its next_seq. */
  baseData: MessagesResponse | undefined
  limit: number
  subjectFilter?: string
  contentFilter?: string
  direction: 'forward' | 'backward'
}

/**
 * Incremental "Load more" on top of the single-page history query: keeps the
 * base useMessages query untouched and appends follow-up pages fetched with
 * start_seq = next_seq. Any base-page change (refetch, filter/limit/stream
 * switch) discards the accumulated pages.
 */
export function useLoadMoreMessages({
  streamName,
  connectionId,
  baseData,
  limit,
  subjectFilter,
  contentFilter,
  direction,
}: UseLoadMoreMessagesOptions) {
  const [extraPages, setExtraPages] = useState<MessagesResponse[]>([])
  const [isLoadingMore, setIsLoadingMore] = useState(false)
  // Guards a late loadMore response from landing on a newer base page.
  const generationRef = useRef(0)

  useEffect(() => {
    generationRef.current++
    // Bail out on already-empty state: a fresh [] every run would re-render
    // forever when baseData identity churns per render.
    setExtraPages((prev) => (prev.length === 0 ? prev : []))
    setIsLoadingMore((prev) => (prev ? false : prev))
  }, [baseData])

  const lastPage = extraPages.length > 0 ? extraPages[extraPages.length - 1] : baseData
  const hasMore = lastPage?.has_more ?? false
  const nextSeq = lastPage?.next_seq

  const loadMore = useCallback(async () => {
    if (!streamName || !connectionId || !hasMore || nextSeq == null || isLoadingMore) return
    const generation = generationRef.current
    setIsLoadingMore(true)
    try {
      // Always seq-anchored: jump-to-time only positions the first page, and
      // start_time/start_seq are mutually exclusive server-side.
      const page = await getMessages(streamName, {
        connection_id: connectionId,
        start_seq: nextSeq,
        limit,
        subject_filter: subjectFilter,
        content_filter: contentFilter,
        direction,
      })
      if (generationRef.current === generation) {
        setExtraPages((prev) => [...prev, page])
      }
    } catch (err) {
      if (generationRef.current === generation) {
        toast.error(getErrorMessage(err))
      }
    } finally {
      if (generationRef.current === generation) {
        setIsLoadingMore(false)
      }
    }
  }, [streamName, connectionId, hasMore, nextSeq, isLoadingMore, limit, subjectFilter, contentFilter, direction])

  const messages = useMemo<Message[]>(() => {
    if (!baseData) return []
    if (extraPages.length === 0) return baseData.messages
    const seen = new Set<number>()
    const out: Message[] = []
    for (const page of [baseData, ...extraPages]) {
      for (const m of page.messages) {
        if (!seen.has(m.sequence)) {
          seen.add(m.sequence)
          out.push(m)
        }
      }
    }
    return out
  }, [baseData, extraPages])

  return { messages, hasMore, loadMore, isLoadingMore }
}
