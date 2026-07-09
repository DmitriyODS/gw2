<template>
  <AppDialog
    :model-value="modelValue"
    title="Поделиться заметкой" icon="share" size="md"
    :actions="[{ kind: 'cancel', label: 'Закрыть' }]"
    @cancel="close" @update:model-value="(v) => !v && close()"
  >
    <div class="ns">
      <!-- ── Пользователям сервиса (адресный шаринг, совместная работа) ── -->
      <section class="ns-section">
        <h4 class="ns-title">
          <span class="material-symbols-outlined">group</span>
          Пользователям сервиса
        </h4>
        <p class="ns-note">
          Заметка появится у адресата во вкладке «Поделились». С правом
          «Редактирование» вы работаете над заметкой вместе — правки и курсоры
          видны друг другу вживую.
        </p>

        <div class="ns-user-search">
          <span class="material-symbols-outlined">search</span>
          <input
            v-model="userQuery"
            class="ns-user-input"
            placeholder="Найти пользователя — имя или логин"
          />
        </div>
        <ul v-if="userQuery && userResults.length" class="ns-user-results">
          <li v-for="u in userResults" :key="u.id">
            <button class="ns-user-row" type="button" @click="addMember(u)">
              <img class="ns-avatar" :src="avatarOf(u)" :alt="u.fio" />
              <span class="ns-user-fio">{{ u.fio }}</span>
              <span class="ns-user-login">@{{ u.login }}</span>
              <span class="material-symbols-outlined ns-user-add">person_add</span>
            </button>
          </li>
        </ul>
        <p v-else-if="userQuery && !searching" class="ns-empty">Никого не нашли.</p>

        <ul v-if="members.length" class="ns-members">
          <li v-for="m in members" :key="m.user_id" class="ns-member">
            <img class="ns-avatar" :src="avatarOf({ id: m.user_id, avatar_path: m.avatar_path })" :alt="m.fio" />
            <span class="ns-user-fio">{{ m.fio }}</span>
            <!-- Право доступа — переключается на месте (идемпотентный upsert) -->
            <button
              class="chip-tint ns-access"
              :class="m.can_edit ? 'chip-tint--warning' : 'chip-tint--primary'"
              type="button"
              :title="m.can_edit ? 'Сделать «только чтение»' : 'Разрешить редактирование'"
              @click="toggleAccess(m)"
            >
              <span class="material-symbols-outlined">{{ m.can_edit ? 'edit' : 'visibility' }}</span>
              {{ m.can_edit ? 'Редактирование' : 'Чтение' }}
            </button>
            <button class="ns-icon danger" title="Закрыть доступ" @click="memberToRemove = m">
              <span class="material-symbols-outlined">person_remove</span>
            </button>
          </li>
        </ul>
        <p v-else-if="!loadingMembers && !userQuery" class="ns-empty">Пока ни с кем не поделились.</p>
      </section>

      <div class="ns-divider" />

      <!-- ── Публичные ссылки ── -->
      <section class="ns-section">
        <h4 class="ns-title">
          <span class="material-symbols-outlined">link</span>
          Публичные ссылки
        </h4>
        <p class="ns-note">
          По ссылке заметку увидит любой — даже без входа в систему. Режим
          выбирается при создании; ссылку можно отозвать в любой момент.
        </p>

        <div class="ns-create">
          <button class="ns-btn" :disabled="busy" @click="createLink('view')">
            <span class="material-symbols-outlined">visibility</span> Только чтение
          </button>
          <button class="ns-btn" :disabled="busy" @click="createLink('edit')">
            <span class="material-symbols-outlined">edit</span> Чтение и редактирование
          </button>
        </div>

        <div v-if="loading" class="ns-empty">Загрузка…</div>
        <ul v-else-if="shares.length" class="ns-shares">
          <li v-for="s in shares" :key="s.id" class="ns-share">
            <span class="chip-tint" :class="s.access === 'edit' ? 'chip-tint--warning' : 'chip-tint--primary'">
              <span class="material-symbols-outlined">{{ s.access === 'edit' ? 'edit' : 'visibility' }}</span>
              {{ s.access === 'edit' ? 'Правка' : 'Чтение' }}
            </span>
            <input class="ns-url" :value="shareUrl(s.code)" readonly @focus="$event.target.select()" />
            <button class="ns-icon" title="Копировать" @click="copy(s.code)">
              <span class="material-symbols-outlined">content_copy</span>
            </button>
            <a class="ns-icon" :href="shareUrl(s.code)" target="_blank" rel="noopener" title="Открыть">
              <span class="material-symbols-outlined">open_in_new</span>
            </a>
            <button class="ns-icon danger" title="Отозвать" @click="shareToRevoke = s">
              <span class="material-symbols-outlined">delete</span>
            </button>
          </li>
        </ul>
        <p v-else class="ns-empty">Ссылок пока нет.</p>
      </section>
    </div>

    <!-- Критические действия — с подтверждением -->
    <ConfirmDialog
      :visible="!!memberToRemove"
      header="Закрыть доступ?"
      :message="`${memberToRemove?.fio} больше не увидит эту заметку во вкладке «Поделились».`"
      confirm-label="Закрыть доступ"
      danger-confirm
      @confirm="confirmRemoveMember"
      @cancel="memberToRemove = null"
    />
    <ConfirmDialog
      :visible="!!shareToRevoke"
      header="Отозвать ссылку?"
      :message="'Все, у кого была эта ссылка, потеряют доступ к заметке.'"
      confirm-label="Отозвать"
      danger-confirm
      @confirm="confirmRevoke"
      @cancel="shareToRevoke = null"
    />
  </AppDialog>
</template>

<script setup>
import { ref, watch } from 'vue'
import AppDialog from '@/components/common/AppDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import * as api from '@/api/notes.js'
import { getDirectory } from '@/api/users.js'
import { useNotificationsStore } from '@/stores/notifications.js'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  noteId: { type: [Number, String, null], default: null },
})
const emit = defineEmits(['update:modelValue'])

const notif = useNotificationsStore()
const shares = ref([])
const loading = ref(false)
const busy = ref(false)

// ── Адресный шаринг ──
const members = ref([])
const loadingMembers = ref(false)
const userQuery = ref('')
const userResults = ref([])
const searching = ref(false)
const memberToRemove = ref(null)
const shareToRevoke = ref(null)
let searchTimer = null

function shareUrl(code) { return `${location.origin}/note/${code}` }
function avatarOf(u) {
  return u.avatar_path ? `/uploads/${u.avatar_path}` : `/api/users/${u.id}/identicon`
}

watch(() => props.modelValue, (open) => {
  if (!open || props.noteId == null) return
  userQuery.value = ''
  userResults.value = []
  load()
  loadMembers()
})

watch(userQuery, (q) => {
  clearTimeout(searchTimer)
  if (!q.trim()) { userResults.value = []; return }
  searchTimer = setTimeout(searchUsers, 200)
})

async function searchUsers() {
  searching.value = true
  try {
    const list = await getDirectory(userQuery.value.trim(), /* excludeSelf */ true, { global: true })
    // Уже добавленных не предлагаем повторно.
    const has = new Set(members.value.map((m) => m.user_id))
    userResults.value = list.filter((u) => !has.has(u.id))
  } catch { /* поиск не критичен */ } finally {
    searching.value = false
  }
}

async function loadMembers() {
  loadingMembers.value = true
  try {
    const data = await api.getMembers(props.noteId)
    members.value = data.members ?? []
  } catch (e) {
    notif.error(e?.message || 'Не удалось загрузить доступы')
  } finally {
    loadingMembers.value = false
  }
}

async function addMember(u) {
  try {
    const m = await api.upsertMember(props.noteId, u.id, false)
    members.value.unshift(m)
    userQuery.value = ''
    userResults.value = []
    notif.success(`Доступ открыт: ${u.fio}`)
  } catch (e) {
    notif.error(e?.message || 'Не удалось поделиться')
  }
}

async function toggleAccess(m) {
  try {
    const updated = await api.upsertMember(props.noteId, m.user_id, !m.can_edit)
    const i = members.value.findIndex((x) => x.user_id === m.user_id)
    if (i !== -1) members.value[i] = updated
  } catch (e) {
    notif.error(e?.message || 'Не удалось изменить право')
  }
}

async function confirmRemoveMember() {
  const m = memberToRemove.value
  memberToRemove.value = null
  if (!m) return
  try {
    await api.removeMember(props.noteId, m.user_id)
    members.value = members.value.filter((x) => x.user_id !== m.user_id)
  } catch (e) {
    notif.error(e?.message || 'Не удалось закрыть доступ')
  }
}

// ── Публичные ссылки ──
async function load() {
  loading.value = true
  try {
    const data = await api.getShares(props.noteId)
    shares.value = data.shares ?? []
  } catch (e) {
    notif.error(e?.message || 'Не удалось загрузить ссылки')
  } finally {
    loading.value = false
  }
}

async function createLink(access) {
  busy.value = true
  try {
    const s = await api.createShare(props.noteId, access)
    shares.value.unshift(s)
    await copy(s.code)
  } catch (e) {
    notif.error(e?.message || 'Не удалось создать ссылку')
  } finally {
    busy.value = false
  }
}

async function confirmRevoke() {
  const s = shareToRevoke.value
  shareToRevoke.value = null
  if (!s) return
  try {
    await api.revokeShare(props.noteId, s.id)
    shares.value = shares.value.filter((x) => x.id !== s.id)
  } catch (e) {
    notif.error(e?.message || 'Не удалось отозвать ссылку')
  }
}

async function copy(code) {
  try { await navigator.clipboard.writeText(shareUrl(code)); notif.success('Ссылка скопирована') } catch { /* ignore */ }
}

function close() { emit('update:modelValue', false) }
</script>

<style scoped>
.ns { display: flex; flex-direction: column; gap: 14px; }
.ns-section { display: flex; flex-direction: column; gap: 10px; }
.ns-title {
  margin: 0;
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13.5px;
  font-weight: 700;
  color: var(--color-text);
}
.ns-title .material-symbols-outlined { font-size: 18px; color: var(--color-primary); }
.ns-divider { height: 1px; background: var(--color-outline-dim); }
.ns-note { margin: 0; font-size: 13px; color: var(--color-text-dim); line-height: 1.5; }
.ns-create { display: flex; gap: 8px; flex-wrap: wrap; }
.ns-btn {
  display: inline-flex; align-items: center; gap: 6px; height: 38px; padding: 0 16px;
  border: none; border-radius: var(--radius-full); background: var(--color-primary); color: var(--color-on-primary);
  font-weight: 600; font-size: 13.5px; cursor: pointer;
}
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

/* ── Пользователи ── */
.ns-user-search { position: relative; display: flex; align-items: center; }
.ns-user-search .material-symbols-outlined {
  position: absolute;
  left: 12px;
  font-size: 19px;
  color: var(--color-text-dim);
  pointer-events: none;
}
.ns-user-input {
  width: 100%;
  height: 38px;
  padding: 0 12px 0 38px;
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-md);
  background: var(--color-surface-low);
  color: var(--color-text);
  font: inherit;
  font-size: 13.5px;
  outline: none;
}
.ns-user-input:focus { border-color: var(--color-primary); }

.ns-user-results {
  list-style: none;
  margin: 0;
  padding: 0;
  max-height: 180px;
  overflow-y: auto;
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-md);
}
.ns-user-row {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  padding: 8px 10px;
  border: none;
  background: transparent;
  font: inherit;
  text-align: left;
  cursor: pointer;
  color: var(--color-text);
}
.ns-user-row:hover { background: var(--color-surface-low); }
.ns-user-fio {
  font-size: 13.5px;
  font-weight: 600;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.ns-user-login { font-size: 12px; color: var(--color-text-dim); flex-shrink: 0; }
.ns-user-add { margin-left: auto; font-size: 18px; color: var(--color-primary); }

.ns-members { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: 8px; }
.ns-member { display: flex; align-items: center; gap: 10px; }
.ns-member .ns-user-fio { flex: 1; min-width: 0; }
.ns-avatar { width: 30px; height: 30px; border-radius: 50%; object-fit: cover; flex-shrink: 0; }
.ns-access { cursor: pointer; border: none; font: inherit; flex-shrink: 0; }
</style>
