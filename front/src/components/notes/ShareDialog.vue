<template>
  <AppDialog
    :model-value="modelValue"
    :title="isFolder ? 'Поделиться папкой' : 'Поделиться заметкой'"
    icon="share" size="md"
    :actions="[{ kind: 'cancel', label: 'Закрыть' }]"
    @cancel="close" @update:model-value="(v) => !v && close()"
  >
    <div class="ns">
      <p v-if="isFolder" class="ns-hint">
        <span class="material-symbols-outlined">info</span>
        Доступ к папке распространяется на все вложенные папки и заметки.
      </p>

      <!-- ── Пользователи ── -->
      <section class="ns-section">
        <h4 class="ns-title"><span class="material-symbols-outlined">group</span>Пользователям</h4>
        <div class="ns-user-search">
          <span class="material-symbols-outlined">search</span>
          <input v-model="userQuery" class="ns-user-input" placeholder="Найти пользователя — имя или логин" />
        </div>
        <ul v-if="userQuery && userResults.length" class="ns-user-results">
          <li v-for="u in userResults" :key="u.id">
            <button class="ns-user-row" type="button" @click="addUser(u)">
              <img class="ns-avatar" :src="avatarOf(u)" :alt="u.fio" />
              <span class="ns-user-fio">{{ u.fio }}</span>
              <span class="ns-user-login">@{{ u.login }}</span>
              <span class="material-symbols-outlined ns-user-add">person_add</span>
            </button>
          </li>
        </ul>
        <p v-else-if="userQuery && !searching" class="ns-empty">Никого не нашли.</p>

        <ul v-if="userMembers.length" class="ns-members">
          <li v-for="m in userMembers" :key="'u' + m.user_id" class="ns-member">
            <img class="ns-avatar" :src="avatarOf({ id: m.user_id, avatar_path: m.avatar_path })" :alt="m.fio" />
            <span class="ns-user-fio">{{ m.fio }}</span>
            <button class="chip-tint ns-access" :class="m.can_edit ? 'chip-tint--warning' : 'chip-tint--primary'"
              type="button" @click="toggleUser(m)">
              <span class="material-symbols-outlined">{{ m.can_edit ? 'edit' : 'visibility' }}</span>
              {{ m.can_edit ? 'Редактирование' : 'Чтение' }}
            </button>
            <button class="ns-icon danger" title="Закрыть доступ" @click="removeUser(m)">
              <span class="material-symbols-outlined">person_remove</span>
            </button>
          </li>
        </ul>
      </section>

      <div class="ns-divider" />

      <!-- ── Компании ── -->
      <section class="ns-section">
        <h4 class="ns-title"><span class="material-symbols-outlined">apartment</span>Компаниям</h4>
        <p class="ns-note">Появится у всех сотрудников выбранной компании — текущих и будущих.</p>
        <div v-if="addableCompanies.length" class="ns-company-add">
          <button v-for="c in addableCompanies" :key="c.id" class="ns-company-chip" @click="addCompany(c)">
            <span class="material-symbols-outlined">add</span>{{ c.name }}
          </button>
        </div>
        <ul v-if="companyMembers.length" class="ns-members">
          <li v-for="m in companyMembers" :key="'c' + m.company_id" class="ns-member">
            <span class="ns-company-ic material-symbols-outlined">apartment</span>
            <span class="ns-user-fio">{{ m.company_name }}</span>
            <button class="chip-tint ns-access" :class="m.can_edit ? 'chip-tint--warning' : 'chip-tint--primary'"
              type="button" @click="toggleCompany(m)">
              <span class="material-symbols-outlined">{{ m.can_edit ? 'edit' : 'visibility' }}</span>
              {{ m.can_edit ? 'Редактирование' : 'Чтение' }}
            </button>
            <button class="ns-icon danger" title="Закрыть доступ" @click="removeCompany(m)">
              <span class="material-symbols-outlined">delete</span>
            </button>
          </li>
        </ul>
        <p v-else-if="!addableCompanies.length" class="ns-empty">У вас нет компаний для шаринга.</p>
      </section>

      <!-- ── Публичные ссылки (только заметка) ── -->
      <template v-if="!isFolder">
        <div class="ns-divider" />
        <section class="ns-section">
          <h4 class="ns-title"><span class="material-symbols-outlined">link</span>Публичные ссылки</h4>
          <p class="ns-note">По ссылке заметку увидит любой — даже без входа в систему.</p>
          <div class="ns-create">
            <button class="ns-btn" :disabled="busy" @click="createLink('view')">
              <span class="material-symbols-outlined">visibility</span> Только чтение
            </button>
            <button class="ns-btn" :disabled="busy" @click="createLink('edit')">
              <span class="material-symbols-outlined">edit</span> Чтение и правка
            </button>
          </div>
          <ul v-if="shares.length" class="ns-shares">
            <li v-for="s in shares" :key="s.id" class="ns-share">
              <span class="chip-tint" :class="s.access === 'edit' ? 'chip-tint--warning' : 'chip-tint--primary'">
                <span class="material-symbols-outlined">{{ s.access === 'edit' ? 'edit' : 'visibility' }}</span>
                {{ s.access === 'edit' ? 'Правка' : 'Чтение' }}
              </span>
              <input class="ns-url" :value="shareUrl(s.code)" readonly @focus="$event.target.select()" />
              <button class="ns-icon" title="Копировать" @click="copy(s.code)">
                <span class="material-symbols-outlined">content_copy</span>
              </button>
              <button class="ns-icon danger" title="Отозвать" @click="revoke(s)">
                <span class="material-symbols-outlined">delete</span>
              </button>
            </li>
          </ul>
        </section>
      </template>
    </div>
  </AppDialog>
</template>

<script setup>
import { computed, ref, watch } from 'vue'
import AppDialog from '@/components/common/AppDialog.vue'
import * as api from '@/api/notes.js'
import { getDirectory } from '@/api/users.js'
import { useNotificationsStore } from '@/stores/notifications.js'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  subjectType: { type: String, default: 'note' }, // note | folder
  subjectId: { type: [Number, String, null], default: null },
})
const emit = defineEmits(['update:modelValue', 'changed'])
const notif = useNotificationsStore()

const isFolder = computed(() => props.subjectType === 'folder')

const members = ref([])
const myCompanies = ref([])
const shares = ref([])
const busy = ref(false)
const userQuery = ref('')
const userResults = ref([])
const searching = ref(false)
let searchTimer = null

const userMembers = computed(() => members.value.filter((m) => m.target === 'user'))
const companyMembers = computed(() => members.value.filter((m) => m.target === 'company'))
const addableCompanies = computed(() => {
  const has = new Set(companyMembers.value.map((m) => m.company_id))
  return myCompanies.value.filter((c) => !has.has(c.id))
})

function shareUrl(code) { return `${location.origin}/note/${code}` }
function avatarOf(u) { return u.avatar_path ? `/uploads/${u.avatar_path}` : `/api/users/${u.id}/identicon` }

watch(() => props.modelValue, (open) => {
  if (!open || props.subjectId == null) return
  userQuery.value = ''
  userResults.value = []
  loadMembers()
  loadCompanies()
  if (!isFolder.value) loadShares()
})

watch(userQuery, (q) => {
  clearTimeout(searchTimer)
  if (!q.trim()) { userResults.value = []; return }
  searchTimer = setTimeout(searchUsers, 200)
})

async function searchUsers() {
  searching.value = true
  try {
    const list = await getDirectory(userQuery.value.trim(), true, { global: true })
    const has = new Set(userMembers.value.map((m) => m.user_id))
    userResults.value = list.filter((u) => !has.has(u.id))
  } catch { /* поиск не критичен */ } finally {
    searching.value = false
  }
}

async function loadMembers() {
  try {
    const data = isFolder.value
      ? await api.getFolderMembers(props.subjectId)
      : await api.getNoteMembers(props.subjectId)
    members.value = data.members ?? []
  } catch (e) { notif.error(e?.message || 'Не удалось загрузить доступы') }
}
async function loadCompanies() {
  try { myCompanies.value = (await api.getMyCompanies()).companies ?? [] } catch { myCompanies.value = [] }
}
async function loadShares() {
  try { const d = await api.getShares(props.subjectId); shares.value = d.shares ?? [] } catch { shares.value = [] }
}

function upsertMember(m) {
  const key = m.target === 'user' ? 'user_id' : 'company_id'
  const i = members.value.findIndex((x) => x.target === m.target && x[key] === m[key])
  if (i === -1) members.value = [m, ...members.value]
  else members.value[i] = m
  emit('changed')
}

async function addUser(u) {
  try {
    const fn = isFolder.value ? api.shareFolderWithUser : api.shareNoteWithUser
    const m = await fn(props.subjectId, u.id, false)
    upsertMember(m)
    userQuery.value = ''
    userResults.value = []
    notif.success(`Доступ открыт: ${u.fio}`)
  } catch (e) { notif.error(e?.message || 'Не удалось поделиться') }
}
async function toggleUser(m) {
  try {
    const fn = isFolder.value ? api.shareFolderWithUser : api.shareNoteWithUser
    upsertMember(await fn(props.subjectId, m.user_id, !m.can_edit))
  } catch (e) { notif.error(e?.message || 'Не удалось изменить право') }
}
async function removeUser(m) {
  try {
    const fn = isFolder.value ? api.unshareFolderUser : api.unshareNoteUser
    await fn(props.subjectId, m.user_id)
    members.value = members.value.filter((x) => !(x.target === 'user' && x.user_id === m.user_id))
    emit('changed')
  } catch (e) { notif.error(e?.message || 'Не удалось закрыть доступ') }
}

async function addCompany(c) {
  try {
    const fn = isFolder.value ? api.shareFolderWithCompany : api.shareNoteWithCompany
    upsertMember(await fn(props.subjectId, c.id, false))
    notif.success(`Открыто компании: ${c.name}`)
  } catch (e) { notif.error(e?.message || 'Не удалось поделиться') }
}
async function toggleCompany(m) {
  try {
    const fn = isFolder.value ? api.shareFolderWithCompany : api.shareNoteWithCompany
    upsertMember(await fn(props.subjectId, m.company_id, !m.can_edit))
  } catch (e) { notif.error(e?.message || 'Не удалось изменить право') }
}
async function removeCompany(m) {
  try {
    const fn = isFolder.value ? api.unshareFolderCompany : api.unshareNoteCompany
    await fn(props.subjectId, m.company_id)
    members.value = members.value.filter((x) => !(x.target === 'company' && x.company_id === m.company_id))
    emit('changed')
  } catch (e) { notif.error(e?.message || 'Не удалось закрыть доступ') }
}

async function createLink(access) {
  busy.value = true
  try {
    const s = await api.createShare(props.subjectId, access)
    shares.value = [s, ...shares.value]
    await copy(s.code)
  } catch (e) { notif.error(e?.message || 'Не удалось создать ссылку') } finally { busy.value = false }
}
async function revoke(s) {
  try { await api.revokeShare(props.subjectId, s.id); shares.value = shares.value.filter((x) => x.id !== s.id) }
  catch (e) { notif.error(e?.message || 'Не удалось отозвать') }
}
async function copy(code) {
  try { await navigator.clipboard.writeText(shareUrl(code)); notif.success('Ссылка скопирована') } catch { /* ignore */ }
}

function close() { emit('update:modelValue', false) }
</script>

<style scoped>
.ns { display: flex; flex-direction: column; gap: 14px; }
.ns-hint { margin: 0; display: flex; align-items: center; gap: 8px; font-size: 12.5px; color: var(--color-text-dim); }
.ns-hint .material-symbols-outlined { font-size: 18px; color: var(--color-primary); }
.ns-section { display: flex; flex-direction: column; gap: 10px; }
.ns-title { margin: 0; display: flex; align-items: center; gap: 8px; font-size: 13.5px; font-weight: 700; color: var(--color-text); }
.ns-title .material-symbols-outlined { font-size: 18px; color: var(--color-primary); }
.ns-divider { height: 1px; background: var(--color-outline-dim); }
.ns-note { margin: 0; font-size: 13px; color: var(--color-text-dim); line-height: 1.5; }
.ns-create { display: flex; gap: 8px; flex-wrap: wrap; }
.ns-btn { display: inline-flex; align-items: center; gap: 6px; height: 38px; padding: 0 16px; border: none; border-radius: var(--radius-full); background: var(--color-primary); color: var(--color-on-primary); font-weight: 600; font-size: 13.5px; cursor: pointer; }
.ns-btn .material-symbols-outlined { font-size: 18px; }
.ns-btn:disabled { opacity: 0.6; }
.ns-empty { margin: 0; font-size: 13px; color: var(--color-text-dim); }
.ns-shares { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: 8px; }
.ns-share { display: flex; align-items: center; gap: 6px; }
.ns-share .chip-tint { flex-shrink: 0; }
.ns-url { flex: 1; min-width: 0; height: 38px; padding: 0 12px; border: 1px solid var(--color-outline-dim); border-radius: var(--radius-md); background: var(--color-surface-low); color: var(--color-text); font-size: 13px; }
.ns-icon { flex-shrink: 0; width: 36px; height: 36px; display: grid; place-items: center; border: 1px solid var(--color-outline-dim); border-radius: var(--radius-full); background: var(--color-surface); color: var(--color-text-dim); cursor: pointer; }
.ns-icon:hover { background: var(--color-surface-high); color: var(--color-text); }
.ns-icon.danger { color: var(--color-error); }
.ns-user-search { position: relative; display: flex; align-items: center; }
.ns-user-search > .material-symbols-outlined { position: absolute; left: 12px; font-size: 19px; color: var(--color-text-dim); pointer-events: none; }
.ns-user-input { width: 100%; height: 38px; padding: 0 12px 0 38px; border: 1px solid var(--color-outline-dim); border-radius: var(--radius-md); background: var(--color-surface-low); color: var(--color-text); font: inherit; font-size: 13.5px; outline: none; }
.ns-user-input:focus { border-color: var(--color-primary); }
.ns-user-results { list-style: none; margin: 0; padding: 0; max-height: 180px; overflow-y: auto; border: 1px solid var(--color-outline-dim); border-radius: var(--radius-md); }
.ns-user-row { display: flex; align-items: center; gap: 10px; width: 100%; padding: 8px 10px; border: none; background: transparent; font: inherit; text-align: left; cursor: pointer; color: var(--color-text); }
.ns-user-row:hover { background: var(--color-surface-low); }
.ns-user-fio { font-size: 13.5px; font-weight: 600; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.ns-user-login { font-size: 12px; color: var(--color-text-dim); flex-shrink: 0; }
.ns-user-add { margin-left: auto; font-size: 18px; color: var(--color-primary); }
.ns-members { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: 8px; }
.ns-member { display: flex; align-items: center; gap: 10px; }
.ns-member .ns-user-fio { flex: 1; min-width: 0; }
.ns-avatar { width: 30px; height: 30px; border-radius: 50%; object-fit: cover; flex-shrink: 0; }
.ns-company-ic { width: 30px; height: 30px; display: grid; place-items: center; border-radius: 50%; background: var(--color-primary-container); color: var(--color-on-primary-container); font-size: 18px; flex-shrink: 0; }
.ns-access { cursor: pointer; border: none; font: inherit; flex-shrink: 0; }
.ns-company-add { display: flex; flex-wrap: wrap; gap: 8px; }
.ns-company-chip { display: inline-flex; align-items: center; gap: 4px; height: 34px; padding: 0 12px; border: 1px dashed var(--color-outline); border-radius: var(--radius-full); background: transparent; color: var(--color-text); font: inherit; font-size: 13px; font-weight: 600; cursor: pointer; }
.ns-company-chip:hover { border-color: var(--color-primary); color: var(--color-primary); }
.ns-company-chip .material-symbols-outlined { font-size: 17px; }
</style>
