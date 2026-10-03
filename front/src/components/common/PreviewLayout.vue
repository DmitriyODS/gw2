<template>
  <!-- Раскладка «настройки + миниатюра» (supporting pane Material 3): широко —
       настройки слева, миниатюра справа и прилипает к краю прокрутки; узко —
       миниатюра сверху и тоже прилипает. Результат правки всегда на виду, какую
       бы настройку ни крутили. Ширина считается от САМОЙ раскладки: она живёт
       и в панели настроек, и в диалоге, и в узком окне. -->
  <div class="pl">
    <div class="pl-grid">
      <aside class="pl-aside">
        <div class="pl-preview"><slot name="preview" /></div>
        <div v-if="$slots.actions" class="pl-actions"><slot name="actions" /></div>
      </aside>
      <div class="pl-main"><slot /></div>
    </div>
  </div>
</template>

<style scoped>
.pl { container-type: inline-size; }

.pl-grid {
  display: flex;
  flex-direction: column;
  gap: 18px;
}

/* Узко миниатюра не растёт выше разумного — иначе на телефоне она заняла бы
   пол-экрана; подложка отделяет её от прокручиваемого под ней содержимого. */
.pl-aside {
  position: sticky;
  top: 0;
  z-index: 2;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
  padding: 8px;
  border-radius: var(--radius-xl);
  background: var(--window-bg);
  box-shadow: var(--shadow-sm);
}

.pl-preview { width: 100%; max-width: 360px; }

.pl-actions {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 8px;
}

.pl-main {
  display: flex;
  flex-direction: column;
  gap: 22px;
  min-width: 0;
}

@container (min-width: 720px) {
  .pl-grid {
    display: grid;
    grid-template-columns: minmax(0, 1fr) minmax(260px, 42%);
    align-items: start;
    gap: 24px;
  }

  .pl-aside {
    order: 2;
    align-items: stretch;
    padding: 0;
    background: none;
    box-shadow: none;
  }

  .pl-preview { max-width: none; }
  .pl-actions { justify-content: flex-end; }
}
</style>
