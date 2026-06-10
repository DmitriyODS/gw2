<template>
  <section class="live-bar">
    <span class="live-label">
      <span class="presence-pulse" />
      Сейчас в эфире
    </span>

    <span
      v-if="groove.zapsLeft != null"
      class="live-zap-budget"
      :class="{ empty: groove.zapsLeft === 0 }"
      :title="`Ваши заряды на сегодня: ${groove.zapsLeft} из ${groove.zapsMax}. Каждый день выдаётся по ${groove.zapsMax} — кидайте их коллегам в эфире (+1 грув получателю)`"
    >
      ⚡ {{ groove.zapsLeft }}/{{ groove.zapsMax }}
    </span>

    <div v-if="entries.length" class="live-list">
      <div v-for="entry in entries" :key="entry.unit_id" class="live-item">
        <img class="live-avatar" :src="avatarUrl(entry.user)" :alt="entry.user?.fio || ''" />
        <div class="live-info">
          <span class="live-name">{{ firstName(entry.user?.fio) }}</span>
          <span class="live-unit" :title="entry.unit_name">{{ entry.unit_name }} · {{ elapsed(entry) }}</span>
        </div>
        <button
          v-if="entry.user?.id !== groove.myId"
          class="live-zap"
          type="button"
          :disabled="groove.zapsLeft === 0"
          :title="groove.zapsLeft === 0
            ? 'Заряды на сегодня закончились — новые появятся завтра'
            : 'Кинуть заряд энергии (+1 грув коллеге)'"
          @click="zap(entry)"
        >
          ⚡<span v-if="entry.zaps" class="live-zap-count">{{ entry.zaps }}</span>
        </button>
        <span v-else-if="entry.zaps" class="live-zap mine" title="Ваши заряды">
          ⚡<span class="live-zap-count">{{ entry.zaps }}</span>
        </span>
      </div>
    </div>

    <span v-else class="live-empty">Тишина в эфире — самое время запустить юнит 😉</span>
  </section>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useGrooveStore } from '@/stores/groove.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { avatarUrl } from '@/utils/groove.js'

const groove = useGrooveStore()
const notify = useNotificationsStore()

const entries = computed(() => groove.live)

// Тикер раз в 30с, чтобы «N мин в эфире» не застывало.
const now = ref(Date.now())
let timer = null
onMounted(() => { timer = setInterval(() => { now.value = Date.now() }, 30000) })
onBeforeUnmount(() => clearInterval(timer))

function firstName(fio) {
  if (!fio) return ''
  const parts = fio.split(' ')
  return parts.length > 1 ? `${parts[0]} ${parts[1]}` : fio
}

function elapsed(entry) {
  if (!entry.started_at) return ''
  const min = Math.max(0, Math.floor((now.value - new Date(entry.started_at).getTime()) / 60000))
  if (min < 60) return `${min} мин`
  return `${Math.floor(min / 60)} ч ${min % 60} мин`
}

async function zap(entry) {
  try {
    await groove.zap(entry.user.id)
  } catch (e) {
    notify.warn(e?.message || 'Заряд не долетел')
  }
}
</script>

<style scoped>
.live-bar {
  display: flex;
  align-items: center;
  gap: 14px;
  background: var(--color-surface);
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-lg, 16px);
  padding: 10px 14px;
  min-width: 0;
}
.live-label {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  font-weight: 600;
  color: var(--color-text-dim);
  flex-shrink: 0;
}
.live-list {
  display: flex;
  gap: 10px;
  overflow-x: auto;
  min-width: 0;
  scrollbar-width: thin;
}
.live-item {
  display: flex;
  align-items: center;
  gap: 8px;
  background: var(--color-surface-high);
  border-radius: var(--radius-full);
  padding: 5px 8px 5px 5px;
  flex-shrink: 0;
}
.live-avatar {
  width: 30px;
  height: 30px;
  border-radius: 50%;
  object-fit: cover;
}
.live-info { display: flex; flex-direction: column; min-width: 0; }
.live-name { font-size: 12.5px; font-weight: 600; white-space: nowrap; }
.live-unit {
  font-size: 11px;
  color: var(--color-text-dim);
  max-width: 180px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.live-zap {
  border: none;
  background: color-mix(in oklch, var(--color-warning) 22%, transparent);
  border-radius: var(--radius-full);
  padding: 4px 9px;
  font-size: 13px;
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  gap: 3px;
  transition: transform 0.12s;
  flex-shrink: 0;
}
.live-zap:not(.mine):not(:disabled):hover { transform: scale(1.1); }
.live-zap:not(.mine):not(:disabled):active { transform: scale(0.9); }
.live-zap:disabled { opacity: 0.4; cursor: default; }
.live-zap.mine { cursor: default; }
.live-zap-budget {
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
  gap: 3px;
  padding: 3px 10px;
  border-radius: var(--radius-full);
  background: color-mix(in oklch, var(--color-warning) 22%, transparent);
  font-size: 12px;
  font-weight: 700;
  cursor: help;
}
.live-zap-budget.empty {
  background: var(--color-surface-high);
  color: var(--color-text-dim);
}
.live-zap-count { font-size: 11.5px; font-weight: 700; }
.live-empty { font-size: 13px; color: var(--color-text-dim); }

@media (max-width: 768px) {
  .live-bar { flex-direction: column; align-items: stretch; gap: 8px; }
}
</style>
