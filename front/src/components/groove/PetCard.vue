<template>
  <section class="pet-card" v-if="pet">
    <div class="pet-stage-bg" aria-hidden="true"></div>

    <div class="pet-top">
      <div class="pet-figure" :class="{ bounce: justFed, sick: pet.sick }">
        <span class="pet-emoji">{{ petEmoji(pet) }}</span>
        <span v-if="hatEmoji" class="pet-hat">{{ hatEmoji }}</span>
        <span v-if="pet.sick" class="pet-sick-badge" title="Грувик болеет">🤒</span>
      </div>
      <span v-if="xpGain" :key="xpGainTick" class="pet-xp-float">+{{ xpGain }} XP</span>
      <span v-if="beansGain" :key="'b' + beansGainTick" class="pet-beans-float">+{{ beansGain }} 🫘</span>

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
    <p v-if="personality" class="pet-personality">{{ personality.emoji }} {{ personality.title }}</p>

    <div v-if="pet.sick" class="pet-sick-bar">
      <div class="pet-sick-head">
        <span class="material-symbols-outlined">healing</span>
        Грувик приболел — хозяин давно не работал
      </div>
      <div class="pet-sick-progress">
        <span
          v-for="i in pet.recovery_target"
          :key="i"
          class="pet-sick-dot"
          :class="{ filled: i <= pet.recovery }"
        ></span>
        <span class="pet-sick-count">{{ pet.recovery }}/{{ pet.recovery_target }}</span>
      </div>
      <p class="pet-sick-hint">Лечат: юнит от 15 минут, закрытая задача, бульон и поглаживания коллег</p>
    </div>

    <!-- Дневной квест от Грувика: главный мотиватор сделать что-то
         конкретное сегодня. Награда — бонус-грувы поверх обычных капов. -->
    <div v-if="quest" class="pet-quest" :class="{ done: quest.done, claimed: quest.claimed }">
      <div class="pet-quest-head">
        <span class="material-symbols-outlined pet-quest-ico">
          {{ quest.claimed ? 'check_circle' : (quest.done ? 'rocket_launch' : 'flag') }}
        </span>
        <span class="pet-quest-title">{{ quest.title }}</span>
        <span class="pet-quest-reward">+{{ quest.reward }} 🫘</span>
      </div>
      <div class="pet-quest-bar">
        <div class="pet-quest-fill" :style="{ width: questPercent + '%' }"></div>
      </div>
      <div class="pet-quest-meta">
        <span>{{ quest.progress }} / {{ quest.target }} {{ quest.unit }}</span>
        <button
          v-if="quest.done && !quest.claimed"
          class="pet-quest-claim"
          type="button"
          :disabled="claiming"
          @click="claim"
        >Забрать награду</button>
        <span v-else-if="quest.claimed" class="pet-quest-claimed">Награда получена 🎉</span>
        <span v-else class="pet-quest-hint">{{ quest.hint }}</span>
      </div>
    </div>

    <div v-if="!pet.sick" class="pet-xp">
      <div class="pet-xp-meta">
        <span>{{ pet.stage >= maxStage ? 'Максимальная форма' : 'До эволюции' }}</span>
        <span v-if="pet.next_stage_xp">{{ pet.xp }} / {{ pet.next_stage_xp }} XP</span>
      </div>
      <div class="pet-xp-row">
        <div class="pet-xp-bar">
          <div class="pet-xp-fill" :style="{ width: xpPercent + '%' }"></div>
        </div>
        <span
          v-if="pet.next_stage_xp"
          class="pet-next"
          :class="{ egg: pet.stage === 0 }"
          :title="nextHint"
        >{{ pet.stage === 0 ? '🐣' : '?' }}</span>
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
      <span
        v-if="pet.feeds_left != null"
        class="pet-chip"
        :title="`Кормлений осталось сегодня: ${pet.feeds_left} из ${feedsMax}. Счётчик обновляется каждый день`"
      >
        <span class="material-symbols-outlined">restaurant</span>
        {{ pet.feeds_left }}/{{ feedsMax }}
      </span>
    </div>

    <div v-if="weekDays.length" class="pet-week" title="Кормления за последние 7 дней">
      <div
        v-for="d in weekDays"
        :key="d.key"
        class="pet-week-day"
        :class="{ fed: d.fed, today: d.today }"
      >
        <span class="pet-week-dot"></span>
        <span class="pet-week-label">{{ d.label }}</span>
      </div>
    </div>

    <div class="pet-actions">
      <button
        class="pet-feed-btn"
        type="button"
        :disabled="!canFeed || feeding"
        @click="feed"
      >
        <span class="pet-chip-emoji">{{ pet.sick ? '🍲' : '🥕' }}</span>
        {{ pet.sick ? 'Дать бульон · 1 грув' : 'Покормить · 3 грува' }}
      </button>
      <button class="pet-icon-btn shop" type="button" @click="openChat" aria-label="Поговорить с Грувиком" title="Поговорить">
        <span class="material-symbols-outlined">forum</span>
      </button>
      <button class="pet-icon-btn shop" type="button" @click="$emit('open-shop')" aria-label="Магазин аксессуаров" title="Магазин">
        <span class="material-symbols-outlined">storefront</span>
      </button>
    </div>

    <p v-if="feedHint" class="pet-feed-hint">{{ feedHint }}</p>

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
import { computed, nextTick, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useGrooveStore } from '@/stores/groove.js'
import { useMessengerStore } from '@/stores/messenger.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { petEmoji, PET_STAGES, PET_SPECIES, PERSONALITIES, SHOP_ITEMS } from '@/utils/groove.js'

defineEmits(['open-shop'])

const groove = useGrooveStore()
const notify = useNotificationsStore()
const router = useRouter()

const pet = computed(() => groove.pet)
const maxStage = PET_STAGES.length - 1

const phrase = ref('')
const justFed = ref(false)
const feeding = ref(false)
const claiming = ref(false)
const renaming = ref(false)
const newName = ref('')
const nameInput = ref(null)
const xpGain = ref(0)
const xpGainTick = ref(0)
const beansGain = ref(0)
const beansGainTick = ref(0)
let phraseTimer = null
let xpGainTimer = null
let beansGainTimer = null

watch(() => pet.value?.beans, (val, old) => {
  if (old != null && val != null && val > old) {
    beansGain.value = val - old
    beansGainTick.value++
    clearTimeout(beansGainTimer)
    beansGainTimer = setTimeout(() => { beansGain.value = 0 }, 1600)
  }
})

const quest = computed(() => pet.value?.quest || null)
const questPercent = computed(() => {
  const q = quest.value
  if (!q || !q.target) return 0
  return Math.min(100, Math.round((q.progress / q.target) * 100))
})

async function claim() {
  if (claiming.value) return
  claiming.value = true
  try {
    await groove.claimQuest()
    notify.success(`+${quest.value?.reward || 20} 🫘 за квест от Грувика`)
  } catch (e) {
    notify.warn(e?.message || 'Не удалось забрать награду')
  } finally {
    claiming.value = false
  }
}

const stageTitle = computed(() => PET_STAGES[pet.value?.stage] || '')
const speciesTitle = computed(() =>
  pet.value?.stage >= 2 ? PET_SPECIES[pet.value.species]?.title : ''
)
const hatEmoji = computed(() =>
  pet.value?.hat ? SHOP_ITEMS[pet.value.hat]?.emoji : null
)
const personality = computed(() =>
  pet.value?.personality ? PERSONALITIES[pet.value.personality] : null
)

async function openChat() {
  try {
    const convId = await useMessengerStore().openPetChat()
    router.push(`/messenger/${convId}`)
  } catch (e) {
    notify.warn(e?.message || 'Чат с Грувиком не открылся')
  }
}
const ownedItems = computed(() => pet.value?.accessories || [])

const xpPercent = computed(() => {
  if (!pet.value) return 0
  if (!pet.value.next_stage_xp) return 100
  return Math.min(100, Math.round((pet.value.xp / pet.value.next_stage_xp) * 100))
})

// Тизер эволюции: следующая форма — загадка (вид пересчитывается на эволюции).
const nextHint = computed(() => {
  const p = pet.value
  if (!p?.next_stage_xp) return ''
  if (p.stage === 0) return `Кто-то вылупится на ${p.next_stage_xp} XP`
  if (p.stage === 1) return `Вид питомца проявится на ${p.next_stage_xp} XP`
  return `Следующая форма откроется на ${p.next_stage_xp} XP`
})

// Календарь кормлений: последние 7 дней, «накормленные» восстанавливаются
// из стрика — он по определению непрерывен и заканчивается в last_fed_date.
const dateKey = (d) => `${d.getFullYear()}-${d.getMonth()}-${d.getDate()}`

const weekDays = computed(() => {
  const p = pet.value
  if (!p) return []
  const fed = new Set()
  if (p.last_fed_date && p.feed_streak > 0) {
    const [y, m, d] = String(p.last_fed_date).slice(0, 10).split('-').map(Number)
    if (y && m && d) {
      for (let i = 0; i < Math.min(p.feed_streak, 14); i++) {
        fed.add(dateKey(new Date(y, m - 1, d - i)))
      }
    }
  }
  const now = new Date()
  const out = []
  for (let i = 6; i >= 0; i--) {
    const dt = new Date(now.getFullYear(), now.getMonth(), now.getDate() - i)
    out.push({
      key: dateKey(dt),
      label: dt.toLocaleDateString('ru-RU', { weekday: 'narrow' }),
      fed: fed.has(dateKey(dt)),
      today: i === 0,
    })
  }
  return out
})

const canFeed = computed(() => {
  if (!pet.value) return false
  const cost = pet.value.sick ? 1 : 3
  return pet.value.beans >= cost && (pet.value.feeds_left == null || pet.value.feeds_left > 0)
})

const feedsMax = computed(() => pet.value?.feeds_max ?? (pet.value?.sick ? 2 : 6))

// Кнопка кормления раньше дизейблилась молча — пользователи с грувами в
// копилке не понимали, почему покормить нельзя. Объясняем причину текстом.
const feedHint = computed(() => {
  const p = pet.value
  if (!p) return ''
  if (p.feeds_left === 0) {
    return p.sick
      ? 'Бульон — не больше двух мисок в день. Завтра счётчик обновится.'
      : 'Грувик сыт: лимит кормлений на сегодня исчерпан, завтра обновится.'
  }
  if (p.beans < (p.sick ? 1 : 3)) {
    return 'Не хватает грувов — их приносят юниты, закрытые задачи и реакции коллег.'
  }
  return ''
})

async function feed() {
  if (feeding.value) return
  feeding.value = true
  const xpBefore = pet.value?.xp ?? 0
  try {
    const res = await groove.feedPet()
    justFed.value = true
    setTimeout(() => { justFed.value = false }, 700)
    const gained = (res.xp ?? xpBefore) - xpBefore
    if (gained > 0) {
      xpGain.value = gained
      xpGainTick.value++
      clearTimeout(xpGainTimer)
      xpGainTimer = setTimeout(() => { xpGain.value = 0 }, 1300)
    }
    if (res.phrase) {
      phrase.value = res.phrase
      clearTimeout(phraseTimer)
      phraseTimer = setTimeout(() => { phrase.value = '' }, 6000)
    }
    if (res.evolved) {
      groove.celebrate('pet_evolved', { pet_name: res.name, stage: res.stage, species: res.species })
    }
    if (res.recovered) {
      groove.celebrate('pet_recovered', { pet_name: res.name })
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
/* «Сцена»: градиентный мир питомца с мягким свечением под ним. */
.pet-stage-bg {
  position: absolute;
  inset: 0 0 auto 0;
  height: 172px;
  background:
    radial-gradient(100px 60px at 50% 84px,
      color-mix(in oklch, var(--color-tertiary) 16%, transparent), transparent 72%),
    linear-gradient(160deg,
      color-mix(in oklch, var(--color-primary-container) 75%, transparent),
      color-mix(in oklch, var(--color-tertiary-container) 45%, transparent) 62%,
      transparent);
  mask-image: linear-gradient(to bottom, black 45%, transparent 100%);
  pointer-events: none;
}
.pet-top { position: relative; display: flex; flex-direction: column; align-items: center; }
.pet-figure {
  position: relative;
  width: 96px;
  height: 96px;
  border-radius: 50%;
  background: var(--color-surface);
  display: grid;
  place-items: center;
  box-shadow: 0 6px 18px color-mix(in oklch, var(--color-primary) 22%, transparent);
  animation: pet-idle 3.8s ease-in-out infinite;
}
.pet-figure.bounce { animation: pet-bounce 0.65s cubic-bezier(0.34, 1.56, 0.64, 1); }
.pet-figure.sick { animation: none; }
@keyframes pet-idle {
  0%, 100% { transform: translateY(0) scale(1); }
  50% { transform: translateY(-4px) scale(1.015); }
}
@keyframes pet-bounce {
  0% { transform: scale(1); }
  35% { transform: scale(1.18) rotate(-4deg); }
  70% { transform: scale(0.96) rotate(2deg); }
  100% { transform: scale(1); }
}
@media (prefers-reduced-motion: reduce) {
  .pet-figure { animation: none; }
}
.pet-figure.sick .pet-emoji { filter: grayscale(0.55) brightness(0.92); }
.pet-xp-float {
  position: absolute;
  top: 4px;
  right: -52px;
  font-size: 13.5px;
  font-weight: 800;
  color: var(--color-success);
  pointer-events: none;
  animation: xp-rise 1.25s ease-out forwards;
}
.pet-beans-float {
  position: absolute;
  top: 4px;
  left: -60px;
  font-size: 13.5px;
  font-weight: 800;
  color: var(--color-warning);
  pointer-events: none;
  animation: xp-rise 1.5s ease-out forwards;
}
@keyframes xp-rise {
  0% { transform: translateY(8px); opacity: 0; }
  25% { opacity: 1; }
  100% { transform: translateY(-20px); opacity: 0; }
}
.pet-sick-badge {
  position: absolute;
  bottom: -4px;
  left: -4px;
  font-size: 22px;
}
.pet-emoji { font-size: 52px; line-height: 1; }
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

.pet-personality {
  margin: 4px 0 0;
  font-size: 12px;
  font-weight: 600;
  padding: 3px 10px;
  border-radius: var(--radius-full);
  background: var(--color-tertiary-container);
  color: var(--color-on-tertiary-container);
}
.pet-sick-bar {
  width: 100%;
  margin-top: 12px;
  border: 1px dashed color-mix(in oklch, var(--color-error) 45%, transparent);
  border-radius: 14px;
  padding: 10px 12px;
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.pet-sick-head {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12.5px;
  font-weight: 600;
  color: var(--color-error);
}
.pet-sick-head .material-symbols-outlined { font-size: 17px; }
.pet-sick-progress { display: flex; align-items: center; gap: 6px; }
.pet-sick-dot {
  width: 14px;
  height: 14px;
  border-radius: 50%;
  background: var(--color-surface-high);
  border: 1.5px solid var(--color-outline-dim);
}
.pet-sick-dot.filled {
  background: var(--color-success);
  border-color: var(--color-success);
}
.pet-sick-count { font-size: 12px; font-weight: 700; margin-left: 2px; }
.pet-sick-hint { margin: 0; font-size: 11.5px; color: var(--color-text-dim); line-height: 1.4; }
.pet-xp { width: 100%; margin-top: 14px; }
.pet-xp-meta {
  display: flex;
  justify-content: space-between;
  font-size: 11.5px;
  color: var(--color-text-dim);
  margin-bottom: 4px;
}
.pet-xp-row { display: flex; align-items: center; gap: 8px; }
.pet-xp-bar {
  flex: 1;
  height: 8px;
  border-radius: var(--radius-full);
  background: var(--color-surface-high);
  overflow: hidden;
}
.pet-next {
  width: 26px;
  height: 26px;
  border-radius: 50%;
  border: 1.5px dashed var(--color-outline-dim);
  background: var(--color-surface-high);
  display: grid;
  place-items: center;
  font-size: 13px;
  font-weight: 800;
  color: var(--color-text-dim);
  cursor: help;
  flex-shrink: 0;
}
.pet-next.egg {
  font-size: 14px;
  filter: grayscale(1) brightness(0.6);
  opacity: 0.85;
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

/* ── Календарь кормлений за неделю ─────────────────────────── */
.pet-week { display: flex; gap: 9px; margin-top: 12px; }
.pet-week-day { display: flex; flex-direction: column; align-items: center; gap: 3px; }
.pet-week-dot {
  width: 15px;
  height: 15px;
  border-radius: 50%;
  background: var(--color-surface-high);
  border: 1.5px solid var(--color-outline-dim);
  transition: background 0.2s, border-color 0.2s;
}
.pet-week-day.fed .pet-week-dot {
  background: color-mix(in oklch, var(--color-warning) 78%, transparent);
  border-color: var(--color-warning);
}
.pet-week-day.today .pet-week-dot {
  box-shadow: 0 0 0 2px color-mix(in oklch, var(--color-primary) 40%, transparent);
}
.pet-week-label {
  font-size: 9.5px;
  font-weight: 600;
  text-transform: uppercase;
  color: var(--color-text-dim);
}

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

.pet-feed-hint {
  margin: 8px 0 0;
  font-size: 11.5px;
  color: var(--color-text-dim);
  text-align: center;
  line-height: 1.4;
}

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

/* ── Квест дня ──────────────────────────────────────────────── */
.pet-quest {
  width: 100%;
  margin-top: 14px;
  border: 1px solid color-mix(in oklch, var(--color-tertiary) 35%, var(--color-outline-dim));
  border-radius: 14px;
  padding: 10px 12px;
  background: color-mix(in oklch, var(--color-tertiary-container) 35%, transparent);
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.pet-quest.done {
  border-color: var(--color-success);
  background: color-mix(in oklch, var(--color-success) 14%, transparent);
}
.pet-quest.claimed {
  border-color: var(--color-outline-dim);
  background: var(--color-surface-high);
  opacity: 0.85;
}
.pet-quest-head {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  font-weight: 700;
}
.pet-quest-ico { font-size: 18px; color: var(--color-tertiary); }
.pet-quest.done .pet-quest-ico { color: var(--color-success); }
.pet-quest-title { flex: 1; min-width: 0; }
.pet-quest-reward {
  font-size: 12px;
  font-weight: 700;
  color: var(--color-text-dim);
  white-space: nowrap;
}
.pet-quest-bar {
  height: 6px;
  border-radius: var(--radius-full);
  background: var(--color-surface);
  overflow: hidden;
}
.pet-quest-fill {
  height: 100%;
  border-radius: inherit;
  background: var(--color-tertiary);
  transition: width 0.5s cubic-bezier(0.34, 1.56, 0.64, 1);
}
.pet-quest.done .pet-quest-fill { background: var(--color-success); }
.pet-quest-meta {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 8px;
  font-size: 11.5px;
  color: var(--color-text-dim);
}
.pet-quest-hint { text-align: right; line-height: 1.35; }
.pet-quest-claim {
  border: none;
  border-radius: var(--radius-full);
  background: var(--color-success);
  color: var(--color-on-primary);
  font-size: 12px;
  font-weight: 700;
  padding: 5px 12px;
  cursor: pointer;
  transition: transform 0.1s;
}
.pet-quest-claim:active { transform: scale(0.95); }
.pet-quest-claim:disabled { opacity: 0.5; cursor: default; }
.pet-quest-claimed { font-weight: 700; color: var(--color-success); }
</style>
