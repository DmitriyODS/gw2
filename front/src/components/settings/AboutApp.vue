<template>
  <div class="about-app">
    <div class="about-hero">
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

    <a class="about-mobile" href="/apps/groovework.apk" download>
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
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useMessengerStore } from '@/stores/messenger.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { useChangelog } from '@/composables/useChangelog.js'
import { useTutorial } from '@/composables/useTutorial.js'
import Logo from '@/components/common/Logo.vue'

const router = useRouter()
const messenger = useMessengerStore()
const notif = useNotificationsStore()
const changelog = useChangelog()
const tutorial = useTutorial()

// Версию продукта берём только с сервера (первая запись changelog), не из бандла.
const appVersion = changelog.latestVersion
onMounted(changelog.loadLatest)

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

.about-hero {
  display: flex;
  align-items: center;
  gap: 24px;
  padding: 24px;
  background: linear-gradient(135deg,
    var(--color-primary-container) 0%,
    var(--color-tertiary-container) 100%);
  border-radius: var(--radius-xl);
}

.about-logo {
  width: 84px;
  height: 84px;
  border-radius: var(--radius-lg);
  background: var(--color-surface);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  box-shadow: var(--shadow-sm);
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
  color: var(--color-on-primary-container);
}

.about-tagline {
  margin: 0 0 12px;
  font-size: 14px;
  color: var(--color-on-primary-container);
  opacity: 0.85;
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
  background: var(--color-surface);
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
  border: 1px solid color-mix(in oklch, var(--color-on-primary-container) 30%, transparent);
  color: var(--color-on-primary-container);
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  transition: background 0.15s;
}

.version-link:hover {
  background: color-mix(in oklch, var(--color-on-primary-container) 10%, transparent);
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
  background: linear-gradient(135deg,
    var(--color-tertiary-container) 0%,
    var(--color-secondary-container) 100%);
  text-decoration: none;
  transition: transform 0.12s, box-shadow 0.15s;
}

.about-mobile:hover {
  transform: translateY(-1px);
  box-shadow: var(--shadow-sm);
}

.about-mobile-icon {
  width: 48px;
  height: 48px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: var(--radius-md);
  background: var(--color-surface);
  color: var(--color-tertiary);
  flex-shrink: 0;
}

.about-mobile-icon .material-symbols-outlined { font-size: 28px; }

.about-mobile-text { flex: 1; min-width: 0; }

.about-mobile-text h4 {
  margin: 0 0 2px;
  font-size: 15px;
  font-weight: 700;
  color: var(--color-on-tertiary-container);
}

.about-mobile-text p {
  margin: 0;
  font-size: 13px;
  line-height: 1.35;
  color: var(--color-on-tertiary-container);
  opacity: 0.85;
}

.about-mobile-btn {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 10px 20px;
  border-radius: var(--radius-full);
  background: var(--color-primary);
  color: var(--color-on-primary);
  font-size: 14px;
  font-weight: 600;
  flex-shrink: 0;
}

.about-mobile-btn .material-symbols-outlined { font-size: 18px; }

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
  background: var(--color-surface);
  border: 1px solid var(--color-outline-dim);
  cursor: pointer;
  text-align: left;
  transition: background 0.15s, transform 0.12s, box-shadow 0.15s;
}

.about-card:not(:disabled):hover {
  background: var(--color-surface-low);
  transform: translateY(-1px);
  box-shadow: var(--shadow-sm);
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
