<template>
  <AppDialog
    :model-value="modelValue"
    tone="tertiary"
    icon="task"
    size="md"
    title="Прикрепить задачу"
    subtitle="Можно прикрепить только задачу из той же компании, что и диалог."
    @update:model-value="$emit('update:modelValue', $event)"
  >
    <div class="attach-task">
      <div class="attach-task-search">
        <span class="material-symbols-outlined">search</span>
        <input
          v-model="q"
          placeholder="Название задачи"
          class="attach-task-input"
          autofocus
        />
      </div>
      <div v-if="loading" class="attach-task-empty">
        <BrandLoader :size="48" />
      </div>
      <div v-else-if="!results.length" class="attach-task-empty">
        <span class="material-symbols-outlined">task</span>
        <p>{{ q ? 'Ничего не нашли' : 'Введите название задачи' }}</p>
      </div>
      <ul v-else class="attach-task-results">
        <li
          v-for="t in results"
          :key="t.id"
          class="attach-task-item"
          :style="cardStyle(t)"
          @click="pick(t)"
        >
          <div class="task-color-strip" :style="stripStyle(t)" />
          <div class="task-info">
            <div class="task-name">{{ t.name }}</div>
            <div class="task-meta">
              <span v-if="t.responsible_fio">
                <span class="material-symbols-outlined">person</span>
                {{ t.responsible_fio }}
              </span>
              <span v-if="t.is_archived" class="archived">
                <span class="material-symbols-outlined">inventory_2</span>
                в архиве
              </span>
            </div>
          </div>
        </li>
      </ul>
    </div>
  </AppDialog>
</template>

<script setup>
import { ref, watch } from 'vue'
import BrandLoader from '@/components/common/BrandLoader.vue'
import AppDialog from '@/components/common/AppDialog.vue'
import { getTasks } from '@/api/tasks.js'
import { TASK_COLOR_IDS } from '@/utils/taskColors.js'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  // company_id диалога — задачу можно прикрепить только из той же компании,
  // что и сам диалог (бизнес-правило бэка). Для Администратора системы
  // явный company_id перебивает выбранную в селекторе компанию.
  companyId: { type: Number, default: null },
})

const emit = defineEmits(['update:modelValue', 'pick'])

const q = ref('')
const results = ref([])
const loading = ref(false)
let debounceTimer = null

async function search() {
  loading.value = true
  try {
    const params = { search: q.value.trim(), per_page: 20 }
    if (props.companyId != null) params.company_id = props.companyId
    const data = await getTasks(params)
    // Сервер отдаёт пагинированный ответ; на разных страницах формат
    // отличается, поддержим оба.
    results.value = Array.isArray(data) ? data : (data.items || [])
  } catch {
    results.value = []
  } finally {
    loading.value = false
  }
}

watch(() => props.modelValue, (v) => {
  if (v) {
    q.value = ''
    search()
  }
})

watch(q, () => {
  clearTimeout(debounceTimer)
  debounceTimer = setTimeout(search, 250)
})

function cardStyle(t) {
  if (!t.color || !TASK_COLOR_IDS.includes(t.color)) return {}
  return {
    '--_card-bg': `var(--tag-${t.color}-surface)`,
    '--_card-border': `var(--tag-${t.color}-border)`,
  }
}

function stripStyle(t) {
  if (t.color && TASK_COLOR_IDS.includes(t.color)) {
    return { background: `var(--tag-${t.color}-accent)` }
  }
  return { background: 'var(--color-outline-dim)' }
}

function pick(t) {
  emit('pick', t)
  emit('update:modelValue', false)
}
</script>

<style scoped>
/* Контейнер модалки: фикс. высота с flex-row layout — строка поиска
   закреплена сверху, список задач скроллится отдельно. */
.attach-task {
  display: flex;
  flex-direction: column;
  max-height: 70dvh;
  min-height: 320px;
}

.attach-task-search {
  position: relative;
  display: flex;
  align-items: center;
  margin-bottom: 12px;
  flex-shrink: 0;
}

.attach-task-search .material-symbols-outlined {
  position: absolute;
  left: 12px;
  color: var(--color-text-dim);
  font-size: 20px;
  pointer-events: none;
}

.attach-task-input {
  width: 100%;
  padding: 10px 12px 10px 40px;
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-md);
  background: var(--color-surface-low);
  color: var(--color-text);
  font: inherit;
  font-size: 14px;
  outline: none;
}

.attach-task-input:focus { border-color: var(--color-primary); }

.attach-task-results {
  list-style: none;
  padding: 0 4px 4px 0;
  margin: 0;
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.attach-task-item {
  /* flex-shrink: 0 обязателен — иначе как flex-дети внутри scroll-контейнера
     элементы сжимаются по высоте, лишь бы все вместить, и скролл не появляется. */
  flex-shrink: 0;
  display: flex;
  gap: 0;
  align-items: stretch;
  cursor: pointer;
  border-radius: var(--radius-md);
  background: var(--_card-bg, var(--color-surface-low));
  border: 1px solid var(--_card-border, var(--color-outline-dim));
  overflow: hidden;
  transition: transform 0.12s, box-shadow 0.15s;
}

.attach-task-item:hover {
  transform: translateY(-1px);
  box-shadow: var(--shadow-sm);
}

.task-color-strip {
  width: 4px;
  flex-shrink: 0;
}

.task-info {
  padding: 10px 12px;
  flex: 1;
  min-width: 0;
}

.task-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--color-text);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.task-meta {
  margin-top: 4px;
  display: flex;
  gap: 10px;
  font-size: 12px;
  color: var(--color-text-dim);
  align-items: center;
  flex-wrap: wrap;
}

.task-meta span {
  display: inline-flex;
  align-items: center;
  gap: 4px;
}

.task-meta .material-symbols-outlined {
  font-size: 14px;
}

.task-meta .archived {
  color: var(--color-warning);
}

.attach-task-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  padding: 28px 16px;
  color: var(--color-text-dim);
}

.attach-task-empty .material-symbols-outlined { font-size: 40px; }
</style>
