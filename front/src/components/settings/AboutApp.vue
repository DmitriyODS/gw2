<template>
  <div class="about-app">
    <div class="about-hero">
      <!-- Волна Groove — фирменный мотив логотипа в движении (как на /promo). -->
      <div class="about-waves" aria-hidden="true">
        <div class="about-wave about-wave--soft"><svg viewBox="0 0 2880 140" preserveAspectRatio="none"><path :d="WAVE_PATH" /></svg></div>
        <div class="about-wave about-wave--mid"><svg viewBox="0 0 2880 140" preserveAspectRatio="none"><path :d="WAVE_PATH" /></svg></div>
        <div class="about-wave about-wave--deep"><svg viewBox="0 0 2880 140" preserveAspectRatio="none"><path :d="WAVE_PATH" /></svg></div>
      </div>
      <div class="about-logo">
        <Logo :size="56" />
      </div>
      <div class="about-hero-text">
        <h3>Groove Work</h3>
        <p class="about-tagline">Внутренняя платформа учёта задач, времени и общения команды.</p>
        <div class="about-version">
          <span class="version-badge" v-if="appVersion">v{{ appVersion }}</span>
          <button class="version-link" @click="changelog.open()">
            <span class="material-symbols-outlined">history</span>
            Что нового
          </button>
        </div>
      </div>
    </div>

    <!-- Внутри самого мобильного приложения карточка не нужна. -->
    <a v-if="showApkCard" class="about-mobile" href="/apps/mobile/groovework.apk" :download="apkDownloadName">
      <div class="about-mobile-icon">
        <span class="material-symbols-outlined">android</span>
      </div>
      <div class="about-mobile-text">
        <h4>Мобильное приложение для Android</h4>
        <p>Установите Groove Work на смартфон — задачи, юниты, чаты и звонки всегда под рукой.</p>
      </div>
      <span class="about-mobile-btn">
        <span class="material-symbols-outlined">download</span>
        Скачать APK
      </span>
    </a>

    <!-- Внутри обёрток (мобильная Capacitor / десктопный Electron) — обновление
         самого приложения: принудительная проверка без ожидания автопроверки,
         скачивание и установка нативно. -->
    <div v-if="hasShellUpdate" class="about-mobile">
      <div class="about-mobile-icon">
        <span class="material-symbols-outlined">system_update</span>
      </div>
      <div class="about-mobile-text">
        <h4>Обновление приложения</h4>
        <p v-if="appBuild">
          Установлена {{ appBuild }}<template v-if="updateInfo">
            · {{ updateInfo.updateAvailable ? `доступна ${updateInfo.latest}` : 'это последняя версия' }}</template>
        </p>
        <p v-else>Оболочка Groove Work</p>
      </div>
      <button
        class="about-mobile-btn about-update-btn"
        :class="{ downloading: updProgress != null && updProgress >= 0 }"
        :style="updateBtnStyle"
        :disabled="updBusy"
        @click="onUpdateClick"
      >
        <span class="material-symbols-outlined">{{ updateInfo?.updateAvailable ? 'download' : 'refresh' }}</span>
        {{ updateBtnLabel }}
      </button>
    </div>

    <!-- Внутри самого десктоп-приложения и на мобильных карточка не нужна. -->
    <div v-if="showDesktopCard" class="about-mobile about-desktop">
      <div class="about-mobile-icon">
        <span class="material-symbols-outlined">desktop_windows</span>
      </div>
      <div class="about-mobile-text">
        <h4>Приложение для компьютера</h4>
        <p>Groove Work отдельным окном: иконка в трее, системные уведомления и звонки — даже когда браузер закрыт.</p>
        <p class="about-desktop-links">
          Все платформы:
          <a :href="desktopFileHref('mac')" download>macOS</a> ·
          <a :href="desktopFileHref('win')" download>Windows</a> ·
          <a :href="desktopFileHref('linux')" download>Linux</a>
        </p>
      </div>
      <a class="about-mobile-btn" :href="desktopFileHref(desktopOs)" download>
        <span class="material-symbols-outlined">download</span>
        Скачать для {{ DESKTOP_OS_LABELS[desktopOs] }}
      </a>
    </div>

    <div class="about-grid">
      <button class="about-card" @click="openSupport" :disabled="opening">
        <div class="about-card-icon" data-tone="primary">
          <span class="material-symbols-outlined">support_agent</span>
        </div>
        <div class="about-card-text">
          <h4>Чат с техподдержкой</h4>
          <p>Личный диалог с командой разработчиков. Опишите проблему или предложите улучшение — ответ придёт прямо сюда, в мессенджер.</p>
        </div>
        <span class="material-symbols-outlined about-card-chev">chevron_right</span>
      </button>

      <button class="about-card" @click="tutorial.open()">
        <div class="about-card-icon" data-tone="secondary">
          <span class="material-symbols-outlined">tour</span>
        </div>
        <div class="about-card-text">
          <h4>Пройти тур по платформе</h4>
          <p>Короткое знакомство с разделами и горячими действиями.</p>
        </div>
        <span class="material-symbols-outlined about-card-chev">chevron_right</span>
      </button>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { getNativeBuild, checkNativeUpdate, installNativeUpdate } from '@/utils/nativeApp.js'
import { useMessengerStore } from '@/stores/messenger.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { useChangelog } from '@/composables/useChangelog.js'
import { useTutorial } from '@/composables/useTutorial.js'
import Logo from '@/components/common/Logo.vue'
import { WAVE_PATH } from '@/utils/wavePath.js'

const router = useRouter()
const messenger = useMessengerStore()
const notif = useNotificationsStore()
const changelog = useChangelog()
const tutorial = useTutorial()

// Версию продукта берём только с сервера (первая запись changelog), не из бандла.
const appVersion = changelog.latestVersion
onMounted(changelog.loadLatest)

/* Десктоп-клиент: артефакты лежат в /apps/desktop/ (заливает make
   deploy-desktop). Имена файлов СОДЕРЖАТ ВЕРСИЮ и приезжают картой files из
   version.json — скачавший всегда видит, какая версия у него в загрузках
   (безымянные имена однажды дали раздачу старого установщика из кэша).
   Кнопка предлагает сборку под ОС посетителя, остальные — ссылками. */
const desktopFiles = ref({
  mac: 'GrooveWork-mac.dmg',
  win: 'GrooveWork-win.exe',
  linux: 'GrooveWork-linux.AppImage',
})
const DESKTOP_OS_LABELS = { mac: 'macOS', win: 'Windows', linux: 'Linux' }
const desktopFileHref = (os) => `/apps/desktop/${desktopFiles.value[os]}`

// Имя сохраняемого APK — с номером сборки (сам файл на сервере канонический
// groovework.apk: его же качает автообновление старых обёрток).
const apkDownloadName = ref('groovework.apk')

onMounted(async () => {
  try {
    const meta = await (await fetch('/apps/desktop/version.json', { cache: 'no-store' })).json()
    if (meta?.files?.mac) desktopFiles.value = meta.files
  } catch { /* карта не приехала — останутся легаси-имена */ }
  try {
    const meta = await (await fetch('/apps/mobile/version.json', { cache: 'no-store' })).json()
    if (meta?.current_build) apkDownloadName.value = `groovework-${meta.current_build}.apk`
  } catch { /* noop */ }
})

const ua = navigator.userAgent
const desktopOs = /Mac/i.test(navigator.platform || ua) ? 'mac' : /Win/i.test(navigator.platform || ua) ? 'win' : 'linux'
// В самом Electron-клиенте и на телефонах предлагать установщик бессмысленно.
const showDesktopCard = !/Electron/i.test(ua) && !/Android|iPhone|iPad/i.test(ua)
// Внутри мобильной обёртки (Capacitor) карточка скачивания APK не нужна —
// приложение уже установлено. Признаки: инжектированный мост window.Capacitor
// (надёжный) и метка GrooveWorkApp в UA (appendUserAgent, страховка).
const showApkCard = !window.Capacitor?.isNativePlatform?.() && !/GrooveWorkApp/i.test(ua)

/* Обновление обёртки изнутри приложения. Мобильная (Capacitor) — нативный
   плагин NativeShell (сборки 2607104+); десктопная (Electron) — мост
   window.GrooveDesktop из preload (версии 1.0.2+). Обвязка общая, различается
   только транспорт. */
const hasNativeShell = !!window.Capacitor?.Plugins?.NativeShell
const desktopShell = window.GrooveDesktop
const hasShellUpdate = hasNativeShell || !!desktopShell
const appBuild = ref(null)
const updateInfo = ref(null)
const updBusy = ref(false)
const updProgress = ref(null)

onMounted(async () => {
  if (hasNativeShell) {
    appBuild.value = `сборка ${await getNativeBuild()}`
  } else if (desktopShell) {
    const { version } = await desktopShell.getVersion().catch(() => ({}))
    if (version) appBuild.value = `версия ${version}`
  }
})

// Десктопный мост сообщает об ошибках полем error — приводим к исключению,
// как у мобильного плагина.
async function shellCheck() {
  if (hasNativeShell) return checkNativeUpdate()
  const r = await desktopShell.checkUpdate()
  if (r?.error) throw new Error(r.error)
  return r
}

async function shellInstall(onProgress) {
  if (hasNativeShell) return installNativeUpdate(onProgress)
  const r = await desktopShell.downloadUpdate(onProgress)
  if (r?.error) throw new Error(r.error)
  return r
}

const updateBtnLabel = computed(() => {
  if (updBusy.value && updProgress.value != null) {
    return updProgress.value >= 0 ? `${Math.round(updProgress.value * 100)}%` : 'Скачивание…'
  }
  if (updBusy.value) return 'Проверяем…'
  if (updateInfo.value?.updateAvailable) return 'Обновить'
  return 'Проверить обновления'
})

// Кнопка-прогресс: пока идёт скачивание, кнопка заливается цветом слева
// направо по проценту (--dl), поверх — сам процент.
const updateBtnStyle = computed(() => {
  if (updProgress.value == null || updProgress.value < 0) return {}
  return { '--dl': `${Math.round(updProgress.value * 100)}%` }
})

async function onUpdateClick() {
  updBusy.value = true
  try {
    if (updateInfo.value?.updateAvailable) {
      updProgress.value = -1
      const { status } = await shellInstall((p) => { updProgress.value = p })
      if (status === 'needs_permission') {
        notif.notify({
          severity: 'info',
          summary: 'Нужно разрешение',
          detail: 'Разрешите установку из этого источника в открывшихся настройках и нажмите «Обновить» ещё раз.',
          life: 9000,
        })
      }
    } else {
      updateInfo.value = await shellCheck()
    }
  } catch (e) {
    notif.error(e?.message || 'Не удалось проверить обновления')
  } finally {
    updBusy.value = false
    updProgress.value = null
  }
}

const opening = ref(false)

async function openSupport() {
  opening.value = true
  try {
    // Открываем личный dev-чат пользователя — бэк создаёт его при первом
    // обращении. id возвращается сразу, навигация — на /messenger/<id>.
    const convId = await messenger.openDevChat()
    router.push(`/messenger/${convId}`)
  } catch (e) {
    notif.error(e.message || 'Не удалось открыть чат техподдержки')
  } finally {
    opening.value = false
  }
}
</script>

<style scoped>
.about-app {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

/* Матовый стеклянный hero — цвет несут только волны внизу. */
.about-hero {
  display: flex;
  align-items: center;
  gap: 24px;
  padding: 24px;
  background: var(--acrylic-card-bg);
  background: var(--glass-bg);
  box-shadow: var(--glass-edge);
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-xl);
  position: relative;
  overflow: hidden;
}

/* Контент hero — над волнами. */
.about-hero > :not(.about-waves) { position: relative; z-index: 1; }

/* Волны — нижняя треть hero, медленный бесшовный дрейф (ширина 200%,
   сдвиг на половину), кверху растворяются маской. */
.about-waves {
  position: absolute;
  inset: auto 0 0 0;
  height: 68%;
  pointer-events: none;
  -webkit-mask-image: linear-gradient(180deg, transparent 0%, black 70%);
  mask-image: linear-gradient(180deg, transparent 0%, black 70%);
}
.about-wave {
  position: absolute;
  inset: 0;
  /* Не уже 2200px: на узком hero слой 200% давал слишком частый период —
     волны «скукоживались» в рябь. Сдвиг на 50% слоя бесшовен при любой
     ширине (в половину укладывается целое число периодов). */
  width: max(200%, 2200px);
  animation: about-wave-drift linear infinite;
}
.about-wave svg { width: 100%; height: 100%; display: block; }
.about-wave--soft { animation-duration: 30s; }
.about-wave--soft path { fill: var(--color-primary-container); opacity: 0.3; }
.about-wave--mid { animation-duration: 20s; animation-delay: -6s; top: 14%; }
.about-wave--mid path { fill: color-mix(in oklch, var(--color-primary) 55%, var(--color-tertiary-container)); opacity: 0.16; }
.about-wave--deep { animation-duration: 13s; animation-delay: -3s; top: 30%; }
.about-wave--deep path { fill: var(--color-primary); opacity: 0.18; }

@keyframes about-wave-drift {
  from { transform: translateX(0); }
  to { transform: translateX(-50%); }
}

@media (prefers-reduced-motion: reduce) {
  .about-wave { animation: none; }
}

/* Подложка круглая, как сама эмблема — иначе на мобильной колонке hero
   логотип читался квадратом. */
.about-logo {
  width: 84px;
  height: 84px;
  border-radius: 50%;
  background: var(--acrylic-card-bg);
  background: var(--glass-bg);
  box-shadow: var(--glass-edge);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}


.about-hero-text {
  flex: 1;
  min-width: 0;
}

.about-hero-text h3 {
  margin: 0 0 4px;
  font-size: 22px;
  font-weight: 800;
  letter-spacing: -0.3px;
  color: var(--color-text);
}

.about-tagline {
  margin: 0 0 12px;
  font-size: 14px;
  color: var(--color-text-dim);
}

.about-version {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}

.version-badge {
  display: inline-flex;
  align-items: center;
  padding: 4px 12px;
  border-radius: var(--radius-full);
  background: var(--acrylic-card-bg);
  color: var(--color-text);
  font-size: 13px;
  font-weight: 700;
  letter-spacing: 0.3px;
}

.version-link {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 12px;
  border-radius: var(--radius-full);
  background: transparent;
  border: 1px solid var(--acrylic-border);
  color: var(--color-text);
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  transition: background 0.15s;
}

.version-link:hover {
  background: var(--glass-bg);
}

.version-link .material-symbols-outlined {
  font-size: 16px;
}

.about-mobile {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 18px 20px;
  border-radius: var(--radius-lg);
  background: var(--acrylic-card-bg);
  background: var(--glass-bg);
  box-shadow: var(--glass-edge);
  border: 1px solid var(--acrylic-border);
  text-decoration: none;
  transition: box-shadow 0.15s;
}

.about-mobile:hover {
  box-shadow: var(--glass-edge), var(--shadow-sm);
}

.about-mobile-icon {
  width: 48px;
  height: 48px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: var(--radius-md);
  background: var(--acrylic-card-bg);
  color: var(--color-tertiary);
  flex-shrink: 0;
}

.about-mobile-icon .material-symbols-outlined { font-size: 28px; }

.about-mobile-text { flex: 1; min-width: 0; }

.about-mobile-text h4 {
  margin: 0 0 2px;
  font-size: 15px;
  font-weight: 700;
  color: var(--color-text);
}

.about-mobile-text p {
  margin: 0;
  font-size: 13px;
  line-height: 1.35;
  color: var(--color-text-dim);
}

.about-mobile-btn {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 10px 20px;
  border-radius: var(--radius-full);
  /* Единственный цветовой акцент карточки — кнопка на градиенте. */
  background: var(--color-primary);
  background: var(--grad-primary);
  color: var(--color-on-primary);
  font-size: 14px;
  font-weight: 600;
  flex-shrink: 0;
  /* Кнопка десктоп-карточки — ссылка <a>: гасим подчёркивание. */
  text-decoration: none;
}

.about-mobile-btn .material-symbols-outlined { font-size: 18px; }

/* Кнопка обновления обёртки — тот же вид, но это <button>. */
.about-update-btn {
  border: 0;
  cursor: pointer;
  font-family: inherit;
}
.about-update-btn:disabled { opacity: 0.6; cursor: progress; }

/* Скачивание: кнопка становится прогресс-баром — заполняется цветом по
   проценту (--dl), текст показывает процент. */
.about-update-btn.downloading {
  opacity: 1;
  min-width: 160px;
  justify-content: center;
  background:
    linear-gradient(90deg,
      var(--color-primary) 0%,
      var(--color-primary) var(--dl, 0%),
      color-mix(in oklch, var(--color-primary) 25%, var(--color-surface)) var(--dl, 0%),
      color-mix(in oklch, var(--color-primary) 25%, var(--color-surface)) 100%);
  color: var(--color-on-primary);
  transition: background 0.2s linear;
}

/* Карточка десктоп-клиента — тот же матовый каркас, отличается иконкой. */
.about-desktop .about-mobile-icon { color: var(--color-secondary); }

.about-desktop-links,
.about-desktop-links a {
  color: var(--color-text-dim);
}

.about-mobile-text p.about-desktop-links {
  margin-top: 6px;
  font-size: 12px;
  opacity: 0.75;
}

.about-desktop-links a {
  font-weight: 600;
  text-decoration: underline;
  text-underline-offset: 2px;
}

.about-grid {
  display: grid;
  grid-template-columns: 1fr;
  gap: 12px;
}

.about-card {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 16px 20px;
  border-radius: var(--radius-lg);
  background: var(--acrylic-card-bg);
  background: var(--glass-bg);
  box-shadow: var(--glass-edge);
  border: 1px solid var(--acrylic-border);
  cursor: pointer;
  text-align: left;
  transition: background 0.15s, box-shadow 0.15s;
}

.about-card:not(:disabled):hover {
  background: var(--color-surface-low);
  box-shadow: var(--glass-edge), var(--shadow-sm);
}

.about-card:disabled { opacity: 0.6; cursor: progress; }

.about-card-icon {
  width: 44px;
  height: 44px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: var(--radius-md);
  flex-shrink: 0;
}

.about-card-icon[data-tone="primary"] {
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
}

.about-card-icon[data-tone="secondary"] {
  background: var(--color-secondary-container);
  color: var(--color-on-secondary-container);
}

.about-card-icon .material-symbols-outlined { font-size: 24px; }

.about-card-text { flex: 1; min-width: 0; }

.about-card-text h4 {
  margin: 0 0 2px;
  font-size: 15px;
  font-weight: 700;
  color: var(--color-text);
}

.about-card-text p {
  margin: 0;
  font-size: 13px;
  color: var(--color-text-dim);
  line-height: 1.35;
}

.about-card-chev {
  color: var(--color-text-dim);
  font-size: 22px;
  flex-shrink: 0;
}

@media (max-width: 600px) {
  .about-hero {
    flex-direction: column;
    text-align: center;
    align-items: center;
  }
  .about-version { justify-content: center; }
  .about-card { padding: 14px; }
  .about-mobile { flex-direction: column; text-align: center; }
  .about-mobile-btn { width: 100%; justify-content: center; }
}
</style>
