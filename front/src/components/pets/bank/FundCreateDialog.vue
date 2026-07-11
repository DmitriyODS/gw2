<template>
  <AppDialog
    :model-value="modelValue"
    title="Новый сбор"
    subtitle="Общая цель компании — коллеги скидываются кудосами"
    icon="volunteer_activism"
    tone="primary"
    size="sm"
    @update:model-value="$emit('update:modelValue', $event)"
  >
    <div class="fcd">
      <div class="fcd-emojis">
        <button
          v-for="e in EMOJIS"
          :key="e"
          class="fcd-emoji"
          :class="{ active: emoji === e }"
          type="button"
          @click="emoji = e"
        >{{ e }}</button>
      </div>
      <input
        v-model="title"
        type="text" maxlength="60"
        class="fcd-input"
        placeholder="Название: «Пицца за релиз», «Помощь приюту»…"
      />
      <textarea
        v-model="description"
        maxlength="300" rows="3"
        class="fcd-input fcd-textarea"
        placeholder="Что будет, когда соберём (необязательно)"
      ></textarea>
      <div class="fcd-targets">
        <button
          v-for="t in QUICK_TARGETS"
          :key="t"
          class="fcd-chip"
          :class="{ active: target === t }"
          type="button"
          @click="target = t"
        ><KudosCoin /> {{ t }}</button>
        <AmountInput v-model="target" :max="100000" placeholder="Цель" size="sm" class="fcd-target" />
      </div>
      <p class="fcd-hint">Взносы не возвращаются — это благотворительность. Сбор видят все коллеги.</p>
    </div>

    <template #footer>
      <div class="fcd-footer">
        <button
          class="btn-grad"
          :disabled="busy || !title.trim() || !validTarget"
          @click="create"
        >Объявить сбор</button>
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

const EMOJIS = ['💝', '🍕', '🎂', '🌳', '🐾', '🚑', '🎄', '🏆']
const QUICK_TARGETS = [500, 1000, 3000, 5000]

const pets = usePetsStore()
const notify = useNotificationsStore()

const busy = ref(false)
const title = ref('')
const description = ref('')
const emoji = ref(EMOJIS[0])
const target = ref(null)

const validTarget = computed(() =>
  Number.isFinite(target.value) && target.value >= 1 && target.value <= 100000)

watch(() => props.modelValue, (open) => {
  if (!open) return
  title.value = ''
  description.value = ''
  emoji.value = EMOJIS[0]
  target.value = null
})

async function create() {
  busy.value = true
  try {
    await pets.createFund({
      title: title.value, description: description.value,
      emoji: emoji.value, target: target.value,
    })
    notify.success('Сбор объявлен — коллеги увидят его в банке')
    emit('update:modelValue', false)
  } catch (e) {
    notify.error(e?.message || 'Не получилось объявить сбор')
  } finally {
    busy.value = false
  }
}
</script>

<style scoped>
.fcd { display: flex; flex-direction: column; gap: 12px; }
.fcd-hint { margin: 0; font-size: 12px; color: var(--color-text-dim); line-height: 1.4; }

.fcd-emojis { display: flex; gap: 6px; flex-wrap: wrap; }
.fcd-emoji {
  width: 40px; height: 40px;
  border: 1px solid var(--color-outline-dim);
  border-radius: 12px;
  background: var(--color-surface);
  background: var(--glass-bg);
  box-shadow: var(--glass-edge);
  font-size: 20px;
  cursor: pointer;
  display: grid; place-items: center;
}
.fcd-emoji.active {
  border-color: var(--color-primary);
  background: var(--color-primary-container);
}

.fcd-input {
  width: 100%;
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-sm);
  background: var(--color-surface);
  background: var(--glass-bg);
  box-shadow: var(--glass-edge);
  color: var(--color-text);
  font: inherit; font-size: 13px;
  padding: 9px 11px;
}
.fcd-textarea { resize: vertical; }
.fcd-target { flex: 1 1 100%; width: 100%; }
.fcd-input:focus { outline: none; border-color: var(--color-primary); }

.fcd-targets { display: flex; gap: 8px; flex-wrap: wrap; align-items: center; }
.fcd-chip {
  border: 1px solid var(--color-outline-dim);
  background: var(--color-surface);
  background: var(--glass-bg);
  box-shadow: var(--glass-edge);
  color: var(--color-text);
  border-radius: var(--radius-full);
  font: inherit; font-size: 12.5px; font-weight: 600;
  padding: 6px 12px; cursor: pointer;
  display: inline-flex; align-items: center; gap: 4px;
}
.fcd-chip.active {
  border-color: var(--color-primary);
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
}

.fcd-footer { display: flex; justify-content: flex-end; width: 100%; }
</style>
