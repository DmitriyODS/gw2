<template>
  <AppDialog
    model-value
    :tone="tone"
    :show-icon="false"
    size="md"
    mobile="sheet"
    :actions="actions"
    @update:model-value="(v) => !v && $emit('close')"
    @cancel="$emit('close')"
    @confirm="goToTasks"
  >
    <div class="mb" :class="`mood-${b.mood}`">
      <!-- Сцена: Грувик под мягким свечением + настроенческий декор. -->
      <div class="mb-stage">
        <span class="mb-glow" aria-hidden="true" />
        <template v-if="decor">
          <i
            v-for="n in decor.count"
            :key="n"
            class="mb-decor"
            :style="decorStyle(n)"
            aria-hidden="true"
          >{{ decor.emoji }}</i>
        </template>
        <div class="mb-pet" :class="{ sick: pet.sick }">
          <span class="mb-pet-emoji">{{ emoji }}</span>
          <span v-if="hatEmoji" class="mb-pet-hat">{{ hatEmoji }}</span>
          <span v-if="pet.sick" class="mb-pet-badge" title="Грувик приболел">🤒</span>
        </div>
      </div>

      <h2 class="mb-greeting">{{ b.greeting }}, {{ b.first_name }}!</h2>

      <!-- Реплика Грувика. -->
      <div class="mb-bubble">
        <p class="mb-message">{{ b.message }}</p>
        <span class="mb-bubble-author">— {{ pet.name }}</span>
      </div>

      <!-- Сводка цифрами. В выходной рабочие цифры не показываем. -->
      <div v-if="b.mood !== 'weekend'" class="mb-stats">
        <div class="mb-stat">
          <span class="mb-stat-num">{{ b.open_count }}</span>
          <span class="mb-stat-label">{{ plural(b.open_count, 'задача', 'задачи', 'задач') }} в работе</span>
        </div>
        <div v-if="b.stale_count" class="mb-stat accent">
          <span class="mb-stat-num">{{ b.stale_count }}</span>
          <span class="mb-stat-label">{{ plural(b.stale_count, 'засиделась', 'засиделись', 'засиделись') }}</span>
        </div>
      </div>

      <!-- Засидевшиеся задачи — кликабельны. -->
      <ul v-if="b.stale?.length" class="mb-list">
        <li v-for="t in b.stale" :key="t.id" class="mb-item" @click="open(t)">
          <div class="mb-item-main">
            <span class="mb-item-name">{{ t.name }}</span>
            <span v-if="t.department?.name" class="mb-item-dept">{{ t.department.name }}</span>
          </div>
          <span class="mb-days">{{ daysLabel(t.days_pending) }}</span>
          <span class="material-symbols-outlined mb-arrow">chevron_right</span>
        </li>
      </ul>
    </div>
  </AppDialog>
</template>

<script setup>
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import AppDialog from '@/components/common/AppDialog.vue'
import { petEmoji, SHOP_ITEMS } from '@/utils/groove.js'

const props = defineProps({
  briefing: { type: Object, required: true },
})
const emit = defineEmits(['close'])
const router = useRouter()

const b = computed(() => props.briefing)
const pet = computed(() => props.briefing.pet || {})
const emoji = computed(() => petEmoji(pet.value))
const hatEmoji = computed(() => (pet.value.hat ? SHOP_ITEMS[pet.value.hat]?.emoji : null))

// Тон диалога и кнопки зависят от настроения Грувика.
const TONE_BY_MOOD = {
  sick: 'warning',
  buried: 'warning',
  reminder: 'primary',
  fresh: 'success',
  weekend: 'success',
}
const tone = computed(() => TONE_BY_MOOD[b.value.mood] || 'primary')

// В выходной к задачам не зовём — единственная кнопка закрывает модалку.
const actions = computed(() => b.value.mood === 'weekend'
  ? [{ kind: 'confirm', label: 'Отдыхаем!', icon: 'beach_access' }]
  : [
      { kind: 'cancel', label: 'Позже' },
      b.value.stale_count
        ? { kind: 'confirm', label: 'Разобрать задачи', icon: 'cleaning_services' }
        : { kind: 'confirm', label: 'К задачам', icon: 'bolt' },
    ])

// Парящий декор вокруг питомца под настроение.
const DECOR_BY_MOOD = {
  buried: { emoji: '📄', count: 5 },
  sick: { emoji: '💤', count: 3 },
  fresh: { emoji: '✨', count: 5 },
  weekend: { emoji: '🌴', count: 4 },
}
const decor = computed(() => DECOR_BY_MOOD[b.value.mood] || null)

// Детерминированный разброс декора кольцом вокруг питомца (без рандома —
// стабильно между рендерами, в пределах сцены).
function decorStyle(n) {
  const angle = (n * 67) % 360
  const radius = 40 + ((n * 13) % 18)
  const x = 50 + radius * Math.cos((angle * Math.PI) / 180) * 0.7
  const y = 50 + radius * Math.sin((angle * Math.PI) / 180) * 0.42
  return {
    left: `${x}%`,
    top: `${y}%`,
    animationDelay: `${(n % 5) * 0.45}s`,
    fontSize: `${16 + ((n * 7) % 10)}px`,
  }
}

function plural(n, one, few, many) {
  const mod10 = n % 10, mod100 = n % 100
  if (mod10 === 1 && mod100 !== 11) return one
  if (mod10 >= 2 && mod10 <= 4 && (mod100 < 10 || mod100 >= 20)) return few
  return many
}

function daysLabel(days) {
  const d = Math.max(1, days || 0)
  return `${d} ${plural(d, 'день', 'дня', 'дней')}`
}

function open(task) {
  emit('close')
  router.push({ path: '/tasks', query: { open: task.id } })
}

function goToTasks() {
  emit('close')
  if (b.value.mood === 'weekend') return // в выходной никуда не гоним
  router.push('/tasks')
}
</script>

<style scoped>
.mb {
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  padding-top: 4px;
}

/* ── Сцена с питомцем ── */
.mb-stage {
  position: relative;
  width: 100%;
  height: 132px;
  display: grid;
  place-items: center;
  margin-bottom: 4px;
}

.mb-glow {
  position: absolute;
  width: 150px;
  height: 150px;
  border-radius: 50%;
  background: radial-gradient(
    circle,
    color-mix(in oklch, var(--glow-color) 45%, transparent) 0%,
    transparent 68%
  );
  filter: blur(2px);
}

.mood-fresh,
.mood-weekend { --glow-color: var(--color-success); }
.mood-reminder { --glow-color: var(--color-primary); }
.mood-buried,
.mood-sick { --glow-color: var(--color-warning, var(--color-tertiary)); }

.mb-pet {
  position: relative;
  width: 96px;
  height: 96px;
  border-radius: 50%;
  display: grid;
  place-items: center;
  background: var(--color-surface-low);
  box-shadow: var(--shadow-md, 0 8px 24px rgba(0, 0, 0, 0.12));
  animation: mb-bob 3.4s ease-in-out infinite;
}

.mb-pet.sick {
  animation: none;
  filter: grayscale(0.5) brightness(0.95);
}

.mb-pet-emoji {
  font-size: 54px;
  line-height: 1;
}

.mb-pet-hat {
  position: absolute;
  top: -14px;
  right: 4px;
  font-size: 26px;
  transform: rotate(12deg);
}

.mb-pet-badge {
  position: absolute;
  bottom: -2px;
  right: -4px;
  font-size: 26px;
}

.mb-decor {
  position: absolute;
  font-style: normal;
  opacity: 0.85;
  animation: mb-float 3.8s ease-in-out infinite;
  pointer-events: none;
}

/* ── Приветствие ── */
.mb-greeting {
  margin: 8px 0 0;
  font-size: 24px;
  font-weight: 800;
  letter-spacing: -0.3px;
  color: var(--color-text);
}

/* ── Облачко реплики ── */
.mb-bubble {
  position: relative;
  margin-top: 14px;
  max-width: 420px;
  padding: 14px 18px 10px;
  border-radius: var(--radius-lg, 20px);
  background: var(--color-surface-high);
  border: 1px solid var(--color-outline-dim);
}

.mb-bubble::before {
  content: '';
  position: absolute;
  top: -8px;
  left: 50%;
  width: 16px;
  height: 16px;
  background: var(--color-surface-high);
  border-left: 1px solid var(--color-outline-dim);
  border-top: 1px solid var(--color-outline-dim);
  transform: translateX(-50%) rotate(45deg);
}

.mb-message {
  margin: 0;
  font-size: 15px;
  line-height: 1.5;
  color: var(--color-text);
}

.mb-bubble-author {
  display: block;
  margin-top: 6px;
  font-size: 12px;
  font-weight: 600;
  color: var(--color-text-dim);
}

/* ── Сводка цифрами ── */
.mb-stats {
  display: flex;
  gap: 10px;
  margin-top: 18px;
}

.mb-stat {
  display: flex;
  flex-direction: column;
  align-items: center;
  min-width: 96px;
  padding: 10px 16px;
  border-radius: var(--radius-md, 16px);
  background: var(--color-surface-low);
}

.mb-stat.accent {
  background: var(--color-warning-container, var(--color-tertiary-container));
}

.mb-stat-num {
  font-size: 26px;
  font-weight: 800;
  line-height: 1.1;
  color: var(--color-text);
}

.mb-stat.accent .mb-stat-num {
  color: var(--color-on-warning-container, var(--color-on-tertiary-container));
}

.mb-stat-label {
  margin-top: 2px;
  font-size: 12px;
  color: var(--color-text-dim);
}

.mb-stat.accent .mb-stat-label {
  color: var(--color-on-warning-container, var(--color-on-tertiary-container));
}

/* ── Список засидевшихся ── */
.mb-list {
  list-style: none;
  width: 100%;
  margin: 18px 0 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.mb-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 10px 10px 14px;
  border-radius: var(--radius-md, 16px);
  background: var(--color-surface-low);
  border-left: 3px solid var(--color-warning, var(--color-tertiary));
  cursor: pointer;
  text-align: left;
  transition: background 0.15s, transform 0.12s;
}

.mb-item:hover {
  background: var(--color-surface-high);
  transform: translateX(2px);
}

.mb-item-main { flex: 1; min-width: 0; display: flex; flex-direction: column; gap: 2px; }

.mb-item-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--color-text);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.mb-item-dept {
  font-size: 12px;
  color: var(--color-text-dim);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.mb-days {
  flex-shrink: 0;
  font-size: 12px;
  font-weight: 700;
  color: var(--color-on-warning-container, var(--color-on-tertiary-container));
  background: var(--color-warning-container, var(--color-tertiary-container));
  padding: 3px 10px;
  border-radius: var(--radius-full);
  white-space: nowrap;
}

.mb-arrow { font-size: 20px; color: var(--color-text-dim); flex-shrink: 0; }

@keyframes mb-bob {
  0%, 100% { transform: translateY(0); }
  50% { transform: translateY(-7px); }
}

@keyframes mb-float {
  0%, 100% { transform: translateY(0) rotate(-6deg); opacity: 0.85; }
  50% { transform: translateY(-9px) rotate(6deg); opacity: 1; }
}

@media (prefers-reduced-motion: reduce) {
  .mb-pet, .mb-decor { animation: none; }
}
</style>
