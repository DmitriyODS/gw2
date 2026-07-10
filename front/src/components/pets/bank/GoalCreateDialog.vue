<template>
  <AppDialog
    :model-value="modelValue"
    title="Новая копилка"
    subtitle="Копите на мечту — кудосы лежат отдельно от кошелька"
    icon="savings"
    tone="primary"
    size="sm"
    @update:model-value="$emit('update:modelValue', $event)"
  >
    <div class="gcd">
      <div class="gcd-emojis">
        <button
          v-for="e in EMOJIS"
          :key="e"
          class="gcd-emoji"
          :class="{ active: emoji === e }"
          type="button"
          @click="emoji = e"
        >{{ e }}</button>
      </div>
      <input
        v-model="title"
        type="text" maxlength="40"
        class="gcd-input"
        placeholder="На что копим? Например: «Легендарный облик»"
      />
      <div class="gcd-targets">
        <button
          v-for="t in QUICK_TARGETS"
          :key="t"
          class="gcd-chip"
          :class="{ active: target === t }"
          type="button"
          @click="target = t"
        ><KudosCoin /> {{ t }}</button>
      </div>
      <AmountInput v-model="target" :max="10000" placeholder="Цель, кудосов" size="sm" class="gcd-target" />
    </div>

    <template #footer>
      <div class="gcd-footer">
        <button
          class="btn-grad"
          :disabled="busy || !title.trim() || !validTarget"
          @click="create"
        >Создать копилку</button>
      </div>
    </template>
  </AppDialog>
</template>

<script setup>
import { computed, ref, watch } from 'vue'
import AppDialog from '@/components/common/AppDialog.vue'
import KudosCoin from '@/components/pets/KudosCoin.vue'
import AmountInput from '@/components/pets/bank/AmountInput.vue'
import { usePetsStore } from '@/stores/pets.js'
import { useNotificationsStore } from '@/stores/notifications.js'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
})
const emit = defineEmits(['update:modelValue'])

const EMOJIS = ['🎯', '🐉', '🏰', '🎁', '🌟', '🧸', '🚀', '🍕']
const QUICK_TARGETS = [100, 300, 500, 1000]

const pets = usePetsStore()
const notify = useNotificationsStore()

const busy = ref(false)
const title = ref('')
const emoji = ref(EMOJIS[0])
const target = ref(null)

const validTarget = computed(() =>
  Number.isFinite(target.value) && target.value >= 1 && target.value <= 10000)

watch(() => props.modelValue, (open) => {
  if (!open) return
  title.value = ''
  emoji.value = EMOJIS[0]
  target.value = null
})

async function create() {
  busy.value = true
  try {
    await pets.createGoal(title.value, emoji.value, target.value)
    notify.success('Копилка создана — пополняйте с кошелька')
    emit('update:modelValue', false)
  } catch (e) {
    notify.error(e?.message || 'Не получилось создать копилку')
  } finally {
    busy.value = false
  }
}
</script>

<style scoped>
.gcd { display: flex; flex-direction: column; gap: 12px; }

.gcd-emojis { display: flex; gap: 6px; flex-wrap: wrap; }
.gcd-emoji {
  width: 40px; height: 40px;
  border: 1px solid var(--color-outline-dim);
  border-radius: 12px;
  background: var(--color-surface);
  font-size: 20px;
  cursor: pointer;
  display: grid; place-items: center;
}
.gcd-emoji.active {
  border-color: var(--color-primary);
  background: var(--color-primary-container);
}

.gcd-input {
  width: 100%;
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-sm);
  background: var(--color-surface);
  color: var(--color-text);
  font: inherit; font-size: 13px;
  padding: 9px 11px;
}
.gcd-input:focus { outline: none; border-color: var(--color-primary); }

.gcd-targets { display: flex; gap: 8px; flex-wrap: wrap; align-items: center; }
.gcd-chip {
  border: 1px solid var(--color-outline-dim);
  background: var(--color-surface);
  color: var(--color-text);
  border-radius: var(--radius-full);
  font: inherit; font-size: 12.5px; font-weight: 600;
  padding: 6px 12px; cursor: pointer;
  display: inline-flex; align-items: center; gap: 4px;
}
.gcd-chip.active {
  border-color: var(--color-primary);
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
}

.gcd-target :deep(input) { font-size: 13px; }
.gcd-footer { display: flex; justify-content: flex-end; width: 100%; }
</style>
