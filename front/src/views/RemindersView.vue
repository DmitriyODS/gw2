<script setup>
/* Раздел «Напоминания»: что ждёт впереди и что уже сработало. Срок считает
   сервер (планировщик remindersvc), здесь — только управление: создать,
   отложить, отметить сделанным, удалить. */
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import AppButton from '@/components/ui/AppButton.vue'
import AppInfoBar from '@/components/ui/AppInfoBar.vue'
import AppPage from '@/components/ui/AppPage.vue'
import AppRow from '@/components/ui/AppRow.vue'
import AppStack from '@/components/ui/AppStack.vue'
import AppTabs from '@/components/ui/AppTabs.vue'
import ConfirmDialog from '@/components/ui/ConfirmDialog.vue'
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

const tabs = computed(() => [
  { value: 'active', label: 'Впереди', badge: store.items.length || undefined },
  { value: 'done', label: 'Сработавшие' },
])

const commands = [
  { key: 'new', label: 'Напоминание', icon: 'add_alert', variant: 'filled', primary: true, fab: true },
]

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
  <AppPage
    title="Напоминания"
    :commands="commands"
    :loading="store.loading"
    @command="openNew"
  >
    <template #subhead>
      <AppTabs v-model="tab" :tabs="tabs" />
    </template>

    <AppStack :gap="12">
      <!-- Сработало прямо сейчас: полоса живёт до ответа пользователя. -->
      <AppInfoBar
        v-for="f in store.fired"
        :key="f.id"
        tone="info"
        icon="notifications_active"
        :title="f.title"
        :message="f.note"
        closable
        @close="store.dismissFired(f.id)"
      >
        <template #actions>
          <AppButton size="sm" label="Через 10 мин" @click="snooze(f, 10)" />
          <AppButton size="sm" variant="filled" label="Готово" @click="complete(f)" />
        </template>
      </AppInfoBar>

      <EmptyState
        v-if="!list.length"
        icon="alarm"
        tone="soft"
        :title="tab === 'active' ? 'Напоминаний нет' : 'Журнал пуст'"
        :subtitle="tab === 'active'
          ? 'Создайте напоминание — придёт уведомление в приложении, на рабочем столе и в телефоне.'
          : 'Здесь будут напоминания, которые уже сработали.'"
      >
        <AppButton
          v-if="tab === 'active'"
          variant="filled"
          icon="add_alert"
          label="Создать напоминание"
          @click="openNew"
        />
      </EmptyState>

      <AppStack v-else :gap="8">
        <AppRow
          v-for="r in list"
          :key="r.id"
          :title="r.title"
          :icon="r.repeat?.kind && r.repeat.kind !== 'none' ? 'repeat' : 'alarm'"
          :tone="isOverdue(r) ? 'danger' : 'neutral'"
          clickable
          inline
          @click="openEdit(r)"
        >
          <template #hint>
            {{ whenLabel(r.remind_at) }}
            <template v-if="repeatLabel(r)"> · {{ repeatLabel(r) }}</template>
            <template v-if="r.link?.kind && r.link.kind !== 'none'">
              · {{ r.link.kind === 'calendar' ? 'событие календаря' : 'дело ежедневника' }}
            </template>
            <template v-if="r.note"> · {{ r.note }}</template>
          </template>

          <AppButton
            v-if="tab === 'active'"
            variant="icon"
            size="sm"
            icon="snooze"
            aria-label="Отложить на 10 минут"
            title="Отложить на 10 минут"
            @click.stop="snooze(r, 10)"
          />
          <AppButton
            v-if="tab === 'active'"
            variant="icon"
            size="sm"
            icon="check_circle"
            aria-label="Готово"
            title="Готово"
            @click.stop="complete(r)"
          />
          <AppButton
            variant="icon"
            size="sm"
            tone="danger"
            icon="delete"
            aria-label="Удалить"
            title="Удалить"
            @click.stop="confirmTarget = r"
          />
        </AppRow>
      </AppStack>
    </AppStack>

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
  </AppPage>
</template>

