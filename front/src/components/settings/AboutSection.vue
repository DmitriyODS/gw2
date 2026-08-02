<template>
  <div class="ab">
    <!-- ── Карточка продукта ─────────────────────────────────── -->
    <section class="ab-card ab-hero">
      <!-- Фирменная волна Groove (мотив логотипа) дрейфует ЗА акриловым
           стеклом: слой волн → матовая пелена → контент. -->
      <div class="ab-waves" aria-hidden="true">
        <div class="ab-wave ab-wave--soft"><svg viewBox="0 0 2880 140" preserveAspectRatio="none"><path :d="WAVE_PATH" /></svg></div>
        <div class="ab-wave ab-wave--mid"><svg viewBox="0 0 2880 140" preserveAspectRatio="none"><path :d="WAVE_PATH" /></svg></div>
        <div class="ab-wave ab-wave--deep"><svg viewBox="0 0 2880 140" preserveAspectRatio="none"><path :d="WAVE_PATH" /></svg></div>
      </div>
      <div class="ab-frost" aria-hidden="true" />

      <div class="ab-brand">
        <Logo :size="64" />
        <h3 class="ab-brand-name">
          <span>Groove Work</span>
          <span v-if="majorVersion" class="ab-brand-major">{{ majorVersion }}</span>
        </h3>
      </div>

      <p class="ab-about">
        Groove Work собирает рабочий день в одном окне: задачи и учёт времени,
        статистика по людям и отделам, мессенджер со звонками, корпоративный
        портал, заметки, доски, ежедневники, календари и напоминания. Один
        аккаунт живёт сразу в нескольких компаниях — переключайтесь между ними,
        не выходя из системы; личные разделы и переписка остаются с вами в любой
        из них. Работает в браузере, на компьютере и на телефоне.
      </p>

      <!-- Информация о приложении: версия, сборка и дата выпуска — ровно три
           бейджа, источник один: data/changelog.json. -->
      <div class="ab-badges">
        <span v-if="appVersion" class="ab-badge primary">
          <span class="material-symbols-outlined">verified</span>
          <span class="ab-badge-text">
            <small>Версия</small>
            <strong>{{ appVersion }}</strong>
          </span>
        </span>
        <span v-if="appBuild" class="ab-badge">
          <span class="material-symbols-outlined">tag</span>
          <span class="ab-badge-text">
            <small>Сборка</small>
            <strong>{{ appBuild }}</strong>
          </span>
        </span>
        <span v-if="releaseDate" class="ab-badge">
          <span class="material-symbols-outlined">event</span>
          <span class="ab-badge-text">
            <small>Дата выпуска</small>
            <strong>{{ releaseDate }}</strong>
          </span>
        </span>
      </div>
    </section>

    <!-- ── Что нового: только текущий выпуск, истории версий больше нет ── -->
    <section v-if="release" class="ab-card ab-news">
      <header class="ab-news-head">
        <span class="ab-news-spark">
          <span class="material-symbols-outlined">auto_awesome</span>
        </span>
        <div class="ab-news-title">
          <h3 class="ab-h">Что нового?</h3>
          <span v-if="release.title" class="ab-news-tag">{{ release.title }}</span>
        </div>
        <!-- Мини-счётчик выпуска: сколько нововведений в этой версии. -->
        <span v-if="highlights.length" class="ab-news-count">
          <strong>{{ highlights.length }}</strong>
          <small>{{ countWord }}</small>
        </span>
      </header>

      <p v-if="release.description" class="ab-news-text">{{ release.description }}</p>

      <!-- Каждому пункту свой номер и цвет из палитры меток: список читается
           как инфографика, а не как простыня текста. -->
      <ol v-if="highlights.length" class="ab-news-grid">
        <li
          v-for="(item, i) in highlights"
          :key="i"
          class="ab-news-item"
          :style="{ '--hue': HIGHLIGHT_HUES[i % HIGHLIGHT_HUES.length], '--delay': `${i * 70}ms` }"
        >
          <span class="ab-news-num">{{ String(i + 1).padStart(2, '0') }}</span>
          <span class="ab-news-body">{{ item }}</span>
        </li>
      </ol>
    </section>

    <!-- ── Обновление обёртки (внутри мобильного/десктопного клиента) ── -->
    <SettingRow v-if="hasShellUpdate" title="Обновление приложения">
      <template #hint>
        <template v-if="shellBuild">
          Установлена {{ shellBuild }}<template v-if="updateInfo">
            · {{ updateInfo.updateAvailable ? `доступна ${updateInfo.latest}` : 'это последняя версия' }}</template>
        </template>
        <template v-else>Оболочка Groove Work</template>
      </template>

      <button
        class="ab-row-btn"
        :class="{ downloading: updProgress != null && updProgress >= 0 }"
        :style="updateBtnStyle"
        :disabled="updBusy"
        type="button"
        @click="onUpdateClick"
      >
        <span class="material-symbols-outlined">{{ updateInfo?.updateAvailable ? 'download' : 'refresh' }}</span>
        {{ updateBtnLabel }}
      </button>
    </SettingRow>

    <!-- ── Приложения для устройств ──────────────────────────── -->
    <SettingRow
      v-if="showApkCard"
      title="Приложение для Android"
      hint="Задачи, юниты, чаты и звонки на смартфоне — с пуш-уведомлениями."
    >
      <a class="ab-row-btn" :href="APK_HREF" :download="apkDownloadName">
        <span class="material-symbols-outlined">download</span>
        Скачать APK
      </a>
    </SettingRow>

    <SettingRow v-if="showDesktopCard" title="Приложение для компьютера">
      <template #hint>
        Отдельное окно, значок в трее, системные уведомления и звонки — даже
        когда браузер закрыт.
        <span class="ab-os-links">
          Все платформы:
          <a :href="desktopFileHref('mac')" download>macOS</a> ·
          <a :href="desktopFileHref('win')" download>Windows</a> ·
          <a :href="desktopFileHref('linux')" download>Linux</a>
        </span>
      </template>

      <a class="ab-row-btn" :href="desktopFileHref(desktopOs)" download>
        <span class="material-symbols-outlined">download</span>
        Скачать для {{ DESKTOP_OS_LABELS[desktopOs] }}
      </a>
    </SettingRow>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { getNativeBuild, checkNativeUpdate, installNativeUpdate } from '@/utils/nativeApp.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { useAppVersion } from '@/composables/useAppVersion.js'
import { useAppDownloads } from '@/composables/useAppDownloads.js'
import Logo from '@/components/common/Logo.vue'
import SettingRow from '@/components/common/SettingRow.vue'
import { WAVE_PATH } from '@/utils/wavePath.js'

const notif = useNotificationsStore()

/* Версия, сборка и дата выпуска — только с сервера (data/changelog.json),
   не из бандла. */
const { release, version: appVersion, build: releaseBuild, majorVersion, load: loadVersion } = useAppVersion()
const releaseDate = computed(() => {
  const raw = release.value?.date
  if (!raw) return ''
  const d = new Date(raw)
  return Number.isNaN(d.getTime()) ? raw : d.toLocaleDateString('ru-RU')
})
const highlights = computed(() => (release.value?.highlights || []).slice(0, 4))

// Цвета пунктов — из палитры меток (--tag-*-h), по кругу.
const HIGHLIGHT_HUES = ['var(--tag-blue-h)', 'var(--tag-violet-h)', 'var(--tag-teal-h)', 'var(--tag-amber-h)']

const countWord = computed(() => {
  const n = highlights.value.length
  const last = n % 10
  const tens = n % 100
  if (tens >= 11 && tens <= 14) return 'нововведений'
  if (last === 1) return 'нововведение'
  if (last >= 2 && last <= 4) return 'нововведения'
  return 'нововведений'
})

onMounted(loadVersion)

/* ── Скачивание приложений ── */
const {
  APK_HREF, apkDownloadName, desktopFileHref, desktopOs, DESKTOP_OS_LABELS,
  showApk: showApkCard, showDesktop: showDesktopCard,
} = useAppDownloads()

/* ── Обновление обёртки изнутри приложения. Мобильная (Capacitor) — нативный
   плагин NativeShell (сборки 2607104+); десктопная (Electron) — мост
   window.GrooveDesktop из preload (версии 1.0.2+). Обвязка общая, различается
   только транспорт. */
const hasNativeShell = !!window.Capacitor?.Plugins?.NativeShell
const desktopShell = window.GrooveDesktop
const hasShellUpdate = hasNativeShell || !!desktopShell
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
</script>

<style scoped>
.ab {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.ab-card {
  padding: 20px;
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-xl);
  background: var(--acrylic-card-bg);
  color: var(--color-text);
}

/* ── Продукт ── */
.ab-hero {
  position: relative;
  overflow: hidden;
}

/* Контент и пелена — над волнами. */
.ab-hero > :not(.ab-waves) { position: relative; z-index: 1; }

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

/* Акриловое стекло поверх волн: размывает их, оставляя мягкое свечение.
   -webkit- ПЕРЕД стандартным — иначе минификатор выбросит стандартное. */
.ab-frost {
  position: absolute;
  inset: 0;
  pointer-events: none;
  background: var(--glass-bg);
  -webkit-backdrop-filter: blur(18px) saturate(1.3);
  backdrop-filter: blur(18px) saturate(1.3);
}

@media (prefers-reduced-motion: reduce) {
  .ab-wave { animation: none; }
}

.ab-brand {
  display: flex;
  align-items: center;
  gap: 18px;
  padding: 6px 0 2px;
}

.ab-brand-name {
  display: flex;
  align-items: baseline;
  flex-wrap: wrap;
  gap: 0.28em;
  margin: 0;
  font-size: 2.1rem;
  line-height: 1.15;
  font-weight: 800;
  letter-spacing: -0.01em;
  color: var(--color-primary);
}

.ab-brand-major { color: var(--color-text); }

.ab-about {
  margin: 16px 0 0;
  font-size: 0.92rem;
  line-height: 1.6;
  color: var(--color-text);
}

/* Факты выпуска — бейджи-пилюли. */
.ab-badges {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  margin-top: 20px;
}

.ab-badge {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  padding: 9px 16px 9px 12px;
  border: 1px solid var(--acrylic-border);
  border-radius: 999px;
  background: var(--glass-bg), var(--acrylic-card-bg);
  box-shadow: var(--glass-edge);
  color: var(--color-text);
}

.ab-badge.primary {
  border-color: color-mix(in oklch, var(--color-primary) 40%, var(--acrylic-border));
  background: var(--glass-bg), var(--color-primary-container);
  color: var(--color-on-primary-container);
}

.ab-badge .material-symbols-outlined { font-size: 20px; }

.ab-badge-text {
  display: flex;
  flex-direction: column;
  line-height: 1.15;
}

.ab-badge-text small {
  font-size: 0.7rem;
  letter-spacing: 0.02em;
  opacity: 0.75;
}

.ab-badge-text strong { font-size: 0.92rem; font-weight: 700; }

/* ── Что нового ── */
.ab-news { position: relative; overflow: hidden; }

/* Мягкое свечение в углу карточки — «праздничная» подсветка выпуска. */
.ab-news::before {
  content: '';
  position: absolute;
  top: -60px;
  right: -40px;
  width: 220px;
  height: 220px;
  border-radius: 50%;
  background: radial-gradient(
    circle,
    color-mix(in oklch, var(--color-primary) 22%, transparent),
    transparent 70%
  );
  pointer-events: none;
}

.ab-news > * { position: relative; }

.ab-news-head {
  display: flex;
  align-items: center;
  gap: 14px;
}

.ab-news-spark {
  display: grid;
  place-items: center;
  width: 44px;
  min-width: 44px;
  max-width: 44px;
  height: 44px;
  min-height: 44px;
  max-height: 44px;
  border-radius: 50%;
  background: linear-gradient(
    140deg,
    var(--color-primary),
    color-mix(in oklch, var(--color-tertiary) 80%, var(--color-primary))
  );
  color: var(--color-on-primary);
  box-shadow: var(--shadow-md);
}

.ab-news-spark .material-symbols-outlined {
  font-size: 24px;
  animation: ab-spark 4.5s ease-in-out infinite;
}

@keyframes ab-spark {
  0%, 72%, 100% { transform: scale(1) rotate(0deg); }
  80% { transform: scale(1.18) rotate(-12deg); }
  88% { transform: scale(1.08) rotate(8deg); }
}

.ab-news-title {
  display: flex;
  flex-direction: column;
  gap: 4px;
  flex: 1;
  min-width: 0;
}

.ab-news-count {
  display: flex;
  flex-direction: column;
  align-items: center;
  line-height: 1;
  color: var(--color-primary);
}

.ab-news-count strong { font-size: 1.9rem; font-weight: 800; }

.ab-news-count small {
  font-size: 0.68rem;
  font-weight: 600;
  color: var(--color-text-dim);
}

.ab-h {
  margin: 0;
  font-size: 1.35rem;
  font-weight: 700;
}

.ab-news-tag {
  align-self: flex-start;
  padding: 5px 12px;
  border-radius: 999px;
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
  font-size: 0.78rem;
  font-weight: 600;
}

.ab-news-text {
  margin: 16px 0 0;
  font-size: 0.95rem;
  line-height: 1.6;
  color: var(--color-text);
}

.ab-news-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(230px, 1fr));
  gap: 12px;
  margin: 18px 0 0;
  padding: 0;
  list-style: none;
}

/* Пункт-«карточка»: цветная кромка слева, крупный номер, текст.
   Появляются лесенкой — выпуск «раскрывается», а не вываливается. */
.ab-news-item {
  display: flex;
  gap: 12px;
  padding: 14px;
  border: 1px solid oklch(0.85 0.05 var(--hue) / 0.5);
  border-left: 3px solid oklch(0.62 0.16 var(--hue));
  border-radius: var(--radius-md);
  background: var(--glass-bg), oklch(0.96 0.03 var(--hue) / 0.55);
  box-shadow: var(--glass-edge);
  font-size: 0.85rem;
  line-height: 1.45;
  color: var(--color-text);
  opacity: 0;
  animation: ab-news-in 0.42s cubic-bezier(0.2, 0, 0, 1) forwards;
  animation-delay: var(--delay);
  transition: transform 0.2s ease;
}

.ab-news-item:hover { box-shadow: var(--shadow-md), var(--glass-edge); }

.ab-news-num {
  font-size: 1.15rem;
  font-weight: 800;
  line-height: 1;
  color: oklch(0.55 0.17 var(--hue));
}

[data-dark="true"] .ab-news-item {
  border-color: oklch(0.45 0.06 var(--hue) / 0.6);
  background: var(--glass-bg), oklch(0.32 0.04 var(--hue) / 0.5);
}

[data-dark="true"] .ab-news-num { color: oklch(0.82 0.13 var(--hue)); }

.ab-news-body { flex: 1; min-width: 0; }

@keyframes ab-news-in {
  from { opacity: 0; transform: translateY(10px); }
  to { opacity: 1; transform: translateY(0); }
}

@media (prefers-reduced-motion: reduce) {
  .ab-news-item { opacity: 1; animation: none; }
  .ab-news-spark .material-symbols-outlined { animation: none; }
}

/* ── Строки-карточки устройств (общий SettingRow) ── */
.ab-os-links { display: block; margin-top: 3px; }

.ab-os-links a {
  color: var(--color-primary);
  text-decoration: none;
}

.ab-os-links a:hover { text-decoration: underline; }

.ab-row-btn {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 10px 18px;
  border: none;
  border-radius: 999px;
  background: var(--color-primary);
  color: var(--color-on-primary);
  font-size: 0.86rem;
  font-weight: 600;
  text-decoration: none;
  white-space: nowrap;
  cursor: pointer;
}

.ab-row-btn:disabled { opacity: 0.7; cursor: progress; }

.ab-row-btn .material-symbols-outlined { font-size: 19px; }

/* Прогресс скачивания заливкой кнопки. */
.ab-row-btn.downloading {
  background: linear-gradient(
    to right,
    var(--color-primary) var(--dl, 0%),
    var(--color-surface-high) var(--dl, 0%)
  );
  color: var(--color-text);
}

/* Узкой бывает ПАНЕЛЬ раздела, а не экран (настройки живут окном рабочего
   стола) — поэтому перенос считаем от контейнера; @media оставлен дублем для
   старого WebView, который @container не знает. */
@container (max-width: 620px) {
  .ab-row-btn { width: 100%; justify-content: center; }
  .ab-news-head { flex-wrap: wrap; }
  .ab-badge { flex: 1 1 auto; }
}

@media (max-width: 620px) {
  .ab-row-btn { width: 100%; justify-content: center; }
}
</style>
