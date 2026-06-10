<template>
  <section class="zoo">
    <header class="zoo-head">
      <h3 class="zoo-title">
        <span class="material-symbols-outlined">pets</span>
        Зоопарк команды
      </h3>
      <span class="zoo-hint">Погладьте чужого Грувика — по груву обоим. Больных 🤒 поглаживание лечит</span>
    </header>

    <div class="zoo-strip">
      <div v-for="p in pets" :key="p.user_id" class="zoo-pet">
        <button
          class="zoo-figure"
          type="button"
          :disabled="p.user_id === groove.myId || p.stroked_by_me"
          :title="strokeTitle(p)"
          @click="stroke(p)"
        >
          <span class="zoo-emoji" :class="{ sick: p.sick }">{{ petEmoji(p) }}</span>
          <span v-if="hatOf(p)" class="zoo-hat">{{ hatOf(p) }}</span>
          <span v-if="p.sick" class="zoo-sick" title="Болеет — погладьте, это лечит">🤒</span>
          <span
            v-if="p.user_id !== groove.myId"
            class="zoo-stroke-badge"
            :class="{ done: p.stroked_by_me }"
          >
            <span class="material-symbols-outlined">{{ p.stroked_by_me ? 'favorite' : 'waving_hand' }}</span>
          </span>
        </button>
        <span class="zoo-pet-name">{{ p.name }}</span>
        <span class="zoo-owner">{{ firstName(p.user?.fio) }}</span>
        <span v-if="p.strokes_today" class="zoo-strokes">❤ {{ p.strokes_today }}</span>
      </div>

      <p v-if="!pets.length" class="zoo-empty">
        Пока пусто — Грувики вылупляются при первом заходе хозяев в «Мой Groove»
      </p>
    </div>
  </section>
</template>

<script setup>
import { computed } from 'vue'
import { useGrooveStore } from '@/stores/groove.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { petEmoji, SHOP_ITEMS } from '@/utils/groove.js'

const groove = useGrooveStore()
const notify = useNotificationsStore()

const pets = computed(() => groove.zoo)

const hatOf = (p) => (p.hat ? SHOP_ITEMS[p.hat]?.emoji : null)

function firstName(fio) {
  if (!fio) return ''
  const parts = fio.split(' ')
  return parts.length > 1 ? `${parts[0]} ${parts[1]}` : fio
}

function strokeTitle(p) {
  if (p.user_id === groove.myId) return 'Это ваш Грувик'
  return p.stroked_by_me ? 'Сегодня уже погладили' : 'Погладить'
}

async function stroke(p) {
  try {
    await groove.strokePet(p.user_id)
    notify.success(`Вы погладили «${p.name}» — вам обоим по груву ❤`)
  } catch (e) {
    notify.warn(e?.message || 'Не получилось погладить')
  }
}
</script>

<style scoped>
.zoo {
  background: var(--color-surface);
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-lg, 16px);
  padding: 14px 16px;
  display: flex;
  flex-direction: column;
  gap: 10px;
  min-width: 0;
}
.zoo-head { display: flex; flex-direction: column; gap: 2px; }
.zoo-title {
  margin: 0;
  font-size: 14.5px;
  font-weight: 700;
  display: flex;
  align-items: center;
  gap: 6px;
}
.zoo-title .material-symbols-outlined { font-size: 18px; color: var(--color-primary); }
.zoo-hint { font-size: 11.5px; color: var(--color-text-dim); }

.zoo-strip {
  display: flex;
  gap: 14px;
  overflow-x: auto;
  padding: 6px 2px 4px;
  scrollbar-width: thin;
}
.zoo-pet {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 2px;
  flex-shrink: 0;
  width: 78px;
}
.zoo-figure {
  position: relative;
  width: 56px;
  height: 56px;
  border-radius: 50%;
  border: none;
  background: var(--color-primary-container);
  display: grid;
  place-items: center;
  cursor: pointer;
  transition: transform 0.15s;
}
.zoo-figure:not(:disabled):hover { transform: scale(1.08); }
.zoo-figure:disabled { cursor: default; }
.zoo-emoji { font-size: 28px; line-height: 1; }
.zoo-emoji.sick { filter: grayscale(0.55) brightness(0.92); }
.zoo-sick {
  position: absolute;
  bottom: -5px;
  left: -5px;
  font-size: 15px;
}
.zoo-hat {
  position: absolute;
  top: -9px;
  right: -2px;
  font-size: 17px;
  transform: rotate(12deg);
}
.zoo-stroke-badge {
  position: absolute;
  bottom: -4px;
  right: -4px;
  width: 22px;
  height: 22px;
  border-radius: 50%;
  background: var(--color-surface);
  border: 1px solid var(--color-outline-dim);
  display: grid;
  place-items: center;
}
.zoo-stroke-badge .material-symbols-outlined {
  font-size: 13px;
  color: var(--color-text-dim);
}
.zoo-stroke-badge.done .material-symbols-outlined { color: var(--color-error); }
.zoo-pet-name {
  font-size: 12px;
  font-weight: 600;
  max-width: 100%;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.zoo-owner {
  font-size: 10.5px;
  color: var(--color-text-dim);
  max-width: 100%;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.zoo-strokes { font-size: 10.5px; color: var(--color-error); }
.zoo-empty { margin: 0; font-size: 12.5px; color: var(--color-text-dim); }
</style>
