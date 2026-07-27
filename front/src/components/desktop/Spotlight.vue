<template>
  <div class="sp-backdrop" @pointerdown.self="close">
    <section class="spotlight" role="dialog" aria-label="Поиск">
      <div class="sp-field">
        <span class="material-symbols-outlined sp-field-icon">search</span>
        <input
          ref="inputEl"
          v-model="query"
          class="sp-input"
          type="text"
          placeholder="Глобальный поиск, или выполнение команд"
          autocomplete="off"
          spellcheck="false"
          @keydown.down.prevent="move(1)"
          @keydown.up.prevent="move(-1)"
          @keydown.enter.prevent="submit"
          @keydown.esc.prevent="close"
        />
        <span v-if="loading" class="sp-spinner" aria-hidden="true" />
        <button v-else-if="query" class="sp-clear" type="button" title="Очистить" @click="query = ''">
          <span class="material-symbols-outlined">close</span>
        </button>
      </div>

      <!-- Калькулятор: считает прямо во время ввода, Enter копирует результат. -->
      <button v-if="calc !== null" class="sp-calc" type="button" @click="copyCalc">
        <span class="material-symbols-outlined">calculate</span>
        <span class="sp-calc-value">= {{ calcText }}</span>
        <span class="sp-calc-hint">{{ copied ? 'Скопировано' : 'Enter — скопировать' }}</span>
      </button>

      <div v-if="sections.length" class="sp-results">
        <section v-for="section in sections" :key="section.key" class="sp-section">
          <h3 class="sp-section-title">{{ section.label }}</h3>
          <button
            v-for="item in section.items"
            :key="item.key"
            class="sp-item"
            :class="{ active: flat[cursor]?.key === item.key, command: item.command }"
            type="button"
            @click="go(item)"
            @mouseenter="cursor = flat.findIndex((i) => i.key === item.key)"
          >
            <img v-if="item.avatar" class="sp-item-avatar" :src="item.avatar" :alt="item.title" />
            <span v-else class="sp-item-icon material-symbols-outlined">{{ item.icon }}</span>
            <span class="sp-item-text">
              <span class="sp-item-title">{{ item.title }}</span>
              <span v-if="item.subtitle" class="sp-item-sub">{{ item.subtitle }}</span>
            </span>
            <span class="material-symbols-outlined sp-item-go">{{ item.command ? 'bolt' : 'arrow_outward' }}</span>
          </button>
        </section>
      </div>

      <p v-else-if="query.trim() && !loading && calc === null" class="sp-empty">Ничего не нашлось</p>

      <p v-else-if="!query.trim()" class="sp-hint">
        Ищем по всем разделам. Умеем «создай задачу …», «напомни … завтра в 9»,
        «напиши Васе …» и «1200×3».
      </p>
    </section>
  </div>
</template>

<script setup>
import { computed, nextTick, onMounted, ref, watch } from 'vue'
import { useDesktopStore } from '@/stores/desktop.js'
import { usePermission, ROLES } from '@/composables/usePermission.js'
import { useCompanySettings } from '@/composables/useCompanySettings.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { useAuthStore } from '@/stores/auth.js'
import { useMessengerStore } from '@/stores/messenger.js'
import { useNotesStore } from '@/stores/notes.js'
import { useBoardsStore } from '@/stores/boards.js'
import { useRemindersStore } from '@/stores/reminders.js'
import { APPS, appById } from '@/desktop/apps.js'
import { calculate, formatResult } from '@/utils/calc.js'
import { parseQuickCommand } from '@/utils/quickCommands.js'
import { humanWhen } from '@/utils/naturalDate.js'
import { resolveRecipients, searchStem } from '@/utils/recipients.js'
import { stripMarkdown } from '@/utils/markdown.js'
import { settingsSections } from '@/utils/settingsSections.js'
import { getTasks } from '@/api/tasks.js'
import { getNotes } from '@/api/notes.js'
import { getBoards } from '@/api/boards.js'
import { searchEntries } from '@/api/diaries.js'
import { searchRecords } from '@/api/registries.js'
import { getPosts } from '@/api/portal.js'
import { getDirectory } from '@/api/users.js'
import { useBreakpoint } from '@/composables/useBreakpoint.js'

const LIMIT = 5
const DEBOUNCE = 220

const emit = defineEmits(['close'])

const desktop = useDesktopStore()
const notif = useNotificationsStore()
const auth = useAuthStore()
const messenger = useMessengerStore()
const notes = useNotesStore()
const boards = useBoardsStore()
const reminders = useRemindersStore()
const { isSuperAdmin, hasActiveCompany, isAtLeast } = usePermission()
const { settings } = useCompanySettings()
const { isMobile } = useBreakpoint()

const inputEl = ref(null)
const query = ref('')
const loading = ref(false)
const cursor = ref(0)
const copied = ref(false)
const hits = ref(emptyHits())

function emptyHits() {
  return { tasks: [], notes: [], boards: [], diaries: [], registries: [], portal: [], people: [], messages: [] }
}

onMounted(() => nextTick(() => inputEl.value?.focus()))

/* ── Калькулятор ────────────────────────────────────────────── */
const calc = computed(() => calculate(query.value))
const calcText = computed(() => (calc.value === null ? '' : formatResult(calc.value)))

async function copyCalc() {
  try {
    await navigator.clipboard.writeText(String(calc.value))
    copied.value = true
    setTimeout(() => { copied.value = false }, 1500)
  } catch {
    notif.warn('Не удалось скопировать результат')
  }
}

/* ── Поиск ──────────────────────────────────────────────────────
   Разделы ищем по названию локально, содержимое — параллельными запросами:
   задачи, заметки, записи ежедневников и реестров (у последних двух поиск
   глобальный, серверный — без обхода списков на клиенте). */
const ctx = computed(() => ({
  hasCompany: hasActiveCompany(),
  isSuperAdmin: isSuperAdmin(),
  settings: settings.value,
}))

const needle = computed(() => query.value.trim().toLowerCase())

const appHits = computed(() => {
  const q = needle.value
  if (!q) return []
  return APPS
    .filter((a) => a.available(ctx.value) && a.title.toLowerCase().includes(q))
    .slice(0, LIMIT)
    .map((a) => ({ key: `app-${a.id}`, icon: a.icon, title: a.title, subtitle: 'Открыть раздел', path: a.path }))
})

/* Разделы настроек ищем по тому же каталогу, что рисует экран настроек, —
   локально, без запроса. */
const settingHits = computed(() => {
  const q = needle.value
  if (!q) return []
  return settingsSections({
    isMobile: isMobile.value,
    hasCompany: hasActiveCompany(),
    isAdmin: isAtLeast(ROLES.ADMIN),
    isSuperAdmin: isSuperAdmin(),
  })
    .filter((s) => [s.title, s.desc].some((t) => t.toLowerCase().includes(q)))
    .slice(0, LIMIT)
    .map((s) => ({
      key: `set-${s.key}`,
      icon: s.icon,
      title: s.title,
      subtitle: `Настройки · ${s.desc}`,
      path: `/settings?section=${s.key}`,
    }))
})

/* Переписки — по уже загруженному списку чатов: сеть не трогаем, а названия
   диалогов и групп клиент и так знает. */
const chatHits = computed(() => {
  const q = needle.value
  if (!q) return []
  return messenger.conversations
    .filter((c) => chatName(c).toLowerCase().includes(q))
    .slice(0, LIMIT)
    .map((c) => ({
      key: `conv-${c.id}`,
      icon: c.is_dev_chat ? 'support_agent' : (c.is_group ? 'groups' : 'forum'),
      avatar: chatAvatar(c),
      title: chatName(c),
      subtitle: c.last_message?.text?.trim() || (c.is_group ? 'Групповой чат' : 'Личная переписка'),
      path: `/messenger/${c.id}`,
    }))
})

function chatName(c) {
  if (c.is_dev_chat) return 'Техподдержка'
  return (c.is_group ? c.title : c.other_user?.fio) || 'Чат'
}

function chatAvatar(c) {
  if (c.is_group) return c.avatar_path ? `/uploads/${c.avatar_path}` : null
  const u = c.other_user
  if (!u || c.is_dev_chat) return null
  return u.avatar_path ? `/uploads/${u.avatar_path}` : `/api/users/${u.id}/identicon`
}

/* Быстрые команды: «создай задачу …», «добавь заметку …», «создай доску …»,
   «напомни … завтра в 9», «напиши Васе …». Что откроется сразу, а что формой,
   решает сам вид: заметку и доску создаём молча (у них нет обязательных
   полей), задаче нужен отдел, напоминанию — срок. */
const command = computed(() => parseQuickCommand(query.value))

const commandHits = computed(() => {
  const cmd = command.value
  // Адресатов сообщения знает только сервер — их строки собирает search().
  if (!cmd || cmd.kind === 'message') return []
  const quoted = cmd.title ? ` «${cmd.title}»` : ''

  if (cmd.kind === 'task') {
    if (!appById('tasks').available(ctx.value) || auth.user?.on_vacation) return []
    return [{
      key: 'cmd-task',
      command: true,
      icon: 'add_task',
      title: `Создать задачу${quoted}`,
      subtitle: 'Откроется форма с заполненным названием',
      path: `/tasks?new=1&title=${encodeURIComponent(cmd.title)}`,
    }]
  }

  if (cmd.kind === 'board') {
    return [{
      key: 'cmd-board',
      command: true,
      icon: 'gesture',
      title: `Создать доску${quoted}`,
      subtitle: 'Доска появится в разделе и сразу откроется',
      run: async () => {
        const board = await boards.createBoard(cmd.title || 'Новая доска')
        desktop.open(`/boards/${board.id}`)
      },
    }]
  }

  if (cmd.kind === 'reminder') {
    // Срок в фразе не назвали — без него напоминание бессмысленно, поэтому
    // отдаём форму с готовым названием (как задаче).
    if (!cmd.at) {
      return [{
        key: 'cmd-reminder',
        command: true,
        icon: 'alarm_add',
        title: `Создать напоминание${quoted}`,
        subtitle: 'Откроется форма — останется выбрать время',
        path: `/reminders?new=1&title=${encodeURIComponent(cmd.title)}`,
      }]
    }
    return [{
      key: 'cmd-reminder',
      command: true,
      icon: 'alarm_add',
      title: `Напомнить${quoted}`,
      subtitle: [humanWhen(cmd.at), repeatLabel(cmd.repeat)].filter(Boolean).join(' · '),
      run: async () => {
        await reminders.create({
          title: cmd.title || 'Напоминание',
          note: '',
          remind_at: cmd.at.toISOString(),
          timezone: Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC',
          repeat: cmd.repeat || { kind: 'none', interval: 1, days: [] },
        })
        notif.success(`Напомню ${humanWhen(cmd.at)}`)
      },
    }]
  }

  return [{
    key: 'cmd-note',
    command: true,
    icon: 'note_add',
    title: `Создать заметку${quoted}`,
    subtitle: 'Заметка появится в разделе и сразу откроется',
    run: async () => {
      const note = await notes.createNote(cmd.title)
      desktop.open(`/notes/${note.id}`)
    },
  }]
})

const REPEAT_LABELS = {
  daily: 'каждый день', weekdays: 'по рабочим дням', weekly: 'каждую неделю',
  monthly: 'каждый месяц', yearly: 'каждый год',
}

const repeatLabel = (repeat) => (repeat ? REPEAT_LABELS[repeat.kind] || '' : '')

/* Кому написать. Своих собеседников (личные чаты и группы) знаем локально,
   остальных — из каталога компании; имя в дательном падеже разбирает
   utils/recipients.js. */
function recipientPool(dirUsers) {
  const pool = []
  const seen = new Set()
  for (const c of messenger.conversations) {
    if (c.is_dev_chat) continue
    if (c.is_group) {
      pool.push({
        key: `g${c.id}`,
        names: [c.title || ''],
        title: c.title || 'Групповой чат',
        icon: 'groups',
        avatar: chatAvatar(c),
        conversationId: c.id,
      })
    } else if (c.other_user?.id) {
      const u = c.other_user
      seen.add(u.id)
      pool.push({
        key: `u${u.id}`,
        names: [u.fio || '', u.login || ''],
        title: u.fio || u.login,
        icon: 'forum',
        avatar: chatAvatar(c),
        conversationId: c.id,
        userId: u.id,
      })
    }
  }
  for (const u of dirUsers) {
    if (!u?.id || seen.has(u.id) || u.id === auth.userId) continue
    pool.push({
      key: `u${u.id}`,
      names: [u.fio || '', u.login || ''],
      title: u.fio || u.login,
      icon: 'account_circle',
      avatar: u.avatar_path ? `/uploads/${u.avatar_path}` : `/api/users/${u.id}/identicon`,
      userId: u.id,
    })
  }
  return pool
}

function messageHits(cmd, dirUsers) {
  const { text, matches } = resolveRecipients(cmd.rest, recipientPool(dirUsers))
  return matches.slice(0, LIMIT).map((p) => ({
    key: `msg-${p.key}`,
    command: true,
    icon: p.icon,
    avatar: p.avatar,
    title: text ? `${p.title} — «${text}»` : p.title,
    subtitle: text ? 'Отправить и открыть чат' : 'Открыть переписку',
    run: () => sendTo(p, text),
  }))
}

async function sendTo(p, text) {
  const conversationId = p.conversationId ?? await messenger.openWith(p.userId)
  if (text) {
    await messenger.send(conversationId, { text })
    notif.success(`Отправлено: ${p.title}`)
  }
  desktop.open(`/messenger/${conversationId}`)
}

const sections = computed(() => [
  { key: 'messages', label: 'Написать', items: hits.value.messages },
  { key: 'commands', label: 'Быстрые действия', items: commandHits.value },
  { key: 'apps', label: 'Разделы', items: appHits.value },
  { key: 'tasks', label: 'Задачи', items: hits.value.tasks },
  { key: 'notes', label: 'Заметки', items: hits.value.notes },
  { key: 'boards', label: 'Доски', items: hits.value.boards },
  { key: 'diaries', label: 'Ежедневники', items: hits.value.diaries },
  { key: 'registries', label: 'Реестры', items: hits.value.registries },
  { key: 'portal', label: 'Портал', items: hits.value.portal },
  { key: 'chats', label: 'Переписки', items: chatHits.value },
  { key: 'people', label: 'Сотрудники', items: hits.value.people },
  { key: 'settings', label: 'Настройки', items: settingHits.value },
].filter((s) => s.items.length))

const flat = computed(() => sections.value.flatMap((s) => s.items))

let timer = null
let ctrl = null
let seq = 0

watch(query, () => {
  clearTimeout(timer)
  ctrl?.abort()
  cursor.value = 0
  const q = query.value.trim()
  if (q.length < 2) {
    hits.value = emptyHits()
    loading.value = false
    return
  }
  timer = setTimeout(() => search(q), DEBOUNCE)
})

async function search(q) {
  const my = ++seq
  ctrl = new AbortController()
  const opt = { signal: ctrl.signal }
  loading.value = true

  const withCompany = hasActiveCompany()
  /* «напиши васе …»: в каталоге ищем по основе имени («вас»), а не по всей
     фразе. Тот же запрос кормит и секцию «Сотрудники» — второй раз каталог не
     дёргаем, просто адресаты вытесняют её из выдачи. */
  const cmd = command.value
  const stem = cmd?.kind === 'message' ? searchStem(cmd.rest) : ''
  const dirQuery = stem || q

  const [tasks, noteHits, boardHits, diaries, registries, portal, people] = await Promise.allSettled([
    withCompany ? getTasks({ search: q, per_page: LIMIT }, opt) : Promise.resolve(null),
    getNotes({ search: q }, opt),
    getBoards({ search: q }, opt),
    searchEntries(q, LIMIT, opt),
    withCompany ? searchRecords(q, LIMIT, opt) : Promise.resolve(null),
    withCompany ? getPosts({ search: q, limit: LIMIT }, opt) : Promise.resolve(null),
    withCompany && dirQuery ? getDirectory(dirQuery, true) : Promise.resolve(null),
  ])
  if (my !== seq) return

  hits.value = {
    messages: cmd?.kind === 'message' ? messageHits(cmd, value(people) ?? []) : [],
    tasks: (value(tasks)?.tasks ?? value(tasks)?.items ?? []).slice(0, LIMIT).map((t) => ({
      key: `task-${t.id}`,
      icon: 'dashboard_customize',
      title: t.name || `Задача #${t.id}`,
      subtitle: t.department_name || t.stage_name || 'Задача',
      path: `/tasks/${t.id}`,
    })),
    notes: (value(noteHits)?.notes ?? []).slice(0, LIMIT).map((n) => ({
      key: `note-${n.id}`,
      icon: 'filter_none',
      title: n.title || 'Без названия',
      subtitle: (n.text_content || '').slice(0, 90) || 'Заметка',
      path: `/notes/${n.id}`,
    })),
    boards: (value(boardHits)?.boards ?? []).slice(0, LIMIT).map((b) => ({
      key: `board-${b.id}`,
      icon: 'gesture',
      title: b.title || 'Без названия',
      subtitle: (b.excerpt || '').slice(0, 90) || 'Доска',
      path: `/boards/${b.id}`,
    })),
    diaries: (value(diaries)?.items ?? []).map((e) => ({
      key: `diary-${e.entry_id}`,
      icon: 'event_list',
      title: e.title,
      subtitle: `${e.diary_name} · ${e.entry_date}`,
      path: `/diaries?diary=${e.diary_id}&q=${encodeURIComponent(e.title)}`,
    })),
    registries: (value(registries)?.items ?? []).map((r) => ({
      key: `record-${r.record_id}`,
      icon: 'list_alt_add',
      title: r.snippet || `Запись #${r.record_id}`,
      subtitle: r.registry_name,
      path: `/registries?registry=${r.registry_id}&q=${encodeURIComponent(q)}`,
    })),
    // Лента отдаёт закреплённые отдельным списком — в поиске они такие же посты.
    portal: [...(value(portal)?.pinned ?? []), ...(value(portal)?.posts ?? [])]
      .slice(0, LIMIT)
      .map((p) => {
        const text = stripMarkdown(p.body || '').trim()
        return {
          key: `post-${p.id}`,
          icon: 'web_stories',
          title: p.title || text.slice(0, 80) || `Публикация #${p.id}`,
          subtitle: (p.title ? text : '').slice(0, 90) || 'Публикация портала',
          path: `/portal/${p.id}`,
        }
      }),
    people: (cmd?.kind === 'message' ? [] : value(people) ?? []).slice(0, LIMIT).map((u) => ({
      key: `user-${u.id}`,
      icon: 'account_circle',
      avatar: u.avatar_path ? `/uploads/${u.avatar_path}` : `/api/users/${u.id}/identicon`,
      title: u.fio || u.login,
      subtitle: u.post || 'Сотрудник — открыть карточку',
      path: `/employees?user=${u.id}`,
    })),
  }
  loading.value = false
}

// Упавший источник не должен ронять весь поиск: остальные результаты покажем.
function value(result) {
  return result.status === 'fulfilled' ? result.value : null
}

/* ── Навигация и переходы ───────────────────────────────────── */
function move(delta) {
  if (!flat.value.length) return
  const next = cursor.value + delta
  cursor.value = (next + flat.value.length) % flat.value.length
}

function submit() {
  const item = flat.value[cursor.value]
  if (item) go(item)
  else if (calc.value !== null) copyCalc()
}

/* Обычный результат — открыть окно раздела; команда — выполнить действие и
   закрыться только после него (пользователь видит, что работа идёт). */
async function go(item) {
  if (!item.run) {
    desktop.open(item.path)
    close()
    return
  }
  if (loading.value) return
  loading.value = true
  try {
    await item.run()
    close()
  } catch (e) {
    notif.error(e?.message || 'Не удалось выполнить команду')
  } finally {
    loading.value = false
  }
}

function close() {
  clearTimeout(timer)
  ctrl?.abort()
  emit('close')
}
</script>

<style scoped>
.sp-backdrop {
  position: fixed;
  inset: 0;
  z-index: 1000;
  background: color-mix(in oklch, var(--color-text) 22%, transparent);
  -webkit-backdrop-filter: blur(2px);
  backdrop-filter: blur(2px);
}

/* Окно поиска — по центру верхней трети экрана, как Spotlight. */
.spotlight {
  position: absolute;
  left: 50%;
  top: 14vh;
  transform: translateX(-50%);
  width: min(720px, calc(100vw - 32px));
  max-height: min(70dvh, 720px);
  display: flex;
  flex-direction: column;
  overflow: hidden;
  background: var(--acrylic-bg-strong);
  -webkit-backdrop-filter: var(--acrylic-blur);
  backdrop-filter: var(--acrylic-blur);
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-xl);
  /* Нижний отступ у самой панели: список внутри прокручивается, и его padding
     у нижнего края не виден — рамка бы упиралась в последний результат. */
  padding-bottom: 12px;
  box-shadow: 0 20px 60px color-mix(in oklch, var(--color-text) 16%, transparent);
  transform-origin: top center;
  transition: opacity 0.18s ease, translate 0.2s cubic-bezier(0.2, 0, 0, 1),
    scale 0.2s cubic-bezier(0.2, 0, 0, 1);
}

.sp-enter-from .spotlight,
.sp-leave-to .spotlight {
  opacity: 0;
  translate: 0 -14px;
  scale: 0.97;
}

.sp-enter-from,
.sp-leave-to { opacity: 0; }

.sp-backdrop { transition: opacity 0.18s ease; }

/* ── Поле ввода ── */
.sp-field {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 0 18px;
  height: 64px;
  flex-shrink: 0;
  border-bottom: 1px solid color-mix(in oklch, var(--acrylic-border) 70%, transparent);
}

.sp-field-icon { font-size: 24px; color: var(--color-text-dim); flex-shrink: 0; }

.sp-input {
  flex: 1;
  min-width: 0;
  height: 100%;
  border: none;
  outline: none;
  background: transparent;
  color: var(--color-text);
  font-size: 18px;
  font-family: inherit;
}

.sp-input::placeholder { color: color-mix(in oklch, var(--color-text-dim) 80%, transparent); }

.sp-clear {
  width: 32px;
  min-width: 32px;
  max-width: 32px;
  height: 32px;
  min-height: 32px;
  max-height: 32px;
  display: grid;
  place-items: center;
  border: none;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--color-text-dim);
  cursor: pointer;
}

.sp-clear:hover { background: color-mix(in oklch, var(--color-primary) 12%, transparent); color: var(--color-primary); }

.sp-spinner {
  width: 18px;
  height: 18px;
  flex-shrink: 0;
  border: 2px solid color-mix(in oklch, var(--color-primary) 30%, transparent);
  border-top-color: var(--color-primary);
  border-radius: 50%;
  animation: spSpin 0.7s linear infinite;
}

@keyframes spSpin { to { rotate: 360deg; } }

/* ── Калькулятор ── */
.sp-calc {
  display: flex;
  align-items: center;
  gap: 12px;
  margin: 12px 12px 0;
  padding: 14px 16px;
  flex-shrink: 0;
  border: 1px solid color-mix(in oklch, var(--color-primary) 26%, transparent);
  border-radius: var(--radius-lg);
  background: color-mix(in oklch, var(--color-primary) 8%, transparent);
  color: var(--color-text);
  cursor: pointer;
  text-align: left;
}

.sp-calc .material-symbols-outlined { font-size: 22px; color: var(--color-primary); }
.sp-calc-value { flex: 1; font-size: 20px; font-weight: 700; font-variant-numeric: tabular-nums; }
.sp-calc-hint { font-size: 12px; color: var(--color-text-dim); }

/* ── Результаты ──
   Отбиты от строки ввода сверху; снизу отступ даёт сама панель (padding-bottom),
   иначе он удваивался бы. */
.sp-results {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding: 14px 10px 0;
  scrollbar-width: thin;
}

.sp-section + .sp-section { margin-top: 10px; }

.sp-section-title {
  margin: 0 0 4px;
  padding: 6px 10px 0;
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0.4px;
  text-transform: uppercase;
  color: var(--color-text-dim);
}

.sp-item {
  display: flex;
  align-items: center;
  gap: 12px;
  width: 100%;
  padding: 10px 12px;
  border: none;
  border-radius: var(--radius-md);
  background: transparent;
  text-align: left;
  cursor: pointer;
  transition: background 0.12s;
}

.sp-item.active { background: color-mix(in oklch, var(--color-primary) 12%, transparent); }

.sp-item-icon { font-size: 22px; color: var(--color-text-dim); flex-shrink: 0; }
.sp-item.active .sp-item-icon { color: var(--color-primary); }

/* Команда — не переход, а действие: подсвечиваем её и без наведения. */
.sp-item.command .sp-item-icon,
.sp-item.command .sp-item-go { color: var(--color-primary); opacity: 1; }

.sp-item-avatar {
  width: 26px;
  height: 26px;
  flex-shrink: 0;
  border-radius: 50%;
  object-fit: cover;
}

.sp-item-text { flex: 1; min-width: 0; display: flex; flex-direction: column; }

.sp-item-title {
  font-size: 14.5px;
  font-weight: 500;
  color: var(--color-text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sp-item-sub {
  font-size: 12.5px;
  color: var(--color-text-dim);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sp-item-go { font-size: 18px; color: var(--color-text-dim); opacity: 0; }
.sp-item.active .sp-item-go { opacity: 1; }

.sp-empty,
.sp-hint {
  margin: 0;
  padding: 26px 22px 30px;
  text-align: center;
  font-size: 13.5px;
  color: var(--color-text-dim);
}
</style>
