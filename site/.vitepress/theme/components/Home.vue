<script setup>
import { onMounted, ref } from 'vue'
import { withBase } from 'vitepress'
import { VPButton } from 'vitepress/theme'

const INSTALL = 'brew install dmit-4884/tap/natscope'

const facts = [
  { title: 'Protobuf as JSON', text: 'Point it at your .proto files and binary payloads read as JSON in history, live tail and publish.' },
  { title: 'All of JetStream', text: 'Streams, consumers, relations, Key/Value and object stores: browse, edit, purge, publish.' },
  { title: 'One local binary', text: 'The UI is inside, state lives in one file, credentials go to the OS keychain.' },
  { title: 'MCP for agents', text: 'Claude Code or Cursor read your streams over MCP, with payloads decoded.' }
]

const video = ref(null)
const reducedMotion = ref(false)
const copied = ref(false)

onMounted(() => {
  reducedMotion.value = window.matchMedia('(prefers-reduced-motion: reduce)').matches
  const el = video.value
  if (!el) return
  el.muted = true
  if (reducedMotion.value) el.pause()
  else el.play().catch(() => {})
})

const copy = async () => {
  try {
    await navigator.clipboard.writeText(INSTALL)
    copied.value = true
    setTimeout(() => (copied.value = false), 1500)
  } catch {}
}

const fullscreen = () => {
  const el = video.value
  if (!el) return
  if (el.requestFullscreen) el.requestFullscreen()
  else if (el.webkitEnterFullscreen) el.webkitEnterFullscreen()
}
</script>

<template>
  <div class="ns-home">
    <section class="ns-hero">
      <h1>Natscope</h1>
      <p class="ns-lead">Web UI for NATS JetStream</p>
      <p class="ns-sub">Browse, tail, publish and manage streams. Protobuf payloads decoded.</p>
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

    <video
      ref="video"
      class="ns-demo"
      :src="withBase('/media/demo.mp4')"
      :poster="withBase('/media/demo.jpg')"
      :autoplay="!reducedMotion"
      :controls="reducedMotion"
      aria-label="Natscope walkthrough: live tail, stream relations, consumers, Protobuf decoding, publishing and Key/Value history."
      muted
      loop
      playsinline
      preload="auto"
      width="1920"
      height="1080"
      @click="fullscreen"
    />

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

.ns-demo {
  display: block;
  width: 100%;
  max-width: 1440px;
  height: auto;
  margin: 0 auto;
  border: 1px solid var(--vp-c-divider);
  border-radius: 12px;
  background: var(--vp-c-bg-alt);
  box-shadow: 0 24px 64px -24px rgba(17, 24, 39, 0.25);
  cursor: zoom-in;
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

  .ns-facts {
    grid-template-columns: minmax(0, 1fr);
    gap: 24px;
    margin-top: 40px;
  }
}
</style>
