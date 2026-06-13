<template>
  <div class="invite-settings">
    <header class="inv-head">
      <h2>Ссылка-приглашение</h2>
      <p>Любой авторизованный пользователь, перешедший по ссылке, вступит в вашу компанию как Сотрудник. Перевыпуск делает старую ссылку недействительной.</p>
    </header>

    <div class="inv-row">
      <input class="inv-input" :value="inviteUrl" readonly placeholder="Ссылка ещё не создана" />
      <button class="inv-btn" :disabled="!code" title="Скопировать" @click="copy">
        <span class="material-symbols-outlined">{{ copied ? 'check' : 'content_copy' }}</span>
      </button>
      <button class="inv-btn primary" :disabled="busy" @click="regen">
        <span class="material-symbols-outlined">{{ code ? 'autorenew' : 'add_link' }}</span>
        <span>{{ code ? 'Перевыпустить' : 'Создать ссылку' }}</span>
      </button>
    </div>

    <p v-if="error" class="inv-err">{{ error }}</p>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useAuthStore } from '@/stores/auth.js'
import { getCompanyInvite, regenerateCompanyInvite } from '@/api/companies.js'

const auth = useAuthStore()
const code = ref('')
const busy = ref(false)
const copied = ref(false)
const error = ref('')

const inviteUrl = computed(() => (code.value ? `${window.location.origin}/join/${code.value}` : ''))

onMounted(load)

async function load() {
  if (auth.companyId == null) return
  try {
    const res = await getCompanyInvite(auth.companyId)
    code.value = res.code || ''
  } catch (e) {
    error.value = e?.message || 'Не удалось загрузить ссылку'
  }
}

async function regen() {
  if (auth.companyId == null) return
  busy.value = true
  error.value = ''
  try {
    const res = await regenerateCompanyInvite(auth.companyId)
    code.value = res.code || ''
  } catch (e) {
    error.value = e?.message || 'Не удалось создать ссылку'
  } finally {
    busy.value = false
  }
}

async function copy() {
  if (!inviteUrl.value) return
  try {
    await navigator.clipboard.writeText(inviteUrl.value)
    copied.value = true
    setTimeout(() => { copied.value = false }, 1500)
  } catch { /* ignore */ }
}
</script>

<style scoped>
.invite-settings { display: flex; flex-direction: column; gap: 16px; max-width: 640px; }
.inv-head h2 { margin: 0 0 6px; font-size: 18px; font-weight: 700; color: var(--color-text); }
.inv-head p { margin: 0; font-size: 13px; color: var(--color-text-dim); line-height: 1.5; }

.inv-row { display: flex; gap: 8px; align-items: center; flex-wrap: wrap; }
.inv-input {
  flex: 1;
  min-width: 220px;
  height: 44px;
  padding: 0 14px;
  border-radius: var(--radius-md, 12px);
  border: 1px solid var(--color-outline-dim);
  background: var(--color-surface-high);
  color: var(--color-text);
  font: inherit;
  font-size: 13px;
}
.inv-btn {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  height: 44px;
  padding: 0 16px;
  border-radius: var(--radius-md, 12px);
  border: 1px solid var(--color-outline-dim);
  background: var(--color-surface);
  color: var(--color-text);
  cursor: pointer;
  font: inherit;
  font-weight: 600;
}
.inv-btn:hover:not(:disabled) { border-color: var(--color-primary); color: var(--color-primary); }
.inv-btn.primary { background: var(--color-primary); color: var(--color-on-primary); border-color: var(--color-primary); }
.inv-btn.primary:hover:not(:disabled) { background: var(--color-primary-hover); color: var(--color-on-primary); }
.inv-btn:disabled { opacity: 0.5; cursor: not-allowed; }
.inv-btn .material-symbols-outlined { font-size: 20px; }

.inv-err {
  margin: 0;
  padding: 10px 12px;
  border-radius: var(--radius-md, 12px);
  background: var(--color-error-container);
  color: var(--color-on-error-container);
  font-size: 13px;
}
</style>
