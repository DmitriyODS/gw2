<template>
  <section class="pet-card" v-if="pet">
    <div class="pet-stage-bg" aria-hidden="true"></div>

    <div class="pet-top">
      <div class="pet-figure" :class="{ bounce: justFed }">
        <span class="pet-emoji">{{ petEmoji(pet) }}</span>
        <span v-if="hatEmoji" class="pet-hat">{{ hatEmoji }}</span>
      </div>

      <transition name="phrase">
        <div v-if="phrase" class="pet-phrase">{{ phrase }}</div>
      </transition>
    </div>

    <div class="pet-name-row">
      <template v-if="renaming">
        <input
          ref="nameInput"
          v-model.trim="newName"
          class="pet-name-input"
          maxlength="50"
          @keyup.enter="saveName"
          @keyup.esc="renaming = false"
        />
        <button class="pet-icon-btn" type="button" @click="saveName" aria-label="Сохранить имя">
          <span class="material-symbols-outlined">check</span>
        </button>
      </template>
      <template v-else>
        <h3 class="pet-name">{{ pet.name }}</h3>
        <button class="pet-icon-btn" type="button" @click="startRename" aria-label="Переименовать">
          <span class="material-symbols-outlined">edit</span>
        </button>
      </template>
    </div>
    <p class="pet-subtitle">{{ stageTitle }}<template v-if="speciesTitle"> · {{ speciesTitle }}</template></p>

    <div class="pet-xp">
      <div class="pet-xp-meta">
        <span>{{ pet.stage >= maxStage ? 'Максимальная форма' : 'До эволюции' }}</span>
        <span v-if="pet.next_stage_xp">{{ pet.xp }} / {{ pet.next_stage_xp }} XP</span>
      </div>
      <div class="pet-xp-bar">
        <div class="pet-xp-fill" :style="{ width: xpPercent + '%' }"></div>
      </div>
    </div>

    <div class="pet-stats">
      <span class="pet-chip beans" title="Грувы — внутренняя валюта за работу">
        <span class="pet-chip-emoji">🫘</span> {{ pet.beans }}
      </span>
      <span class="pet-chip streak" title="Стрик кормления">
        <span class="material-symbols-outlined">local_fire_department</span>
        {{ pet.feed_streak }} дн.
      </span>
      <span v-if="pet.feeds_left != null" class="pet-chip" title="Кормлений осталось сегодня">
        <span class="material-symbols-outlined">restaurant</span>
        {{ pet.feeds_left }}
      </span>
    </div>

    <div class="pet-actions">
      <button
        class="pet-feed-btn"
        type="button"
        :disabled="!canFeed || feeding"
        @click="feed"
      >
        <span class="pet-chip-emoji">🥕</span>
        Покормить · 3 грува
      </button>
      <button class="pet-icon-btn shop" type="button" @click="$emit('open-shop')" aria-label="Магазин аксессуаров" title="Магазин">
        <span class="material-symbols-outlined">storefront</span>
      </button>
    </div>

    <div v-if="ownedItems.length" class="pet-closet">
      <button
        v-for="item in ownedItems"
        :key="item"
        class="pet-closet-item"
        :class="{ active: pet.hat === item }"
        type="button"
        :title="SHOP_ITEMS[item]?.title || item"
        @click="toggleEquip(item)"
      >{{ SHOP_ITEMS[item]?.emoji || '🎁' }}</button>
    </div>
  </section>
</template>

<script setup>
import { computed, nextTick, ref } from 'vue'
import { useGrooveStore } from '@/stores/groove.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { petEmoji, PET_STAGES, PET_SPECIES, SHOP_ITEMS } from '@/utils/groove.js'

defineEmits(['open-shop'])

const groove = useGrooveStore()
const notify = useNotificationsStore()

const pet = computed(() => groove.pet)
const maxStage = PET_STAGES.length - 1

const phrase = ref('')
const justFed = ref(false)
const feeding = ref(false)
const renaming = ref(false)
const newName = ref('')
const nameInput = ref(null)
let phraseTimer = null

const stageTitle = computed(() => PET_STAGES[pet.value?.stage] || '')
const speciesTitle = computed(() =>
  pet.value?.stage >= 2 ? PET_SPECIES[pet.value.species]?.title : ''
)
const hatEmoji = computed(() =>
  pet.value?.hat ? SHOP_ITEMS[pet.value.hat]?.emoji : null
)
const ownedItems = computed(() => pet.value?.accessories || [])

const xpPercent = computed(() => {
  if (!pet.value) return 0
  if (!pet.value.next_stage_xp) return 100
  return Math.min(100, Math.round((pet.value.xp / pet.value.next_stage_xp) * 100))
})

const canFeed = computed(() =>
  pet.value && pet.value.beans >= 3 && (pet.value.feeds_left == null || pet.value.feeds_left > 0)
)

async function feed() {
  if (feeding.value) return
  feeding.value = true
  try {
    const res = await groove.feedPet()
    justFed.value = true
    setTimeout(() => { justFed.value = false }, 700)
    if (res.phrase) {
      phrase.value = res.phrase
      clearTimeout(phraseTimer)
      phraseTimer = setTimeout(() => { phrase.value = '' }, 6000)
    }
    if (res.evolved) {
      notify.success(`«${res.name}» эволюционировал! Теперь это ${PET_STAGES[res.stage]} 🎉`)
    }
  } catch (e) {
    notify.warn(e?.message || 'Покормить не получилось')
  } finally {
    feeding.value = false
  }
}

function startRename() {
  newName.value = pet.value?.name || ''
  renaming.value = true
  nextTick(() => nameInput.value?.focus())
}

async function saveName() {
  if (!newName.value) { renaming.value = false; return }
  try {
    await groove.renamePet(newName.value)
  } catch (e) {
    notify.error(e?.message || 'Не удалось переименовать')
  }
  renaming.value = false
}

async function toggleEquip(item) {
  try {
    await groove.equipItem(pet.value.hat === item ? null : item)
  } catch (e) {
    notify.error(e?.message || 'Не получилось')
  }
}
</script>

<style scoped>
.pet-card {
  position: relative;
  background: var(--color-surface);
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-lg, 16px);
  padding: 18px 16px 16px;
  display: flex;
  flex-direction: column;
  align-items: center;
  overflow: hidden;
}
.pet-stage-bg {
  position: absolute;
  inset: 0 0 auto 0;
  height: 96px;
  background: linear-gradient(180deg,
    color-mix(in oklch, var(--color-primary-container) 55%, transparent),
    transparent);
  pointer-events: none;
}
.pet-top { position: relative; display: flex; flex-direction: column; align-items: center; }
.pet-figure {
  position: relative;
  width: 86px;
  height: 86px;
  border-radius: 50%;
  background: var(--color-primary-container);
  display: grid;
  place-items: center;
  box-shadow: var(--shadow-sm, none);
}
.pet-figure.bounce { animation: pet-bounce 0.65s cubic-bezier(0.34, 1.56, 0.64, 1); }
@keyframes pet-bounce {
  0% { transform: scale(1); }
  35% { transform: scale(1.18) rotate(-4deg); }
  70% { transform: scale(0.96) rotate(2deg); }
  100% { transform: scale(1); }
}
.pet-emoji { font-size: 46px; line-height: 1; }
.pet-hat {
  position: absolute;
  top: -12px;
  right: 2px;
  font-size: 24px;
  transform: rotate(12deg);
}
.pet-phrase {
  position: relative;
  margin-top: 10px;
  max-width: 240px;
  background: var(--color-surface-high);
  border: 1px solid var(--color-outline-dim);
  border-radius: 14px;
  padding: 8px 12px;
  font-size: 13px;
  font-style: italic;
  text-align: center;
  line-height: 1.4;
}
.phrase-enter-active, .phrase-leave-active { transition: opacity 0.25s, transform 0.25s; }
.phrase-enter-from, .phrase-leave-to { opacity: 0; transform: translateY(4px); }

.pet-name-row { display: flex; align-items: center; gap: 4px; margin-top: 10px; }
.pet-name { margin: 0; font-size: 18px; font-weight: 700; }
.pet-name-input {
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-full);
  padding: 5px 12px;
  font-size: 15px;
  width: 160px;
  background: var(--color-surface);
  color: var(--color-text);
  outline: none;
}
.pet-icon-btn {
  width: 30px;
  height: 30px;
  border-radius: 50%;
  border: none;
  background: none;
  display: grid;
  place-items: center;
  cursor: pointer;
  color: var(--color-text-dim);
}
.pet-icon-btn:hover { background: var(--color-surface-high); }
.pet-icon-btn .material-symbols-outlined { font-size: 18px; }
.pet-icon-btn.shop {
  width: 44px;
  height: 44px;
  background: var(--color-secondary-container);
  color: var(--color-on-secondary-container);
  border-radius: 14px;
}
.pet-subtitle {
  margin: 2px 0 0;
  font-size: 12.5px;
  color: var(--color-text-dim);
}

.pet-xp { width: 100%; margin-top: 14px; }
.pet-xp-meta {
  display: flex;
  justify-content: space-between;
  font-size: 11.5px;
  color: var(--color-text-dim);
  margin-bottom: 4px;
}
.pet-xp-bar {
  height: 8px;
  border-radius: var(--radius-full);
  background: var(--color-surface-high);
  overflow: hidden;
}
.pet-xp-fill {
  height: 100%;
  border-radius: inherit;
  background: var(--color-primary);
  transition: width 0.6s cubic-bezier(0.34, 1.56, 0.64, 1);
}

.pet-stats { display: flex; gap: 8px; margin-top: 12px; flex-wrap: wrap; justify-content: center; }
.pet-chip {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 4px 10px;
  border-radius: var(--radius-full);
  background: var(--color-surface-high);
  font-size: 13px;
  font-weight: 600;
}
.pet-chip .material-symbols-outlined { font-size: 16px; }
.pet-chip.beans { background: color-mix(in oklch, var(--color-success) 18%, transparent); }
.pet-chip.streak { background: color-mix(in oklch, var(--color-warning) 22%, transparent); }
.pet-chip-emoji { font-size: 14px; }

.pet-actions { display: flex; gap: 8px; margin-top: 14px; width: 100%; }
.pet-feed-btn {
  flex: 1;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  border: none;
  border-radius: var(--radius-full);
  background: var(--color-primary);
  color: var(--color-on-primary);
  font-size: 14px;
  font-weight: 600;
  padding: 11px 16px;
  cursor: pointer;
  transition: transform 0.1s, opacity 0.15s;
}
.pet-feed-btn:active { transform: scale(0.97); }
.pet-feed-btn:disabled { opacity: 0.45; cursor: default; }

.pet-closet { display: flex; gap: 6px; margin-top: 12px; flex-wrap: wrap; justify-content: center; }
.pet-closet-item {
  width: 38px;
  height: 38px;
  border-radius: 12px;
  border: 1.5px solid var(--color-outline-dim);
  background: var(--color-surface);
  font-size: 19px;
  cursor: pointer;
  display: grid;
  place-items: center;
  transition: border-color 0.15s, background 0.15s;
}
.pet-closet-item:hover { background: var(--color-surface-high); }
.pet-closet-item.active {
  border-color: var(--color-primary);
  background: var(--color-primary-container);
}
</style>
