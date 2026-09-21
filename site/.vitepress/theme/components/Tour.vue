<script setup>
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { withBase } from 'vitepress'

const INSTALL_CMD = 'brew install dmit-4884/tap/natscope'
const STEP_MS = 7000
const VIDEO_GRACE_MS = 4000
const REVEAL_RATIO = 0.12
const PLAY_RATIO = 0.3

const chapters = [
  {
    id: 'streams',
    title: 'Streams and messages',
    lede: 'The sidebar lists every JetStream stream on the server. Open one and the Messages tab shows what is inside.',
    tint: '#e8effd',
    tintDark: '#1a2540',
    steps: [
      {
        id: 'browse',
        title: 'Browse and filter',
        body: 'Stored messages arrive newest first, with paging in both directions. Filters narrow the list by subject pattern, payload text, start sequence or timestamp.',
        media: { type: 'image', src: '/media/messages-filters.png', alt: 'Message list with the filter panel open' }
      },
      {
        id: 'read',
        title: 'Read a payload',
        body: 'Click a row and the side panel shows headers, timing and the payload. When a Protobuf mapping matches the subject, the payload renders as JSON.',
        media: { type: 'image', src: '/media/message-decoded.png', alt: 'Message viewer showing a Protobuf payload decoded to JSON' }
      },
      {
        id: 'hex',
        title: 'See the raw bytes',
        body: 'The same payload as raw text, hex or Base64 when you need to see the exact bytes on the wire.',
        media: { type: 'image', src: '/media/message-hex.png', alt: 'Message viewer in hex mode' }
      },
      {
        id: 'tail',
        title: 'Tail it live',
        body: 'Flip the toolbar from History to Realtime and messages stream in as they arrive. A display-rate limit keeps a busy queue readable without slowing the subscription.',
        media: { type: 'video', src: '/media/live.mp4', poster: '/media/live.jpg', alt: 'Realtime tail of a stream', seconds: 12 }
      },
      {
        id: 'config',
        title: 'Edit the config',
        body: 'Change a stream in place. Saving shows a field-level diff before anything reaches the server. Purge, seal and delete sit under the same tab, each behind a confirmation.',
        media: { type: 'image', src: '/media/streams-info.png', alt: 'Stream config tab' }
      },
      {
        id: 'create',
        title: 'Create a stream',
        body: 'A form with every config field, or a JSON view that takes a full JetStream config document.',
        media: { type: 'image', src: '/media/stream-create.png', alt: 'Create stream form' }
      }
    ]
  },
  {
    id: 'proto',
    title: 'Protobuf decoding',
    lede: 'Binary payloads read as JSON everywhere a message shows up: the browser, live tail and the publish preview.',
    tint: '#e3f3ea',
    tintDark: '#16291f',
    steps: [
      {
        id: 'sources',
        title: 'Add a proto source',
        body: 'Point Natscope at a Git repository, a local directory or an uploaded set of .proto files. It compiles them, resolves buf.lock dependencies and names the file and line when something fails.',
        media: { type: 'image', src: '/media/proto-sources.png', alt: 'Proto sources in Settings' }
      },
      {
        id: 'mappings',
        title: 'Map subjects to types',
        body: 'Bind a subject pattern to a message type, wildcards included. Each mapping reports its own health, so a missing type shows up in the list instead of failing at decode time.',
        media: { type: 'image', src: '/media/proto-mappings.png', alt: 'Subject mappings in Settings' }
      },
      {
        id: 'decode',
        title: 'Watch it decode',
        body: 'With a source and a mapping in place, the binary payload in the message viewer turns into JSON. Edit a watched .proto on disk and running tails re-decode with the new schema without reconnecting.',
        media: { type: 'video', src: '/media/proto.mp4', poster: '/media/proto.jpg', alt: 'A binary payload decoding after a mapping is added', seconds: 24 }
      }
    ]
  },
  {
    id: 'publish',
    title: 'Publishing',
    lede: 'The Publish tab sends a message from the same screen you read them on. With a mapping resolved, JSON is encoded to Protobuf on the way out.',
    tint: '#fdf0dc',
    tintDark: '#2c2416',
    steps: [
      {
        id: 'editor',
        title: 'Write against the schema',
        body: 'Field autocompletion and live validation against the resolved message type. Add NATS headers as key/value pairs and send with Cmd+Enter.',
        media: { type: 'image', src: '/media/publish-form.png', alt: 'Publish form with schema validation' }
      },
      {
        id: 'templates',
        title: 'Save templates',
        body: 'Keep a draft as a template with its subject, message type, body and headers. Load it back from the Templates dropdown on any stream.',
        media: { type: 'image', src: '/media/templates-list.png', alt: 'Template list in Settings' }
      },
      {
        id: 'history',
        title: 'Keep the history',
        body: 'Every publish is logged with its subject, encoding, payload and result. Load an entry back into the form, or copy its payload.',
        media: { type: 'image', src: '/media/publish-history.png', alt: 'Publish history panel' }
      }
    ]
  },
  {
    id: 'jetstream',
    title: 'Consumers, KV and Object Store',
    lede: 'The rest of JetStream from the same sidebar, without dropping to the nats CLI.',
    tint: '#eeebfa',
    tintDark: '#221f36',
    steps: [
      {
        id: 'consumers',
        title: 'Manage consumers',
        body: 'Pull and push, durable and ephemeral. Create one, edit it with a config diff, pause it until a chosen time, or copy it as a nats consumer add command.',
        media: { type: 'image', src: '/media/consumers-detail.png', alt: 'Consumer detail view' }
      },
      {
        id: 'kv',
        title: 'Edit KV values',
        body: 'Browse buckets and keys. Writes are compare-and-set on the revision you loaded, and History lists every put, delete and purge of a key.',
        media: { type: 'image', src: '/media/kv-history.png', alt: 'KV key revision history' }
      },
      {
        id: 'objects',
        title: 'Move objects in and out',
        body: 'Upload and download objects, delete them, and seal a bucket when it should stop accepting writes.',
        media: { type: 'image', src: '/media/objects-list.png', alt: 'Object bucket listing' }
      }
    ]
  },
  {
    id: 'local',
    title: 'Runs on your machine',
    lede: 'One binary with the UI embedded. State lives in a single bbolt file, secrets in the OS keychain.',
    tint: '#eef1f5',
    tintDark: '#1e2229',
    steps: [
      {
        id: 'connections',
        title: 'Save connections',
        body: 'Keep as many NATS endpoints as you need with user/pass, token, NKey or credentials auth, plus TLS or mTLS. Test the round trip before you connect.',
        media: { type: 'image', src: '/media/connections-list.png', alt: 'Saved connections' }
      },
      {
        id: 'workspace',
        title: 'Carry your workspace',
        body: 'Export connections, proto sources, mappings, templates and settings as one JSON file and import it on another machine. Secrets never leave the vault.',
        media: { type: 'image', src: '/media/workspace-export.png', alt: 'Workspace export' }
      },
      {
        id: 'preferences',
        title: 'Set your preferences',
        body: 'Paging, live tail, density, timestamp format and per-action confirmations, saved on the backend so they follow you across browsers.',
        media: { type: 'image', src: '/media/settings.png', alt: 'Preferences' }
      }
    ]
  }
]

const active = ref(chapters.map(() => 0))
const auto = ref(chapters.map(() => true))
const paused = ref(chapters.map(() => false))
const revealed = ref(chapters.map(() => false))
const runs = ref(chapters.map(() => 0))
const mounted = ref(false)
const reducedMotion = ref(false)
const zoomed = ref(-1)
const pillVisible = ref(false)
const copied = ref(false)
const rootEl = ref(null)

const inView = chapters.map(() => false)
const timers = chapters.map(() => 0)
const chapterEls = []
const tabEls = chapters.map(() => [])
let observer = null
let frame = 0

const stepOf = (ci) => chapters[ci].steps[active.value[ci]]
const durationOf = (step) => (step.media.seconds ? step.media.seconds * 1000 : STEP_MS)
const zoomable = (ci) => !(reducedMotion.value && stepOf(ci).media.type === 'video')
const running = (ci) => mounted.value && auto.value[ci] && !paused.value[ci] && inView[ci] && !reducedMotion.value && !document.hidden

const clearTimer = (ci) => {
  if (timers[ci]) {
    clearTimeout(timers[ci])
    timers[ci] = 0
  }
}

const schedule = (ci) => {
  clearTimer(ci)
  if (!running(ci)) return
  runs.value[ci] += 1
  const step = stepOf(ci)
  const delay = durationOf(step) + (step.media.type === 'video' ? VIDEO_GRACE_MS : 0)
  timers[ci] = setTimeout(() => {
    timers[ci] = 0
    advance(ci)
  }, delay)
}

const advance = (ci) => {
  active.value[ci] = (active.value[ci] + 1) % chapters[ci].steps.length
  schedule(ci)
}

const select = (ci, si) => {
  auto.value[ci] = false
  clearTimer(ci)
  active.value[ci] = si
}

const onVideoEnded = (ci) => {
  if (running(ci)) advance(ci)
}

const pause = (ci) => {
  paused.value[ci] = true
  clearTimer(ci)
}

const resume = (ci) => {
  paused.value[ci] = false
  schedule(ci)
}

const onTabKey = (ci, e) => {
  const count = chapters[ci].steps.length
  const current = active.value[ci]
  let next = null
  if (e.key === 'ArrowRight') next = (current + 1) % count
  else if (e.key === 'ArrowLeft') next = (current - 1 + count) % count
  else if (e.key === 'Home') next = 0
  else if (e.key === 'End') next = count - 1
  if (next === null) return
  e.preventDefault()
  select(ci, next)
  tabEls[ci][next]?.focus()
}

const setChapterRef = (el, ci) => {
  chapterEls[ci] = el
}

const setTabRef = (el, ci, si) => {
  tabEls[ci][si] = el
}

const onKey = (e) => {
  if (e.key === 'Escape') zoomed.value = -1
}

watch(zoomed, (ci, previous) => {
  if (ci >= 0) {
    document.addEventListener('keydown', onKey)
    document.body.style.overflow = 'hidden'
    pause(ci)
  } else {
    document.removeEventListener('keydown', onKey)
    document.body.style.overflow = ''
    if (previous >= 0) {
      chapterEls[previous]?.querySelector('.ns-zoom-btn')?.focus({ preventScroll: true })
    }
  }
})

const updatePill = () => {
  frame = 0
  const install = document.getElementById('install')
  const pastHero = rootEl.value ? rootEl.value.getBoundingClientRect().top < 0 : false
  const beforeInstall = install ? install.getBoundingClientRect().top > window.innerHeight : true
  pillVisible.value = pastHero && beforeInstall
}

const schedulePill = () => {
  if (frame) return
  frame = requestAnimationFrame(updatePill)
}

const onVisibility = () => {
  chapters.forEach((_, ci) => schedule(ci))
}

const copyCommand = async () => {
  try {
    await navigator.clipboard.writeText(INSTALL_CMD)
    copied.value = true
    setTimeout(() => {
      copied.value = false
    }, 1600)
  } catch {
    copied.value = false
  }
}

onMounted(() => {
  reducedMotion.value = window.matchMedia('(prefers-reduced-motion: reduce)').matches
  mounted.value = true
  observer = new IntersectionObserver(
    (entries) => {
      for (const entry of entries) {
        const ci = chapterEls.indexOf(entry.target)
        if (ci < 0) continue
        if (entry.isIntersecting && entry.intersectionRatio >= REVEAL_RATIO) revealed.value[ci] = true
        const visible = entry.isIntersecting && entry.intersectionRatio >= PLAY_RATIO
        if (visible === inView[ci]) continue
        inView[ci] = visible
        if (visible) schedule(ci)
        else clearTimer(ci)
      }
    },
    { threshold: [0, REVEAL_RATIO, PLAY_RATIO, 0.6] }
  )
  chapterEls.forEach((el) => el && observer.observe(el))
  window.addEventListener('scroll', schedulePill, { passive: true })
  window.addEventListener('resize', schedulePill)
  document.addEventListener('visibilitychange', onVisibility)
  updatePill()
})

onBeforeUnmount(() => {
  observer?.disconnect()
  chapters.forEach((_, ci) => clearTimer(ci))
  window.removeEventListener('scroll', schedulePill)
  window.removeEventListener('resize', schedulePill)
  document.removeEventListener('visibilitychange', onVisibility)
  document.removeEventListener('keydown', onKey)
  document.body.style.overflow = ''
  if (frame) cancelAnimationFrame(frame)
})
</script>

<template>
  <section ref="rootEl" class="ns-tour" :class="{ 'is-mounted': mounted }">
    <article
      v-for="(chapter, ci) in chapters"
      :key="chapter.id"
      :ref="(el) => setChapterRef(el, ci)"
      class="ns-chapter"
      :class="{ 'is-revealed': revealed[ci], 'is-paused': paused[ci] }"
      :style="{ '--ns-tint': chapter.tint, '--ns-tint-dark': chapter.tintDark }"
      @mouseenter="pause(ci)"
      @mouseleave="resume(ci)"
    >
      <div class="ns-chapter-head">
        <h2 class="ns-chapter-title">{{ chapter.title }}</h2>
        <p class="ns-chapter-lede">{{ chapter.lede }}</p>
      </div>

      <div class="ns-tabs" role="tablist" :aria-label="chapter.title" @keydown="onTabKey(ci, $event)">
        <button
          v-for="(step, si) in chapter.steps"
          :id="`ns-tab-${chapter.id}-${step.id}`"
          :key="step.id"
          :ref="(el) => setTabRef(el, ci, si)"
          type="button"
          role="tab"
          class="ns-tab"
          :class="{ 'is-active': si === active[ci] }"
          :aria-selected="si === active[ci]"
          :aria-controls="`ns-panel-${chapter.id}`"
          :tabindex="si === active[ci] ? 0 : -1"
          @click="select(ci, si)"
        >
          {{ step.title }}
          <span
            v-if="si === active[ci]"
            :key="runs[ci]"
            class="ns-tab-bar"
            :class="{ 'is-auto': auto[ci] && !reducedMotion }"
            :style="{ '--ns-step-ms': `${durationOf(step)}ms` }"
            aria-hidden="true"
          ></span>
        </button>
      </div>

      <div :id="`ns-panel-${chapter.id}`" class="ns-panel" role="tabpanel" :aria-labelledby="`ns-tab-${chapter.id}-${stepOf(ci).id}`">
        <p v-for="(step, si) in chapter.steps" :key="step.id" class="ns-step-body" :hidden="si !== active[ci]">{{ step.body }}</p>
        <div class="ns-mat">
          <div class="ns-frame">
            <div
              v-for="(step, si) in chapter.steps"
              :key="step.id"
              class="ns-layer"
              :class="{ 'is-active': si === active[ci] }"
              :aria-hidden="si !== active[ci]"
            >
              <img
                :src="withBase(step.media.type === 'video' ? step.media.poster : step.media.src)"
                :alt="step.media.alt"
                loading="lazy"
                decoding="async"
              />
              <video
                v-if="step.media.type === 'video' && si === active[ci]"
                :src="withBase(step.media.src)"
                :poster="withBase(step.media.poster)"
                :autoplay="!reducedMotion"
                :controls="reducedMotion"
                muted
                playsinline
                @ended="onVideoEnded(ci)"
              />
            </div>
            <button v-if="zoomable(ci)" type="button" class="ns-zoom-btn" aria-label="Show the full screenshot" @click="zoomed = ci">
              <span class="ns-zoom-hint" aria-hidden="true">Full screen</span>
            </button>
          </div>
        </div>
      </div>
    </article>
  </section>

  <Teleport to="body">
    <div class="ns-pill" :class="{ 'is-visible': pillVisible }" :aria-hidden="!pillVisible">
      <code class="ns-pill-cmd">{{ INSTALL_CMD }}</code>
      <button type="button" class="ns-pill-copy" :tabindex="pillVisible ? 0 : -1" @click="copyCommand">
        {{ copied ? 'Copied' : 'Copy' }}
      </button>
      <a class="ns-pill-cta" :href="withBase('/guide/installation')" :tabindex="pillVisible ? 0 : -1">Install guide</a>
    </div>
  </Teleport>

  <Teleport to="body">
    <div v-if="zoomed >= 0" class="ns-tour-zoom" @click="zoomed = -1">
      <video
        v-if="stepOf(zoomed).media.type === 'video'"
        :src="withBase(stepOf(zoomed).media.src)"
        :poster="withBase(stepOf(zoomed).media.poster)"
        controls
        autoplay
        muted
        loop
        playsinline
        @click.stop
      />
      <img v-else :src="withBase(stepOf(zoomed).media.src)" :alt="stepOf(zoomed).media.alt" />
    </div>
  </Teleport>
</template>

<style>
.vp-doc .ns-tour {
  padding-top: 8px;
}

.vp-doc .ns-chapter {
  padding: 40px 0 48px;
}

.vp-doc .ns-tour.is-mounted .ns-chapter {
  opacity: 0;
  transform: translateY(18px);
  transition: opacity 0.6s ease, transform 0.6s ease;
}

.vp-doc .ns-tour.is-mounted .ns-chapter.is-revealed {
  opacity: 1;
  transform: none;
}

.vp-doc .ns-chapter-head {
  max-width: 64ch;
}

.vp-doc .ns-tour h2.ns-chapter-title {
  margin: 0;
  padding: 0;
  border: 0;
  font-size: 28px;
  font-weight: 600;
  line-height: 1.2;
  letter-spacing: -0.02em;
  color: var(--vp-c-text-1);
}

.vp-doc .ns-tour p.ns-chapter-lede {
  margin: 12px 0 0;
  font-size: 17px;
  line-height: 1.55;
  color: var(--vp-c-text-2);
}

.vp-doc .ns-tabs {
  position: sticky;
  top: 0;
  z-index: 5;
  display: flex;
  gap: 4px;
  margin-top: 24px;
  border-bottom: 1px solid var(--vp-c-divider);
  background: var(--vp-c-bg);
  overflow-x: auto;
  scrollbar-width: none;
}

.vp-doc .ns-tabs::-webkit-scrollbar {
  display: none;
}

.vp-doc .ns-tab {
  position: relative;
  flex: none;
  padding: 10px 14px 12px;
  border: 0;
  background: none;
  font: inherit;
  font-size: 15px;
  font-weight: 500;
  line-height: 1.4;
  color: var(--vp-c-text-2);
  white-space: nowrap;
  cursor: pointer;
  transition: color 0.2s ease;
}

.vp-doc .ns-tab:first-child {
  padding-left: 0;
}

.vp-doc .ns-tab:hover,
.vp-doc .ns-tab.is-active {
  color: var(--vp-c-text-1);
}

.vp-doc .ns-tab:focus-visible {
  outline: 2px solid var(--vp-c-brand-1);
  outline-offset: -2px;
  border-radius: 4px;
}

.vp-doc .ns-tab-bar {
  position: absolute;
  right: 0;
  bottom: -1px;
  left: 0;
  height: 2px;
  background: var(--vp-c-brand-1);
  transform-origin: left;
}

.vp-doc .ns-tab-bar.is-auto {
  transform: scaleX(0);
  animation: ns-tab-progress var(--ns-step-ms) linear forwards;
}

.vp-doc .ns-chapter.is-paused .ns-tab-bar.is-auto {
  animation-play-state: paused;
}

@keyframes ns-tab-progress {
  to {
    transform: scaleX(1);
  }
}

.vp-doc .ns-tour p.ns-step-body {
  margin: 16px 0 18px;
  max-width: 72ch;
  min-height: 3.2em;
  font-size: 16px;
  line-height: 1.6;
  color: var(--vp-c-text-2);
}

.vp-doc .ns-mat {
  padding: clamp(12px, 1.8vw, 32px);
  border-radius: 24px;
  background: var(--ns-tint);
}

.dark .vp-doc .ns-mat {
  background: var(--ns-tint-dark);
}

.vp-doc .ns-frame {
  position: relative;
  aspect-ratio: 16 / 9;
  overflow: hidden;
  border: 1px solid rgba(15, 23, 42, 0.1);
  border-radius: 12px;
  background: var(--vp-c-bg);
  box-shadow: 0 30px 60px -30px rgba(15, 23, 42, 0.45);
}

.dark .vp-doc .ns-frame {
  border-color: rgba(255, 255, 255, 0.08);
  box-shadow: 0 30px 60px -30px rgba(0, 0, 0, 0.8);
}

.vp-doc .ns-layer {
  position: absolute;
  inset: 0;
  opacity: 0;
  transition: opacity 0.4s ease;
}

.vp-doc .ns-layer.is-active {
  opacity: 1;
}

.vp-doc .ns-layer img,
.vp-doc .ns-layer video {
  position: absolute;
  inset: 0;
  display: block;
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.vp-doc .ns-zoom-btn {
  position: absolute;
  inset: 0;
  z-index: 2;
  padding: 0;
  border: 0;
  background: transparent;
  cursor: zoom-in;
}

.vp-doc .ns-zoom-btn:focus-visible {
  outline: 2px solid var(--vp-c-brand-1);
  outline-offset: -2px;
}

.vp-doc .ns-zoom-hint {
  position: absolute;
  right: 12px;
  bottom: 12px;
  padding: 5px 11px;
  border: 1px solid rgba(255, 255, 255, 0.25);
  border-radius: 8px;
  background: rgba(17, 24, 39, 0.62);
  color: #fff;
  font-size: 12px;
  line-height: 1.4;
  opacity: 0;
  transition: opacity 0.15s ease;
}

.vp-doc .ns-frame:hover .ns-zoom-hint,
.vp-doc .ns-zoom-btn:focus-visible .ns-zoom-hint {
  opacity: 1;
}

.ns-tour-zoom {
  position: fixed;
  inset: 0;
  z-index: 200;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
  background: rgba(10, 14, 24, 0.88);
  cursor: zoom-out;
}

.ns-tour-zoom img,
.ns-tour-zoom video {
  max-width: min(1920px, 96vw);
  max-height: 94vh;
  border-radius: 8px;
  box-shadow: 0 24px 80px rgba(0, 0, 0, 0.5);
}

.ns-tour-zoom video {
  cursor: default;
}

@media (min-width: 960px) {
  .vp-doc .ns-tour {
    padding-top: 24px;
  }

  .vp-doc .ns-chapter {
    padding: 56px 0 64px;
  }

  .vp-doc .ns-tour h2.ns-chapter-title {
    font-size: 36px;
  }

  .vp-doc .ns-tour p.ns-chapter-lede {
    font-size: 18px;
  }

  .vp-doc .ns-tabs {
    top: var(--vp-nav-height);
    margin-top: 32px;
    gap: 8px;
  }

  .vp-doc .ns-tab {
    padding: 12px 16px 14px;
    font-size: 16px;
  }

  .vp-doc .ns-tour p.ns-step-body {
    margin: 18px 0 22px;
    font-size: 17px;
    min-height: 3.2em;
  }
}

@media (min-width: 1280px) {
  .vp-doc .ns-tour {
    --ns-tour-width: min(1840px, 100vw - 96px);
    width: var(--ns-tour-width);
    margin-left: calc((100% - var(--ns-tour-width)) / 2);
  }
}

.ns-pill {
  position: fixed;
  left: 50%;
  bottom: 24px;
  z-index: 25;
  display: flex;
  align-items: center;
  gap: 8px;
  max-width: calc(100vw - 32px);
  padding: 8px 8px 8px 16px;
  border: 1px solid var(--vp-c-divider);
  border-radius: 999px;
  background: color-mix(in srgb, var(--vp-c-bg) 78%, transparent);
  box-shadow: 0 16px 40px -16px rgba(15, 23, 42, 0.4);
  -webkit-backdrop-filter: blur(14px);
  backdrop-filter: blur(14px);
  opacity: 0;
  transform: translate(-50%, 24px);
  pointer-events: none;
  transition: opacity 0.3s ease, transform 0.3s ease;
}

.ns-pill.is-visible {
  opacity: 1;
  transform: translate(-50%, 0);
  pointer-events: auto;
}

.ns-pill-cmd {
  font-family: var(--vp-font-family-mono);
  font-size: 13px;
  color: var(--vp-c-text-1);
  white-space: nowrap;
}

.ns-pill-copy {
  padding: 6px 12px;
  border: 1px solid var(--vp-c-divider);
  border-radius: 999px;
  background: var(--vp-c-bg-soft);
  color: var(--vp-c-text-1);
  font: inherit;
  font-size: 13px;
  font-weight: 500;
  line-height: 1.4;
  cursor: pointer;
}

.ns-pill-copy:hover {
  border-color: var(--vp-c-brand-2);
}

.ns-pill-cta {
  padding: 7px 16px;
  border-radius: 999px;
  background: var(--vp-button-brand-bg);
  color: var(--vp-button-brand-text);
  font-size: 13px;
  font-weight: 600;
  line-height: 1.4;
  text-decoration: none;
  white-space: nowrap;
  transition: background-color 0.2s ease;
}

.ns-pill-cta:hover {
  background: var(--vp-button-brand-hover-bg);
  color: var(--vp-button-brand-hover-text);
}

.ns-pill-copy:focus-visible,
.ns-pill-cta:focus-visible {
  outline: 2px solid var(--vp-c-brand-1);
  outline-offset: 2px;
}

@media (max-width: 639px) {
  .ns-pill {
    padding-left: 8px;
  }

  .ns-pill-cmd {
    display: none;
  }
}

@media (prefers-reduced-motion: reduce) {
  .vp-doc .ns-tour.is-mounted .ns-chapter {
    opacity: 1;
    transform: none;
    transition: none;
  }

  .vp-doc .ns-layer,
  .vp-doc .ns-tab,
  .vp-doc .ns-zoom-hint,
  .ns-pill,
  .ns-pill-cta {
    transition: none;
  }

  .vp-doc .ns-tab-bar.is-auto {
    animation: none;
    transform: none;
  }
}
</style>
