<script setup>
/* Раздел «Напоминания»: что ждёт впереди и что уже сработало. Срок считает
   сервер (планировщик remindersvc), здесь — только управление: создать,
   отложить, отметить сделанным, удалить. */
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import BrandLoader from '@/components/common/BrandLoader.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import ReminderEditDialog from '@/components/reminders/ReminderEditDialog.vue'
import { useRemindersStore } from '@/stores/reminders.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { humanWhen } from '@/utils/naturalDate.js'

const store = useRemindersStore()
const notify = useNotificationsStore()
const route = useRoute()
const router = useRouter()

const tab = ref('active')
const editOpen = ref(false)
const editing = ref(null)
const presetTitle = ref('')
const confirmTarget = ref(null)

const list = computed(() => (tab.value === 'active' ? store.items : store.done))

const REPEAT_LABELS = {
  none: '', daily: 'каждый день', weekdays: 'по рабочим дням',
  weekly: 'по дням недели', monthly: 'каждый месяц', yearly: 'каждый год',
}

const WEEKDAY_SHORT = ['', 'пн', 'вт', 'ср', 'чт', 'пт', 'сб', 'вс']

onMounted(() => {
  store.fetchAll()
  consumeCreateQuery()
})

/* Форма создания с готовым названием: `/reminders?new=1&title=…` — команда
   «создай напоминание …» из строки поиска рабочего стола, когда срок в фразе
   не назвали. Роут уже открыт — компонент не пересоздаётся, поэтому следим и
   за самим query. */
function consumeCreateQuery() {
  if (!route.query.new) return
  presetTitle.value = String(route.query.title || '')
  editing.value = null
  editOpen.value = true
  router.replace({ path: '/reminders' }).catch(() => {})
}

watch(() => route.query.new, (v) => {
  if (v) consumeCreateQuery()
})

function openNew() {
  editing.value = null
  presetTitle.value = ''
  editOpen.value = true
}

function openEdit(r) {
  editing.value = r
  editOpen.value = true
}

// Человеческий срок («сегодня в 14:30») — общий с разбором фраз строки поиска.
const whenLabel = (iso) => humanWhen(iso)

/** Полное описание повтора для подписи карточки. */
function repeatLabel(r) {
  const kind = r.repeat?.kind || 'none'
  if (kind === 'none') return ''
  const interval = r.repeat?.interval || 1
  if (kind === 'weekly' && r.repeat?.days?.length) {
    return `по ${r.repeat.days.map((d) => WEEKDAY_SHORT[d]).join(', ')}`
  }
  if (interval > 1) {
    const unit = { daily: 'дн.', weekly: 'нед.', monthly: 'мес.', yearly: 'г.' }[kind] || ''
    return `каждые ${interval} ${unit}`
  }
  return REPEAT_LABELS[kind] || ''
}

// Просрочено — сервер уже отправил уведомление, но пользователь его не закрыл.
const isOverdue = (r) => r.active && new Date(r.remind_at) < new Date()

async function snooze(r, minutes) {
  try {
    await store.snooze(r.id, minutes)
    notify.success(`Отложено на ${minutes} мин.`)
  } catch {
    notify.error('Не удалось отложить')
  }
}

async function complete(r) {
  try {
    await store.complete(r.id)
  } catch {
    notify.error('Не удалось отметить')
  }
}

async function confirmDelete() {
  const r = confirmTarget.value
  confirmTarget.value = null
  if (!r) return
  try {
    await store.remove(r.id)
    notify.success('Напоминание удалено')
  } catch {
    notify.error('Не удалось удалить')
  }
}
</script>

<template>
  <div class="rv">
    <header class="rv-head">
      <div class="rv-tabs">
        <button
          type="button"
          class="rv-tab"
          :class="{ 'is-active': tab === 'active' }"
          @click="tab = 'active'"
        >
          Впереди
          <span v-if="store.items.length" class="rv-count">{{ store.items.length }}</span>
        </button>
        <button
          type="button"
          class="rv-tab"
          :class="{ 'is-active': tab === 'done' }"
          @click="tab = 'done'"
        >Сработавшие</button>
      </div>
      <button type="button" class="btn-grad rv-new" @click="openNew">
        <span class="material-symbols-outlined">add_alert</span> Напоминание
      </button>
    </header>

    <!-- Сработало прямо сейчас: баннер живёт до ответа пользователя. -->
    <section v-if="store.fired.length" class="rv-fired">
      <article v-for="f in store.fired" :key="f.id" class="rv-fired-card">
        <span class="material-symbols-outlined rv-fired-ic">notifications_active</span>
        <div class="rv-fired-body">
          <h3 class="rv-fired-title">{{ f.title }}</h3>
          <p v-if="f.note" class="rv-fired-note">{{ f.note }}</p>
        </div>
        <div class="rv-fired-actions">
          <button type="button" class="btn-glass" @click="snooze(f, 10)">Через 10 мин</button>
          <button type="button" class="btn-grad" @click="complete(f)">Готово</button>
          <button type="button" class="rv-icon" title="Скрыть" @click="store.dismissFired(f.id)">
            <span class="material-symbols-outlined">close</span>
          </button>
        </div>
      </article>
    </section>

    <BrandLoader v-if="store.loading" :size="64" class="rv-loader" />

    <EmptyState
      v-else-if="!list.length"
      class="rv-empty"
      icon="alarm"
      tone="soft"
      :title="tab === 'active' ? 'Напоминаний нет' : 'Журнал пуст'"
      :subtitle="tab === 'active'
        ? 'Создайте напоминание — придёт уведомление в приложении, на рабочем столе и в телефоне.'
        : 'Здесь будут напоминания, которые уже сработали.'"
    >
      <button v-if="tab === 'active'" type="button" class="btn-grad" @click="openNew">
        <span class="material-symbols-outlined">add_alert</span> Создать напоминание
      </button>
    </EmptyState>

    <ul v-else class="rv-list">
      <li
        v-for="r in list"
        :key="r.id"
        class="rv-item"
        :class="{ 'is-overdue': isOverdue(r) }"
      >
        <button type="button" class="rv-main" @click="openEdit(r)">
          <span class="material-symbols-outlined rv-item-ic">
            {{ r.repeat?.kind && r.repeat.kind !== 'none' ? 'repeat' : 'alarm' }}
          </span>
          <span class="rv-item-body">
            <span class="rv-item-title">{{ r.title }}</span>
            <span class="rv-item-meta">
              {{ whenLabel(r.remind_at) }}
              <template v-if="repeatLabel(r)"> · {{ repeatLabel(r) }}</template>
              <template v-if="r.link?.kind && r.link.kind !== 'none'">
                · {{ r.link.kind === 'calendar' ? 'событие календаря' : 'дело ежедневника' }}
              </template>
            </span>
            <span v-if="r.note" class="rv-item-note">{{ r.note }}</span>
          </span>
        </button>

        <div class="rv-item-actions">
          <button v-if="tab === 'active'" type="button" class="rv-icon" title="Отложить на 10 минут" @click="snooze(r, 10)">
            <span class="material-symbols-outlined">snooze</span>
          </button>
          <button v-if="tab === 'active'" type="button" class="rv-icon" title="Готово" @click="complete(r)">
            <span class="material-symbols-outlined">check_circle</span>
          </button>
          <button type="button" class="rv-icon rv-icon--danger" title="Удалить" @click="confirmTarget = r">
            <span class="material-symbols-outlined">delete</span>
          </button>
        </div>
      </li>
    </ul>

    <ReminderEditDialog v-model="editOpen" :reminder="editing" :preset-title="presetTitle" />

    <ConfirmDialog
      :visible="!!confirmTarget"
      header="Удалить напоминание?"
      :message="`«${confirmTarget?.title || ''}» больше не напомнит о себе.`"
      confirm-label="Удалить"
      danger-confirm
      @confirm="confirmDelete"
      @cancel="confirmTarget = null"
    />
  </div>
</template>

<style scoped>
.rv {
  display: flex;
  flex-direction: column;
  gap: 12px;
  height: 100%;
  min-height: 0;
  padding: 12px;
  overflow-y: auto;
}

.rv-head { display: flex; align-items: center; gap: 12px; }
.rv-tabs { display: flex; gap: 4px; flex: 1; }

.rv-tab {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 8px 14px;
  border: none;
  border-radius: 999px;
  background: transparent;
  color: var(--color-text-muted);
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
}

.rv-tab.is-active { background: var(--color-primary-container); color: var(--color-on-primary-container); }

.rv-count {
  padding: 0 6px;
  border-radius: 999px;
  background: var(--color-primary);
  color: var(--color-on-primary);
  font-size: 11px;
}

.rv-new { gap: 6px; }

.rv-fired { display: flex; flex-direction: column; gap: 8px; }

.rv-fired-card {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 14px;
  border: 1px solid var(--color-primary);
  border-radius: var(--radius-lg);
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
}

.rv-fired-ic { font-size: 26px; }
.rv-fired-body { flex: 1; min-width: 0; }
.rv-fired-title { margin: 0; font-size: 15px; font-weight: 600; }
.rv-fired-note { margin: 2px 0 0; font-size: 13px; opacity: 0.85; }
.rv-fired-actions { display: flex; align-items: center; gap: 6px; }

.rv-list { display: flex; flex-direction: column; gap: 8px; margin: 0; padding: 0; list-style: none; }

.rv-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 4px 8px 4px 4px;
  border: 1px solid var(--glass-edge);
  border-radius: var(--radius-lg);
  background: var(--acrylic-card-bg);
}

.rv-item.is-overdue { border-color: var(--color-error); }

.rv-main {
  display: flex;
  align-items: center;
  gap: 12px;
  flex: 1;
  min-width: 0;
  padding: 10px;
  border: none;
  border-radius: var(--radius-md);
  background: transparent;
  color: var(--color-text);
  text-align: left;
  cursor: pointer;
}

.rv-item-ic { font-size: 22px; color: var(--color-primary); }
.rv-item-body { display: flex; flex-direction: column; gap: 2px; min-width: 0; }
.rv-item-title { font-size: 14px; font-weight: 600; }
.rv-item-meta { font-size: 12px; color: var(--color-text-muted); }

.rv-item-note {
  font-size: 12px;
  color: var(--color-text-muted);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.rv-item-actions { display: flex; gap: 2px; }

.rv-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 36px;
  max-width: 36px;
  min-height: 36px;
  max-height: 36px;
  border: none;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--color-text-muted);
  cursor: pointer;
}

.rv-icon:hover { background: var(--color-surface-variant); color: var(--color-text); }
.rv-icon--danger:hover { color: var(--color-error); }
.rv-loader { margin: auto; }
.rv-empty { margin: auto; }

@media (max-width: 768px) {
  .rv { padding: 8px; }
  .rv-fired-card { flex-wrap: wrap; }
  .rv-fired-actions { width: 100%; justify-content: flex-end; }
}
</style>
