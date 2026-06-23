<template>
  <div v-if="unit" class="unit-banner" role="status" aria-label="Идёт работа над юнитом">
    <button class="ub-main" title="Развернуть" @click="expand">
      <span class="ub-rec" aria-hidden="true"></span>
      <span class="ub-text">
        <span class="ub-label">Идёт работа</span>
        <span class="ub-name">
          {{ unit.name }}
          <span class="ub-task">· {{ unit.task_name || `Задача #${unit.task_id}` }}</span>
        </span>
      </span>
      <span class="ub-timer">{{ clock }}</span>
    </button>
    <div class="ub-actions">
      <button class="ub-btn ub-expand" title="Развернуть" @click="expand">
        <span class="material-symbols-outlined">open_in_full</span>
        <span class="ub-btn-label">Открыть</span>
      </button>
      <button class="ub-btn ub-stop" title="Завершить" :disabled="stopping" @click="confirmStop = true">
        <span class="material-symbols-outlined">check</span>
        <span class="ub-btn-label">Завершить</span>
      </button>
    </div>

    <ConfirmDialog
      :visible="confirmStop"
      header="Завершить юнит"
      :message="`Завершить «${unit.name}»? Учёт времени остановится.`"
      confirm-label="Завершить"
      @confirm="handleStop"
      @cancel="confirmStop = false"
    />
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { useUnitsStore } from '@/stores/units.js'
import { useActiveUnit } from '@/composables/useActiveUnit.js'
import { useElapsed } from '@/composables/useElapsed.js'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'

const unitsStore = useUnitsStore()
const { stopping, stop, expand } = useActiveUnit()

const unit = computed(() => unitsStore.activeUnit)
const { clock } = useElapsed(() => unit.value?.datetime_start)

const confirmStop = ref(false)

async function handleStop() {
  confirmStop.value = false
  await stop()
}
</script>

<style scoped>
.unit-banner {
  flex-shrink: 0;
  position: relative;
  z-index: 2;
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 16px;
  color: var(--color-on-primary);
  background: linear-gradient(
    100deg,
    var(--color-primary) 0%,
    color-mix(in oklch, var(--color-primary) 55%, var(--color-secondary)) 100%
  );
  box-shadow: var(--shadow-md);
}

/* Кликабельная зона «развернуть» — занимает всё свободное место. */
.ub-main {
  flex: 1;
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 0;
  border: none;
  background: transparent;
  color: inherit;
  font: inherit;
  text-align: left;
  cursor: pointer;
}

.ub-rec {
  flex-shrink: 0;
  width: 12px;
  height: 12px;
  border-radius: 50%;
  background: var(--color-on-primary);
  box-shadow: 0 0 0 0 color-mix(in oklch, var(--color-on-primary) 60%, transparent);
  animation: ubPulse 1.6s ease-in-out infinite;
}

@keyframes ubPulse {
  0%, 100% { box-shadow: 0 0 0 0 color-mix(in oklch, var(--color-on-primary) 55%, transparent); }
  50%      { box-shadow: 0 0 0 7px color-mix(in oklch, var(--color-on-primary) 0%, transparent); }
}

.ub-text {
  min-width: 0;
  display: flex;
  flex-direction: column;
  line-height: 1.25;
}

.ub-label {
  font-size: 11px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.6px;
  opacity: 0.85;
}

.ub-name {
  font-size: 14px;
  font-weight: 600;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.ub-task { font-weight: 400; opacity: 0.85; }

.ub-timer {
  flex-shrink: 0;
  margin-left: auto;
  font-size: 20px;
  font-weight: 700;
  font-variant-numeric: tabular-nums;
  letter-spacing: 0.5px;
}

.ub-actions {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  gap: 8px;
}

.ub-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 8px 14px;
  border: none;
  border-radius: var(--radius-full, 999px);
  font: inherit;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  transition: transform 0.12s, filter 0.15s, opacity 0.15s;
}

.ub-btn:hover:not(:disabled) { transform: translateY(-1px); }
.ub-btn:active:not(:disabled) { transform: translateY(0); }
.ub-btn:disabled { opacity: 0.55; cursor: not-allowed; }
.ub-btn .material-symbols-outlined { font-size: 18px; }

.ub-expand {
  background: color-mix(in oklch, var(--color-on-primary) 22%, transparent);
  color: var(--color-on-primary);
}

.ub-stop {
  background: var(--color-on-primary);
  color: var(--color-primary);
}

/* На узких экранах прячем вторичный текст и подписи кнопок — остаётся
   таймер, точка записи и круглые иконки. */
@media (max-width: 640px) {
  .unit-banner { gap: 8px; padding: 9px 12px; }
  .ub-task { display: none; }
  .ub-btn-label { display: none; }
  .ub-btn { padding: 9px; }
  .ub-timer { font-size: 17px; }
}
</style>
