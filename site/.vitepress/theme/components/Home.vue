<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { withBase } from 'vitepress'
import { VPButton } from 'vitepress/theme'

const INSTALL = 'brew install dmit-4884/tap/natscope'

const facts = [
  { title: 'Protobuf as JSON', text: 'Point it at your .proto files and binary payloads read as JSON in history, live tail and publish.' },
  { title: 'All of JetStream', text: 'Streams, consumers, relations, Key/Value and object stores: browse, edit, purge, publish.' },
  { title: 'One local binary', text: 'The UI is inside, state lives in one file, credentials go to the OS keychain.' },
  { title: 'MCP for agents', text: 'Claude Code or Cursor read your streams over MCP, with payloads decoded.' }
]

const player = ref(null)
const video = ref(null)
const reducedMotion = ref(false)
const copied = ref(false)
const playing = ref(false)
const current = ref(0)
const duration = ref(0)
let frame = 0

onMounted(() => {
  reducedMotion.value = window.matchMedia('(prefers-reduced-motion: reduce)').matches
  const el = video.value
  if (!el) return
  el.muted = true
  if (el.readyState >= 1) duration.value = el.duration
  if (reducedMotion.value) el.pause()
  else el.play().catch(() => {})
})

onBeforeUnmount(() => cancelAnimationFrame(frame))

const copy = async () => {
  try {
    await navigator.clipboard.writeText(INSTALL)
    copied.value = true
    setTimeout(() => (copied.value = false), 1500)
  } catch {}
}

const tick = () => {
  current.value = video.value?.currentTime ?? 0
  frame = requestAnimationFrame(tick)
}

const onPlay = () => {
  playing.value = true
  cancelAnimationFrame(frame)
  frame = requestAnimationFrame(tick)
}

const onPause = () => {
  playing.value = false
  cancelAnimationFrame(frame)
  current.value = video.value?.currentTime ?? 0
}

const onMetadata = () => {
  duration.value = video.value?.duration ?? 0
}

const toggle = () => {
  const el = video.value
  if (!el) return
  if (el.paused) el.play().catch(() => {})
  else el.pause()
}

const seek = (event) => {
  const el = video.value
  if (!el) return
  el.currentTime = Number(event.target.value)
  current.value = el.currentTime
}

const clock = (seconds) => {
  const s = Math.max(0, Math.floor(seconds || 0))
  return `${Math.floor(s / 60)}:${String(s % 60).padStart(2, '0')}`
}

const progress = computed(() => (duration.value ? (current.value / duration.value) * 100 : 0))

const fullscreen = () => {
  const box = player.value
  const el = video.value
  if (document.fullscreenElement) document.exitFullscreen()
  else if (box?.requestFullscreen) box.requestFullscreen()
  else if (el?.webkitEnterFullscreen) el.webkitEnterFullscreen()
}
</script>

<template>
  <div class="ns-home">
    <section class="ns-hero">
      <h1>Natscope</h1>
      <p class="ns-lead">Web UI for NATS JetStream</p>
      <p class="ns-sub">Browse, tail, publish and manage streams, with Protobuf payloads decoded.</p>
      <div class="ns-actions">
        <VPButton tag="a" size="big" theme="brand" text="Get started" :href="withBase('/guide/what-is-natscope')" />
        <VPButton tag="a" size="big" theme="alt" text="GitHub" href="https://github.com/dmit-4884/natscope" />
      </div>
      <div class="ns-install">
        <code><span aria-hidden="true">$ </span>{{ INSTALL }}</code>
        <button type="button" :aria-label="copied ? 'Copied' : 'Copy install command'" @click="copy">
          {{ copied ? 'Copied' : 'Copy' }}
        </button>
      </div>
    </section>

    <div ref="player" class="ns-player" :class="{ 'is-paused': !playing }">
      <video
        ref="video"
        class="ns-demo"
        :src="withBase('/media/demo.mp4')"
        :poster="withBase('/media/demo.jpg')"
        :autoplay="!reducedMotion"
        aria-label="Natscope walkthrough: live tail, stream relations, a Protobuf message in wire and decoded views, and publishing with schema completion."
        muted
        loop
        playsinline
        preload="auto"
        width="1920"
        height="1080"
        @click="toggle"
        @play="onPlay"
        @pause="onPause"
        @loadedmetadata="onMetadata"
      />
      <div class="ns-controls">
        <button type="button" class="ns-control" :aria-label="playing ? 'Pause' : 'Play'" @click="toggle">
          <svg v-if="playing" viewBox="0 0 24 24" aria-hidden="true"><path d="M7 5h3.5v14H7zM13.5 5H17v14h-3.5z" /></svg>
          <svg v-else viewBox="0 0 24 24" aria-hidden="true"><path d="M8 5.2v13.6a.8.8 0 0 0 1.2.7l10.6-6.8a.8.8 0 0 0 0-1.4L9.2 4.5a.8.8 0 0 0-1.2.7z" /></svg>
        </button>
        <input
          class="ns-seek"
          type="range"
          min="0"
          :max="duration || 0"
          step="0.01"
          :value="current"
          :style="{ '--progress': `${progress}%` }"
          aria-label="Seek"
          :aria-valuetext="`${clock(current)} of ${clock(duration)}`"
          @input="seek"
        />
        <span class="ns-time">{{ clock(current) }} / {{ clock(duration) }}</span>
        <button type="button" class="ns-control" aria-label="Full screen" @click="fullscreen">
          <svg viewBox="0 0 24 24" aria-hidden="true">
            <path d="M4 9V4h5M15 4h5v5M20 15v5h-5M9 20H4v-5" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" />
          </svg>
        </button>
      </div>
    </div>

    <ul class="ns-facts">
      <li v-for="f in facts" :key="f.title">
        <h2>{{ f.title }}</h2>
        <p>{{ f.text }}</p>
      </li>
    </ul>
  </div>
</template>

<style scoped>
.ns-home {
  padding: 0 24px;
}

.ns-hero {
  max-width: 760px;
  margin: 0 auto;
  padding: 64px 0 40px;
  text-align: center;
}

.ns-hero h1 {
  font-size: 56px;
  line-height: 1.1;
  font-weight: 700;
  letter-spacing: -0.02em;
  color: var(--vp-c-text-1);
}

.ns-lead {
  margin-top: 12px;
  font-size: 26px;
  line-height: 1.3;
  font-weight: 600;
  color: var(--vp-c-text-1);
}

.ns-sub {
  margin-top: 12px;
  font-size: 17px;
  line-height: 1.5;
  color: var(--vp-c-text-2);
}

.ns-actions {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 12px;
  margin-top: 28px;
}

.ns-install {
  display: inline-flex;
  align-items: center;
  gap: 12px;
  max-width: 100%;
  margin-top: 20px;
  padding: 8px 8px 8px 16px;
  border: 1px solid var(--vp-c-divider);
  border-radius: 10px;
  background: var(--vp-c-bg-soft);
}

.ns-install code {
  overflow-x: auto;
  white-space: nowrap;
  font-family: var(--vp-font-family-mono);
  font-size: 14px;
  color: var(--vp-c-text-1);
}

.ns-install code span {
  color: var(--vp-c-text-3);
}

.ns-install button {
  flex-shrink: 0;
  padding: 4px 10px;
  border: 1px solid var(--vp-c-divider);
  border-radius: 6px;
  background: var(--vp-c-bg);
  font-size: 13px;
  font-weight: 500;
  color: var(--vp-c-text-2);
  transition: color 0.2s, border-color 0.2s;
}

.ns-install button:hover {
  border-color: var(--vp-c-brand-2);
  color: var(--vp-c-brand-1);
}

.ns-player {
  position: relative;
  max-width: 1440px;
  margin: 0 auto;
  overflow: hidden;
  border: 1px solid var(--vp-c-divider);
  border-radius: 12px;
  background: var(--vp-c-bg-alt);
  box-shadow: 0 24px 64px -24px rgba(17, 24, 39, 0.25);
}

.ns-demo {
  display: block;
  width: 100%;
  height: auto;
  cursor: pointer;
}

.ns-player:fullscreen {
  display: flex;
  align-items: center;
  max-width: none;
  border: 0;
  border-radius: 0;
  background: #000;
}

.ns-player:fullscreen .ns-demo {
  max-height: 100%;
}

.ns-controls {
  position: absolute;
  right: 0;
  bottom: 0;
  left: 0;
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 28px 16px 12px;
  background: linear-gradient(to top, rgba(17, 24, 39, 0.72), rgba(17, 24, 39, 0));
  color: #fff;
  opacity: 0;
  transition: opacity 0.2s;
}

.ns-player:hover .ns-controls,
.ns-player:focus-within .ns-controls,
.ns-player.is-paused .ns-controls {
  opacity: 1;
}

@media (hover: none) {
  .ns-controls {
    opacity: 1;
  }
}

.ns-control {
  display: flex;
  flex-shrink: 0;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  border-radius: 8px;
  color: #fff;
  transition: background-color 0.2s;
}

.ns-control:hover {
  background: rgba(255, 255, 255, 0.16);
}

.ns-control:focus-visible,
.ns-seek:focus-visible {
  outline: 2px solid var(--vp-c-brand-1);
  outline-offset: 2px;
}

.ns-control svg {
  width: 20px;
  height: 20px;
  fill: currentColor;
}

.ns-seek {
  flex: 1;
  min-width: 0;
  height: 4px;
  margin: 0;
  border-radius: 9999px;
  background: linear-gradient(to right, var(--vp-c-brand-1) var(--progress), rgba(255, 255, 255, 0.35) var(--progress));
  cursor: pointer;
  appearance: none;
  -webkit-appearance: none;
}

.ns-seek::-webkit-slider-thumb {
  width: 14px;
  height: 14px;
  border: 0;
  border-radius: 50%;
  background: #fff;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.4);
  -webkit-appearance: none;
}

.ns-seek::-moz-range-thumb {
  width: 14px;
  height: 14px;
  border: 0;
  border-radius: 50%;
  background: #fff;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.4);
}

.ns-seek::-moz-range-track {
  background: transparent;
}

.ns-time {
  flex-shrink: 0;
  font-family: var(--vp-font-family-mono);
  font-size: 13px;
  font-variant-numeric: tabular-nums;
  color: rgba(255, 255, 255, 0.9);
}

.ns-facts {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 32px;
  max-width: 1440px;
  margin: 56px auto 0;
  padding: 0;
  list-style: none;
}

.ns-facts h2 {
  font-size: 16px;
  font-weight: 600;
  color: var(--vp-c-text-1);
}

.ns-facts p {
  margin-top: 6px;
  font-size: 14px;
  line-height: 1.6;
  color: var(--vp-c-text-2);
}

@media (max-width: 960px) {
  .ns-facts {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 640px) {
  .ns-hero {
    padding: 40px 0 32px;
  }

  .ns-hero h1 {
    font-size: 40px;
  }

  .ns-lead {
    font-size: 20px;
  }

  .ns-sub {
    font-size: 15px;
  }

  .ns-controls {
    gap: 8px;
    padding: 20px 8px 6px;
  }

  .ns-time {
    display: none;
  }

  .ns-facts {
    grid-template-columns: minmax(0, 1fr);
    gap: 24px;
    margin-top: 40px;
  }
}
</style>
