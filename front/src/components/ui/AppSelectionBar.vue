<template>
  <Transition name="selbar">
    <div v-if="count > 0" class="selbar" role="status">
      <span class="selbar-count">
        <span class="material-symbols-outlined">checklist</span>
        <span class="selbar-word">Выбрано:</span> <strong>{{ count }}</strong>
      </span>

      <span class="selbar-tools">
        <AppButton
          v-if="canSelectAll"
          size="sm"
          variant="text"
          :label="`Выбрать все (${total})`"
          @click="$emit('select-all')"
        />
        <slot />
        <AppButton
          variant="icon"
          size="sm"
          icon="close"
          title="Очистить выбор"
          aria-label="Очистить выбор"
          @click="$emit('clear')"
        />
      </span>
    </div>
  </Transition>
</template>

<script setup>
/* Плашка массового выбора: всплывает над списком, когда что-то отмечено, и
   держится при переходе по страницам — выбор их переживает.

   Своих действий у неё нет: удаление, выгрузка и прочее приходят слотом от
   раздела, а плашка отвечает лишь за счётчик, «выбрать всё» и сброс. */
import { computed } from 'vue'
import AppButton from './AppButton.vue'

const props = defineProps({
  /** Сколько записей отмечено. */
  count: { type: Number, default: 0 },
  /** Сколько всего по текущему фильтру — предложение «выбрать все». */
  total: { type: Number, default: 0 },
  /** Уже выбрано всё (или раздел не умеет выбирать за пределами страницы). */
  allSelected: { type: Boolean, default: false },
})
defineEmits(['select-all', 'clear'])

const canSelectAll = computed(() => !props.allSelected && props.total > props.count)
</script>

<style scoped>
.selbar {
  position: absolute;
  left: 50%;
  bottom: 16px;
  transform: translateX(-50%);
  z-index: 5;
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  /* Счётчик и действия — по противоположным краям пилюли, а не кучей слева. */
  justify-content: space-between;
  gap: 10px;
  max-width: min(680px, calc(100% - 28px));
  padding: 6px 6px 6px 16px;
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-full);
  /* Плашка лежит ВНУТРИ акриловой панели раздела, а та — backdrop root:
     настоящий backdrop-filter здесь размывать нечего, и плашка выглядела
     просто прозрачной. Поэтому матовость — «иней» (--glass-bg поверх плотной
     подложки) плюс blur для случая, когда панели-акрила над нами нет.
     -webkit-префикс идёт ПЕРЕД стандартным — минификатор иначе выбрасывает
     стандартное свойство. */
  background: var(--color-surface-high);
  background: var(--glass-bg), var(--acrylic-bg-strong);
  -webkit-backdrop-filter: var(--acrylic-blur);
  backdrop-filter: var(--acrylic-blur);
  box-shadow: var(--glass-edge), var(--shadow-lg, var(--shadow-md));
  color: var(--color-text);
}

.selbar-count {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  white-space: nowrap;
}
.selbar-count .material-symbols-outlined { font-size: 18px; color: var(--color-primary); }

.selbar-tools { display: inline-flex; align-items: center; gap: 6px; }

/* Выезжает снизу — плашка принадлежит списку, а не экрану. */
.selbar-enter-active,
.selbar-leave-active { transition: opacity 0.16s ease, transform 0.16s ease; }
.selbar-enter-from,
.selbar-leave-to { opacity: 0; transform: translate(-50%, 12px); }

/* Телефон: места мало — плашка идёт по ширине экрана, компактнее и без
   слова «Выбрано» (иконка и число говорят сами за себя). */
@media (max-width: 560px) {
  .selbar {
    left: 8px;
    right: 8px;
    bottom: 8px;
    transform: none;
    max-width: none;
    gap: 6px;
    padding: 4px 4px 4px 12px;
  }
  .selbar-word { display: none; }
  .selbar-count { font-size: 12.5px; }
  .selbar-enter-from,
  .selbar-leave-to { transform: translateY(12px); }
}
</style>
