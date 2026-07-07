<template>
  <AppDialog
    :model-value="modelValue"
    tone="primary"
    :icon="mode === 'form' ? (editing ? 'edit' : 'add_circle') : 'tune'"
    size="md"
    :title="mode === 'form' ? (editing ? 'Изменить раздел' : 'Новый раздел') : 'Разделы портала'"
    :subtitle="mode === 'form' ? '' : subtitleText"
    @update:model-value="onDialogToggle"
  >
    <!-- Режим «список» -->
    <template v-if="mode === 'list'">
      <div v-if="portal.loadingTopics" class="topic-status">
        <ProgressSpinner style="width:28px;height:28px" />
      </div>

      <EmptyState
        v-else-if="!portal.topics.length"
        icon="label"
        title="Разделов пока нет"
        :subtitle="canManage ? 'Создайте первый — посты можно будет группировать по темам' : 'Администратор компании ещё не создал разделы'"
      />

      <div v-else class="topic-list">
        <component
          :is="canManage ? 'button' : 'div'"
          v-for="t in portal.topics"
          :key="t.id"
          class="topic-row"
          :type="canManage ? 'button' : undefined"
          :aria-label="canManage ? `Изменить раздел «${t.name}»` : undefined"
          @click="canManage && openForm(t)"
        >
          <span class="topic-icon" :style="colorStyle(t.color)">
            <span class="material-symbols-outlined">{{ t.icon || 'label' }}</span>
          </span>
          <span class="topic-name">{{ t.name }}</span>
          <template v-if="canManage">
            <span class="topic-edit-hint material-symbols-outlined" aria-hidden="true">edit</span>
            <button
              class="topic-icon-btn danger"
              type="button"
              :aria-label="`Удалить раздел «${t.name}»`"
              @click.stop="confirmDelete(t)"
            >
              <span class="material-symbols-outlined">delete</span>
            </button>
          </template>
        </component>
      </div>

      <button v-if="canManage" class="topic-add-btn" type="button" @click="openForm(null)">
        <span class="material-symbols-outlined">add</span>
        Новый раздел
      </button>
    </template>

    <!-- Режим «форма» (создание/редактирование) -->
    <form v-else class="topic-form" @submit.prevent="submit">
      <div class="topic-preview">
        <span class="topic-icon lg" :style="colorStyle(color)">
          <span class="material-symbols-outlined">{{ icon }}</span>
        </span>
        <input
          ref="nameInput"
          v-model="name"
          class="topic-input"
          placeholder="Название раздела"
          maxlength="60"
        />
      </div>

      <div class="topic-group">
        <span class="topic-group-label">Иконка</span>
        <div class="topic-icons">
          <button
            v-for="ic in ICONS"
            :key="ic.key"
            type="button"
            class="topic-icon-pick"
            :class="{ active: icon === ic.key }"
            :title="ic.label"
            :aria-label="ic.label"
            :aria-pressed="icon === ic.key"
            @click="icon = ic.key"
          >
            <span class="material-symbols-outlined">{{ ic.key }}</span>
          </button>
        </div>
      </div>

      <div class="topic-group">
        <span class="topic-group-label">Цвет</span>
        <div class="topic-colors">
          <button
            type="button"
            class="topic-color-pick none"
            :class="{ active: color === null }"
            title="Без цвета"
            aria-label="Без цвета"
            :aria-pressed="color === null"
            @click="color = null"
          >
            <span class="material-symbols-outlined">format_color_reset</span>
          </button>
          <button
            v-for="c in TASK_COLORS"
            :key="c.id"
            type="button"
            class="topic-color-pick"
            :class="{ active: color === c.id }"
            :style="{ background: `var(--tag-${c.id}-accent)` }"
            :title="c.label"
            :aria-label="c.label"
            :aria-pressed="color === c.id"
            @click="color = c.id"
          />
        </div>
      </div>

      <div class="topic-form-actions">
        <button type="button" class="topic-btn-text" @click="backToList">
          <span class="material-symbols-outlined">arrow_back</span> Назад
        </button>
        <button type="submit" class="topic-btn-primary" :disabled="!name.trim() || saving">
          {{ editing ? 'Сохранить' : 'Создать' }}
        </button>
      </div>
    </form>

    <AppDialog
      v-model="deleteOpen"
      tone="danger"
      icon="delete"
      size="sm"
      title="Удалить раздел?"
      subtitle="Посты раздела останутся, но потеряют привязку к нему."
      :actions="[{ kind: 'cancel', label: 'Отмена' }, { kind: 'confirm', label: 'Удалить', icon: 'delete' }]"
      @confirm="doDelete"
    />
  </AppDialog>
</template>

<script setup>
import { computed, nextTick, ref, watch } from 'vue'
import ProgressSpinner from 'primevue/progressspinner'
import AppDialog from '@/components/common/AppDialog.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import { usePortalStore } from '@/stores/portal.js'
import { usePermission } from '@/composables/usePermission.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { TASK_COLORS } from '@/utils/taskColors.js'

const props = defineProps({ modelValue: { type: Boolean, default: false } })
const emit = defineEmits(['update:modelValue'])

const portal = usePortalStore()
const { isAdmin } = usePermission()
const canManage = computed(() => isAdmin())

const subtitleText = computed(() => (canManage.value
  ? 'Тематические разделы для постов — тап по разделу, чтобы изменить'
  : 'Тематические разделы для постов — их ведёт администратор компании'))

const ICONS = [
  { key: 'campaign', label: 'Объявления' },
  { key: 'celebration', label: 'Праздники' },
  { key: 'groups', label: 'Команда' },
  { key: 'work', label: 'Работа' },
  { key: 'event', label: 'События' },
  { key: 'info', label: 'Информация' },
  { key: 'emoji_events', label: 'Достижения' },
  { key: 'volunteer_activism', label: 'Благодарности' },
]

const mode = ref('list')
const editing = ref(null)
const name = ref('')
const icon = ref(ICONS[0].key)
const color = ref(null) // null — раздел без цвета
const saving = ref(false)
const nameInput = ref(null)

// Каждое открытие диалога начинается со списка, не с прошлого состояния.
watch(() => props.modelValue, (open) => {
  if (open) backToList()
})

function onDialogToggle(v) {
  emit('update:modelValue', v)
}

function colorStyle(c) {
  return c ? { background: `var(--tag-${c}-surface)`, color: `var(--tag-${c}-accent)` } : {}
}

function openForm(t) {
  editing.value = t
  name.value = t?.name || ''
  icon.value = t?.icon || ICONS[0].key
  color.value = t?.color || null
  mode.value = 'form'
  nextTick(() => nameInput.value?.focus())
}

function backToList() {
  mode.value = 'list'
  editing.value = null
}

async function submit() {
  const n = name.value.trim()
  if (!n) return
  saving.value = true
  try {
    if (editing.value) await portal.updateTopic(editing.value.id, { name: n, color: color.value, icon: icon.value })
    else await portal.createTopic({ name: n, color: color.value, icon: icon.value })
    backToList()
  } catch (e) {
    useNotificationsStore().error(e?.message || 'Не удалось сохранить раздел')
  } finally {
    saving.value = false
  }
}

const deleteOpen = ref(false)
const deletingTopic = ref(null)

function confirmDelete(t) {
  deletingTopic.value = t
  deleteOpen.value = true
}

async function doDelete() {
  try {
    await portal.deleteTopic(deletingTopic.value.id)
  } catch (e) {
    useNotificationsStore().error(e?.message || 'Не удалось удалить раздел')
  } finally {
    deleteOpen.value = false
  }
}
</script>

<style scoped>
.topic-status { display: flex; justify-content: center; padding: 20px 0; }

.topic-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.topic-row {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  padding: 8px;
  min-height: 48px;
  border: none;
  border-radius: var(--radius-md);
  background: transparent;
  font: inherit;
  color: var(--color-text);
  text-align: left;
}
button.topic-row { cursor: pointer; }
button.topic-row:hover { background: var(--color-surface-low); }
button.topic-row:hover .topic-edit-hint { opacity: 1; }

.topic-icon {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  display: grid;
  place-items: center;
  background: var(--color-surface-high);
  flex-shrink: 0;
}
.topic-icon .material-symbols-outlined { font-size: 19px; }
.topic-icon.lg { width: 44px; height: 44px; }
.topic-icon.lg .material-symbols-outlined { font-size: 22px; }

.topic-name { flex: 1; min-width: 0; font-size: 14px; font-weight: 600; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }

.topic-edit-hint {
  font-size: 17px;
  color: var(--color-text-dim);
  opacity: 0;
  transition: opacity 0.15s;
}

.topic-icon-btn {
  width: 40px;
  height: 40px;
  border: none;
  border-radius: 50%;
  background: transparent;
  color: var(--color-text-dim);
  cursor: pointer;
  display: grid;
  place-items: center;
  flex-shrink: 0;
}
.topic-icon-btn:hover { background: var(--color-surface-high); color: var(--color-text); }
.topic-icon-btn.danger:hover { color: var(--color-error); }
.topic-icon-btn .material-symbols-outlined { font-size: 19px; }

.topic-add-btn {
  margin-top: 12px;
  margin-bottom: 10px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  width: 100%;
  min-height: 44px;
  border: 1px dashed var(--color-outline-dim);
  border-radius: var(--radius-md);
  background: transparent;
  color: var(--color-primary);
  font: inherit;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
}
.topic-add-btn:hover { background: var(--color-surface-low); border-color: var(--color-primary); }
.topic-add-btn .material-symbols-outlined { font-size: 19px; }

/* ── Форма ── */
.topic-form { display: flex; flex-direction: column; gap: 16px; padding-bottom: 10px; }

.topic-preview { display: flex; align-items: center; gap: 12px; }

.topic-input {
  flex: 1;
  min-width: 0;
  padding: 11px 14px;
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-md);
  background: var(--color-surface);
  color: var(--color-text);
  font: inherit;
  font-size: 15px;
  outline: none;
  box-sizing: border-box;
}
.topic-input:focus { border-color: var(--color-primary); }

.topic-group { display: flex; flex-direction: column; gap: 8px; }
.topic-group-label { font-size: 12px; font-weight: 700; color: var(--color-text-dim); text-transform: uppercase; letter-spacing: 0.03em; }

.topic-icons {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.topic-icon-pick {
  width: 44px;
  height: 44px;
  border-radius: 50%;
  border: 1.5px solid var(--color-outline-dim);
  background: var(--color-surface);
  color: var(--color-text-dim);
  cursor: pointer;
  display: grid;
  place-items: center;
}
.topic-icon-pick.active { border-color: var(--color-primary); background: var(--color-primary-container); color: var(--color-on-primary-container); }
.topic-icon-pick .material-symbols-outlined { font-size: 20px; }
.topic-icon-pick:focus-visible { outline: 2px solid var(--color-primary); outline-offset: 2px; }

.topic-colors {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.topic-color-pick {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  border: 2px solid transparent;
  cursor: pointer;
  display: grid;
  place-items: center;
  padding: 0;
}
.topic-color-pick.active { border-color: var(--color-text); }
.topic-color-pick.none {
  background: var(--color-surface);
  border-color: var(--color-outline-dim);
  color: var(--color-text-dim);
}
.topic-color-pick.none.active { border-color: var(--color-text); color: var(--color-text); }
.topic-color-pick.none .material-symbols-outlined { font-size: 17px; }
.topic-color-pick:focus-visible { outline: 2px solid var(--color-primary); outline-offset: 2px; }

.topic-form-actions {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 8px;
  margin-top: 4px;
}

.topic-btn-text {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  border: none;
  background: transparent;
  color: var(--color-primary);
  font: inherit;
  font-size: 13.5px;
  font-weight: 600;
  cursor: pointer;
  padding: 10px 12px;
  border-radius: var(--radius-full);
}
.topic-btn-text:hover { background: var(--color-surface-low); }
.topic-btn-text .material-symbols-outlined { font-size: 17px; }

.topic-btn-primary {
  border: none;
  border-radius: var(--radius-full);
  background: var(--color-primary);
  color: var(--color-on-primary);
  font: inherit;
  font-size: 13.5px;
  font-weight: 700;
  cursor: pointer;
  padding: 11px 22px;
}
.topic-btn-primary:disabled { opacity: 0.55; cursor: not-allowed; }
</style>
