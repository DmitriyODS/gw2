<template>
  <!-- Доступ к файлу или папке: ссылка и адресная выдача. Доступ, выданный на
       папку, действует на всё её содержимое — об этом честно говорим. -->
  <AppDialog
    :model-value="true"
    title="Доступ"
    :subtitle="name"
    :actions="[{ kind: 'confirm', label: 'Готово' }]"
    @confirm="$emit('close')"
    @cancel="$emit('close')"
    @update:model-value="(v) => !v && $emit('close')"
  >
    <BrandLoader v-if="loading" block :size="48" />

    <AppStack v-else :gap="14">
      <AppInfoBar
        v-if="kind === 'folder'"
        tone="info"
        message="Доступ к папке открывает и всё, что внутри неё."
      />

      <!-- Публичная ссылка. -->
      <AppCard title="Ссылка" hint="Открыть сможет любой, у кого есть адрес">
        <AppStack v-if="links.length" :gap="8">
          <AppRow v-for="link in links" :key="link.id" :title="linkURL(link.code)" dense inline>
            <AppButton variant="icon" icon="content_copy" aria-label="Скопировать" @click="copy(link.code)" />
            <AppButton variant="icon" icon="link_off" tone="danger" aria-label="Убрать" @click="removeLink(link.id)" />
          </AppRow>
        </AppStack>
        <AppButton v-else label="Создать ссылку" icon="add_link" @click="addLink" />
      </AppCard>

      <!-- Адресный доступ. -->
      <AppCard title="Люди и компании" hint="Видно только тем, кого вы добавите">
        <AppStack :gap="8">
          <AppRow
            v-for="m in members"
            :key="m.id"
            :title="m.user_name || m.company_name || 'Без имени'"
            :hint="m.can_edit ? 'Может изменять' : 'Только просмотр'"
            dense
            inline
          >
            <AppButton variant="icon" icon="person_remove" tone="danger" aria-label="Убрать" @click="revoke(m.id)" />
          </AppRow>

          <div class="add-row">
            <InputText
              v-model="query"
              class="add-input"
              placeholder="Имя или логин сотрудника"
              @input="searchUsers"
            />
            <AppSwitch v-model="canEdit" label="Может изменять" />
          </div>

          <ul v-if="found.length" class="found">
            <li v-for="u in found" :key="u.id">
              <button type="button" class="found-btn" @click="addUser(u)">
                <span>{{ u.fio }}</span>
                <span class="found-login">{{ u.login }}</span>
              </button>
            </li>
          </ul>
        </AppStack>
      </AppCard>
    </AppStack>
  </AppDialog>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import InputText from 'primevue/inputtext'
import BrandLoader from '@/components/common/BrandLoader.vue'
import AppButton from '@/components/ui/AppButton.vue'
import AppCard from '@/components/ui/AppCard.vue'
import AppDialog from '@/components/ui/AppDialog.vue'
import AppInfoBar from '@/components/ui/AppInfoBar.vue'
import AppRow from '@/components/ui/AppRow.vue'
import AppStack from '@/components/ui/AppStack.vue'
import AppSwitch from '@/components/ui/AppSwitch.vue'
import * as api from '@/api/drive.js'
import { useNotificationsStore } from '@/stores/notifications.js'

const props = defineProps({
  kind: { type: String, required: true }, // file | folder
  id: { type: Number, required: true },
  name: { type: String, default: '' },
})

defineEmits(['close'])

const notif = useNotificationsStore()
const loading = ref(true)
const links = ref([])
const members = ref([])
const query = ref('')
const found = ref([])
const canEdit = ref(false)

// Поиск людей ждёт паузы в наборе: запрос на каждую букву — лишняя нагрузка.
let searchTimer = null

async function load() {
  try {
    const res = await api.getAccess(props.kind, props.id)
    links.value = res.links || []
    members.value = res.members || []
  } catch (e) {
    notif.error(e.message || 'Не удалось загрузить доступ')
  } finally {
    loading.value = false
  }
}

function linkURL(code) {
  return `${location.origin}/drive/s/${code}`
}

async function copy(code) {
  try {
    await navigator.clipboard.writeText(linkURL(code))
    notif.success('Ссылка скопирована')
  } catch {
    notif.error('Не удалось скопировать — выделите адрес вручную')
  }
}

async function addLink() {
  try {
    links.value = [...links.value, await api.createLink(props.kind, props.id)]
  } catch (e) {
    notif.error(e.message || 'Не удалось создать ссылку')
  }
}

async function removeLink(id) {
  try {
    await api.deleteLink(id)
    links.value = links.value.filter((l) => l.id !== id)
  } catch (e) {
    notif.error(e.message || 'Не удалось убрать ссылку')
  }
}

function searchUsers() {
  clearTimeout(searchTimer)
  const q = query.value.trim()
  if (q.length < 2) {
    found.value = []
    return
  }
  searchTimer = setTimeout(async () => {
    try {
      const res = await api.searchUsers(q)
      found.value = res.items || []
    } catch { /* поиск не критичен */ }
  }, 250)
}

async function addUser(user) {
  try {
    await api.shareTo(props.kind, props.id, { user_id: user.id, can_edit: canEdit.value })
    query.value = ''
    found.value = []
    await load()
  } catch (e) {
    notif.error(e.message || 'Не удалось открыть доступ')
  }
}

async function revoke(id) {
  try {
    await api.revokeAccess(id)
    members.value = members.value.filter((m) => m.id !== id)
  } catch (e) {
    notif.error(e.message || 'Не удалось убрать доступ')
  }
}

onMounted(load)
</script>

<style scoped>
.add-row {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

.add-input {
  flex: 1;
  min-width: min(220px, 100%);
}

.found {
  margin: 0;
  padding: 0;
  list-style: none;
  max-height: 180px;
  overflow-y: auto;
}

.found-btn {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  width: 100%;
  padding: 8px 10px;
  border: none;
  border-radius: var(--radius-sm);
  background: none;
  color: var(--color-text);
  cursor: pointer;
  text-align: left;
}

.found-btn:hover { background: var(--color-surface-variant); }

.found-login {
  color: var(--color-text-dim);
  font-size: 0.82rem;
}
</style>
