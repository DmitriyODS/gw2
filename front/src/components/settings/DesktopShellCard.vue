<template>
  <!-- Каркас рабочего стола: где висит панель задач и как открывается меню
       «Пуск». Настройка личная и синхронизируется между устройствами. -->
  <AppCard
    title="Панель задач и меню «Пуск»"
    hint="Панель можно прижать к любому краю экрана — окна сами подстроятся под свободное место."
  >
    <div class="ds-sides">
      <button
        v-for="side in SIDES"
        :key="side.key"
        class="ds-side"
        :class="{ 'is-active': prefs.taskbarSide === side.key }"
        type="button"
        @click="prefs.setTaskbarSide(side.key)"
      >
        <span class="ds-preview" :data-side="side.key"><i /></span>
        <span>{{ side.label }}</span>
      </button>
    </div>

    <AppSwitchRow
      :model-value="prefs.startFullscreen"
      title="Меню «Пуск» во весь экран"
      hint="Иначе меню открывается панелью, а развернуть его можно кнопкой в его шапке."
      @update:model-value="prefs.setStartFullscreen"
    />
  </AppCard>
</template>

<script setup>
import { useDesktopPrefsStore } from '@/stores/desktopPrefs.js'
import AppCard from '@/components/ui/AppCard.vue'
import AppSwitchRow from '@/components/ui/AppSwitchRow.vue'

const prefs = useDesktopPrefsStore()

const SIDES = [
  { key: 'bottom', label: 'Снизу' },
  { key: 'top', label: 'Сверху' },
  { key: 'left', label: 'Слева' },
  { key: 'right', label: 'Справа' },
]
</script>

<style scoped>
.ds-sides {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 10px;
}

.ds-side {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  padding: 12px 8px;
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-md);
  background: var(--color-surface-variant);
  color: var(--color-text);
  font-size: 0.82rem;
  font-weight: 600;
  cursor: pointer;
  transition: border-color 0.2s ease, background 0.2s ease;
}

.ds-side.is-active {
  border-color: var(--color-primary);
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
}

/* Схема экрана: рамка с полоской панели у выбранного края. */
.ds-preview {
  position: relative;
  width: 100%;
  height: 46px;
  border: 1px solid var(--color-outline, var(--acrylic-border));
  border-radius: 6px;
  background: var(--color-surface);
  overflow: hidden;
}

.ds-preview i {
  position: absolute;
  border-radius: 3px;
  background: var(--color-primary);
}

.ds-preview[data-side='bottom'] i { left: 20%; right: 20%; bottom: 4px; height: 7px; }
.ds-preview[data-side='top'] i { left: 20%; right: 20%; top: 4px; height: 7px; }
.ds-preview[data-side='left'] i { top: 20%; bottom: 20%; left: 4px; width: 7px; }
.ds-preview[data-side='right'] i { top: 20%; bottom: 20%; right: 4px; width: 7px; }

@media (max-width: 560px) {
  .ds-sides { grid-template-columns: repeat(2, minmax(0, 1fr)); }
}
</style>
