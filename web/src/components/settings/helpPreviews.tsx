/* eslint-disable react-refresh/only-export-components -- previews are tiny,
 * change rarely, and live next to the map keying them; not worth a file split. */
import { useEffect, useMemo, useRef, useState, type ReactElement } from 'react'
import { CopyButton } from '@/components/ui'
import { formatCount, formatTimestamp as formatTimestampSetting } from '@/utils/formatters'
import { plural } from '@/utils/plural'

// Interactive HelpModal previews for visual settings: toggle option values and
// copy rendered output.

export const HELP_PREVIEWS: Record<string, () => ReactElement> = {
  'display.density': () => <DensityPreview />,
  'display.timestampFormat': () => <TimestampFormatPreview />,
  'display.jsonIndentSize': () => <JsonIndentPreview />,
  'display.defaultViewMode': () => <DefaultViewModePreview />,
  'display.autoScrollLive': () => <AutoScrollLivePreview />,
  'messages.defaultPageSize': () => <DefaultPageSizePreview />,
  'messages.defaultDirection': () => <DefaultDirectionPreview />,
}

interface PreviewCardProps {
  /** Where in the app this setting is visible — shown as a breadcrumb-like label. */
  context: string
  children: React.ReactNode
}

function PreviewCard({ context, children }: PreviewCardProps) {
  return (
    <div className="mb-4 border border-blue-200 bg-accent-light/40 rounded-lg p-3">
      <div className="flex items-center justify-between mb-2 gap-2">
        <span className="inline-flex items-center gap-1 text-xs font-medium text-accent-text">
          <PulseDot />
          Live preview
        </span>
        <span className="text-2xs text-content-tertiary font-medium truncate" title={context}>
          {context}
        </span>
      </div>
      {children}
    </div>
  )
}

function PulseDot() {
  return (
    <span className="relative inline-flex w-2 h-2">
      <span className="absolute inset-0 rounded-full bg-blue-400 opacity-60 animate-ping" />
      <span className="relative inline-block w-2 h-2 rounded-full bg-blue-500" />
    </span>
  )
}

interface OptionToggleProps<T extends string | number> {
  options: ReadonlyArray<{ value: T; label: string }>
  value: T
  onChange: (v: T) => void
}

function OptionToggle<T extends string | number>({ options, value, onChange }: OptionToggleProps<T>) {
  return (
    <div className="inline-flex bg-surface-primary border border-border rounded-md p-0.5 shadow-sm">
      {options.map((opt) => (
        <button
          key={String(opt.value)}
          type="button"
          onClick={() => onChange(opt.value)}
          className={`px-3 py-1 text-xs font-medium rounded transition-colors ${
            value === opt.value
              ? 'bg-accent text-content-inverse shadow-sm'
              : 'text-content-secondary hover:text-content-primary hover:bg-surface-secondary'
          }`}
        >
          {opt.label}
        </button>
      ))}
    </div>
  )
}

const MOCK_ROWS = [
  { seq: 1022, subject: 'PROFILES.PERSONS.UPDATES.e28c1843', received: '13h ago', size: '404 B' },
  { seq: 1021, subject: 'PROFILES.PERSONS.NEW.0e7cc5cb', received: '18h ago', size: '178 B' },
  { seq: 1020, subject: 'PROFILES.PERSONS.NEW.7a6f663d', received: '1d ago', size: '160 B' },
]

function DensityPreview() {
  const [density, setDensity] = useState<'comfortable' | 'compact'>('comfortable')
  const padding = density === 'comfortable' ? 'py-3' : 'py-1.5'
  const headerPad = density === 'comfortable' ? 'py-2' : 'py-1'

  return (
    <PreviewCard context="Affects all lists app-wide (CSS density variables)">
      <div className="flex items-center justify-between mb-3">
        <OptionToggle
          value={density}
          onChange={setDensity}
          options={[
            { value: 'comfortable', label: 'Comfortable' },
            { value: 'compact', label: 'Compact' },
          ]}
        />
        <span className="text-xs text-content-tertiary">
          {density === 'comfortable' ? '~52px rows' : '~36px rows'}
        </span>
      </div>
      <div className="border border-border rounded-md bg-surface-primary overflow-hidden">
        <div className={`flex items-center gap-3 px-3 ${headerPad} bg-surface-secondary border-b border-border text-2xs uppercase tracking-wide text-content-tertiary font-medium`}>
          <span className="w-14 shrink-0">Seq</span>
          <span className="flex-1 min-w-0">Subject</span>
          <span className="w-20 shrink-0">Received</span>
          <span className="w-14 shrink-0 text-right">Size</span>
        </div>
        {MOCK_ROWS.map((row, i) => (
          <div
            key={row.seq}
            className={`flex items-center gap-3 px-3 ${padding} text-xs ${
              i < MOCK_ROWS.length - 1 ? 'border-b border-gray-100' : ''
            }`}
          >
            <span className="font-mono text-gray-700 w-14 shrink-0">{row.seq}</span>
            <span className="font-mono text-content-primary truncate flex-1 min-w-0">{row.subject}</span>
            <span className="text-content-tertiary w-20 shrink-0 font-mono">{row.received}</span>
            <span className="text-content-tertiary w-14 shrink-0 text-right">{row.size}</span>
          </div>
        ))}
      </div>
    </PreviewCard>
  )
}

const TIMESTAMP_ROWS = [
  { seq: 1022, subject: 'PROFILES.PERSONS.UPDATES.e28c1843', offsetMs: 13 * 60 * 60_000, relative: '13h ago', size: '404 B' },
  { seq: 1021, subject: 'PROFILES.PERSONS.NEW.0e7cc5cb', offsetMs: 18 * 60 * 60_000, relative: '18h ago', size: '178 B' },
  { seq: 1020, subject: 'PROFILES.PERSONS.NEW.7a6f663d', offsetMs: 24 * 60 * 60_000, relative: '1d ago', size: '160 B' },
]

function TimestampFormatPreview() {
  const [fmt, setFmt] = useState<'relative' | 'absolute' | 'iso'>('relative')
  const now = useMemo(() => new Date(), [])

  const rows = useMemo(
    () =>
      TIMESTAMP_ROWS.map((r) => ({
        ...r,
        date: new Date(now.getTime() - r.offsetMs),
      })),
    [now],
  )

  return (
    <PreviewCard context="Stream → Messages (history) · Received column">
      <div className="flex items-center justify-between mb-3 gap-2 flex-wrap">
        <OptionToggle
          value={fmt}
          onChange={setFmt}
          options={[
            { value: 'relative', label: 'Relative' },
            { value: 'absolute', label: 'Absolute' },
            { value: 'iso', label: 'ISO 8601' },
          ]}
        />
        <CopyButton
          variant="button"
          size="sm"
          label={`Copy ${fmt}`}
          value={() => rows.map((r) => formatTimestamp(r.date, fmt, r.relative)).join('\n')}
        />
      </div>

      <div className="border border-border rounded-md bg-surface-primary overflow-hidden">
        <div className="flex items-center gap-3 px-3 py-1.5 bg-surface-secondary border-b border-border text-2xs uppercase tracking-wide text-content-tertiary font-medium">
          <span className="w-14 shrink-0">Seq</span>
          <span className="flex-1 min-w-0">Subject</span>
          <span className="w-44 shrink-0 inline-flex items-center gap-1">
            Received
            <span className="text-accent normal-case tracking-normal">← affected</span>
          </span>
          <span className="w-12 shrink-0 text-right">Size</span>
        </div>
        {rows.map((r, i) => {
          const formatted = formatTimestamp(r.date, fmt, r.relative)
          return (
            <div
              key={r.seq}
              className={`flex items-center gap-3 px-3 py-2 text-xs ${
                i < rows.length - 1 ? 'border-b border-gray-100' : ''
              }`}
            >
              <span className="font-mono text-gray-700 w-14 shrink-0">{r.seq}</span>
              <span className="font-mono text-content-primary truncate flex-1 min-w-0">{r.subject}</span>
              <span className="font-mono text-gray-700 ring-2 ring-blue-400/70 bg-accent-light/60 rounded px-1.5 py-0.5 select-text w-44 shrink-0 truncate">
                {formatted}
              </span>
              <span className="text-content-tertiary w-12 shrink-0 text-right">{r.size}</span>
            </div>
          )
        })}
      </div>
    </PreviewCard>
  )
}

function formatTimestamp(date: Date, fmt: 'relative' | 'absolute' | 'iso', relative: string): string {
  if (fmt === 'relative') return relative
  return formatTimestampSetting(date, fmt)
}

const JSON_SAMPLE = {
  order: {
    id: 'abc-123',
    items: [
      { sku: 'WIDGET-01', qty: 2 },
      { sku: 'GADGET-7', qty: 1 },
    ],
  },
}

function JsonIndentPreview() {
  const [indent, setIndent] = useState<2 | 4>(2)
  const formatted = useMemo(() => JSON.stringify(JSON_SAMPLE, null, indent), [indent])

  return (
    <PreviewCard context="Message detail → Decoded / JSON viewer (collapsed nodes)">
      <div className="flex items-center justify-between mb-2">
        <OptionToggle
          value={indent}
          onChange={(v) => setIndent(v as 2 | 4)}
          options={[
            { value: 2, label: '2 spaces' },
            { value: 4, label: '4 spaces' },
          ]}
        />
        <CopyButton variant="button" size="sm" value={formatted} label="Copy JSON" />
      </div>
      <pre className="bg-surface-inverse text-gray-100 rounded-md text-xs font-mono p-3 overflow-x-auto whitespace-pre leading-5 m-0 select-text">
        {formatted}
      </pre>
    </PreviewCard>
  )
}

const MOCK_SEQS = [
  { seq: 1, subject: 'orders.create', label: 'first order ever' },
  { seq: 2, subject: 'orders.create', label: 'second order' },
  { seq: 3, subject: 'orders.update', label: 'third event' },
  { seq: 1003, subject: 'orders.delete', label: 'older order' },
  { seq: 1004, subject: 'orders.update', label: 'previous order' },
  { seq: 1005, subject: 'orders.create', label: 'latest order' },
]

function DefaultDirectionPreview() {
  const [direction, setDirection] = useState<'backward' | 'forward'>('backward')
  const ordered = useMemo(
    () => (direction === 'backward' ? [...MOCK_SEQS].reverse() : MOCK_SEQS),
    [direction],
  )

  return (
    <PreviewCard context="Stream → Messages — initial sort order">
      <div className="flex items-center justify-between mb-3">
        <OptionToggle
          value={direction}
          onChange={setDirection}
          options={[
            { value: 'backward', label: 'Backward (newest first)' },
            { value: 'forward', label: 'Forward (oldest first)' },
          ]}
        />
        <span className="text-xs text-content-tertiary">
          {direction === 'backward' ? '↓ newest at top' : '↑ oldest at top'}
        </span>
      </div>
      <div className="border border-border rounded-md bg-surface-primary overflow-hidden">
        {ordered.map((row, i) => (
          <div
            key={row.seq}
            className={`flex items-center gap-3 px-3 py-2 text-xs ${
              i < ordered.length - 1 ? 'border-b border-gray-100' : ''
            } ${i === 0 ? 'bg-accent-light/50' : ''}`}
          >
            <span className="font-mono text-content-tertiary w-14 shrink-0">#{row.seq}</span>
            <span className="font-mono text-accent-text w-32 shrink-0 truncate">{row.subject}</span>
            <span className="text-gray-700 truncate flex-1 min-w-0">{row.label}</span>
            {i === 0 && (
              <span className="text-2xs uppercase tracking-wide text-accent font-semibold shrink-0">
                shown first
              </span>
            )}
          </div>
        ))}
      </div>
    </PreviewCard>
  )
}

interface MockMsg {
  id: number
  seq: number
  subject: string
  received: string
  size: string
}

const HISTORY_SEED: ReadonlyArray<Omit<MockMsg, 'id'>> = [
  { seq: 1022, subject: 'PROFILES.PERSONS.UPDATES.e28c1843', received: '13h ago', size: '404 B' },
  { seq: 1021, subject: 'PROFILES.PERSONS.NEW.0e7cc5cb', received: '18h ago', size: '178 B' },
  { seq: 1020, subject: 'PROFILES.PERSONS.NEW.7a6f663d', received: '1d ago', size: '160 B' },
  { seq: 1019, subject: 'PROFILES.PERSONS.NEW.29b3b513', received: '1d ago', size: '179 B' },
]

const REALTIME_PATTERNS: ReadonlyArray<{ subject: string; size: string }> = [
  { subject: 'PROFILES.PERSONS.NEW.6a91-c4b2', size: '178 B' },
  { subject: 'PROFILES.PERSONS.UPDATES.7f33-d201', size: '404 B' },
  { subject: 'PROFILES.PERSONS.NEW.b84e-f102', size: '160 B' },
  { subject: 'PROFILES.PERSONS.UPDATES.a02d-9c41', size: '391 B' },
]

function DefaultViewModePreview() {
  const [mode, setMode] = useState<'history' | 'realtime'>('history')
  const [messages, setMessages] = useState<MockMsg[]>(() =>
    HISTORY_SEED.map((m, i) => ({ ...m, id: i + 1 })),
  )
  const [msgsPerSec, setMsgsPerSec] = useState(142)
  const idRef = useRef(100)
  const seqRef = useRef(2000)

  useEffect(() => {
    if (mode === 'history') {
      setMessages(HISTORY_SEED.map((m, i) => ({ ...m, id: i + 1 })))
      return
    }
    // Realtime: seed and start the ticker.
    seqRef.current = 2000
    const seedRows: MockMsg[] = REALTIME_PATTERNS.slice(0, 3).map((p, i) => ({
      id: ++idRef.current,
      seq: seqRef.current - i,
      subject: p.subject,
      size: p.size,
      received: i === 0 ? 'just now' : `${i}s ago`,
    }))
    setMessages(seedRows)
    setMsgsPerSec(142)

    const tick = setInterval(() => {
      seqRef.current += 1
      const pattern = REALTIME_PATTERNS[seqRef.current % REALTIME_PATTERNS.length]
      const newMsg: MockMsg = {
        id: ++idRef.current,
        seq: seqRef.current,
        subject: pattern.subject,
        size: pattern.size,
        received: 'just now',
      }
      setMessages((prev) => {
        const aged = prev.map((m, i) => ({ ...m, received: i === 0 ? '1s ago' : `${i + 1}s ago` }))
        return [newMsg, ...aged].slice(0, 4)
      })
      setMsgsPerSec(120 + Math.floor(Math.random() * 60))
    }, 900)

    return () => clearInterval(tick)
  }, [mode])

  const totalMessages = mode === 'realtime' ? 1024 + (seqRef.current - 2000) : 29

  return (
    <PreviewCard context="Stream → first open · default initial view">
      <div className="mb-3">
        <OptionToggle
          value={mode}
          onChange={setMode}
          options={[
            { value: 'history', label: 'History' },
            { value: 'realtime', label: 'Realtime' },
          ]}
        />
      </div>

      {/* Light-theme mock matching the actual stream view. */}
      <div className="border border-border rounded-md bg-surface-primary overflow-hidden text-xs">
        <div className="px-3 pt-2 pb-1">
          <div className="font-semibold text-content-primary text-sm">Stream: ORDERS</div>
        </div>

        <div className="px-3 flex items-center gap-4 border-b border-border">
          <span className="py-1.5 border-b-2 border-accent text-accent font-medium">Messages</span>
          <span className="py-1.5 text-content-tertiary">Config</span>
          <span className="py-1.5 text-content-tertiary">Consumers</span>
          <span className="py-1.5 text-content-tertiary">Publish</span>
        </div>

        <div className="px-3 py-1.5 border-b border-gray-100 text-2xs text-content-tertiary">
          Messages <span className="text-gray-700 font-medium">{totalMessages}</span>
          {' · '}Size <span className="text-gray-700 font-medium">10.82 KB</span>
          {mode === 'realtime' && (
            <>
              {' · '}Messages/s <span className="text-gray-700 font-medium tabular-nums">{msgsPerSec}</span>
            </>
          )}
        </div>

        {/* Toolbar with Realtime/History toggle */}
        <div className="px-3 py-1.5 flex items-center justify-between border-b border-gray-100 bg-surface-secondary/50">
          <span className="text-2xs text-content-secondary">{totalMessages} messages</span>
          <div className="inline-flex bg-surface-hover p-0.5 rounded text-2xs">
            <span className={`px-2 py-0.5 rounded ${mode === 'realtime' ? 'bg-surface-primary text-content-primary shadow-sm font-medium' : 'text-content-tertiary'}`}>
              Realtime
            </span>
            <span className={`px-2 py-0.5 rounded ${mode === 'history' ? 'bg-accent text-content-inverse shadow-sm font-medium' : 'text-content-tertiary'}`}>
              History
            </span>
          </div>
        </div>

        <div className="flex items-center gap-3 px-3 py-1.5 bg-surface-secondary border-b border-border text-2xs uppercase tracking-wide text-content-tertiary font-medium">
          <span className="w-12 shrink-0">Seq</span>
          <span className="flex-1 min-w-0">Subject</span>
          <span className="w-20 shrink-0 text-right">Received</span>
          <span className="w-14 shrink-0 text-right">Size</span>
        </div>

        <div className="min-h-[120px]">
          {messages.map((m) => (
            <div
              key={m.id}
              className="flex items-center gap-3 px-3 py-1.5 border-b border-gray-100 last:border-b-0 animate-fade-in"
            >
              <span className="font-mono text-gray-700 w-12 shrink-0">{m.seq}</span>
              <span className="font-mono text-accent-text truncate flex-1 min-w-0">{m.subject}</span>
              <span className="text-content-tertiary w-20 shrink-0 text-right">{m.received}</span>
              <span className="text-content-tertiary w-14 shrink-0 text-right tabular-nums">{m.size}</span>
            </div>
          ))}
        </div>

        {/* Footer — pagination for history, live indicator for realtime */}
        <div className="px-3 py-1.5 bg-surface-secondary border-t border-border flex items-center justify-between text-2xs text-content-secondary">
          {mode === 'history' ? (
            <>
              <span>Page 1 of 20</span>
              <span className="flex items-center gap-3">
                <span className="text-content-muted">← Prev</span>
                <span className="text-accent">Next →</span>
              </span>
            </>
          ) : (
            <>
              <span className="inline-flex items-center gap-1.5 text-status-error-text">
                <span className="relative inline-flex w-2 h-2">
                  <span className="absolute inset-0 rounded-full bg-red-500 opacity-75 animate-ping" />
                  <span className="relative inline-block w-2 h-2 rounded-full bg-red-500" />
                </span>
                <span className="font-medium tabular-nums">Live · {msgsPerSec} msgs/sec</span>
              </span>
              <span className="text-content-muted italic">auto-updating…</span>
            </>
          )}
        </div>
      </div>
    </PreviewCard>
  )
}

const AUTOSCROLL_PATTERNS = [
  { subject: 'PROFILES.PERSONS.NEW.6a91c4b2', size: '178 B' },
  { subject: 'PROFILES.PERSONS.UPDATES.7f33d201', size: '404 B' },
  { subject: 'PROFILES.PERSONS.NEW.b84ef102', size: '160 B' },
  { subject: 'PROFILES.PERSONS.UPDATES.a02d9c41', size: '391 B' },
]

interface AutoScrollMsg {
  id: number
  seq: number
  subject: string
  size: string
}

function AutoScrollLivePreview() {
  const [enabled, setEnabled] = useState(true)
  const [topRows, setTopRows] = useState<AutoScrollMsg[]>(() =>
    AUTOSCROLL_PATTERNS.slice(0, 4).map((p, i) => ({
      id: i + 1,
      seq: 1020 - i,
      subject: p.subject,
      size: p.size,
    })),
  )
  const [pendingCount, setPendingCount] = useState(0)
  const idRef = useRef(100)
  const seqRef = useRef(1020)

  useEffect(() => {
    seqRef.current = 1020
    setTopRows(
      AUTOSCROLL_PATTERNS.slice(0, 4).map((p, i) => ({
        id: ++idRef.current,
        seq: 1020 - i,
        subject: p.subject,
        size: p.size,
      })),
    )
    setPendingCount(0)

    const tick = setInterval(() => {
      seqRef.current += 1
      const pattern = AUTOSCROLL_PATTERNS[seqRef.current % AUTOSCROLL_PATTERNS.length]
      const newMsg: AutoScrollMsg = {
        id: ++idRef.current,
        seq: seqRef.current,
        subject: pattern.subject,
        size: pattern.size,
      }
      if (enabled) {
        // Auto-scroll: new message appears at top, list pushed down.
        setTopRows((prev) => [newMsg, ...prev].slice(0, 4))
      } else {
        // No auto-scroll: badge counter grows, list stays put.
        setPendingCount((c) => c + 1)
      }
    }, 1100)
    return () => clearInterval(tick)
  }, [enabled])

  const handleScrollTop = () => {
    setPendingCount(0)
    // Pretend the user scrolled up — flush pending into the visible list.
    seqRef.current += 0
  }

  return (
    <PreviewCard context="Stream → Messages (Realtime mode) · scroll behavior">
      <div className="mb-3">
        <OptionToggle
          value={enabled ? 'on' : 'off'}
          onChange={(v) => setEnabled(v === 'on')}
          options={[
            { value: 'on', label: 'Enabled (default)' },
            { value: 'off', label: 'Disabled' },
          ]}
        />
      </div>

      <div className="border border-border rounded-md bg-surface-primary overflow-hidden text-xs">
        {/* "New messages" badge appears only when disabled and there are pending */}
        {!enabled && pendingCount > 0 && (
          <button
            type="button"
            onClick={handleScrollTop}
            className="w-full py-1.5 bg-accent-light border-b border-blue-200 text-accent-text text-2xs font-medium flex items-center justify-center gap-1 hover:bg-accent-muted animate-fade-in"
          >
            <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 10l7-7m0 0l7 7m-7-7v18" />
            </svg>
            {plural(pendingCount, 'new message')} · click to load
          </button>
        )}

        <div className="flex items-center gap-3 px-3 py-1.5 bg-surface-secondary border-b border-border text-2xs uppercase tracking-wide text-content-tertiary font-medium">
          <span className="w-12 shrink-0">Seq</span>
          <span className="flex-1 min-w-0">Subject</span>
          <span className="w-14 shrink-0 text-right">Size</span>
        </div>

        <div className="min-h-[120px]">
          {topRows.map((m) => (
            <div
              key={m.id}
              className="flex items-center gap-3 px-3 py-1.5 border-b border-gray-100 last:border-b-0 animate-fade-in"
            >
              <span className="font-mono text-gray-700 w-12 shrink-0">{m.seq}</span>
              <span className="font-mono text-accent-text truncate flex-1 min-w-0">{m.subject}</span>
              <span className="text-content-tertiary w-14 shrink-0 text-right tabular-nums">{m.size}</span>
            </div>
          ))}
        </div>
      </div>

      <p className="text-2xs text-content-secondary mt-2">
        {enabled
          ? '↑ New messages appear at the top automatically. The list scrolls so the newest is always visible.'
          : '↑ List stays put. Incoming messages are counted in a banner at the top — click it to jump to the latest.'}
      </p>
    </PreviewCard>
  )
}

const TOTAL_MESSAGES = 1024

function DefaultPageSizePreview() {
  const [pageSize, setPageSize] = useState<25 | 50 | 100 | 250 | 500>(50)
  const totalPages = Math.ceil(TOTAL_MESSAGES / pageSize)
  const startSeq = TOTAL_MESSAGES
  const endSeq = TOTAL_MESSAGES - pageSize + 1

  return (
    <PreviewCard context="Stream → Messages · pagination footer">
      <div className="flex items-center justify-between mb-3 gap-2 flex-wrap">
        <OptionToggle
          value={pageSize}
          onChange={(v) => setPageSize(v as 25 | 50 | 100 | 250 | 500)}
          options={[
            { value: 25, label: '25' },
            { value: 50, label: '50' },
            { value: 100, label: '100' },
            { value: 250, label: '250' },
            { value: 500, label: '500' },
          ]}
        />
        <span className="text-2xs text-content-tertiary">
          Total stream: <span className="font-medium text-gray-700">{formatCount(TOTAL_MESSAGES)}</span> messages
        </span>
      </div>

      <div className="border border-border rounded-md bg-surface-primary overflow-hidden text-xs">
        <div className="flex items-center gap-3 px-3 py-1.5 bg-surface-secondary border-b border-border text-2xs uppercase tracking-wide text-content-tertiary font-medium">
          <span className="w-12 shrink-0">Seq</span>
          <span className="flex-1 min-w-0">Subject</span>
          <span className="w-14 shrink-0 text-right">Size</span>
        </div>

        {/* A few rows just to set context. */}
        <div>
          {[0, 1, 2].map((i) => (
            <div
              key={i}
              className="flex items-center gap-3 px-3 py-1.5 border-b border-gray-100 text-gray-700"
            >
              <span className="font-mono w-12 shrink-0">{startSeq - i}</span>
              <span className="font-mono text-accent-text truncate flex-1 min-w-0">
                PROFILES.PERSONS.UPDATES.{(startSeq - i).toString(16)}
              </span>
              <span className="text-content-tertiary w-14 shrink-0 text-right tabular-nums">
                {180 + ((i * 17) % 200)} B
              </span>
            </div>
          ))}
          <div className="flex items-center gap-3 px-3 py-2 text-2xs text-content-muted italic">
            … {pageSize - 3} more rows on this page …
          </div>
        </div>

        {/* Pagination footer — the part that this setting actually controls. */}
        <div className="px-3 py-2 bg-surface-secondary border-t border-border flex items-center justify-between text-2xs ring-2 ring-blue-400/70 ring-inset">
          <span className="text-content-secondary">
            Showing seqs <span className="font-mono text-content-primary">{endSeq}</span>–
            <span className="font-mono text-content-primary">{startSeq}</span> ({pageSize} per page)
          </span>
          <span className="text-content-secondary">
            Page <span className="font-medium text-content-primary">1</span> of{' '}
            <span className="font-medium text-content-primary">{formatCount(totalPages)}</span>
          </span>
        </div>
      </div>

      <p className="text-2xs text-accent-text mt-2">
        ↑ Page size = {pageSize} → {formatCount(totalPages)} pages for {formatCount(TOTAL_MESSAGES)} messages.
        {pageSize >= 250 && ' Large pages mean more memory in the browser per load.'}
        {pageSize <= 25 && ' Small pages mean more clicks to browse.'}
      </p>
    </PreviewCard>
  )
}

