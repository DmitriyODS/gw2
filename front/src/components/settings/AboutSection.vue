<template>
  <!-- Устройство — как у «Об этом Mac» и «О системе» Windows: крупная марка с
       версией, под ней плоский список сведений «название → значение», дальше —
       что нового и действия с приложениями. Каждый блок про одно. -->
  <div class="ab">
    <!-- ── Марка: фирменные волны за стеклом ── -->
    <section class="ab-hero">
      <!-- Волна Groove (мотив логотипа) дрейфует ЗА акриловым стеклом:
           слой волн → матовая пелена → контент. -->
      <div class="ab-waves" aria-hidden="true">
        <div class="ab-wave ab-wave--soft"><svg viewBox="0 0 2880 140" preserveAspectRatio="none"><path :d="WAVE_PATH" /></svg></div>
        <div class="ab-wave ab-wave--mid"><svg viewBox="0 0 2880 140" preserveAspectRatio="none"><path :d="WAVE_PATH" /></svg></div>
        <div class="ab-wave ab-wave--deep"><svg viewBox="0 0 2880 140" preserveAspectRatio="none"><path :d="WAVE_PATH" /></svg></div>
      </div>
      <div class="ab-frost" aria-hidden="true" />

      <BrandLogo :size="72" class="ab-logo" />
      <h3 class="ab-name">
        <span>Groove Work</span>
        <span v-if="majorVersion" class="ab-name-major">{{ majorVersion }}</span>
      </h3>
      <p class="ab-tagline">Ваш timeline в удобном интерфейсе</p>
    </section>

    <!-- ── Сведения: одна строка — один факт ── -->
    <AppCard title="Сведения" :gap="6">
      <dl class="ab-facts">
        <div v-if="appVersion" class="ab-fact">
          <dt>Версия</dt>
          <dd>{{ appVersion }}</dd>
        </div>
        <!-- Номер сборки — вход в скрытый раздел разработчика: пять быстрых
             нажатий открывают «Настройки → DevTools» (приём мобильных ОС). -->
        <div v-if="appBuild" class="ab-fact ab-fact-tap" @click="onBuildTap">
          <dt>Сборка</dt>
          <dd>{{ appBuild }}</dd>
        </div>
        <div v-if="releaseDate" class="ab-fact">
          <dt>Дата выпуска</dt>
          <dd>{{ releaseDate }}</dd>
        </div>
        <div v-if="hasShellUpdate" class="ab-fact">
          <dt>{{ shellLabel }}</dt>
          <dd>
            {{ shellBuild || '—' }}
            <small v-if="updateInfo" class="ab-fact-note">
              {{ updateInfo.updateAvailable ? `доступна ${updateInfo.latest}` : 'последняя версия' }}
            </small>
          </dd>
        </div>
      </dl>

      <div v-if="hasShellUpdate" class="ab-actions">
        <AppButton
          :variant="updateInfo?.updateAvailable ? 'filled' : 'glass'"
          :icon="updateInfo?.updateAvailable ? 'download' : 'refresh'"
          :label="updateBtnLabel"
          :disabled="updBusy"
          @click="onUpdateClick"
        />
      </div>
    </AppCard>

    <!-- ── Что нового: только текущий выпуск, истории версий нет ── -->
    <AppCard v-if="release" :title="newsTitle" :hint="release.title">
      <p v-if="release.description" class="ab-news-text">{{ release.description }}</p>
      <ul v-if="highlights.length" class="ab-news-list">
        <li v-for="(item, i) in highlights" :key="i">
          <span class="material-symbols-outlined ab-news-mark">check_circle</span>
          <span>{{ item }}</span>
        </li>
      </ul>
    </AppCard>

    <!-- ── Приложения для устройств ── -->
    <AppCard v-if="showApkCard || showDesktopCard" title="Приложения" :gap="6">
      <AppRow
        v-if="showDesktopCard"
        plain
        title="Для компьютера"
        hint="Отдельное окно, значок в трее и системные уведомления — даже когда браузер закрыт."
      >
        <AppButton
          tag="a"
          :href="desktopFileHref(desktopOs)"
          download
          icon="download"
          :label="DESKTOP_OS_LABELS[desktopOs]"
        />
      </AppRow>
      <p v-if="showDesktopCard" class="ab-os-links">
        Другие системы:
        <template v-for="(os, i) in otherOs" :key="os">
          <a :href="desktopFileHref(os)" download>{{ DESKTOP_OS_LABELS[os] }}</a><template v-if="i < otherOs.length - 1"> · </template>
        </template>
      </p>

      <AppRow
        v-if="showApkCard"
        plain
        title="Для Android"
        hint="Задачи, чаты и звонки на смартфоне — с пуш-уведомлениями."
      >
        <AppButton tag="a" :href="APK_HREF" :download="apkDownloadName" icon="download" label="APK" />
      </AppRow>
    </AppCard>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { getNativeBuild, checkNativeUpdate, installNativeUpdate } from '@/utils/nativeApp.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { useAppVersion } from '@/composables/useAppVersion.js'
import { tapBuildNumber } from '@/utils/devTools.js'
import { useAppDownloads } from '@/composables/useAppDownloads.js'
import BrandLogo from '@/components/common/BrandLogo.vue'
import AppRow from '@/components/ui/AppRow.vue'
import AppCard from '@/components/ui/AppCard.vue'
import AppButton from '@/components/ui/AppButton.vue'
import { WAVE_PATH } from '@/utils/wavePath.js'

const notif = useNotificationsStore()

/* Пять быстрых нажатий по номеру сборки открывают скрытый раздел «DevTools».
   Сообщаем об этом только в момент включения — иначе тап по номеру выглядел бы
   как случайно сработавшая кнопка. */
function onBuildTap() {
  if (tapBuildNumber()) {
    notif.success('Раздел появился в списке настроек', 'DevTools включены')
  }
}

/* Версия, сборка и дата выпуска — только с сервера (data/changelog.json),
   не из бандла. */
const { release, version: appVersion, build: releaseBuild, majorVersion, load: loadVersion } = useAppVersion()
const releaseDate = computed(() => {
  const raw = release.value?.date
  if (!raw) return ''
  const d = new Date(raw)
  return Number.isNaN(d.getTime()) ? raw : d.toLocaleDateString('ru-RU')
})
const highlights = computed(() => release.value?.highlights || [])
const newsTitle = computed(() => (majorVersion.value ? `Что нового в Groove Work ${majorVersion.value}` : 'Что нового'))

// Здесь версию показывают целиком («что нового», дата) — читаем с сервера,
// минуя кэш марки: раздел открывают именно чтобы узнать актуальный выпуск.
onMounted(() => loadVersion({ force: true }))

/* ── Скачивание приложений ── */
const {
  APK_HREF, apkDownloadName, desktopFileHref, desktopOs, DESKTOP_OS_LABELS,
  showApk: showApkCard, showDesktop: showDesktopCard,
} = useAppDownloads()

// Своя система — главной кнопкой, остальные — ссылками под ней.
const otherOs = computed(() => Object.keys(DESKTOP_OS_LABELS).filter((os) => os !== desktopOs))

/* ── Обновление обёртки изнутри приложения. Мобильная (Capacitor) — нативный
   плагин NativeShell (сборки 2607104+); десктопная (Electron) — мост
   window.GrooveDesktop из preload (версии 1.0.2+). Обвязка общая, различается
   только транспорт. */
const hasNativeShell = !!window.Capacitor?.Plugins?.NativeShell
const desktopShell = window.GrooveDesktop
const hasShellUpdate = hasNativeShell || !!desktopShell
const shellLabel = hasNativeShell ? 'Приложение для Android' : 'Приложение для компьютера'
const shellBuild = ref(null)
const updateInfo = ref(null)
const updBusy = ref(false)
const updProgress = ref(null)

// Сборка продукта — из данных выпуска; версия установленной обёртки
// (Capacitor/Electron) живёт в своей карточке обновления ниже.
const appBuild = releaseBuild

onMounted(async () => {
  if (hasNativeShell) {
    shellBuild.value = `${await getNativeBuild()}`
  } else if (desktopShell) {
    const { version } = await desktopShell.getVersion().catch(() => ({}))
    if (version) shellBuild.value = version
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
</script>

<style scoped>
.ab {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

/* ── Марка ── */
.ab-hero {
  position: relative;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  padding: 28px 20px 34px;
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-xl);
  background: var(--acrylic-card-bg);
  text-align: center;
  color: var(--color-text);
}

/* Контент и пелена — над волнами. */
.ab-hero > :not(.ab-waves):not(.ab-frost) { position: relative; z-index: 1; }

/* Волны — нижняя часть карточки, медленный бесшовный дрейф (ширина 200%,
   сдвиг на половину), кверху растворяются маской. */
.ab-waves {
  position: absolute;
  inset: auto 0 0 0;
  /* Высота слоя ФИКСИРОВАНА и равна высоте viewBox: тянулась бы в процентах —
     волна меняла бы амплитуду вслед за высотой карточки. */
  height: 140px;
  pointer-events: none;
  overflow: hidden;
  -webkit-mask-image: linear-gradient(180deg, transparent 0%, black 70%);
  mask-image: linear-gradient(180deg, transparent 0%, black 70%);
}

/* Слой рисуется 1:1 с viewBox (2880×140), поэтому при любом размере окна
   форма волны неизменна — двигается только сдвиг. Ширины хватает с запасом:
   период 240, сдвиг ровно на половину (1440 = 6 периодов) бесшовен. */
.ab-wave {
  position: absolute;
  left: 0;
  bottom: 0;
  width: 2880px;
  height: 140px;
  animation: ab-wave-drift linear infinite;
}

.ab-wave svg { width: 2880px; height: 140px; display: block; }

.ab-wave--soft { animation-duration: 30s; }
.ab-wave--soft path { fill: var(--color-primary-container); opacity: 0.3; }
.ab-wave--mid { animation-duration: 20s; animation-delay: -6s; bottom: -18px; }
.ab-wave--mid path { fill: color-mix(in oklch, var(--color-primary) 55%, var(--color-tertiary-container)); opacity: 0.16; }
.ab-wave--deep { animation-duration: 13s; animation-delay: -3s; bottom: -34px; }
.ab-wave--deep path { fill: var(--color-primary); opacity: 0.18; }

@keyframes ab-wave-drift {
  from { transform: translateX(0); }
  to { transform: translateX(-1440px); }
}

/* Пелена поверх волн: приглушает их до мягкого свечения. Размытия здесь нет
   намеренно — волны под ней ДВИЖУТСЯ, и backdrop-filter пересчитывался бы
   каждый кадр всё время, пока раздел открыт. */
.ab-frost {
  position: absolute;
  inset: 0;
  pointer-events: none;
  background: var(--glass-bg);
}

@media (prefers-reduced-motion: reduce) {
  .ab-wave { animation: none; }
}

.ab-logo { margin-bottom: 6px; }

.ab-name {
  display: flex;
  align-items: baseline;
  justify-content: center;
  flex-wrap: wrap;
  gap: 0.28em;
  margin: 0;
  font-size: 1.9rem;
  line-height: 1.15;
  font-weight: 800;
  letter-spacing: -0.01em;
  color: var(--color-primary);
}

.ab-name-major { color: var(--color-text); }

.ab-tagline {
  margin: 0;
  max-width: 420px;
  font-size: 0.92rem;
  line-height: 1.45;
  color: var(--color-text-dim);
}

.ab-version {
  margin-top: 8px;
  padding: 4px 12px;
  border-radius: var(--radius-full);
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
  font-size: 0.8rem;
  font-weight: 600;
}

/* ── Сведения: строки «название → значение», разделённые линией ── */
.ab-facts { margin: 0; }

.ab-fact {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 16px;
  padding: 11px 2px;
  border-bottom: 1px solid var(--color-outline-dim);
}

.ab-fact:last-child { border-bottom: none; }

.ab-fact dt {
  font-size: 0.9rem;
  color: var(--color-text-dim);
}

.ab-fact dd {
  margin: 0;
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  font-size: 0.92rem;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
  text-align: right;
  color: var(--color-text);
}

.ab-fact-note {
  font-size: 0.78rem;
  font-weight: 400;
  color: var(--color-text-dim);
}

/* Строка сборки нажимаемая (вход в DevTools), но выглядит так же: подсказывать
   секретный ход курсором-указателем не надо. Снимаем только выделение текста —
   пять быстрых кликов иначе выделяют номер. */
.ab-fact-tap { user-select: none; -webkit-tap-highlight-color: transparent; }

.ab-actions {
  display: flex;
  justify-content: flex-end;
  padding-top: 12px;
}

/* ── Что нового ── */
.ab-news-text {
  margin: 0;
  font-size: 0.92rem;
  line-height: 1.55;
  color: var(--color-text);
}

.ab-news-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
  margin: 0;
  padding: 0;
  list-style: none;
}

.ab-news-list li {
  display: flex;
  gap: 10px;
  font-size: 0.9rem;
  line-height: 1.45;
  color: var(--color-text);
  overflow-wrap: anywhere;
}

.ab-news-mark {
  flex-shrink: 0;
  font-size: 20px;
  color: var(--color-primary);
}

/* ── Приложения ── */
.ab-os-links {
  margin: -4px 0 4px;
  padding: 0 12px 10px;
  border-bottom: 1px solid var(--color-outline-dim);
  font-size: 0.82rem;
  color: var(--color-text-dim);
}

.ab-os-links:last-child { border-bottom: none; padding-bottom: 0; }

.ab-os-links a {
  color: var(--color-primary);
  text-decoration: none;
}

.ab-os-links a:hover { text-decoration: underline; }

@container (max-width: 520px) {
  .ab-hero { padding: 22px 16px 28px; }
  .ab-name { font-size: 1.6rem; }
}

@media (max-width: 520px) {
  .ab-hero { padding: 22px 16px 28px; }
  .ab-name { font-size: 1.6rem; }
}
</style>
