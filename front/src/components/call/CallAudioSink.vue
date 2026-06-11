<template>
  <!-- Невидимые <audio> для ВСЕХ удалённых участников: звук не должен зависеть
       от того, какие плитки сейчас отрисованы (мини-режим, фокус демонстрации,
       боковые панели). -->
  <div class="call-audio-sink" aria-hidden="true">
    <audio
      v-for="p in remotes"
      :key="p.identity"
      :ref="(el) => setRef(p.identity, el)"
      autoplay
    />
  </div>
</template>

<script setup>
import { computed, watch, onBeforeUnmount } from 'vue'
import { useCallStore } from '@/stores/call.js'
import { callRoom } from '@/services/livekit.js'

const callStore = useCallStore()
const remotes = computed(() => callStore.participantList.filter(p => !p.pending))

const els = new Map()      // identity → <audio>
const attached = new Map() // identity → livekit Track

function setRef(identity, el) {
  if (el) {
    els.set(identity, el)
  } else {
    els.delete(identity)
    attached.delete(identity)
  }
  sync()
}

function sync() {
  for (const p of remotes.value) {
    const el = els.get(p.identity)
    if (!el) continue
    const track = callRoom.getTrack(p.identity, 'audio')
    const prev = attached.get(p.identity)
    if (prev && prev !== track) {
      try { prev.detach(el) } catch {}
      attached.delete(p.identity)
    }
    if (track && prev !== track) {
      track.attach(el)
      attached.set(p.identity, track)
    }
  }
}

// participants заменяется целиком при каждом resync — shallow watch достаточно.
watch(() => callStore.participants, sync, { flush: 'post' })

onBeforeUnmount(() => {
  for (const [identity, track] of attached) {
    const el = els.get(identity)
    if (el) { try { track.detach(el) } catch {} }
  }
  attached.clear()
  els.clear()
})
</script>

<style scoped>
.call-audio-sink { display: none; }
</style>
