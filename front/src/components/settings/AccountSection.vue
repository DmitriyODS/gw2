<template>
  <div class="acc">
    <!-- Кто я: аватар, имя, контакты-чипы и правка одной кнопкой. -->
    <AppCard class="acc-id" :gap="20">
      <button type="button" class="acc-avatar" title="Открыть фото" @click="lightboxOpen = true">
        <img :src="avatarSrc" :alt="user?.fio" />
        <span class="acc-avatar-zoom" aria-hidden="true">
          <span class="material-symbols-outlined">zoom_in</span>
        </span>
      </button>

      <div class="acc-id-body">
        <h3 class="acc-name">{{ user?.fio || '—' }}</h3>
        <p v-if="user?.post" class="acc-post">{{ user.post }}</p>

        <ul class="acc-facts">
          <li v-if="user?.login">@{{ user.login }}</li>
          <li :class="{ empty: !user?.email }">{{ user?.email || 'Email не указан' }}</li>
          <li :class="{ empty: !user?.phone }">{{ user?.phone || 'Телефон не указан' }}</li>
          <li v-if="user?.on_vacation" class="vacation">🏖️ В отпуске</li>
        </ul>
      </div>

      <button type="button" class="acc-edit" title="Редактировать" @click="editOpen = true">
        <span class="material-symbols-outlined">edit</span>
      </button>
    </AppCard>

    <!-- Подписка и внешние аккаунты — ряд пилюль под карточкой. -->
    <div class="acc-links">
      <component
        :is="link.action ? 'button' : 'div'"
        v-for="link in links"
        :key="link.key"
        class="acc-link"
        :class="{ 'is-static': !link.action }"
        :type="link.action ? 'button' : undefined"
        @click="link.action && link.action()"
      >
        <!-- Только чужие фирменные знаки: свои иконки в пилюлях не рисуем. -->
        <YandexLogo v-if="link.logo === 'yandex'" :size="20" />
        <YougileLogo v-else-if="link.logo === 'yougile'" :size="20" />
        {{ link.label }}
        <span class="acc-link-state">{{ link.state }}</span>
      </component>
    </div>

    <!-- Отпуск: человек уходит и возвращается сам. Он привязан к КОМПАНИИ, а не
         к аккаунту, поэтому строки нет, пока активной компании не выбрано. -->
    <AppSwitchRow
      v-if="auth.companyId"
      :model-value="onVacation"
      title="Я в отпуске"
      icon="beach_access"
      :hint="vacationHint"
      :disabled="vacationBusy"
      @update:model-value="toggleVacation"
    />

    <!-- Экран блокировки — про безопасность аккаунта, поэтому рядом с
         сеансами устройств. -->
    <ScreenLockCard />

    <!-- Подключения YouGile: это внешний АККАУНТ человека, а не настройка
         приложения — поэтому живёт здесь, рядом с Яндекс ID. Подключений
         может быть несколько, работает активное. -->
    <AppCard v-if="showYougile" id="acc-yougile-card" title="Интеграция с YouGile">
      <YougileUserSettings />
    </AppCard>

    <AppCard class="acc-sessions" title="Авторизация и сессии">
      <template #head>
        <AppButton
          v-if="otherSessions"
          label="Выйти на всех устройствах"
          icon="logout"
          tone="danger"
          :loading="revokingAll"
          @click="revokeAllOpen = true"
        />
      </template>

      <div class="sess-grid">
        <button type="button" class="sess-tile sess-add" @click="authorizeOpen = true">
          <span class="material-symbols-outlined">devices</span>
          <span>Авторизовать<br />новое устройство</span>
        </button>

        <div v-if="sessionsLoading" class="sess-tile sess-loading">
          <BrandLoader :size="48" />
        </div>

        <article v-for="s in sessions" :key="s.id" class="sess-tile sess-item">
          <header class="sess-head">
            <span class="material-symbols-outlined sess-ico">{{ platformIcon(s.platform) }}</span>
            <span v-if="s.city || s.ip" class="sess-place" :title="s.ip">{{ s.city || s.ip }}</span>
          </header>

          <h4 class="sess-title" :title="s.device">{{ s.title }}</h4>
          <p class="sess-meta">Вход: {{ formatLogin(s.created_at) }}</p>
          <p class="sess-meta">
            Последняя активность: {{ formatSeen(s.last_seen_at) }}
            <span v-if="s.current" class="sess-current">· это устройство</span>
          </p>

          <button type="button" class="sess-end" :disabled="revoking === s.id" @click="revokeTarget = s">
            <span class="material-symbols-outlined">logout</span>
            Завершить сеанс
          </button>
        </article>
      </div>
    </AppCard>

    <ProfileEditDialog v-model="editOpen" />
    <AuthorizeDeviceDialog v-model="authorizeOpen" />
    <ImageLightbox v-model="lightboxOpen" :src="avatarSrc" :caption="user?.fio" />

    <AppDialog
      :model-value="!!revokeTarget"
      tone="danger"
      size="sm"
      title="Завершить сеанс?"
      :subtitle="revokeSubtitle"
      :busy="!!revoking"
      :closable="!revoking"
      :actions="[
        { kind: 'cancel', label: 'Отмена', disabled: !!revoking },
        { kind: 'confirm', label: 'Завершить', icon: 'logout', disabled: !!revoking },
      ]"
      @update:model-value="revokeTarget = null"
      @confirm="confirmRevoke"
    />

    <!-- Выход на всех устройствах: текущее остаётся, чтобы человек не выкинул
         сам себя из настроек. -->
    <AppDialog
      :model-value="revokeAllOpen"
      tone="danger"
      size="sm"
      title="Выйти на всех устройствах?"
      :subtitle="`Сеансов будет завершено: ${otherSessions}. Это устройство останется в системе.`"
      :busy="revokingAll"
      :closable="!revokingAll"
      :actions="[
        { kind: 'cancel', label: 'Отмена', disabled: revokingAll },
        { kind: 'confirm', label: 'Выйти везде', icon: 'logout', disabled: revokingAll },
      ]"
      @update:model-value="revokeAllOpen = false"
      @confirm="confirmRevokeAll"
    />

    <AppDialog
      :model-value="yandexUnlinkAsk"
      tone="danger"
      size="sm"
      title="Отвязать Яндекс-аккаунт?"
      subtitle="Входить кнопкой «Войти через Яндекс» станет нельзя — только по логину и паролю."
      :busy="yandexBusy"
      :closable="!yandexBusy"
      :actions="[
        { kind: 'cancel', label: 'Отмена', disabled: yandexBusy },
        { kind: 'confirm', label: 'Отвязать', icon: 'link_off', disabled: yandexBusy },
      ]"
      @update:model-value="yandexUnlinkAsk = false"
      @confirm="unlinkYandex"
    />
  </div>
</template>

<script setup>
/* Аккаунт — раздел настроек (прежний самостоятельный «Профиль»): личные данные,
   внешние аккаунты и реестр входов. Всё редактирование живёт в диалоге правки,
   поэтому здесь только карточки и действия над сеансами. */
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { avatarUrl } from '@/utils/pets.js'
import { useAuthStore } from '@/stores/auth.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { useBillingStore } from '@/stores/billing.js'
import { usePetsStore } from '@/stores/pets.js'
import { formatUntil } from '@/utils/money.js'
import { SUBSCRIPTIONS_VISIBLE } from '@/utils/release.js'
import { inAppShell } from '@/utils/appShell.js'
import { dayLabel } from '@/utils/chatDates.js'
import {
  listSessions,
  revokeSession,
  revokeOtherSessions,
  yandexConfig,
  yandexAuthURL,
  yandexLinkStatus,
  yandexUnlink,
} from '@/api/auth.js'
import { getYougileStatus } from '@/api/yougile.js'
import AppButton from '@/components/ui/AppButton.vue'
import AppCard from '@/components/ui/AppCard.vue'
import AppDialog from '@/components/ui/AppDialog.vue'
import AppSwitchRow from '@/components/ui/AppSwitchRow.vue'
import ImageLightbox from '@/components/common/ImageLightbox.vue'
import BrandLoader from '@/components/common/BrandLoader.vue'
import AuthorizeDeviceDialog from '@/components/devicelink/AuthorizeDeviceDialog.vue'
import ProfileEditDialog from '@/components/profile/ProfileEditDialog.vue'
import YandexLogo from '@/components/common/YandexLogo.vue'
import YougileLogo from '@/components/common/YougileLogo.vue'
import YougileUserSettings from '@/components/settings/YougileUserSettings.vue'
import ScreenLockCard from '@/components/settings/ScreenLockCard.vue'

const props = defineProps({
  // Личное подключение YouGile показывается рядовому участнику компании:
  // администратор подключает компанию целиком в её карточке.
  showYougile: { type: Boolean, default: false },
})

const auth = useAuthStore()
const notif = useNotificationsStore()
const router = useRouter()
const billing = useBillingStore()
const pets = usePetsStore()

const user = computed(() => auth.user)
const avatarSrc = computed(() => {
  const u = auth.user
  if (!u) return ''
  return u.avatar_path ? `/uploads/${u.avatar_path}` : avatarUrl(u)
})

const editOpen = ref(false)
const lightboxOpen = ref(false)
const authorizeOpen = ref(false)

/* ── Отпуск ──────────────────────────────────────────────────────── */
// Отпуск живёт в членстве (user_companies.on_vacation): в этой компании человек
// отдыхает, в другой продолжает работать — поэтому подпись всегда её называет.
const vacationBusy = ref(false)
// Пока запрос в пути, тумблер показывает ЖЕЛАЕМОЕ положение: иначе он отскакивал
// бы назад до ответа сервера.
const vacationPending = ref(null)
const onVacation = computed(() => vacationPending.value ?? !!user.value?.on_vacation)

const vacationHint = computed(() => {
  const where = auth.companyName ? `в компании «${auth.companyName}»` : 'в активной компании'
  return onVacation.value
    ? `Вы отдыхаете ${where}: задачи и юниты закрыты, грувик на паузе`
    : `Задачи, юниты и уход за грувиком ${where} закроются до возвращения`
})

async function toggleVacation(on) {
  if (vacationBusy.value) return
  vacationBusy.value = true
  vacationPending.value = on
  try {
    await auth.setVacation(on)
    // Плавающий грувик уходит в отпуск вместе с хозяином — обновляем его метку
    // сразу. Только загруженного: GET создал бы питомца тому, кто их не заводил.
    if (pets.pet) pets.fetchPet().catch(() => {})
    notif.success(on
      ? 'Хорошего отдыха! Задачи подождут'
      : 'С возвращением — работа снова доступна')
  } catch (e) {
    notif.error(e?.message || 'Не удалось изменить режим отпуска')
  } finally {
    vacationPending.value = null
    vacationBusy.value = false
  }
}

/* ── Сеансы ──────────────────────────────────────────────────────── */
const sessions = ref([])
const sessionsLoading = ref(true)
const revokeTarget = ref(null)
const revoking = ref(0)
const revokeAllOpen = ref(false)
const revokingAll = ref(false)

// Кнопка «выйти на всех устройствах» нужна, только когда таких устройств
// больше одного: гнать самого себя неоткуда.
const otherSessions = computed(() => sessions.value.filter((s) => !s.current).length)

async function confirmRevokeAll() {
  revokingAll.value = true
  try {
    const res = await revokeOtherSessions()
    sessions.value = sessions.value.filter((s) => s.current)
    notif.success(res.revoked
      ? `Завершено сеансов: ${res.revoked}`
      : 'Других сеансов не было')
    revokeAllOpen.value = false
  } catch (e) {
    notif.error(e.message || 'Не удалось завершить сеансы')
  } finally {
    revokingAll.value = false
  }
}

async function loadSessions() {
  sessionsLoading.value = true
  try {
    const items = await listSessions()
    // Своё устройство — первым: с него чаще всего и завершают чужие сеансы.
    sessions.value = (items || []).sort((a, b) => Number(b.current) - Number(a.current))
  } catch (e) {
    notif.error(e.message || 'Не удалось загрузить список сеансов')
  } finally {
    sessionsLoading.value = false
  }
}

function platformIcon(platform) {
  if (platform === 'mobile') return 'smartphone'
  if (platform === 'desktop') return 'desktop_windows'
  return 'web_asset'
}

function formatLogin(iso) {
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return '—'
  return d.toLocaleString('ru-RU', {
    day: '2-digit', month: '2-digit', year: 'numeric', hour: '2-digit', minute: '2-digit',
  })
}

// «сегодня» / «вчера» / «16 мая» — в строку карточки, поэтому со строчной.
function formatSeen(iso) {
  const label = dayLabel(iso)
  return label ? label.charAt(0).toLowerCase() + label.slice(1) : '—'
}

const revokeSubtitle = computed(() => {
  const s = revokeTarget.value
  if (!s) return ''
  return s.current
    ? 'Это текущее устройство — вы выйдете из системы.'
    : `«${s.title}» перестанет открывать ваш аккаунт: на нём попросят войти заново.`
})

async function confirmRevoke() {
  const s = revokeTarget.value
  if (!s) return
  revoking.value = s.id
  try {
    await revokeSession(s.id)
    if (s.current) {
      // Свой сеанс уже недействителен — уходим на экран входа.
      await auth.logout()
      return
    }
    sessions.value = sessions.value.filter((x) => x.id !== s.id)
    notif.success('Сеанс завершён')
    revokeTarget.value = null
  } catch (e) {
    notif.error(e.message || 'Не удалось завершить сеанс')
  } finally {
    revoking.value = 0
  }
}

/* ── Подписка и внешние аккаунты ─────────────────────────────────── */
const yandex = ref({ enabled: false, client_id: '', linked: false })
const yandexBusy = ref(false)
const yandexUnlinkAsk = ref(false)
const yougile = ref({ connected: false, available: false })

// Тариф берём у биллинга — тот же источник, что у магазина, иначе профиль и
// витрина расходятся. Кнопка ведёт в магазин, где подписка и оформляется.
const plan = computed(() => {
  const name = billing.planName
  if (!name) return '—'
  return billing.expiresAt
    ? `${name} · до ${formatUntil(billing.expiresAt)}`
    : name
})

const links = computed(() => [
  // Подписка скрыта, пока не подключена оплата (см. utils/release.js): вести
  // человека в витрину, где нечего оформить, некуда.
  ...(SUBSCRIPTIONS_VISIBLE ? [{
    key: 'plan',
    label: 'Подписка',
    state: plan.value,
    action: () => router.push('/store?tab=subs'),
  }] : []),
  {
    key: 'yandex',
    label: 'Яндекс аккаунт',
    logo: 'yandex',
    state: !yandex.value.enabled ? 'недоступно' : yandex.value.linked ? 'подключено' : 'не настроено',
    action: yandex.value.enabled && !yandexBusy.value
      ? (yandex.value.linked ? () => { yandexUnlinkAsk.value = true } : linkYandex)
      : null,
  },
  {
    key: 'yougile',
    label: 'YouGile аккаунт',
    logo: 'yougile',
    state: !yougile.value.available ? 'недоступно' : yougile.value.connected ? 'подключено' : 'не настроено',
    action: yougile.value.available ? goToYougile : null,
  },
])

// Пилюля вела на /settings?section=general — панели, вообще не знающей про
// YouGile. Личное подключение (карточка ниже на этой же странице) — просто
// долистать до неё; компанийное настраивает только администратор в карточке
// компании, туда и ведём.
function goToYougile() {
  if (props.showYougile) {
    document.getElementById('acc-yougile-card')?.scrollIntoView({ behavior: 'smooth', block: 'start' })
    return
  }
  router.push(auth.companyId ? `/companies/${auth.companyId}?tab=settings&settingsTab=yougile` : '/companies')
}

function linkYandex() {
  window.location.href = yandexAuthURL(yandex.value.client_id, inAppShell() ? 'app-link' : 'link')
}

async function unlinkYandex() {
  yandexBusy.value = true
  try {
    await yandexUnlink()
    yandex.value.linked = false
    yandexUnlinkAsk.value = false
    notif.success('Яндекс-аккаунт отвязан')
  } catch (e) {
    notif.error(e?.message || 'Не удалось отвязать аккаунт')
  } finally {
    yandexBusy.value = false
  }
}

// Каждая интеграция грузится сама по себе: недоступная (нет компании, нет прав)
// гасит только свою пилюлю, а не весь ряд.
function loadLinks() {
  yandexConfig()
    .then(async (cfg) => {
      yandex.value = { ...yandex.value, ...cfg }
      if (cfg.enabled) yandex.value.linked = (await yandexLinkStatus()).linked
    })
    .catch(() => {})

  getYougileStatus()
    .then((st) => {
      yougile.value = { connected: !!st.connected, available: !!st.company_enabled || !!st.connected }
    })
    .catch(() => {})
}

onMounted(() => {
  loadSessions()
  loadLinks()
  // Тариф в карточке — тот же, что в магазине: общий стор, общий запрос.
  // Пока подписка скрыта, показывать нечего — и запрашивать тоже.
  if (SUBSCRIPTIONS_VISIBLE) billing.fetchShowcase()
})
</script>

<style scoped>
.acc {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

/* ── Кто я ───────────────────────────────────────────────────────── */
/* Строка, а не столбец: AppCard по умолчанию ставит содержимое в колонку,
   а здесь аватар, данные и кнопка правки стоят в ряд. Промежуток задаёт сама
   карточка (:gap) — её класс специфичнее, и отсюда его было не перебить. */
.acc-id {
  flex-direction: row;
  align-items: flex-start;
}

.acc-avatar {
  position: relative;
  flex-shrink: 0;
  width: 116px;
  height: 116px;
  padding: 0;
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-lg);
  background: var(--color-surface-low);
  overflow: hidden;
  cursor: zoom-in;
  transition: transform 0.18s, box-shadow 0.18s;
}

.acc-avatar:hover {
  box-shadow: var(--shadow-md);
}

.acc-avatar img {
  display: block;
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.acc-avatar-zoom {
  position: absolute;
  inset: 0;
  display: grid;
  place-items: center;
  background: color-mix(in oklch, var(--color-scrim) 55%, transparent);
  color: var(--color-on-primary);
  opacity: 0;
  transition: opacity 0.15s;
}

.acc-avatar:hover .acc-avatar-zoom { opacity: 1; }
.acc-avatar-zoom .material-symbols-outlined { font-size: 30px; }

.acc-id-body {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding-top: 4px;
}

.acc-name {
  margin: 0;
  font-size: 1.6rem;
  font-weight: 800;
  line-height: 1.15;
  letter-spacing: -0.4px;
  color: var(--color-primary);
  overflow-wrap: anywhere;
}

.acc-post {
  margin: 0;
  font-size: 0.86rem;
  color: var(--color-text-dim);
}

/* Контакты — пилюли-факты: их не редактируют отсюда, только читают. */
.acc-facts {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin: 4px 0 0;
  padding: 0;
  list-style: none;
}

.acc-facts li {
  padding: 7px 14px;
  border-radius: 999px;
  background: var(--color-surface-high);
  color: var(--color-text);
  font-size: 0.85rem;
  font-weight: 500;
}

.acc-facts li.empty { color: var(--color-text-dim); }

.acc-facts li.vacation {
  background: var(--color-secondary-container);
  color: var(--color-on-secondary-container);
  font-weight: 600;
}

.acc-edit {
  display: grid;
  place-items: center;
  flex-shrink: 0;
  width: 40px;
  min-width: 40px;
  max-width: 40px;
  height: 40px;
  min-height: 40px;
  max-height: 40px;
  padding: 0;
  border: 1px solid var(--acrylic-border);
  border-radius: 50%;
  background: var(--glass-bg), var(--acrylic-card-bg);
  box-shadow: var(--glass-edge);
  color: var(--color-text);
  cursor: pointer;
  transition: background 0.15s, border-color 0.15s;
}

.acc-edit:hover {
  background: var(--glass-hover-bg, var(--glass-bg)), var(--acrylic-card-bg);
  border-color: color-mix(in oklch, var(--color-primary) 30%, var(--acrylic-border));
}

.acc-edit .material-symbols-outlined { font-size: 20px; }

/* ── Подписка и внешние аккаунты ─────────────────────────────────── */
.acc-links {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
}

/* Пилюля внешнего аккаунта: тот же стеклянный чип, что и всюду, но во всю
   ширину колонки — справа у неё состояние подключения. */
.acc-link {
  flex: 1 1 240px;
  display: inline-flex;
  align-items: center;
  justify-content: flex-start;
  gap: 8px;
  padding: 8px 16px 8px 8px;
  border: 1px solid var(--acrylic-border);
  border-radius: 999px;
  background: var(--glass-bg), var(--acrylic-card-bg);
  box-shadow: var(--glass-edge);
  color: var(--color-text);
  font-size: 0.86rem;
  font-weight: 600;
  cursor: pointer;
  transition: background 0.2s ease;
}

.acc-link:hover:not(.is-static) {
  background: var(--glass-hover-bg, var(--glass-bg)), var(--acrylic-card-bg);
}

.acc-link.is-static { cursor: default; }

.acc-link-state {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--color-text-dim);
  font-weight: 500;
}

/* ── Сеансы ──────────────────────────────────────────────────────── */
.acc-sessions {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.sess-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(min(230px, 100%), 1fr));
  gap: 14px;
}

.sess-tile {
  padding: 16px;
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-lg);
  background: var(--color-surface);
  min-height: 164px;
}

.sess-add {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  color: var(--color-text);
  font-size: 0.87rem;
  font-weight: 600;
  line-height: 1.35;
  text-align: center;
  cursor: pointer;
  transition: border-color 0.15s, background 0.15s;
}

.sess-add:hover {
  background: var(--color-surface-low);
  border-color: color-mix(in oklch, var(--color-primary) 35%, var(--color-outline-dim));
}

.sess-add .material-symbols-outlined {
  font-size: 30px;
  color: var(--color-text-dim);
}

.sess-loading { display: grid; place-items: center; }

.sess-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.sess-head {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  margin-bottom: 6px;
}

.sess-ico {
  font-size: 30px;
  color: var(--color-text);
}

.sess-place {
  margin-left: auto;
  max-width: 60%;
  padding: 3px 10px;
  border-radius: var(--radius-full);
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
  font-size: 0.72rem;
  font-weight: 600;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sess-title {
  margin: 0;
  font-size: 0.94rem;
  font-weight: 700;
  color: var(--color-text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sess-meta {
  margin: 0;
  font-size: 0.78rem;
  line-height: 1.4;
  color: var(--color-text-dim);
}

.sess-current {
  color: var(--color-primary);
  font-weight: 600;
}

.sess-end {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  width: 100%;
  margin-top: auto;
  padding: 9px 12px;
  border: none;
  border-radius: var(--radius-sm);
  background: var(--color-error-container);
  color: var(--color-error);
  font-size: 0.83rem;
  font-weight: 600;
  cursor: pointer;
  transition: background 0.15s;
}

.sess-end:hover:not(:disabled) {
  background: color-mix(in oklch, var(--color-error) 22%, var(--color-error-container));
}

.sess-end:disabled { opacity: 0.6; cursor: not-allowed; }
.sess-end .material-symbols-outlined { font-size: 17px; }

/* ── Узкая панель настроек (ширина ОКНА, а не экрана) ────────────── */
@container (max-width: 560px) {
  .acc-id {
    flex-wrap: wrap;
    gap: 14px;
  }

  .acc-avatar { width: 88px; height: 88px; }
  .acc-name { font-size: 1.3rem; }
  .acc-link { flex-basis: 100%; }
  .sess-tile { min-height: 0; }
  .sess-grid { grid-template-columns: minmax(0, 1fr); }
}

/* Заводской WebView старых Android не знает @container — на телефоне окно
   всё равно во весь экран, поэтому дублируем правила media-запросом. */
@media (max-width: 560px) {
  .acc-id {
    flex-wrap: wrap;
    gap: 14px;
  }

  .acc-avatar { width: 88px; height: 88px; }
  .acc-name { font-size: 1.3rem; }
  .acc-link { flex-basis: 100%; }
  .sess-tile { min-height: 0; }
  .sess-grid { grid-template-columns: minmax(0, 1fr); }
}
</style>
