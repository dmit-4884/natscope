<script setup>
import { computed, ref, watch, onUnmounted } from 'vue'
import { withBase } from 'vitepress'

const props = defineProps({
  src: { type: String, required: true },
  alt: { type: String, default: '' }
})

const imgSrc = computed(() => withBase(props.src))
const zoomed = ref(false)

const onKey = (e) => {
  if (e.key === 'Escape') zoomed.value = false
}

watch(zoomed, (open) => {
  if (typeof document === 'undefined') return
  if (open) {
    document.addEventListener('keydown', onKey)
    document.body.style.overflow = 'hidden'
  } else {
    document.removeEventListener('keydown', onKey)
    document.body.style.overflow = ''
  }
})

onUnmounted(() => {
  if (typeof document === 'undefined') return
  document.removeEventListener('keydown', onKey)
  document.body.style.overflow = ''
})
</script>

<template>
  <figure class="ns-shot">
    <img
      :src="imgSrc"
      :alt="alt"
      loading="lazy"
      decoding="async"
      role="button"
      tabindex="0"
      @click="zoomed = true"
      @keydown.enter="zoomed = true"
    />
    <figcaption v-if="alt">{{ alt }} <span class="ns-shot-hint">— click to enlarge</span></figcaption>
  </figure>
  <Teleport to="body">
    <div v-if="zoomed" class="ns-zoom" @click="zoomed = false">
      <img :src="imgSrc" :alt="alt" />
    </div>
  </Teleport>
</template>

<style scoped>
.ns-shot {
  margin: 24px 0;
}

.ns-shot img {
  display: block;
  width: 100%;
  border: 1px solid var(--vp-c-divider);
  border-radius: 12px;
  background: var(--vp-c-bg-alt);
  cursor: zoom-in;
}

.ns-shot figcaption {
  margin-top: 8px;
  font-size: 13px;
  line-height: 1.5;
  color: var(--vp-c-text-2);
}

.ns-shot-hint {
  color: var(--vp-c-text-3);
}

.ns-zoom {
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

.ns-zoom img {
  max-width: min(1920px, 96vw);
  max-height: 94vh;
  border-radius: 8px;
  box-shadow: 0 24px 80px rgba(0, 0, 0, 0.5);
}
</style>
