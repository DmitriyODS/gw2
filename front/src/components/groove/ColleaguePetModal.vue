<template>
  <AppDialog
    :model-value="modelValue"
    :title="pet?.name || 'Грувик'"
    :subtitle="ownerName"
    icon="pets"
    tone="primary"
    size="sm"
    @update:model-value="$emit('update:modelValue', $event)"
  >
    <div class="cpm-figure-wrap">
      <div class="cpm-figure" :class="{ sick: pet?.sick }">
        <span class="cpm-emoji">{{ petEmoji(pet) }}</span>
        <span v-if="hatEmoji" class="cpm-hat">{{ hatEmoji }}</span>
        <span v-if="pet?.sick" class="cpm-sick-badge">🤒</span>
      </div>
    </div>

    <p class="cpm-stage">
      {{ stageTitle }}<template v-if="speciesTitle"> · {{ speciesTitle }}</template>
    </p>

    <p v-if="personality" class="cpm-personality">
      {{ personality.emoji }} {{ personality.title }}
    </p>

    <div v-if="pet && !pet.sick" class="cpm-xp">
      <div class="cpm-xp-meta">
        <span>{{ pet.stage >= maxStage ? 'Максимальная форма' : 'До эволюции' }}</span>
        <span v-if="pet.next_stage_xp">{{ pet.xp }} / {{ pet.next_stage_xp }} XP</span>
      </div>
      <div class="cpm-xp-bar">
        <div class="cpm-xp-fill" :style="{ width: xpPercent + '%' }"></div>
      </div>
    </div>

    <div v-if="pet?.sick" class="cpm-sick-row">
      <span class="material-symbols-outlined">healing</span>
      <div class="cpm-sick-dots">
        <span
          v-for="i in pet.recovery_target"
          :key="i"
          class="cpm-sick-dot"
          :class="{ filled: i <= pet.recovery }"
        ></span>
      </div>
      <span class="cpm-sick-count">{{ pet.recovery }}/{{ pet.recovery_target }}</span>
    </div>

    <div v-if="ownedItems.length" class="cpm-accessories">
      <span
        v-for="item in ownedItems"
        :key="item"
        class="cpm-acc-item"
        :class="{ active: pet?.hat === item }"
        :title="SHOP_ITEMS[item]?.title || item"
      >{{ SHOP_ITEMS[item]?.emoji || '🎁' }}</span>
    </div>

    <div class="cpm-stats">
      <span class="cpm-chip" title="Накоплено грувов"><GrooveCoin /> {{ pet?.beans ?? 0 }}</span>
      <span class="cpm-chip" title="Стрик кормлений">
        <span class="material-symbols-outlined">local_fire_department</span>
        {{ pet?.feed_streak ?? 0 }} дн.
      </span>
    </div>

    <div v-if="isOwn" class="cpm-actions">
      <p class="cpm-own-hint">
        <span class="material-symbols-outlined">waving_hand</span>
        Это ваш Грувик
      </p>
    </div>
  </AppDialog>
</template>

<script setup>
import { computed } from 'vue'
import AppDialog from '@/components/common/AppDialog.vue'
import GrooveCoin from '@/components/groove/GrooveCoin.vue'
import { useGrooveStore } from '@/stores/groove.js'
import { petEmoji, PET_STAGES, PET_SPECIES, PERSONALITIES, SHOP_ITEMS } from '@/utils/groove.js'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  pet: { type: Object, default: null },
})
defineEmits(['update:modelValue'])

const groove = useGrooveStore()

const maxStage = PET_STAGES.length - 1

const isOwn = computed(() => props.pet?.user_id === groove.myId)

const ownerName = computed(() => {
  const fio = props.pet?.user?.fio
  if (!fio) return ''
  const parts = fio.split(' ')
  return parts.length > 1 ? `${parts[0]} ${parts[1]}` : fio
})

const stageTitle = computed(() => PET_STAGES[props.pet?.stage] || '')
const speciesTitle = computed(() =>
  props.pet?.stage >= 2 ? PET_SPECIES[props.pet?.species]?.title : ''
)
const hatEmoji = computed(() => props.pet?.hat ? SHOP_ITEMS[props.pet.hat]?.emoji : null)
const personality = computed(() =>
  props.pet?.personality ? PERSONALITIES[props.pet.personality] : null
)
const ownedItems = computed(() => props.pet?.accessories || [])

const xpPercent = computed(() => {
  const p = props.pet
  if (!p || !p.next_stage_xp) return 100
  return Math.min(100, Math.round((p.xp / p.next_stage_xp) * 100))
})
</script>

<style scoped>
.cpm-figure-wrap {
  display: flex;
  justify-content: center;
  margin-bottom: 10px;
}
.cpm-figure {
  position: relative;
  width: 100px;
  height: 100px;
  border-radius: 50%;
  background: var(--color-primary-container);
  display: grid;
  place-items: center;
  box-shadow: 0 6px 20px color-mix(in oklch, var(--color-primary) 22%, transparent);
  animation: cpm-idle 3.8s ease-in-out infinite;
}
.cpm-figure.sick { animation: none; }
@keyframes cpm-idle {
  0%, 100% { transform: translateY(0) scale(1); }
  50% { transform: translateY(-4px) scale(1.015); }
}
@media (prefers-reduced-motion: reduce) { .cpm-figure { animation: none; } }
.cpm-emoji { font-size: 54px; line-height: 1; }
.cpm-figure.sick .cpm-emoji { filter: grayscale(0.55) brightness(0.92); }
.cpm-hat {
  position: absolute;
  top: -12px;
  right: 2px;
  font-size: 26px;
  transform: rotate(12deg);
}
.cpm-sick-badge {
  position: absolute;
  bottom: -4px;
  left: -4px;
  font-size: 22px;
}

.cpm-stage {
  margin: 0 0 4px;
  font-size: 13px;
  color: var(--color-text-dim);
  text-align: center;
}
.cpm-personality {
  margin: 0 0 10px;
  font-size: 12px;
  font-weight: 600;
  padding: 3px 10px;
  border-radius: var(--radius-full);
  background: var(--color-tertiary-container);
  color: var(--color-on-tertiary-container);
  align-self: center;
}

.cpm-xp { width: 100%; margin-top: 10px; }
.cpm-xp-meta {
  display: flex;
  justify-content: space-between;
  font-size: 11.5px;
  color: var(--color-text-dim);
  margin-bottom: 4px;
}
.cpm-xp-bar {
  width: 100%;
  height: 8px;
  border-radius: var(--radius-full);
  background: var(--color-surface-high);
  overflow: hidden;
}
.cpm-xp-fill {
  height: 100%;
  border-radius: inherit;
  background: var(--color-primary);
  transition: width 0.6s cubic-bezier(0.34, 1.56, 0.64, 1);
}

.cpm-sick-row {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 10px;
  padding: 8px 12px;
  border: 1px dashed color-mix(in oklch, var(--color-error) 40%, transparent);
  border-radius: 12px;
  width: 100%;
}
.cpm-sick-row .material-symbols-outlined { font-size: 18px; color: var(--color-error); }
.cpm-sick-dots { display: flex; gap: 5px; flex: 1; }
.cpm-sick-dot {
  width: 13px;
  height: 13px;
  border-radius: 50%;
  background: var(--color-surface-high);
  border: 1.5px solid var(--color-outline-dim);
}
.cpm-sick-dot.filled { background: var(--color-success); border-color: var(--color-success); }
.cpm-sick-count { font-size: 12px; font-weight: 700; }

.cpm-accessories {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
  justify-content: center;
  margin-top: 10px;
}
.cpm-acc-item {
  width: 36px;
  height: 36px;
  border-radius: 10px;
  background: var(--color-surface-high);
  border: 1.5px solid var(--color-outline-dim);
  display: grid;
  place-items: center;
  font-size: 18px;
}
.cpm-acc-item.active {
  border-color: var(--color-primary);
  background: var(--color-primary-container);
}

.cpm-stats {
  display: flex;
  gap: 8px;
  margin-top: 12px;
  flex-wrap: wrap;
  justify-content: center;
}
.cpm-chip {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 10px;
  border-radius: var(--radius-full);
  background: var(--color-surface-high);
  font-size: 12.5px;
  font-weight: 600;
}
.cpm-chip .material-symbols-outlined { font-size: 15px; }

.cpm-actions { width: 100%; margin-top: 14px; }
.cpm-own-hint {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  margin: 0;
  font-size: 13px;
  font-weight: 600;
  color: var(--color-text-dim);
}
.cpm-own-hint .material-symbols-outlined { font-size: 17px; }
</style>
