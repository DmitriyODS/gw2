<template>
  <AppDialog
    :model-value="modelValue"
    tone="primary"
    icon="badge"
    size="md"
    :title="cropping ? 'Загрузка аватарки' : 'Редактирование профиля'"
    :subtitle="cropping ? 'Выберите фото и подгоните кадр.' : 'Данные и контакты, которые видят коллеги.'"
    :busy="saving"
    :closable="!saving"
    :actions="cropping ? [
      { kind: 'neutral', label: 'Назад', icon: 'arrow_back', onClick: () => (cropping = false) },
    ] : [
      { kind: 'cancel', label: 'Отмена', disabled: saving },
      { kind: 'confirm', label: 'Сохранить', icon: 'check', disabled: saving },
    ]"
    @update:model-value="close"
    @confirm="save"
  >
    <!-- Кроппер живёт в этом же окне (сменой режима), а не вложенным диалогом:
         так не громоздим маски друг на друга. -->
    <AvatarCropper v-if="cropping" @cropped="onCropped" @cancel="cropping = false" />

    <div v-else class="pe-body">
      <div class="pe-avatar-row">
        <img :src="avatarSrc" class="pe-avatar" :alt="auth.user?.fio" />
        <div class="pe-avatar-actions">
          <button type="button" class="btn-glass" @click="cropping = true">
            <span class="material-symbols-outlined">photo_camera</span>
            Загрузить фото
          </button>
          <button
            v-if="auth.user?.avatar_path"
            type="button"
            class="btn-glass danger"
            :disabled="avatarBusy"
            @click="removeAvatar"
          >
            <span class="material-symbols-outlined">delete</span>
            Удалить
          </button>
        </div>
      </div>

      <form class="pe-form" @submit.prevent="save">
        <div class="form-group">
          <label>ФИО</label>
          <InputText v-model="form.fio" class="w-full" placeholder="Иванов Иван Иванович" />
        </div>
        <div class="form-group">
          <label>Должность</label>
          <InputText v-model="form.post" class="w-full" placeholder="Менеджер" />
        </div>
        <div class="form-group">
          <label>Телефон</label>
          <PhoneInput v-model="form.phone" />
        </div>
        <div class="form-group">
          <label>Email</label>
          <InputText
            v-model="form.email"
            class="w-full"
            type="email"
            inputmode="email"
            placeholder="you@example.com"
          />
        </div>
        <p v-if="error" class="error-msg">{{ error }}</p>
      </form>

      <ul class="pe-rows">
        <li class="pe-row">
          <span class="pe-row-ico" data-tone="primary">
            <span class="material-symbols-outlined">alternate_email</span>
          </span>
          <span class="pe-row-text">
            <small>Логин</small>
            <span>{{ auth.user?.login || '—' }}</span>
          </span>
          <button type="button" class="btn-glass pe-row-btn" @click="loginDialog = true">Изменить</button>
        </li>
        <li class="pe-row">
          <span class="pe-row-ico" data-tone="tertiary">
            <span class="material-symbols-outlined">lock</span>
          </span>
          <span class="pe-row-text">
            <small>Пароль</small>
            <span>••••••••</span>
          </span>
          <button type="button" class="btn-glass pe-row-btn" @click="passwordDialog = true">Изменить</button>
        </li>
      </ul>
    </div>

    <template #footer-start>
      <!-- Мобильная оболочка не имеет меню «Пуск» с выходом — держим его здесь. -->
      <button v-if="isMobile && !cropping" type="button" class="btn-glass danger" @click="auth.logout()">
        <span class="material-symbols-outlined">logout</span>
        Выйти
      </button>
    </template>
  </AppDialog>

  <ChangeLoginDialog v-model="loginDialog" :current-login="auth.user?.login || ''" />
  <ChangePasswordDialog v-model="passwordDialog" />
</template>

<script setup>
import { computed, reactive, ref, watch } from 'vue'
import AppDialog from '@/components/common/AppDialog.vue'
import AvatarCropper from '@/components/settings/AvatarCropper.vue'
import ChangeLoginDialog from '@/components/profile/ChangeLoginDialog.vue'
import ChangePasswordDialog from '@/components/profile/ChangePasswordDialog.vue'
import PhoneInput from '@/components/common/PhoneInput.vue'
import InputText from 'primevue/inputtext'
import { useAuthStore } from '@/stores/auth.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { useBreakpoint } from '@/composables/useBreakpoint.js'
import { updateMe, uploadAvatar, deleteAvatar } from '@/api/users.js'

const props = defineProps({ modelValue: { type: Boolean, default: false } })
const emit = defineEmits(['update:modelValue'])

const auth = useAuthStore()
const notif = useNotificationsStore()
const { isMobile } = useBreakpoint()

const form = reactive({ fio: '', post: '', phone: '', email: '' })
const error = ref('')
const saving = ref(false)
const cropping = ref(false)
const avatarBusy = ref(false)
const loginDialog = ref(false)
const passwordDialog = ref(false)

const avatarSrc = computed(() => {
  const u = auth.user
  if (!u) return ''
  return u.avatar_path ? `/uploads/${u.avatar_path}` : `/api/users/${u.id}/identicon`
})

watch(
  () => props.modelValue,
  (open) => {
    if (!open) return
    const u = auth.user || {}
    form.fio = u.fio || ''
    form.post = u.post || ''
    form.phone = u.phone || ''
    form.email = u.email || ''
    error.value = ''
    cropping.value = false
  },
  { immediate: true },
)

function close() {
  if (saving.value) return
  emit('update:modelValue', false)
}

async function save() {
  error.value = ''
  if (!form.fio.trim()) {
    error.value = 'ФИО обязательно'
    return
  }
  saving.value = true
  try {
    await updateMe({
      fio: form.fio.trim(),
      post: form.post.trim(),
      phone: form.phone.trim() || null,
      email: form.email.trim() || null,
    })
    await auth.loadMe()
    notif.success('Профиль обновлён')
    emit('update:modelValue', false)
  } catch (e) {
    error.value = e.message || 'Ошибка сохранения'
  } finally {
    saving.value = false
  }
}

async function onCropped(blob) {
  cropping.value = false
  try {
    await uploadAvatar(blob)
    await auth.loadMe()
    notif.success('Аватарка обновлена')
  } catch (e) {
    notif.error(e.message || 'Ошибка загрузки аватарки')
  }
}

async function removeAvatar() {
  avatarBusy.value = true
  try {
    await deleteAvatar()
    await auth.loadMe()
    notif.success('Аватарка удалена')
  } catch (e) {
    notif.error(e.message || 'Ошибка удаления аватарки')
  } finally {
    avatarBusy.value = false
  }
}

</script>

<style scoped>
.pe-body {
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.pe-avatar-row {
  display: flex;
  align-items: center;
  gap: 16px;
}

.pe-avatar {
  width: 84px;
  height: 84px;
  border-radius: var(--radius-lg);
  object-fit: cover;
  border: 1px solid var(--color-outline-dim);
  background: var(--color-surface-low);
}

.pe-avatar-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.pe-form {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.form-group label {
  font-size: 13px;
  font-weight: 600;
  color: var(--color-text-dim);
}

.w-full { width: 100%; }

.error-msg {
  margin: 0;
  font-size: 13px;
  color: var(--color-on-error-container);
  padding: 8px 12px;
  background: var(--color-error-container);
  border-radius: var(--radius-sm);
}

.pe-rows {
  list-style: none;
  margin: 0;
  padding: 16px 0 0;
  border-top: 1px solid var(--color-outline-dim);
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.pe-row {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
}

.pe-row-ico {
  flex-shrink: 0;
  width: 40px;
  height: 40px;
  border-radius: var(--radius-sm);
  display: grid;
  place-items: center;
  background: var(--tone-bg, var(--color-primary-container));
  color: var(--tone-fg, var(--color-on-primary-container));
}
.pe-row-ico[data-tone="primary"]   { --tone-bg: var(--color-primary-container);   --tone-fg: var(--color-on-primary-container); }
.pe-row-ico[data-tone="secondary"] { --tone-bg: var(--color-secondary-container); --tone-fg: var(--color-on-secondary-container); }
.pe-row-ico[data-tone="tertiary"]  { --tone-bg: var(--color-tertiary-container);  --tone-fg: var(--color-on-tertiary-container); }
.pe-row-ico .material-symbols-outlined { font-size: 20px; }

.pe-row-text {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 1px;
}

.pe-row-text small {
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 0.04em;
  text-transform: uppercase;
  color: var(--color-text-dim);
}

.pe-row-text > span {
  font-size: 13.5px;
  color: var(--color-text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.pe-row-btn { margin-left: auto; flex-shrink: 0; }

.btn-glass.danger {
  color: var(--color-error);
  border-color: color-mix(in oklch, var(--color-error) 30%, var(--color-outline-dim));
}
</style>
