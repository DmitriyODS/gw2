<template>
  <div class="yg-settings">
    <!-- Карточка статуса -->
    <div class="settings-card yg-card">
      <div class="hero-icon" :data-tone="status.connected ? 'primary' : 'secondary'">
        <span class="material-symbols-outlined">{{ status.connected ? 'check_circle' : 'link' }}</span>
      </div>
      <div class="card-text">
        <h3>{{ status.connected ? 'YouGile подключён' : 'Подключение к YouGile' }}</h3>
        <p v-if="status.connected">
          Вход выполнен как <b>{{ status.yg_login }}</b>. Ключ:
          <span class="key-fp">…{{ status.key_fingerprint || '????' }}</span>.
          Проверено {{ formatLast(status.last_validated_at) }}.
        </p>
        <p v-else-if="status.company_enabled">
          Войдите в свой YouGile, чтобы создавать карточки и подтягивать их из YouGile в Groove Work.
          Логин и пароль уходят на сервер только для выдачи ключа и не сохраняются.
        </p>
        <p v-else>
          Интеграция с YouGile пока не настроена в вашей компании. Попросите администратора включить её в настройках.
        </p>
      </div>
    </div>

    <!-- Форма подключения / отключения -->
    <div v-if="!status.connected && status.company_enabled" class="settings-card form-card">
      <form class="yg-form" @submit.prevent="onConnect">
        <div class="field">
          <label class="lbl" for="yg-login">Логин YouGile (email)</label>
          <input id="yg-login" class="ctl" type="email" autocomplete="email"
                 v-model="form.login" :disabled="busy" required />
        </div>
        <div class="field">
          <label class="lbl" for="yg-password">Пароль YouGile</label>
          <input id="yg-password" class="ctl" type="password" autocomplete="current-password"
                 v-model="form.password" :disabled="busy" required />
        </div>
        <div class="actions">
          <button type="submit" class="btn-filled" :disabled="busy || !canSubmit">
            <span class="material-symbols-outlined">link</span>
            Подключить
          </button>
        </div>
      </form>
    </div>

    <div v-if="status.connected" class="settings-card actions-card">
      <div class="hero-icon" data-tone="tertiary">
        <span class="material-symbols-outlined">tune</span>
      </div>
      <div class="card-text">
        <h3>Управление подключением</h3>
        <p>Если ключ перестал работать или вы хотите выйти — сделайте это здесь.</p>
      </div>
      <div class="card-actions">
        <button class="btn-outlined" :disabled="busy" @click="askRotate">
          <span class="material-symbols-outlined">refresh</span>
          Сбросить ключ
        </button>
        <button class="btn-outlined danger" :disabled="busy" @click="onDisconnect">
          <span class="material-symbols-outlined">link_off</span>
          Отвязать
        </button>
      </div>
    </div>

    <!-- Диалог сброса ключа: повторно запрашиваем пароль -->
    <Dialog :visible="showRotate" @update:visible="(v) => v || (showRotate = false)"
            modal :closable="!busy" :style="{ width: '420px', maxWidth: 'calc(100vw - 24px)' }"
            header="Сброс ключа YouGile">
      <p class="dlg-text">
        Введите пароль вашего YouGile-аккаунта ещё раз, чтобы выпустить новый ключ.
        Старый ключ будет отозван.
      </p>
      <input class="ctl" type="password" autocomplete="current-password"
             v-model="rotatePassword" :disabled="busy"
             placeholder="Пароль" />
      <template #footer>
        <button class="btn-text" :disabled="busy" @click="showRotate = false">Отмена</button>
        <button class="btn-filled" :disabled="busy || !rotatePassword" @click="onRotate">
          Сбросить
        </button>
      </template>
    </Dialog>
  </div>
</template>

<script setup>
import { reactive, ref, computed, onMounted } from 'vue'
import Dialog from 'primevue/dialog'
import { useYougileStore } from '@/stores/yougile.js'
import { useNotificationsStore } from '@/stores/notifications.js'

const yg = useYougileStore()
const notif = useNotificationsStore()

const form = reactive({ login: '', password: '' })
const showRotate = ref(false)
const rotatePassword = ref('')
const busy = ref(false)

const status = computed(() => yg.status)
const canSubmit = computed(() => form.login.trim() && form.password)

async function onConnect() {
  busy.value = true
  try {
    await yg.connect({ login: form.login.trim(), password: form.password })
    notif.success('YouGile подключён')
    form.password = ''
  } catch (e) {
    notif.error(e?.data?.message || e?.message || 'Не удалось подключиться')
  } finally {
    busy.value = false
  }
}

async function onDisconnect() {
  busy.value = true
  try {
    await yg.disconnect()
    notif.success('YouGile отвязан')
  } catch (e) {
    notif.error(e?.data?.message || 'Не удалось отвязать')
  } finally {
    busy.value = false
  }
}

function askRotate() {
  rotatePassword.value = ''
  showRotate.value = true
}

async function onRotate() {
  busy.value = true
  try {
    await yg.rotate({ password: rotatePassword.value })
    notif.success('Ключ перевыпущен')
    showRotate.value = false
  } catch (e) {
    notif.error(e?.data?.message || 'Не удалось сбросить ключ')
  } finally {
    busy.value = false
  }
}

function formatLast(iso) {
  if (!iso) return 'недавно'
  try {
    return new Date(iso).toLocaleString('ru-RU', {
      day: '2-digit', month: '2-digit', year: 'numeric',
      hour: '2-digit', minute: '2-digit',
    })
  } catch { return 'недавно' }
}

onMounted(() => { yg.refreshStatus().catch(() => {}) })
</script>

<style scoped>
.yg-settings { display: flex; flex-direction: column; gap: 16px; }

.settings-card {
  display: flex; align-items: flex-start; gap: 18px;
  padding: 20px 22px;
  background: var(--acrylic-card-bg);
  border: 1px solid var(--color-outline-dim);
  border-radius: 20px;
}
.hero-icon {
  flex-shrink: 0; width: 56px; height: 56px;
  border-radius: 16px; display: grid; place-items: center;
  background: var(--tone-bg, var(--color-primary-container));
  color: var(--tone-fg, var(--color-on-primary-container));
}
.hero-icon[data-tone="primary"]   { --tone-bg: var(--color-primary-container);   --tone-fg: var(--color-on-primary-container); }
.hero-icon[data-tone="secondary"] { --tone-bg: var(--color-secondary-container); --tone-fg: var(--color-on-secondary-container); }
.hero-icon[data-tone="tertiary"]  { --tone-bg: var(--color-tertiary-container);  --tone-fg: var(--color-on-tertiary-container); }
.hero-icon .material-symbols-outlined { font-size: 28px; }

.card-text { flex: 1; min-width: 0; }
.card-text h3 { margin: 0 0 4px; font-size: 16px; font-weight: 700; color: var(--color-text); }
.card-text p { margin: 0; font-size: 13px; line-height: 1.5; color: var(--color-text-dim); }
.card-text p b { color: var(--color-text); }
.key-fp { font-family: var(--font-mono, monospace); color: var(--color-text); }

.form-card { flex-direction: column; }
.yg-form { display: flex; flex-direction: column; gap: 14px; }
.field { display: flex; flex-direction: column; gap: 6px; }
.lbl { font-size: 13px; font-weight: 600; color: var(--color-on-surface-variant); }
.ctl {
  appearance: none; width: 100%;
  border: 1px solid var(--color-outline-variant);
  background: var(--acrylic-card-bg); color: var(--color-on-surface);
  padding: 10px 12px; border-radius: var(--radius-md, 12px);
  font: inherit;
}
.ctl:focus { outline: 2px solid var(--color-primary); outline-offset: -1px; }

.actions { display: flex; justify-content: flex-end; }
.actions-card { align-items: center; flex-wrap: wrap; }
.card-actions { display: flex; gap: 10px; flex-wrap: wrap; }

.btn-filled, .btn-outlined, .btn-text {
  display: inline-flex; align-items: center; gap: 8px;
  height: 40px; padding: 0 18px; border-radius: 20px;
  font: inherit; font-weight: 600; cursor: pointer;
  border: 1px solid transparent; transition: background .15s, border-color .15s;
}
.btn-filled { background: var(--color-primary); color: var(--color-on-primary); }
.btn-filled:hover:not(:disabled) { background: color-mix(in oklch, var(--color-primary) 90%, black); }
.btn-outlined { background: transparent; border-color: var(--color-outline-variant); color: var(--color-text); }
.btn-outlined:hover:not(:disabled) { background: var(--color-surface-high); }
.btn-outlined.danger { color: var(--color-error); border-color: var(--color-error); }
.btn-outlined.danger:hover:not(:disabled) { background: var(--color-error-container); color: var(--color-on-error-container); }
.btn-text { background: transparent; color: var(--color-text); }
.btn-text:hover:not(:disabled) { background: var(--color-surface-high); }
button:disabled { opacity: 0.5; cursor: not-allowed; }

.dlg-text { margin: 0 0 12px; font-size: 14px; color: var(--color-text-dim); }
</style>
