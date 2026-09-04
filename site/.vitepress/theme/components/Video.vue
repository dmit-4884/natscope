<script setup>
import { computed, ref } from 'vue'
import { withBase } from 'vitepress'

const props = defineProps({
  src: { type: String, required: true },
  poster: { type: String, default: '' },
  caption: { type: String, default: '' }
})

const videoSrc = computed(() => withBase(props.src))
const posterSrc = computed(() => (props.poster ? withBase(props.poster) : undefined))
const videoRef = ref(null)

const goFullscreen = () => {
  const el = videoRef.value
  if (!el) return
  if (el.requestFullscreen) el.requestFullscreen()
  else if (el.webkitEnterFullscreen) el.webkitEnterFullscreen()
  if (el.paused) el.play()
}
</script>

<template>
  <figure class="ns-video">
    <div class="ns-video-frame">
      <video
        ref="videoRef"
        :src="videoSrc"
        :poster="posterSrc"
        :aria-label="caption || undefined"
        controls
        muted
        playsinline
        preload="none"
      />
      <button type="button" class="ns-video-expand" aria-label="Play fullscreen" @click="goFullscreen">
        ⤢ Fullscreen
      </button>
    </div>
    <figcaption v-if="caption">{{ caption }}</figcaption>
  </figure>
</template>

<style scoped>
.ns-video {
  margin: 24px 0;
}

.ns-video-frame {
  position: relative;
}

.ns-video video {
  display: block;
  width: 100%;
  border: 1px solid var(--vp-c-divider);
  border-radius: 12px;
  background: var(--vp-c-bg-alt);
}

.ns-video-expand {
  position: absolute;
  top: 10px;
  right: 10px;
  padding: 4px 10px;
  border-radius: 8px;
  border: 1px solid rgba(255, 255, 255, 0.25);
  background: rgba(17, 24, 39, 0.62);
  color: #fff;
  font-size: 12px;
  line-height: 1.4;
  cursor: pointer;
  opacity: 0;
  transition: opacity 0.15s ease;
}

.ns-video-frame:hover .ns-video-expand,
.ns-video-expand:focus-visible {
  opacity: 1;
}

.ns-video figcaption {
  margin-top: 8px;
  font-size: 13px;
  line-height: 1.5;
  color: var(--vp-c-text-2);
}
</style>
