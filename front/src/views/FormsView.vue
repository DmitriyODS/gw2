<template>
  <AppListDetail
    v-model:open="detailOpen"
    :loading="store.loadingList && !store.forms.length"
    @narrow-change="narrow = $event"
  >
    <!-- Список форм — общий EntityList: закрепление, папки и порядок личные. -->
    <template #list="{ toggle }">
      <EntityList
        section="forms"
        title="Формы и опросы"
        label-key="title"
        :items="store.forms"
        :selected-id="store.selectedId"
        :scope="store.scope"
        :scopes="SCOPES"
        :icon-of="(f) => (f.quiz ? 'quiz' : 'assignment')"
        :hint="formHint"
        :narrow="narrow"
        :renaming-id="renamingId"
        rename-placeholder="Название формы"
        :rename-maxlength="200"
        create-label="Создать форму"
        :menu-items="formMenu"
        :empty="listEmpty"
        @select="selectForm"
        @update:scope="store.setScope"
        @create="createOpen = true"
        @command="onListCommand"
        @rename="applyRename"
        @rename-cancel="renamingId = null"
        @toggle="toggle"
      >
        <template #chips="{ item }">
          <AppChip size="sm" :tone="statusMeta(item.status).tone" :label="statusMeta(item.status).label" />
          <AppChip v-if="item.responses" size="sm" icon="forum" :label="String(item.responses)" title="Собрано ответов" />
        </template>
      </EntityList>
    </template>

    <template #detail="{ collapsed, toggle }">
      <AppPage
        embedded
        :title="store.selected?.title || ''"
        :back="narrow"
        back-label="К формам"
        :menu="!narrow && collapsed"
        menu-icon="left_panel_open"
        menu-label="Показать список"
        :commands="commands"
        :loading="store.loadingForm && !store.selected"
        @back="detailOpen = false"
        @menu="toggle"
        @command="onCommand"
      >
        <template v-if="store.selected" #subhead>
          <AppTabs v-model="tab" :tabs="tabs" variant="tint" dense />
        </template>

        <template v-if="store.selected && tab === 'responses'" #search="{ narrow: tight }">
          <SearchField
            v-model="searchInput"
            placeholder="Поиск по ответам…"
            :collapsed="tight"
            @update:model-value="onSearch"
            @clear="clearSearch"
          />
        </template>

        <template v-if="store.selected && tab === 'responses' && totalPages > 1" #footer>
          <span class="fv-total">Всего ответов: {{ store.responsesTotal }}</span>
          <div class="fv-pager">
            <AppButton
              variant="icon" size="sm" icon="chevron_left"
              aria-label="Предыдущая страница"
              :disabled="store.filters.page <= 1"
              @click="store.setPage(store.filters.page - 1)"
            />
            <span class="fv-page">{{ store.filters.page }} / {{ totalPages }}</span>
            <AppButton
              variant="icon" size="sm" icon="chevron_right"
              aria-label="Следующая страница"
              :disabled="store.filters.page >= totalPages"
              @click="store.setPage(store.filters.page + 1)"
            />
          </div>
        </template>

        <template v-if="store.selected">
          <FormEditor
            v-if="tab === 'editor'"
            ref="editor"
            :form="store.selected"
            :save="saveStructure"
            @error="notif.error($event)"
            @saved="notif.success('Структура сохранена')"
          />

          <FormFill
            v-else-if="tab === 'fill'"
            :form="fill?.form || store.selected"
            :can-respond="fill?.can_respond ?? false"
            :reason="fill?.reason || ''"
            :mine="fill?.mine || null"
            :answer-keys="fill?.answer_keys || null"
            :booking="fill?.booking || {}"
            :submit="submitAnswer"
            :upload="uploadAnswerFile"
            @error="notif.error($event)"
            @submitted="onSubmitted"
          />

          <FormResponses
            v-else-if="tab === 'responses'"
            :responses="store.responses"
            :loading="store.loadingResponses"
            :quiz="!!store.selected.quiz"
            :can-edit="store.canEdit"
            :search="store.filters.search"
            @open="openResponse"
            @remove="responseToDelete = $event"
          />

          <FormSummary v-else-if="tab === 'summary'" :summary="store.summary" />

          <FormAssignees
            v-else-if="tab === 'assignees'"
            :progress="store.progress"
            :can-edit="store.canEdit"
            @assign="shareUsersOpen = true"
          />

          <FormSettings
            v-else-if="tab === 'settings'"
            :form="store.selected"
            :can-edit="store.canEdit"
            :save="store.updateForm"
            @error="notif.error($event)"
          />
        </template>

        <EmptyState
          v-else
          icon="assignment"
          tone="soft"
          title="Выберите форму слева"
          subtitle="Или создайте новую — кнопка внизу списка"
        />
      </AppPage>
    </template>

    <AppDialog
      v-model="createOpen"
      title="Новая форма"
      subtitle="Название можно поменять в любой момент"
      size="sm"
      :actions="[
        { kind: 'cancel', label: 'Отмена' },
        { kind: 'confirm', label: 'Создать', disabled: !newTitle.trim(), loading: creating },
      ]"
      @cancel="createOpen = false"
      @confirm="doCreate"
    >
      <AppStack :gap="12">
        <InputText v-model="newTitle" placeholder="Например, «Опрос о питании»" maxlength="200" autofocus />
        <AppSwitchRow
          v-model="newQuiz"
          title="Это тест"
          hint="У вопросов появятся баллы и правильные ответы"
        />
      </AppStack>
    </AppDialog>

    <FormResponseDialog
      v-model="responseOpen"
      :form="store.selected"
      :response="activeResponse"
      :can-edit="store.canEdit"
      :publish-grades="store.publishGrades"
      @error="notif.error($event)"
    />

    <FormShareLinkDialog
      v-model="shareLinkOpen"
      :form-id="shareTargetId"
      :form="store.selected"
      @error="notif.error($event)"
      @copied="notif.success('Ссылка скопирована')"
    />
    <FormShareUsersDialog
      v-model="shareUsersOpen"
      :form-id="shareTargetId"
      @error="notif.error($event)"
      @changed="onAccessChanged"
    />

    <!-- Уход с несохранённой структурой: решение обязательно, поэтому диалог
         без крестика и без закрытия по фону — иначе правка теряется молча. -->
    <AppDialog
      v-model="leaveOpen"
      title="Сохранить изменения?"
      subtitle="В конструкторе остались несохранённые правки структуры"
      size="sm"
      :closable="false"
      :show-close="false"
      :busy="leaveBusy"
      :actions="[
        { kind: 'cancel', label: 'Остаться' },
        { kind: 'neutral', label: 'Не сохранять', tone: 'danger' },
        { kind: 'confirm', label: 'Сохранить', loading: leaveBusy },
      ]"
      @cancel="cancelLeave"
      @neutral="leaveWithout"
      @confirm="saveAndLeave"
    />

    <ConfirmDialog
      :visible="!!formToDelete"
      header="Удалить форму?"
      :message="`«${formToDelete?.title || ''}» удалится вместе со всеми собранными ответами. Действие необратимо.`"
      confirm-label="Удалить" danger-confirm
      @confirm="doDeleteForm" @cancel="formToDelete = null"
    />
    <ConfirmDialog
      :visible="!!responseToDelete"
      header="Удалить ответ?"
      message="Ответ и приложенные к нему файлы будут удалены безвозвратно."
      confirm-label="Удалить" danger-confirm
      @confirm="doDeleteResponse" @cancel="responseToDelete = null"
    />
    <ConfirmDialog
      :visible="clearOpen"
      header="Очистить все ответы?"
      message="Форма останется, но собранные ответы и их файлы удалятся безвозвратно."
      confirm-label="Очистить" danger-confirm
      @confirm="doClear" @cancel="clearOpen = false"
    />
  </AppListDetail>
</template>

<script setup>
/* Раздел «Формы и опросы»: слева список, справа открытая форма во вкладках —
   конструктор, заполнение, ответы, сводка, назначения и настройки.

   Что показывать, решает уровень доступа, посчитанный сервером (my_access):
   назначенному видна только вкладка заполнения, соавтору — ещё ответы, автору —
   всё. Клиент его не вычисляет, а только отображает. */
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { onBeforeRouteLeave, useRoute } from 'vue-router'
import InputText from 'primevue/inputtext'
import AppButton from '@/components/ui/AppButton.vue'
import AppChip from '@/components/ui/AppChip.vue'
import AppDialog from '@/components/ui/AppDialog.vue'
import AppListDetail from '@/components/ui/AppListDetail.vue'
import AppPage from '@/components/ui/AppPage.vue'
import AppStack from '@/components/ui/AppStack.vue'
import AppSwitchRow from '@/components/ui/AppSwitchRow.vue'
import AppTabs from '@/components/ui/AppTabs.vue'
import ConfirmDialog from '@/components/ui/ConfirmDialog.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import EntityList from '@/components/common/EntityList.vue'
import SearchField from '@/components/common/SearchField.vue'
import FormAssignees from '@/components/forms/FormAssignees.vue'
import FormEditor from '@/components/forms/FormEditor.vue'
import FormFill from '@/components/forms/FormFill.vue'
import FormResponseDialog from '@/components/forms/FormResponseDialog.vue'
import FormResponses from '@/components/forms/FormResponses.vue'
import FormSettings from '@/components/forms/FormSettings.vue'
import FormShareLinkDialog from '@/components/forms/FormShareLinkDialog.vue'
import FormShareUsersDialog from '@/components/forms/FormShareUsersDialog.vue'
import FormSummary from '@/components/forms/FormSummary.vue'
import { useFormsStore } from '@/stores/forms.js'
import { useAuthStore } from '@/stores/auth.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { exportResponses, getFill, submitResponse, updateMyResponse, uploadFile } from '@/api/forms.js'
import { saveBlob } from '@/utils/download.js'
import { statusMeta } from '@/utils/formFields.js'

const store = useFormsStore()
const route = useRoute()
const authStore = useAuthStore()
const notif = useNotificationsStore()

/* Узкая раскладка — свойство САМОГО раздела (он живёт окном рабочего стола),
   поэтому её сообщает AppListDetail, а не медиазапрос по ширине экрана. */
const narrow = ref(false)
const detailOpen = ref(false)
const tab = ref('editor')
const editor = ref(null)
const searchInput = ref('')

/* Форма живёт в компании, где заведена, поэтому смена активной компании меняет
   сам набор: перечитываем список и закрываем открытую, если её здесь нет. */
watch(() => authStore.companyId, (id, prev) => {
  if (id !== prev) store.reloadForCompany()
})

const tabs = computed(() => {
  const items = []
  if (store.canEdit) items.push({ value: 'editor', label: 'Вопросы' })
  items.push({ value: 'fill', label: store.selected?.my_responded ? 'Мой ответ' : 'Заполнить' })
  if (store.canSeeResponses) {
    items.push({ value: 'responses', label: `Ответы${store.selected?.responses ? ` · ${store.selected.responses}` : ''}` })
    items.push({ value: 'summary', label: 'Сводка' })
    items.push({ value: 'assignees', label: 'Назначения' })
  }
  if (store.canEdit) items.push({ value: 'settings', label: 'Настройки' })
  return items
})

const commands = computed(() => {
  if (!store.selected) return []
  return [
    /* Главное действие — ссылка (её раздают чаще всего), назначение рядом
       отдельной командой. Подменю у главной кнопки не бывает: AppCommandBar
       раскрывает `children` только в меню «ещё», и такая кнопка молчала бы. */
    ...(store.canEdit
      ? [
          {
            key: 'share-link', label: 'Поделиться', icon: 'share',
            variant: 'filled', primary: true,
          },
          { key: 'share-users', label: 'Назначить', icon: 'person_add' },
        ]
      : []),
    ...(store.canSeeResponses && store.selected.responses
      ? [{ key: 'export', label: 'Выгрузить в Excel', icon: 'download' }]
      : []),
    ...(store.canEdit && store.selected.quiz && store.selected.quiz_release === 'manual'
      ? [{ key: 'grades', label: 'Открыть оценки', icon: 'visibility' }]
      : []),
    /* Дублирование, очистка и удаление идут ОТДЕЛЬНЫМИ командами, а не
       подменю «Ещё»: их и так немного, а лишний уровень прячет действия,
       которыми пользуются регулярно (AppCommandBar сам уводит лишние в «ещё»,
       когда места не хватает). */
    ...(store.canEdit
      ? [{ key: 'duplicate', label: 'Дублировать', icon: 'content_copy' }]
      : []),
    ...(store.canEdit && store.selected.responses
      ? [{ key: 'clear', label: 'Очистить ответы', icon: 'delete_sweep' }]
      : []),
    ...(store.isOwner
      ? [{ key: 'delete', label: 'Удалить форму', icon: 'delete' }]
      : []),
  ]
})

function onCommand(key) {
  const actions = {
    'share-link': () => openShare('share-link'),
    'share-users': () => openShare('share-users'),
    export: doExport,
    grades: publishAll,
    duplicate: doDuplicate,
    clear: () => { clearOpen.value = true },
    delete: () => { formToDelete.value = store.selected },
  }
  actions[key]?.()
}

// ── Выбор формы и вкладки ──
async function selectForm(id) {
  if (!(await confirmLeave())) return
  await store.select(id)
  detailOpen.value = true
  tab.value = defaultTab()
  await loadTabData()
}

/* Несохранённая структура — единственное состояние раздела, которое можно
   потерять молча: остальное уходит на сервер сразу. Поэтому любой уход
   (другая вкладка, другая форма, закрытие окна или вкладки браузера)
   спрашивает, что делать с правками, и без ответа не выпускает. */
const leaveOpen = ref(false)
const leaveBusy = ref(false)
let leaveResolve = null

const editorDirty = computed(() => !!editor.value?.dirty)

function confirmLeave() {
  if (!editorDirty.value) return Promise.resolve(true)
  leaveOpen.value = true
  return new Promise((resolve) => { leaveResolve = resolve })
}

function finishLeave(allowed) {
  leaveOpen.value = false
  leaveResolve?.(allowed)
  leaveResolve = null
}

function cancelLeave() {
  finishLeave(false)
}

function leaveWithout() {
  editor.value?.reset()
  finishLeave(true)
}

async function saveAndLeave() {
  leaveBusy.value = true
  try {
    await editor.value?.save()
    finishLeave(true)
  } catch (e) {
    notif.error(e?.message || 'Не удалось сохранить структуру')
    finishLeave(false)
  } finally {
    leaveBusy.value = false
  }
}

// Закрытие вкладки браузера: своего диалога тут не показать — только
// системный запрос подтверждения (его вид задаёт браузер).
function guardUnload(e) {
  if (!editorDirty.value) return
  e.preventDefault()
  e.returnValue = ''
}

onMounted(() => window.addEventListener('beforeunload', guardUnload))
onBeforeUnmount(() => window.removeEventListener('beforeunload', guardUnload))

// Уход из раздела роутером (окно закрыли, открыли другой раздел).
onBeforeRouteLeave(async () => await confirmLeave())

// defaultTab — с чего открывается форма: автору привычнее конструктор,
// назначенному — само заполнение.
function defaultTab() {
  if (store.canEdit) return 'editor'
  return 'fill'
}

/* Смена вкладки уносит конструктор из DOM вместе с черновиком, поэтому она
   тоже спрашивает. Возврат прежнего значения делаем без повторного вопроса —
   отслеживаем флагом. */
let tabRollback = false
watch(tab, async (next, prev) => {
  if (tabRollback) {
    tabRollback = false
    return
  }
  if (prev === 'editor' && !(await confirmLeave())) {
    tabRollback = true
    tab.value = prev
    return
  }
  await loadTabData()
})

const fill = ref(null)

async function loadTabData() {
  if (!store.selectedId) return
  try {
    if (tab.value === 'fill') fill.value = await getFill(store.selectedId)
    if (tab.value === 'responses') await store.fetchResponses()
    if (tab.value === 'summary') await store.fetchSummary()
    if (tab.value === 'assignees') await store.fetchProgress()
  } catch (e) {
    notif.error(e?.message || 'Не удалось загрузить данные формы')
  }
}

// ── Создание, переименование, удаление ──
const createOpen = ref(false)
const newTitle = ref('')
const newQuiz = ref(false)
const creating = ref(false)
const formToDelete = ref(null)
const responseToDelete = ref(null)
const clearOpen = ref(false)

watch(createOpen, (open) => {
  if (open) {
    newTitle.value = ''
    newQuiz.value = false
  }
})

async function doCreate() {
  if (!newTitle.value.trim() || creating.value) return
  creating.value = true
  try {
    const form = await store.createForm(newTitle.value.trim(), newQuiz.value)
    createOpen.value = false
    await selectForm(form.id)
  } catch (e) {
    notif.error(e?.message || 'Не удалось создать форму')
  } finally {
    creating.value = false
  }
}

async function applyRename(f, title) {
  renamingId.value = null
  try {
    if (store.selectedId !== f.id) await store.select(f.id)
    await store.updateForm({ title })
  } catch (e) {
    notif.error(e?.message || 'Не удалось переименовать форму')
  }
}

async function doDeleteForm() {
  const f = formToDelete.value
  formToDelete.value = null
  try {
    await store.removeForm(f.id)
    notif.success('Форма удалена')
  } catch (e) {
    notif.error(e?.message || 'Не удалось удалить форму')
  }
}

async function doDuplicate() {
  try {
    const copy = await store.duplicateForm(store.selectedId)
    notif.success('Копия создана')
    await selectForm(copy.id)
  } catch (e) {
    notif.error(e?.message || 'Не удалось скопировать форму')
  }
}

// ── Пункты списка ──
const renamingId = ref(null)

const SCOPES = [
  { value: 'all', label: 'Все' },
  { value: 'mine', label: 'Мои' },
  { value: 'assigned', label: 'Назначены' },
  { value: 'shared', label: 'Совместные' },
]

const EMPTY = {
  mine: { icon: 'assignment', title: 'Своих форм нет', subtitle: 'Создайте первую — кнопка внизу панели.' },
  assigned: { icon: 'assignment_turned_in', title: 'Вам ничего не назначили', subtitle: 'Здесь появятся формы, которые нужно заполнить.' },
  shared: { icon: 'group', title: 'Совместных форм нет', subtitle: 'Здесь появятся формы, где вам открыли ответы или правку.' },
  all: { icon: 'assignment', title: 'Форм пока нет', subtitle: 'Создайте первую — кнопка внизу панели.' },
}
const listEmpty = computed(() => EMPTY[store.scope] || EMPTY.all)

/* Подпись пункта: у чужой формы — хозяин, у назначенной — срок ответа и
   отметка «отвечено». Своя форма подписи не требует. */
function formHint(f) {
  const parts = []
  if (f.my_access !== 'owner' && f.owner_name) parts.push(f.owner_name)
  if (f.my_due_at) parts.push(f.my_responded ? 'Вы ответили' : `Ответить до ${dueText(f.my_due_at)}`)
  else if (f.my_responded) parts.push('Вы ответили')
  return parts.join(' · ')
}

function dueText(value) {
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return ''
  return d.toLocaleDateString('ru-RU', { day: '2-digit', month: '2-digit' })
}

/* Действия над самой формой; организация списка — забота EntityList. */
function formMenu(f) {
  const manage = ['edit', 'owner'].includes(f.my_access)
  return [
    { label: 'Переименовать', icon: 'edit', action: 'rename', disabled: !manage },
    {
      label: 'Поделиться',
      icon: 'share',
      disabled: !manage,
      children: [
        { label: 'Ссылкой', icon: 'link', action: 'share-link' },
        { label: 'Назначить', icon: 'person_add', action: 'share-users' },
      ],
    },
    { label: 'Дублировать', icon: 'content_copy', action: 'duplicate', disabled: !manage },
    { divider: true },
    { label: 'Удалить', icon: 'delete', danger: true, action: 'delete', disabled: f.my_access !== 'owner' },
  ]
}

async function onListCommand(action, f) {
  if (!f) return
  if (action === 'rename') {
    renamingId.value = f.id
    return
  }
  if (action === 'delete') {
    formToDelete.value = f
    return
  }
  if (action === 'duplicate') {
    if (store.selectedId !== f.id) await store.select(f.id)
    doDuplicate()
    return
  }
  if (action.startsWith('share-')) openShare(action, f)
}

// ── Доступ ──
const shareLinkOpen = ref(false)
const shareUsersOpen = ref(false)
const shareTargetId = ref(null)

function openShare(action, form = store.selected) {
  if (!form) return
  shareTargetId.value = form.id
  shareLinkOpen.value = action === 'share-link'
  shareUsersOpen.value = action === 'share-users'
}

async function onAccessChanged() {
  await store.fetchForms()
  if (tab.value === 'assignees') await store.fetchProgress()
}

// ── Структура и заполнение ──
const saveStructure = (sections) => store.saveStructure(sections)

/* Правка отправленного и новая отправка — разные ручки: PATCH меняет свой
   ответ (когда автор формы это разрешил), POST добавляет ещё один. Что именно
   делает человек, решает само заполнение. */
const submitAnswer = (payload) => (payload.edit && fill.value?.mine
  ? updateMyResponse(store.selectedId, payload)
  : submitResponse(store.selectedId, payload))

// Файл кладём В КОНКРЕТНЫЙ вопрос: от него зависит потолок размера, а квоту
// платит владелец формы.
const uploadAnswerFile = (file, question, onProgress) =>
  uploadFile(store.selectedId, question.id, file, { onProgress })

async function onSubmitted() {
  notif.success('Ответ отправлен')
  fill.value = await getFill(store.selectedId)
  await store.loadForm({ silent: true })
}

// ── Ответы ──
const responseOpen = ref(false)
const activeResponse = ref(null)

function openResponse(r) {
  activeResponse.value = r
  responseOpen.value = true
}

async function doDeleteResponse() {
  const r = responseToDelete.value
  responseToDelete.value = null
  try {
    await store.deleteResponse(r.id)
  } catch (e) {
    notif.error(e?.message || 'Не удалось удалить ответ')
  }
}

async function doClear() {
  clearOpen.value = false
  try {
    await store.clearResponses()
    notif.success('Ответы удалены')
  } catch (e) {
    notif.error(e?.message || 'Не удалось очистить ответы')
  }
}

async function publishAll() {
  try {
    await store.publishGrades(0)
    notif.success('Оценки открыты отвечающим')
  } catch (e) {
    notif.error(e?.message || 'Не удалось открыть оценки')
  }
}

async function doExport() {
  try {
    const res = await exportResponses(store.selectedId)
    await saveBlob(await res.blob(), `${store.selected?.title || 'form'}.xlsx`)
  } catch (e) {
    notif.error(e?.message || 'Не удалось выгрузить ответы')
  }
}

// ── Поиск по ответам ──
let searchTimer = null
function onSearch() {
  clearTimeout(searchTimer)
  searchTimer = setTimeout(() => store.setSearch(searchInput.value.trim()), 300)
}

function clearSearch() {
  clearTimeout(searchTimer)
  searchInput.value = ''
  store.setSearch('')
}

const totalPages = computed(() =>
  Math.max(1, Math.ceil(store.responsesTotal / store.filters.per_page)))

// Переход из строки поиска Hola и из уведомления о назначении.
function applyQuery() {
  const id = Number(route.query.form)
  if (id) selectForm(id)
}

onMounted(() => store.fetchForms().then(applyQuery))
watch(() => route.query, applyQuery)
</script>

<style scoped>
/* Каркас, шапка и вкладки — общие компоненты; здесь остаётся только то, что
   принадлежит самому разделу. */
.fv-total { flex: 1; font-size: 13px; color: var(--color-text-dim); }
.fv-pager { display: flex; align-items: center; gap: 8px; }
.fv-page { min-width: 56px; text-align: center; font-size: 13px; color: var(--color-text-dim); }
</style>
