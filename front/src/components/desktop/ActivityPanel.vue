<template>
  <aside class="ap" aria-label="Моя активность">
    <header class="ap-head">
      <h2 class="ap-title">Моя активность</h2>
      <button
        v-if="items.length"
        class="ap-clear"
        type="button"
        title="Очистить ленту"
        @click="activity.clear()"
      >
        <span class="material-symbols-outlined">delete_sweep</span>
      </button>
    </header>

    <div v-if="items.length" class="ap-list">
      <!-- Открытые разделы идут одним потоком с созданными элементами: лента
           показывает всю работу за сеанс, а не только то, что было создано. -->
      <button
        v-for="item in items"
        :key="item.key"
        class="ap-item"
        type="button"
        :title="`${item.label} · ${fullTime(item.at)}`"
        @click="emit('open', item.path)"
      >
        <span class="ap-item-icon">
          <span class="material-symbols-outlined">{{ item.icon }}</span>
        </span>
        <span class="ap-item-text">
          <span class="ap-item-title">{{ item.label }}</span>
          <span class="ap-item-sub">{{ item.sub }}</span>
        </span>
      </button>
    </div>

    <p v-else class="ap-empty">здесь будут отображаться ваши последние действия в сервисе</p>
  </aside>
</template>

<script setup>
import { computed } from 'vue'
import { useActivityStore, ACTIONS } from '@/stores/activity.js'
import { appById } from '@/desktop/apps.js'
import { timeAgo, fullTime } from '@/utils/time.js'

const emit = defineEmits(['open'])

const activity = useActivityStore()

/* Значок, название и путь раздела берём из реестра приложений — второго списка
   разделов на фронте быть не должно. Событие раздела, которого у пользователя
   больше нет (сменилась компания, отключилась фича), из ленты просто выпадает. */
const items = computed(() => activity.items
  .map((item) => {
    const app = appById(item.section)
    if (!app) return null
    const opened = item.action === 'opened'
    return {
      key: item.key,
      icon: app.icon,
      path: item.path || app.path,
      label: opened ? app.title : (item.title || app.title),
      sub: opened
        ? `${ACTIONS.opened} · ${timeAgo(item.at)}`
        : `${ACTIONS[item.action] || ACTIONS.created} · ${app.title} · ${timeAgo(item.at)}`,
      at: item.at,
    }
  })
  .filter(Boolean))
</script>

<style scoped>
/* Лента — самостоятельная стеклянная панель рядом с плитками. */
.ap {
  display: flex;
  flex-direction: column;
  min-height: 0;
  gap: 10px;
  padding: 16px 12px 12px;
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-xl);
  background: var(--glass-bg), var(--acrylic-card-bg);
  box-shadow: var(--glass-edge);
}

.ap-head {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-shrink: 0;
  padding: 0 4px;
}

.ap-title {
  flex: 1;
  min-width: 0;
  margin: 0;
  font-size: 16px;
  font-weight: 700;
  letter-spacing: -0.2px;
  color: var(--color-text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ap-clear {
  width: 30px;
  min-width: 30px;
  max-width: 30px;
  height: 30px;
  min-height: 30px;
  max-height: 30px;
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
.ap-clear .material-symbols-outlined { font-size: 20px; }

.ap-list {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 4px;
  /* Полоса прокрутки — в своём жёлобе справа, а не поверх строк. */
  scrollbar-gutter: stable;
  padding-right: 4px;
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
  display: grid;
  place-items: center;
  flex-shrink: 0;
  width: 32px;
  height: 32px;
  border-radius: var(--radius-md);
  background: color-mix(in oklch, var(--color-primary) 10%, transparent);
  color: var(--color-text-dim);
  transition: color 0.12s;
}

.ap-item:hover .ap-item-icon { color: var(--color-primary); }
.ap-item-icon .material-symbols-outlined { font-size: 19px; }

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
  flex: 1;
  display: grid;
  place-items: center;
  margin: 0;
  padding: 0 18px;
  font-size: 13px;
  line-height: 1.5;
  text-align: center;
  color: var(--color-text-dim);
}
</style>
