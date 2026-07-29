/**
 * Static help content for settings fields.
 * Keys follow `section.field` naming (e.g. `messages.fetchMethod`).
 */
export interface HelpEntry {
  title: string
  body: string
}

export const HELP: Record<string, HelpEntry> = {
  'messages.fetchMethod': {
    title: 'Fetch Method',
    body: `How messages are retrieved from NATS JetStream streams.

---

**Direct**
Fetches by sequence number via GetMsg API. No consumer created on the server. Fast random access — ideal for browsing.

\`\`\`
Stream: ORDERS  (1,000,000 messages)

  GetMsg(seq=999500)
  GetMsg(seq=999501)
  ...
  GetMsg(seq=999550)

  → 50 messages loaded in ~20ms (20 parallel fetches)
\`\`\`

---

**Consumer** (default)
Creates a temporary ordered consumer with server-side subject filtering. The server sends only matching messages — no wasted bandwidth.

\`\`\`
Stream: ORDERS
  subjects: orders.us.*  orders.eu.*  orders.asia.*

  Consumer filter: "orders.eu.>"

  → Server sends only EU orders
  → Sparse streams handled correctly
\`\`\`

---

**When to use what?**
• **Direct** — general browsing, jumping to specific sequences
• **Consumer** — filtered views, sparse streams with many deleted messages`,
  },
  'messages.defaultPageSize': {
    title: 'Default Page Size',
    body: `Number of messages loaded per page in history mode. Default is 50; max 500 (server hard limit).

Affects both the initial load and each "load more" request. Larger pages use more memory in the browser; smaller pages mean more clicks to browse.`,
  },
  'messages.defaultDirection': {
    title: 'Default Direction',
    body: `Initial sort order when you open a stream.

**Backward** (default) — newest message first.
**Forward** — oldest message first.

You can always toggle direction in the toolbar — this setting only controls the default.`,
  },
  'live.subscriptionMode': {
    title: 'Subscription Mode',
    body: `How the live feed connects to NATS for real-time messages.

---

**Core NATS** (default)
Simple raw subscription. Only receives messages published after connecting.

\`\`\`
  Client ── Subscribe("orders.>") ──→ NATS Server
           ←── only NEW messages ───
\`\`\`

• Lightweight, lowest latency
• No consumer state on the server
• Messages during disconnect are lost

---

**JetStream Ordered**
Creates an ephemeral ordered consumer. Can replay history.

\`\`\`
  Client ── OrderedConsumer(stream) ──→ NATS Server
           ←── history + new msgs ────
\`\`\`

• Replay last message on connect (see Deliver Policy)
• Guaranteed message ordering
• Server-side subject filtering
• Auto-recreates consumer on gaps

---

**When to use what?**
• **Core NATS** — simple monitoring, lowest overhead
• **JetStream Ordered** — need history replay or ordering guarantees`,
  },
  'live.maxDisplayRate': {
    title: 'Max Display Rate',
    body: `Limits how many messages per second are sent from the server to the browser. Uses a token bucket algorithm — messages over the limit are dropped server-side before serialization, saving CPU and bandwidth.

\`\`\`
0      →  unlimited (default, all messages sent)
5      →  5 msg/s (good for monitoring high-throughput streams)
50     →  50 msg/s (balanced for moderate streams)
1000   →  1000 msg/s (light throttle)
\`\`\`

Dropped messages are still counted in total stats. The toolbar shows "X dropped" when throttling is active. Requires reconnecting to the live stream to take effect.

**Important:** the throttle is display-only — dropped messages are skipped in the live view but **remain in the stream**. Real consumers reading from the stream are unaffected.`,
  },
  'display.density': {
    title: 'Density',
    body: `Controls row height and padding in message lists.

**Comfortable** (default) — 52px rows, more whitespace · ~12 messages visible on a 1080p screen.

**Compact** — 36px rows, tighter spacing · ~18 messages visible on a 1080p screen (+50%).`,
  },
  'display.defaultViewMode': {
    title: 'Default View Mode',
    body: `What you see when you first open a stream.

**History** (default) — paginated browse of stored messages, no live updates.
**Realtime** — immediately starts a live subscription; new messages stream in as they arrive.`,
  },
  'display.timestampFormat': {
    title: 'Timestamp Format',
    body: `How timestamps appear in the message list.

**Relative** (default) — human-friendly, auto-updates ("3s ago", "2m ago").
**Absolute** — browser locale format, fixed at the message receive time.
**ISO 8601** — precise, copy-pasteable for log correlation.`,
  },
  'display.jsonIndentSize': {
    title: 'JSON Indent Size',
    body: `Indentation in the message payload viewer.

**2 spaces** (default) — more compact, fits more on screen.
**4 spaces** — easier to read for deeply nested structures.`,
  },
  'display.autoScrollLive': {
    title: 'Auto-scroll Live',
    body: `Controls scroll behavior when new messages arrive in live mode.

**Enabled** (default) — the list scrolls to the top so the newest message is always visible.
**Disabled** — the list stays put; an "↑ N new messages" banner appears at the top. Click it to jump to the latest.

When enabled, scrolling down manually pauses auto-scroll. Scrolling back to the top re-enables it.`,
  },
  'publish.publishTimeoutSec': {
    title: 'Publish Timeout',
    body: `How long to wait for the NATS server to acknowledge a published message.

\`\`\`
You click "Publish"
  → message sent to NATS server
  → server writes to stream
  → server sends ACK back
  → total time must be < timeout

Timeout: 5s   →  strict, fails fast on slow servers
Timeout: 10s  →  balanced (default)
Timeout: 30s  →  tolerant, for slow/remote servers
Timeout: 60s  →  very tolerant, for unreliable networks
\`\`\`

If publishing fails with "timeout", increase this value. Common causes: high server load, network latency, slow disk I/O on the NATS server.`,
  },
}
