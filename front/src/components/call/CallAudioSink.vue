<template>
  <!-- Невидимые <audio> для ВСЕХ удалённых участников: звук не должен зависеть
       от того, какие плитки сейчас отрисованы (мини-режим, фокус демонстрации,
       боковые панели). У каждого — голос и звук его демонстрации экрана. -->
  <div class="call-audio-sink" aria-hidden="true">
    <audio
      v-for="s in sources"
      :key="s.key"
      :ref="(el) => setRef(s.key, el)"
      autoplay
    />
  </div>
</template>

<script setup>
import { computed, watch, onBeforeUnmount } from 'vue'
import { useCallStore } from '@/stores/call.js'
import { callRoom } from '@/services/livekit.js'

const callStore = useCallStore()

const sources = computed(() => callStore.participantList
  .filter(p => !p.pending)
  .flatMap(p => [
    { key: `${p.identity}:audio`, identity: p.identity, source: 'audio' },
    { key: `${p.identity}:screen-audio`, identity: p.identity, source: 'screen-audio' },
  ]))

const els = new Map()      // key → <audio>
const attached = new Map() // key → livekit Track

function setRef(key, el) {
  if (el) {
    els.set(key, el)
  } else {
    els.delete(key)
    attached.delete(key)
  }
  sync()
}

function sync() {
  for (const s of sources.value) {
    const el = els.get(s.key)
    if (!el) continue
    const track = callRoom.getTrack(s.identity, s.source)
    const prev = attached.get(s.key)
    if (prev && prev !== track) {
      try { prev.detach(el) } catch {}
      attached.delete(s.key)
    }
    if (track && prev !== track) {
      track.attach(el)
      attached.set(s.key, track)
    }
  }
}

// participants заменяется целиком при каждом resync — shallow watch достаточно.
watch(() => callStore.participants, sync, { flush: 'post' })

onBeforeUnmount(() => {
  for (const [key, track] of attached) {
    const el = els.get(key)
    if (el) { try { track.detach(el) } catch {} }
  }
  attached.clear()
  els.clear()
})
</script>

<style scoped>
.call-audio-sink { display: none; }
</style>
