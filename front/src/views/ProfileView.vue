<template>
  <div class="profile-view">
    <div class="profile-container">
      <!-- Верхний ряд: кто я + мои интеграции -->
      <div class="top-row">
        <section class="card id-card">
          <button type="button" class="id-avatar" title="Открыть фото" @click="lightboxOpen = true">
            <img :src="avatarSrc" :alt="user?.fio" />
            <span class="id-avatar-zoom" aria-hidden="true">
              <span class="material-symbols-outlined">zoom_in</span>
            </span>
          </button>

          <div class="id-body">
            <h1 class="id-name">{{ user?.fio || '—' }}</h1>
            <p v-if="user?.post" class="id-post">{{ user.post }}</p>

            <ul class="id-contacts">
              <li :class="{ empty: !user?.email }">{{ user?.email || 'Email не указан' }}</li>
              <li :class="{ empty: !user?.phone }">{{ user?.phone || 'Телефон не указан' }}</li>
              <li v-if="user?.login">@{{ user.login }}</li>
            </ul>

            <div class="id-actions">
              <span v-if="user?.on_vacation" class="id-vacation">🏖️ В отпуске</span>
              <button type="button" class="btn-glass id-edit" @click="editOpen = true">
                <span class="material-symbols-outlined">edit</span>
                Редактировать
              </button>
            </div>
          </div>
        </section>

        <section class="card int-card">
          <h2 class="card-title">Интеграции</h2>
          <ul class="int-list">
            <li v-for="it in integrations" :key="it.key">
              <component
                :is="it.action ? 'button' : 'div'"
                class="int-row"
                :class="{ 'is-static': !it.action }"
                :type="it.action ? 'button' : undefined"
                @click="it.action && it.action()"
              >
                <span class="int-ico" :data-tone="it.tone">
                  <span v-if="it.badge" class="int-badge">{{ it.badge }}</span>
                  <span v-else class="material-symbols-outlined">{{ it.icon }}</span>
                </span>
                <span class="int-text">
                  {{ it.label }} <span class="int-state">({{ it.state }})</span>
                </span>
                <span v-if="it.action" class="material-symbols-outlined int-go">chevron_right</span>
              </component>
            </li>
          </ul>
        </section>
      </div>

      <!-- Авторизация и сессии -->
      <section class="card sess-card">
        <h2 class="card-title">Авторизация и сессии</h2>

        <div class="sess-grid">
          <button type="button" class="sess-tile sess-add" @click="showAuthorizeDevice = true">
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

            <h3 class="sess-title" :title="s.device">{{ s.title }}</h3>
            <p class="sess-meta">Вход: {{ formatLogin(s.created_at) }}</p>
            <p class="sess-meta">
              Последняя активность: {{ formatSeen(s.last_seen_at) }}
              <span v-if="s.current" class="sess-current">· это устройство</span>
            </p>

            <button type="button" class="sess-end" :disabled="revoking === s.id" @click="askRevoke(s)">
              <span class="material-symbols-outlined">logout</span>
              Завершить сеанс
            </button>
          </article>
        </div>
      </section>
    </div>

    <ProfileEditDialog v-model="editOpen" />
    <AuthorizeDeviceDialog v-model="showAuthorizeDevice" />

    <ImageLightbox v-model="lightboxOpen" :src="avatarSrc" :caption="user?.fio" />

    <AppDialog
      :model-value="!!revokeTarget"
      tone="danger"
      icon="logout"
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
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { usePermission } from '@/composables/usePermission.js'
import { inAppShell } from '@/utils/appShell.js'
import { dayLabel } from '@/utils/chatDates.js'
import {
  listSessions,
  revokeSession,
  yandexConfig,
  yandexAuthURL,
  yandexLinkStatus,
  yandexUnlink,
} from '@/api/auth.js'
import { getAiStatus } from '@/api/ai.js'
import { getYougileStatus } from '@/api/yougile.js'
import AppDialog from '@/components/common/AppDialog.vue'
import ImageLightbox from '@/components/common/ImageLightbox.vue'
import BrandLoader from '@/components/common/BrandLoader.vue'
import AuthorizeDeviceDialog from '@/components/devicelink/AuthorizeDeviceDialog.vue'
import ProfileEditDialog from '@/components/profile/ProfileEditDialog.vue'

const auth = useAuthStore()
const notif = useNotificationsStore()
const router = useRouter()
const { isAdmin } = usePermission()

const user = computed(() => auth.user)
const avatarSrc = computed(() => {
  const u = auth.user
  if (!u) return ''
  return u.avatar_path ? `/uploads/${u.avatar_path}` : `/api/users/${u.id}/identicon`
})

const editOpen = ref(false)
const lightboxOpen = ref(false)
const showAuthorizeDevice = ref(false)

// ── Сессии ───────────────────────────────────────────────────────
const sessions = ref([])
const sessionsLoading = ref(true)
const revokeTarget = ref(null)
const revoking = ref(0)

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

function askRevoke(s) {
  revokeTarget.value = s
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

// ── Интеграции ───────────────────────────────────────────────────
const yandex = ref({ enabled: false, client_id: '', linked: false })
const yandexBusy = ref(false)
const yougile = ref({ connected: false, available: false })
const aiEnabled = ref(false)

const integrations = computed(() => [
  {
    key: 'yandex',
    label: 'Яндекс аккаунт',
    badge: 'Я',
    tone: 'error',
    state: !yandex.value.enabled ? 'недоступно' : yandex.value.linked ? 'подключено' : 'не настроено',
    action: yandex.value.enabled && !yandexBusy.value
      ? (yandex.value.linked ? unlinkYandex : linkYandex)
      : null,
  },
  {
    key: 'yougile',
    label: 'YouGile аккаунт',
    icon: 'sync_alt',
    tone: 'secondary',
    state: !yougile.value.available ? 'недоступно' : yougile.value.connected ? 'подключено' : 'не настроено',
    action: yougile.value.available ? () => router.push('/settings?section=yougile') : null,
  },
  {
    key: 'ai',
    label: 'ИИ ассистент',
    icon: 'smart_toy',
    tone: 'primary',
    state: aiEnabled.value ? 'подключено' : 'не настроено',
    // Ключ ИИ задаёт администратор компании — остальным строка справочная.
    action: isAdmin() && auth.companyId ? () => router.push(`/companies/${auth.companyId}`) : null,
  },
])

function linkYandex() {
  window.location.href = yandexAuthURL(yandex.value.client_id, inAppShell() ? 'app-link' : 'link')
}

async function unlinkYandex() {
  yandexBusy.value = true
  try {
    await yandexUnlink()
    yandex.value.linked = false
    notif.success('Яндекс-аккаунт отвязан')
  } catch (e) {
    notif.error(e?.message || 'Не удалось отвязать аккаунт')
  } finally {
    yandexBusy.value = false
  }
}

// Каждая интеграция грузится сама по себе: недоступная (нет компании, нет прав)
// гасит только свою строку, а не всю карточку.
async function loadIntegrations() {
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

  getAiStatus()
    .then((st) => { aiEnabled.value = !!st.enabled })
    .catch(() => {})
}

onMounted(() => {
  loadSessions()
  loadIntegrations()
})
</script>

<style scoped>
/* Раздел живёт в окне рабочего стола, поэтому раскладка считается от ЕГО
   ширины (container queries), а не от ширины экрана: media-запросы здесь
   ничего бы не знали про размер окна и карточки сжимались бы в кашу. */
.profile-view {
  padding: 24px;
  height: 100%;
  overflow-y: auto;
  container-type: inline-size;
  container-name: profile;
}

.profile-container {
  max-width: 1280px;
  margin: 0 auto;
  display: flex;
  flex-direction: column;
  gap: 20px;
}

/* Крупные акриловые панели-блоки: карточка «кто я» + интеграции, ниже сеансы. */
.card {
  background: var(--acrylic-card-bg);
  -webkit-backdrop-filter: var(--acrylic-blur);
  backdrop-filter: var(--acrylic-blur);
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-xl);
  padding: 22px;
}

.card-title {
  margin: 0 0 16px;
  font-size: 19px;
  font-weight: 800;
  letter-spacing: -0.2px;
  color: var(--color-text);
}

.top-row {
  display: grid;
  grid-template-columns: minmax(0, 1.55fr) minmax(0, 1fr);
  gap: 20px;
  align-items: stretch;
}

/* ── Кто я ───────────────────────────────────────────────────────── */
.id-card {
  display: flex;
  gap: 22px;
  align-items: flex-start;
}

.id-avatar {
  flex-shrink: 0;
  position: relative;
  width: 152px;
  height: 152px;
  padding: 0;
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-lg);
  overflow: hidden;
  background: var(--color-surface-low);
  cursor: zoom-in;
  transition: transform 0.18s, box-shadow 0.18s;
}

.id-avatar:hover {
  transform: scale(1.02);
  box-shadow: var(--shadow-md);
}

.id-avatar img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}

.id-avatar-zoom {
  position: absolute;
  inset: 0;
  display: grid;
  place-items: center;
  background: color-mix(in oklch, var(--color-scrim) 55%, transparent);
  color: var(--color-on-primary);
  opacity: 0;
  transition: opacity 0.15s;
}
.id-avatar:hover .id-avatar-zoom { opacity: 1; }
.id-avatar-zoom .material-symbols-outlined { font-size: 30px; }

.id-body {
  min-width: 0;
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 8px;
  align-self: stretch;
}

.id-name {
  margin: 0;
  font-size: 26px;
  font-weight: 800;
  line-height: 1.15;
  letter-spacing: -0.4px;
  color: var(--color-primary);
  overflow-wrap: anywhere;
}

.id-post {
  margin: 0;
  font-size: 14px;
  color: var(--color-text-dim);
}

.id-contacts {
  list-style: none;
  margin: 4px 0 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.id-contacts li {
  font-size: 15px;
  color: var(--color-text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.id-contacts li.empty { color: var(--color-text-dim); }

/* Кнопка правки прижата к нижнему правому углу карточки — как в макете. */
.id-actions {
  margin-top: auto;
  padding-top: 12px;
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 10px;
  flex-wrap: wrap;
}

.id-vacation {
  margin-right: auto;
  padding: 5px 12px;
  border-radius: var(--radius-full);
  background: var(--color-secondary-container);
  color: var(--color-on-secondary-container);
  font-size: 12.5px;
  font-weight: 600;
}

.id-edit {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 9px 18px;
}
.id-edit .material-symbols-outlined { font-size: 18px; }

/* ── Интеграции ──────────────────────────────────────────────────── */
.int-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.int-row {
  width: 100%;
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 14px;
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-lg);
  background: var(--color-surface);
  color: var(--color-text);
  font-size: 14.5px;
  text-align: left;
  cursor: pointer;
  transition: border-color 0.15s, background 0.15s;
}

.int-row:hover:not(.is-static) {
  background: var(--color-surface-low);
  border-color: color-mix(in oklch, var(--color-primary) 35%, var(--color-outline-dim));
}

.int-row.is-static { cursor: default; }

.int-ico {
  flex-shrink: 0;
  width: 32px;
  height: 32px;
  border-radius: var(--radius-sm);
  display: grid;
  place-items: center;
  background: var(--tone-bg, var(--color-primary-container));
  color: var(--tone-fg, var(--color-on-primary-container));
}
.int-ico[data-tone="primary"]   { --tone-bg: var(--color-primary-container);   --tone-fg: var(--color-on-primary-container); }
.int-ico[data-tone="secondary"] { --tone-bg: var(--color-secondary-container); --tone-fg: var(--color-on-secondary-container); }
.int-ico[data-tone="error"]     { --tone-bg: var(--color-error-container);     --tone-fg: var(--color-error); }
.int-ico .material-symbols-outlined { font-size: 19px; }

.int-badge {
  font-size: 15px;
  font-weight: 800;
  line-height: 1;
}

.int-text {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.int-state { color: var(--color-text-dim); }

.int-go {
  margin-left: auto;
  font-size: 20px;
  color: var(--color-text-dim);
}

/* ── Сеансы ──────────────────────────────────────────────────────── */
.sess-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
  gap: 16px;
}

.sess-tile {
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-lg);
  background: var(--color-surface);
  padding: 16px;
  min-height: 168px;
}

.sess-add {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  color: var(--color-text);
  font-size: 14px;
  font-weight: 600;
  text-align: center;
  line-height: 1.35;
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
  font-size: 11.5px;
  font-weight: 600;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sess-title {
  margin: 0;
  font-size: 15px;
  font-weight: 700;
  color: var(--color-text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sess-meta {
  margin: 0;
  font-size: 12.5px;
  color: var(--color-text-dim);
  line-height: 1.4;
}

.sess-current {
  color: var(--color-primary);
  font-weight: 600;
}

.sess-end {
  margin-top: auto;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  width: 100%;
  padding: 9px 12px;
  border: none;
  border-radius: var(--radius-sm);
  background: var(--color-error-container);
  color: var(--color-error);
  font-size: 13.5px;
  font-weight: 600;
  cursor: pointer;
  transition: background 0.15s;
}

.sess-end:hover:not(:disabled) {
  background: color-mix(in oklch, var(--color-error) 22%, var(--color-error-container));
}

.sess-end:disabled { opacity: 0.6; cursor: not-allowed; }
.sess-end .material-symbols-outlined { font-size: 17px; }

/* ── Узкое окно: верхний ряд схлопывается в колонку ──────────────── */
/* Двум карточкам нужно ~940px, дальше они начинают жать содержимое —
   складываем их друг под друга, а не сжимаем. */
@container profile (max-width: 940px) {
  .top-row { grid-template-columns: minmax(0, 1fr); }
}

/* Ещё уже — карточка «кто я» становится вертикальной, сетка сеансов в один
   столбец (так же выглядит раздел и на телефоне). */
@container profile (max-width: 620px) {
  .card { padding: 16px; border-radius: var(--radius-lg); }

  .id-card {
    flex-direction: column;
    align-items: center;
    text-align: center;
    gap: 16px;
  }

  .id-avatar { width: 120px; height: 120px; }

  .id-body { align-items: center; width: 100%; }

  .id-contacts li { white-space: normal; }

  .id-actions { justify-content: center; width: 100%; }

  .id-vacation { margin-right: 0; }

  .sess-tile { min-height: 0; }
}

/* Сетка сеансов подстраивается сама (auto-fill), но у совсем узкого окна
   минимальная ширина плитки была бы больше самой сетки. */
@container profile (max-width: 420px) {
  .sess-grid { grid-template-columns: minmax(0, 1fr); }
}

/* Мобильный каркас: отступы — вопрос экрана, а не окна (контент уезжает под
   нижнюю навигацию). Узкая раскладка здесь ПОВТОРЕНА media-запросом намеренно:
   заводской WebView старых Android (см. build.target в vite.config.js) не знает
   @container и без этого показал бы телефону десктопный ряд в две колонки. */
@media (max-width: 768px) {
  .profile-view {
    padding: 12px;
    /* Резерв под нижнюю навигацию (64px) + воздух. */
    padding-bottom: calc(76px + env(safe-area-inset-bottom, 0px));
  }

  .profile-container { gap: 16px; }

  .top-row { grid-template-columns: minmax(0, 1fr); }

  .card { padding: 16px; border-radius: var(--radius-lg); }

  .id-card {
    flex-direction: column;
    align-items: center;
    text-align: center;
    gap: 16px;
  }

  .id-avatar { width: 120px; height: 120px; }

  .id-body { align-items: center; width: 100%; }

  .id-contacts li { white-space: normal; }

  .id-actions { justify-content: center; width: 100%; }

  .id-vacation { margin-right: 0; }

  .sess-grid { grid-template-columns: minmax(0, 1fr); }

  .sess-tile { min-height: 0; }
}
</style>
