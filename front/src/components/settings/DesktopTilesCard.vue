<template>
  <div class="dt-card">
    <header class="dt-head">
      <span class="dt-icon material-symbols-outlined">grid_view</span>
      <div class="dt-head-text">
        <h3>Живые плитки</h3>
        <p>
          Плитки меню «Пуск» по очереди показывают сводку своего раздела:
          дела на сегодня, ближайшие события, непрочитанное. Настройка личная и
          синхронизируется на всех ваших устройствах.
        </p>
      </div>
    </header>

    <SwitchRow
      :model-value="prefs.liveTiles"
      title="Показывать сводки на плитках"
      hint="Выключено — плитки остаются обычными значками, данные не запрашиваются."
      icon="animation"
      @update:model-value="prefs.setLiveTiles"
    />
  </div>
</template>

<script setup>
import { useDesktopPrefsStore } from '@/stores/desktopPrefs.js'
import SwitchRow from '@/components/common/SwitchRow.vue'

const prefs = useDesktopPrefsStore()
</script>

<style scoped>
.dt-card {
  display: flex;
  flex-direction: column;
  gap: 14px;
  padding: 18px;
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-lg);
  background: var(--acrylic-card-bg);
}

.dt-head {
  display: flex;
  align-items: flex-start;
  gap: 12px;
}

.dt-icon {
  display: grid;
  place-items: center;
  width: 40px;
  height: 40px;
  flex: none;
  border-radius: var(--radius-md);
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
  font-size: 22px;
}

.dt-head-text h3 {
  margin: 0 0 4px;
  font-size: 16px;
  font-weight: 650;
  color: var(--color-text);
}

.dt-head-text p {
  margin: 0;
  font-size: 13px;
  line-height: 1.45;
  color: var(--color-text-dim);
}
</style>
