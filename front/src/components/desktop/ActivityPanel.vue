<template>
  <aside class="ap" aria-label="Последние действия">
    <header class="ap-head">
      <span class="ap-title">Последние действия</span>
      <button
        v-if="activity.items.length || recentSections.length"
        class="ap-clear"
        type="button"
        title="Очистить ленту"
        @click="activity.clear()"
      >
        <span class="material-symbols-outlined">delete_sweep</span>
      </button>
    </header>

    <!-- Куда пользователь заходил последним: быстрый возврат в раздел. -->
    <div v-if="recentSections.length" class="ap-sections">
      <button
        v-for="s in recentSections"
        :key="s.id"
        class="ap-chip"
        type="button"
        :title="`Открыть: ${s.title}`"
        @click="emit('open', s.path)"
      >
        <span class="material-symbols-outlined">{{ s.icon }}</span>
        <span class="ap-chip-label">{{ s.title }}</span>
      </button>
    </div>

    <div class="ap-list">
      <button
        v-for="item in activity.items"
        :key="item.key"
        class="ap-item"
        type="button"
        :title="item.title"
        @click="emit('open', item.path)"
      >
        <span class="ap-item-icon material-symbols-outlined">{{ iconOf(item) }}</span>
        <span class="ap-item-text">
          <span class="ap-item-title">{{ item.title || sectionTitle(item) }}</span>
          <span class="ap-item-sub">{{ subtitleOf(item) }}</span>
        </span>
      </button>

      <p v-if="!activity.items.length" class="ap-empty">
        Здесь появятся ваши последние действия: созданные задачи, заметки и записи.
        Из ленты можно вернуться к любому из них.
      </p>
    </div>
  </aside>
</template>

<script setup>
import { computed } from 'vue'
import { useActivityStore, ACTIONS } from '@/stores/activity.js'
import { appById } from '@/desktop/apps.js'
import { timeAgo } from '@/utils/time.js'

const emit = defineEmits(['open'])

const activity = useActivityStore()

/* Недавно открытые разделы: id из журнала, всё остальное — из реестра
   приложений. Раздел, которого у пользователя больше нет (сменилась компания,
   отключилась фича), из строки просто выпадает. */
const recentSections = computed(() => activity.sections
  .map((s) => {
    const app = appById(s.id)
    return app ? { id: s.id, title: app.title, icon: app.icon, path: app.path } : null
  })
  .filter(Boolean))

// Значок и название раздела берём из реестра приложений — второго списка
// разделов на фронте быть не должно.
const iconOf = (item) => appById(item.section)?.icon || 'history'
const sectionTitle = (item) => appById(item.section)?.title || 'Раздел'

function subtitleOf(item) {
  return `${ACTIONS[item.action] || ACTIONS.created} · ${sectionTitle(item)} · ${timeAgo(item.at)}`
}
</script>

<style scoped>
.ap {
  display: flex;
  flex-direction: column;
  min-height: 0;
  gap: 8px;
  padding-left: 16px;
  border-left: 1px solid var(--acrylic-border);
}

.ap-head {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-shrink: 0;
  height: 30px;
}

.ap-title {
  flex: 1;
  min-width: 0;
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0.4px;
  text-transform: uppercase;
  color: var(--color-text-dim);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ap-clear {
  width: 28px;
  min-width: 28px;
  max-width: 28px;
  height: 28px;
  min-height: 28px;
  max-height: 28px;
  display: grid;
  place-items: center;
  border: none;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--color-text-dim);
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
}

.ap-clear:hover { background: color-mix(in oklch, var(--color-primary) 12%, transparent); color: var(--color-primary); }
.ap-clear .material-symbols-outlined { font-size: 18px; }

/* Недавние разделы — компактные чипы, лента ниже остаётся главной. */
.ap-sections {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  flex-shrink: 0;
  padding-bottom: 8px;
  border-bottom: 1px solid color-mix(in oklch, var(--acrylic-border) 70%, transparent);
}

.ap-chip {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  max-width: 100%;
  height: 26px;
  min-height: 26px;
  padding: 0 9px;
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-full);
  background: var(--glass-bg);
  color: var(--color-text-dim);
  font-size: 11.5px;
  cursor: pointer;
  transition: color 0.12s, border-color 0.12s;
}

.ap-chip:hover {
  color: var(--color-primary);
  border-color: color-mix(in oklch, var(--color-primary) 32%, var(--acrylic-border));
}

.ap-chip .material-symbols-outlined { font-size: 15px; }

.ap-chip-label {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ap-list {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 4px;
  scrollbar-width: thin;
}

.ap-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 10px;
  border: none;
  border-radius: var(--radius-md);
  background: transparent;
  text-align: left;
  cursor: pointer;
  transition: background 0.12s;
}

.ap-item:hover { background: color-mix(in oklch, var(--color-primary) 8%, transparent); }

.ap-item-icon {
  font-size: 20px;
  flex-shrink: 0;
  color: var(--color-text-dim);
}

.ap-item:hover .ap-item-icon { color: var(--color-primary); }

.ap-item-text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
}

.ap-item-title {
  font-size: 13px;
  font-weight: 500;
  color: var(--color-text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ap-item-sub {
  font-size: 11px;
  color: var(--color-text-dim);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ap-empty {
  margin: 8px 4px 0;
  font-size: 12px;
  line-height: 1.45;
  color: var(--color-text-dim);
}
</style>
