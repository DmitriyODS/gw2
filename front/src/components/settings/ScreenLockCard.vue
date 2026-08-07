<template>
  <!-- Экран блокировки: пин закрывает приложение, пока вас нет за устройством.
       Сессия при этом остаётся живой — открытые окна и черновики на месте. -->
  <AppCard title="Экран блокировки" hint="Закрывает приложение пин-кодом, пока вас нет за устройством. Из аккаунта при этом не выходит.">
    <template #head>
      <AppChip :tone="lock.enabled.value ? 'success' : 'neutral'" :label="lock.enabled.value ? 'Включён' : 'Выключен'" />
    </template>

    <AppSwitchRow
      :model-value="lock.enabled.value"
      title="Запирать экран"
      hint="Понадобится пин-код: четыре–восемь цифр"
      @update:model-value="onToggle"
    />

    <template v-if="lock.enabled.value">
      <AppRow
        title="Запирать автоматически"
        hint="Через сколько бездействия закрывать экран. «Вручную» — только по кнопке."
        stack
      >
        <AppTabs
          variant="tint"
          dense
          :model-value="String(lock.afterMin.value ?? 0)"
          :tabs="DELAYS"
          @update:model-value="setDelay"
        />
      </AppRow>

      <AppRow
        title="Обои экрана"
        hint="Картинка запертого экрана. Не выбрана — фирменная волна."
      >
        <AppButton label="Выбрать" icon="wallpaper" @click="openWallpaper" />
      </AppRow>

      <AppStack row :gap="10" class="lock-actions">
        <AppButton label="Сменить пин-код" icon="password" @click="openPin('change')" />
        <AppButton label="Запереть сейчас" icon="lock" @click="lock.lock()" />
        <AppButton label="Убрать блокировку" icon="lock_open" tone="danger" @click="openSecret()" />
      </AppStack>
    </template>

    <!-- Обои запертого экрана. Комплект и история картинок общие с рабочим
         столом: набор один, а показываются они в разных местах. -->
    <AppDialog
      v-if="wallpaperOpen"
      v-model="wallpaperOpen"
      size="lg"
      title="Обои экрана блокировки"
      subtitle="Готовый комплект, своя картинка, градиент или узор."
      :actions="wallpaperActions"
      @confirm="applyWallpaper"
      @cancel="wallpaperOpen = false"
    >
      <BackgroundEditor :recipe="wallpaper" :upload-fn="uploadFn" preview="desktop" :presets="WALLPAPERS" />
    </AppDialog>

    <!-- Задание и смена пин-кода. -->
    <AppDialog
      :model-value="pinOpen"
      size="sm"
      :title="pinMode === 'change' ? 'Новый пин-код' : 'Пин-код экрана'"
      subtitle="От четырёх до восьми цифр. Забудете — подойдёт пароль от аккаунта."
      :busy="busy"
      :actions="[
        { kind: 'cancel', label: 'Отмена', disabled: busy },
        { kind: 'confirm', label: 'Сохранить', disabled: busy || !pinValid },
      ]"
      @confirm="savePin"
      @cancel="pinOpen = false"
      @update:model-value="(v) => !v && (pinOpen = false)"
    >
      <InputText
        v-model="pin"
        class="lock-input"
        type="password"
        inputmode="numeric"
        autocomplete="new-password"
        placeholder="Например, 2468"
        @keyup.enter="pinValid && savePin()"
      />
      <p v-if="pin && !pinValid" class="lock-note">Нужны только цифры — от четырёх до восьми.</p>
    </AppDialog>

    <!-- Снятие блокировки требует действующий пин или пароль: иначе её убрал бы
         любой, кто подошёл к запертому экрану. -->
    <AppDialog
      :model-value="secretOpen"
      size="sm"
      tone="danger"
      title="Убрать блокировку?"
      subtitle="Введите текущий пин-код или пароль от аккаунта."
      :busy="busy"
      :actions="[
        { kind: 'cancel', label: 'Отмена', disabled: busy },
        { kind: 'confirm', label: 'Убрать', disabled: busy || !secret },
      ]"
      @confirm="disable"
      @cancel="secretOpen = false"
      @update:model-value="(v) => !v && (secretOpen = false)"
    >
      <InputText
        v-model="secret"
        class="lock-input"
        type="password"
        autocomplete="off"
        placeholder="Пин-код или пароль"
        @keyup.enter="secret && disable()"
      />
    </AppDialog>
  </AppCard>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import InputText from 'primevue/inputtext'
import BackgroundEditor from '@/components/common/BackgroundEditor.vue'
import AppButton from '@/components/ui/AppButton.vue'
import AppCard from '@/components/ui/AppCard.vue'
import AppChip from '@/components/ui/AppChip.vue'
import AppDialog from '@/components/ui/AppDialog.vue'
import AppRow from '@/components/ui/AppRow.vue'
import AppStack from '@/components/ui/AppStack.vue'
import AppSwitchRow from '@/components/ui/AppSwitchRow.vue'
import AppTabs from '@/components/ui/AppTabs.vue'
import { setScreenLock, disableScreenLock } from '@/api/auth.js'
import { useScreenLock } from '@/composables/useScreenLock.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { useDesktopPrefsStore } from '@/stores/desktopPrefs.js'
import { uploadAttachment } from '@/api/messenger.js'
import { cloneRecipe, normalizeRecipe } from '@/utils/chatBackgrounds.js'
import { WALLPAPERS, defaultWallpaperRecipe } from '@/utils/wallpapers.js'

const notif = useNotificationsStore()
const lock = useScreenLock()
const prefs = useDesktopPrefsStore()

// 0 — «вручную»: экран запирается только кнопкой.
const DELAYS = [
  { value: '0', label: 'Вручную' },
  { value: '1', label: '1 мин' },
  { value: '5', label: '5 мин' },
  { value: '15', label: '15 мин' },
  { value: '60', label: '1 час' },
]

const pinOpen = ref(false)
const pinMode = ref('set')
const pin = ref('')
const secretOpen = ref(false)
const secret = ref('')
const busy = ref(false)

const pinValid = computed(() => /^\d{4,8}$/.test(pin.value))

/* ── Обои запертого экрана ── */
const wallpaperOpen = ref(false)
const wallpaper = reactive(defaultWallpaperRecipe())
// Картинка — личный ассет: тот же общий uploads, что у обоев рабочего стола.
const uploadFn = (file) => uploadAttachment(file)

function openWallpaper() {
  const saved = normalizeRecipe(prefs.lockWallpaper)
  Object.assign(wallpaper, saved ? cloneRecipe(saved) : defaultWallpaperRecipe())
  wallpaperOpen.value = true
}

const wallpaperActions = computed(() => [
  { kind: 'cancel', label: 'Отмена' },
  { kind: 'neutral', label: 'Убрать', icon: 'restart_alt', disabled: !prefs.lockWallpaper, onClick: clearWallpaper },
  { kind: 'confirm', label: 'Применить', icon: 'check' },
])

function applyWallpaper() {
  prefs.setLockWallpaper(cloneRecipe(wallpaper))
  wallpaperOpen.value = false
  notif.success('Обои экрана блокировки обновлены')
}

function clearWallpaper() {
  prefs.setLockWallpaper(null)
  Object.assign(wallpaper, defaultWallpaperRecipe())
  wallpaperOpen.value = false
}

function openPin(mode = 'set') {
  pinMode.value = mode
  pin.value = ''
  pinOpen.value = true
}

function openSecret() {
  secret.value = ''
  secretOpen.value = true
}

// Включение просит пин, выключение — подтверждение секретом.
function onToggle(on) {
  if (on) openPin('set')
  else openSecret()
}

async function savePin() {
  busy.value = true
  try {
    lock.apply(await setScreenLock({ pin: pin.value, after_min: lock.afterMin.value ?? 0 }))
    pinOpen.value = false
    notif.success(pinMode.value === 'change' ? 'Пин-код изменён' : 'Блокировка включена')
  } catch (e) {
    notif.error(e.message || 'Не удалось сохранить пин-код')
  } finally {
    busy.value = false
  }
}

async function setDelay(value) {
  const minutes = Number(value)
  try {
    lock.apply(await setScreenLock({ after_min: minutes }))
  } catch (e) {
    notif.error(e.message || 'Не удалось изменить задержку')
  }
}

async function disable() {
  busy.value = true
  try {
    await disableScreenLock(secret.value)
    lock.apply({ enabled: false, after_min: null })
    secretOpen.value = false
    notif.success('Блокировка убрана')
  } catch (e) {
    notif.error(e.message || 'Не подошло — проверьте пин-код или пароль')
  } finally {
    busy.value = false
  }
}

onMounted(() => lock.load())
</script>

<style scoped>
.lock-input { width: 100%; }

.lock-note {
  margin: 8px 0 0;
  color: var(--color-text-dim);
  font-size: 0.85rem;
}

.lock-actions { flex-wrap: wrap; }
</style>
