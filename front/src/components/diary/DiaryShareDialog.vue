<template>
  <AppDialog
    :model-value="modelValue"
    title="Поделиться" icon="share" size="md"
    :actions="[{ kind: 'cancel', label: 'Закрыть' }]"
    @cancel="close" @update:model-value="(v) => !v && close()"
  >
    <div class="ds">
      <!-- Адресный доступ -->
      <section class="ds-sec">
        <h4 class="ds-title"><span class="material-symbols-outlined">group</span> Доступ людям</h4>
        <p class="ds-note">
          Выбранные пользователи увидят этот ежедневник у себя во вкладке
          «Поделились» — только для чтения.
        </p>

        <div class="ds-search">
          <span class="material-symbols-outlined">person_search</span>
          <input v-model="query" type="text" placeholder="Найти пользователя по имени или логину…" />
          <span v-if="searching" class="material-symbols-outlined spin">progress_activity</span>
        </div>
        <div v-if="results.length" class="ds-results">
          <button
            v-for="u in results" :key="u.id" class="ds-result" :disabled="busyUser === u.id"
            @click="add(u)"
          >
            <img :src="avatarOf(u)" class="ds-ava" alt="" />
            <span class="ds-rname">{{ u.fio }}</span>
            <span class="material-symbols-outlined">add</span>
          </button>
        </div>

        <ul v-if="members.length" class="ds-members">
          <li v-for="m in members" :key="m.user_id" class="ds-member">
            <img :src="avatarOf({ id: m.user_id, avatar_path: m.avatar_path })" class="ds-ava" alt="" />
            <span class="ds-mname">{{ m.fio }}</span>
            <button class="ds-remove" title="Закрыть доступ" @click="remove(m.user_id)">
              <span class="material-symbols-outlined">close</span>
            </button>
          </li>
        </ul>
        <p v-else-if="!loadingMembers" class="ds-empty">Пока никому не открыт.</p>
      </section>

      <!-- Публичная ссылка -->
      <section class="ds-sec">
        <h4 class="ds-title"><span class="material-symbols-outlined">link</span> Ссылка для всех</h4>
        <p class="ds-note">
          По ссылке любой (без входа в систему) сможет просматривать ежедневник —
          но не редактировать. Ссылку можно отозвать в любой момент.
        </p>
        <button class="ds-btn" :disabled="sharesBusy" @click="createLink">
          <span class="material-symbols-outlined">add_link</span> Создать ссылку
        </button>
        <div v-if="loadingShares" class="ds-empty">Загрузка…</div>
        <ul v-else-if="shares.length" class="ds-shares">
          <li v-for="s in shares" :key="s.id" class="ds-share">
            <input class="ds-url" :value="shareUrl(s.code)" readonly @focus="$event.target.select()" />
            <button class="ds-icon" title="Копировать" @click="copy(s.code)">
              <span class="material-symbols-outlined">content_copy</span>
            </button>
            <a class="ds-icon" :href="shareUrl(s.code)" target="_blank" rel="noopener" title="Открыть">
              <span class="material-symbols-outlined">open_in_new</span>
            </a>
            <button class="ds-icon danger" title="Отозвать" @click="revoke(s.id)">
              <span class="material-symbols-outlined">delete</span>
            </button>
          </li>
        </ul>
        <p v-else class="ds-empty">Ссылок пока нет.</p>
      </section>
    </div>
  </AppDialog>
</template>

<script setup>
import { ref, watch } from 'vue'
import AppDialog from '@/components/common/AppDialog.vue'
import * as api from '@/api/diaries.js'
import { getDirectory } from '@/api/users.js'
import { useAuthStore } from '@/stores/auth.js'
import { useNotificationsStore } from '@/stores/notifications.js'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  diaryId: { type: [Number, null], default: null },
})
const emit = defineEmits(['update:modelValue'])

const auth = useAuthStore()
const notif = useNotificationsStore()

const members = ref([])
const loadingMembers = ref(false)
const query = ref('')
const results = ref([])
const searching = ref(false)
const busyUser = ref(null)

const shares = ref([])
const loadingShares = ref(false)
const sharesBusy = ref(false)

function avatarOf(u) {
  return u.avatar_path ? `/uploads/${u.avatar_path}` : `/api/users/${u.id}/identicon`
}
function shareUrl(code) { return `${location.origin}/diary/${code}` }

watch(() => props.modelValue, (open) => {
  if (open && props.diaryId != null) loadAll()
  else { query.value = ''; results.value = [] }
})

async function loadAll() {
  loadingMembers.value = true
  loadingShares.value = true
  try {
    const [m, s] = await Promise.all([api.getMembers(props.diaryId), api.getShares(props.diaryId)])
    members.value = m.members ?? []
    shares.value = s.shares ?? []
  } catch (e) {
    notif.error(e?.message || 'Не удалось загрузить')
  } finally {
    loadingMembers.value = false
    loadingShares.value = false
  }
}

let searchTimer = null
watch(query, (q) => {
  clearTimeout(searchTimer)
  const term = q.trim()
  if (term.length < 2) { results.value = []; return }
  searchTimer = setTimeout(async () => {
    searching.value = true
    try {
      const data = await getDirectory(term, true, { global: true })
      const list = Array.isArray(data) ? data : (data?.items || [])
      const taken = new Set([auth.userId, ...members.value.map((m) => m.user_id)])
      results.value = list.filter((u) => !taken.has(u.id)).slice(0, 8)
    } catch { results.value = [] } finally { searching.value = false }
  }, 300)
})

async function add(u) {
  busyUser.value = u.id
  try {
    const m = await api.addMember(props.diaryId, u.id)
    members.value.push(m)
    results.value = results.value.filter((x) => x.id !== u.id)
    query.value = ''
    notif.success(`Доступ открыт: ${u.fio}`)
  } catch (e) {
    notif.error(e?.message || 'Не удалось открыть доступ')
  } finally {
    busyUser.value = null
  }
}

async function remove(userId) {
  try {
    await api.removeMember(props.diaryId, userId)
    members.value = members.value.filter((m) => m.user_id !== userId)
  } catch (e) {
    notif.error(e?.message || 'Не удалось закрыть доступ')
  }
}

async function createLink() {
  sharesBusy.value = true
  try {
    shares.value.unshift(await api.createShare(props.diaryId))
  } catch (e) {
    notif.error(e?.message || 'Не удалось создать ссылку')
  } finally {
    sharesBusy.value = false
  }
}

async function revoke(id) {
  try {
    await api.revokeShare(props.diaryId, id)
    shares.value = shares.value.filter((s) => s.id !== id)
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
.ds { display: flex; flex-direction: column; gap: 22px; }
.ds-sec { display: flex; flex-direction: column; gap: 10px; }
.ds-title { display: inline-flex; align-items: center; gap: 8px; margin: 0; font-size: 15px; font-weight: 700; color: var(--color-text); }
.ds-title .material-symbols-outlined { font-size: 20px; color: var(--color-primary); }
.ds-note { margin: 0; font-size: 13px; color: var(--color-text-dim); line-height: 1.5; }

.ds-search {
  display: flex; align-items: center; gap: 8px; height: 42px; padding: 0 12px;
  background: var(--color-surface-high); border: 1px solid var(--color-outline-variant); border-radius: var(--radius-md, 14px);
}
.ds-search > .material-symbols-outlined { color: var(--color-text-dim); font-size: 20px; }
.ds-search input { flex: 1; min-width: 0; border: none; background: none; outline: none; color: var(--color-text); font: inherit; }

.ds-results { display: flex; flex-direction: column; gap: 2px; border: 1px solid var(--color-outline-dim); border-radius: var(--radius-md); padding: 4px; }
.ds-result {
  display: flex; align-items: center; gap: 10px; width: 100%; padding: 8px; border: none; background: none;
  border-radius: var(--radius-md); cursor: pointer; text-align: left; color: var(--color-text);
}
.ds-result:hover { background: var(--color-surface-high); }
.ds-result:disabled { opacity: 0.5; }
.ds-rname { flex: 1; min-width: 0; font-weight: 600; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.ds-result .material-symbols-outlined { color: var(--color-primary); }

.ds-members { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: 6px; }
.ds-member { display: flex; align-items: center; gap: 10px; padding: 6px 8px; border-radius: var(--radius-md); background: var(--color-surface-high); }
.ds-mname { flex: 1; min-width: 0; font-weight: 600; color: var(--color-text); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.ds-ava { width: 30px; height: 30px; border-radius: 50%; object-fit: cover; flex-shrink: 0; }
.ds-remove { flex-shrink: 0; width: 30px; height: 30px; display: grid; place-items: center; border: none; background: none; cursor: pointer; color: var(--color-text-dim); border-radius: var(--radius-full); }
.ds-remove:hover { background: var(--color-surface); color: var(--color-error); }

.ds-empty { margin: 0; font-size: 13px; color: var(--color-text-dim); }
.ds-btn {
  display: inline-flex; align-items: center; gap: 6px; align-self: flex-start; height: 38px; padding: 0 16px;
  border: none; border-radius: var(--radius-full); background: var(--color-primary); color: var(--color-on-primary);
  font-weight: 600; font-size: 14px; cursor: pointer;
}
.ds-shares { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: 8px; }
.ds-share { display: flex; align-items: center; gap: 6px; }
.ds-url { flex: 1; min-width: 0; height: 38px; padding: 0 12px; border: 1px solid var(--color-outline-dim); border-radius: var(--radius-md); background: var(--color-surface-low); color: var(--color-text); font-size: 13px; }
.ds-icon { flex-shrink: 0; width: 36px; height: 36px; display: grid; place-items: center; border: 1px solid var(--color-outline-dim); border-radius: var(--radius-full); background: var(--color-surface); color: var(--color-text-dim); cursor: pointer; }
.ds-icon:hover { background: var(--color-surface-high); color: var(--color-text); }
.ds-icon.danger { color: var(--color-error); }
.spin { animation: dsspin 1s linear infinite; font-size: 18px; }
@keyframes dsspin { to { transform: rotate(360deg); } }
</style>
