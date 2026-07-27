<template>
  <div class="dw-card">
    <header class="dw-head">
      <span class="dw-icon material-symbols-outlined">wallpaper</span>
      <div class="dw-head-text">
        <h3>Обои рабочего стола</h3>
        <p>
          Своя картинка, градиент или узор под окнами. Оформление личное и
          синхронизируется на всех ваших устройствах.
        </p>
      </div>
    </header>

    <BackgroundEditor :recipe="recipe" :upload-fn="uploadFn" preview="desktop" />

    <section v-if="prefs.wallpapers.length" class="dw-history">
      <h4 class="dw-history-title">Недавние картинки</h4>
      <div class="dw-history-row">
        <button
          v-for="url in prefs.wallpapers"
          :key="url"
          class="dw-thumb"
          :class="{ active: recipe.image?.url === url }"
          type="button"
          title="Вернуть эту картинку"
          @click="useImage(url)"
        >
          <img :src="url" alt="" />
          <span class="dw-thumb-remove" title="Убрать из истории" @click.stop="prefs.forgetWallpaper(url)">
            <span class="material-symbols-outlined">close</span>
          </span>
        </button>
      </div>
    </section>

    <div class="dw-actions">
      <button class="btn-glass" type="button" :disabled="!prefs.wallpaper" @click="reset">
        <span class="material-symbols-outlined">restart_alt</span>
        Сбросить
      </button>
      <button class="btn-grad" type="button" @click="apply">
        <span class="material-symbols-outlined">check</span>
        Применить
      </button>
    </div>
  </div>
</template>

<script setup>
import { onMounted, reactive } from 'vue'
import BackgroundEditor from '@/components/common/BackgroundEditor.vue'
import { useDesktopPrefsStore } from '@/stores/desktopPrefs.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { uploadAttachment } from '@/api/messenger.js'
import { DEFAULT_RECIPE, normalizeRecipe, cloneRecipe } from '@/utils/chatBackgrounds.js'

const prefs = useDesktopPrefsStore()
const notif = useNotificationsStore()

const recipe = reactive(cloneRecipe(DEFAULT_RECIPE))

// Картинка обоев — личный ассет пользователя; грузим через общий uploads
// мессенджера (тот же путь, что у фонов чатов и ленты портала).
const uploadFn = (file) => uploadAttachment(file)

onMounted(() => {
  Object.assign(recipe, cloneRecipe(normalizeRecipe(prefs.wallpaper) || DEFAULT_RECIPE))
})

// Возврат к ранее загруженной картинке: размытие оставляем текущее.
function useImage(url) {
  recipe.image = { url, blur: recipe.image?.blur ?? 0 }
}

function apply() {
  prefs.setWallpaper(cloneRecipe(recipe))
  notif.success('Обои рабочего стола обновлены')
}

function reset() {
  prefs.setWallpaper(null)
  Object.assign(recipe, cloneRecipe(DEFAULT_RECIPE))
}
</script>

<style scoped>
/* Карточка живёт в панели настроек, но её стили (.settings-card и соседи)
   scoped в SettingsView — поэтому раскладка своя, а кнопки берём глобальные. */
.dw-card {
  display: flex;
  flex-direction: column;
  gap: 18px;
  padding: 22px;
  background: var(--glass-bg);
  box-shadow: var(--glass-edge);
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-xl);
}

.dw-head {
  display: flex;
  align-items: flex-start;
  gap: 16px;
}

.dw-icon {
  flex-shrink: 0;
  width: 48px;
  height: 48px;
  display: grid;
  place-items: center;
  border-radius: var(--radius-lg);
  background: var(--color-tertiary-container, var(--color-primary-container));
  color: var(--color-on-tertiary-container, var(--color-on-primary-container));
  font-size: 24px;
}

.dw-head-text h3 {
  margin: 0 0 4px;
  font-size: 16px;
  font-weight: 650;
  color: var(--color-text);
}

.dw-head-text p {
  margin: 0;
  font-size: 13.5px;
  line-height: 1.45;
  color: var(--color-text-dim);
}

/* ── История картинок ── */
.dw-history-title {
  margin: 0 0 10px;
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0.4px;
  text-transform: uppercase;
  color: var(--color-text-dim);
}

.dw-history-row {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}

.dw-thumb {
  position: relative;
  width: 84px;
  height: 56px;
  padding: 0;
  overflow: hidden;
  border: 2px solid var(--acrylic-border);
  border-radius: var(--radius-md);
  background: var(--color-surface-low);
  cursor: pointer;
  transition: border-color 0.15s;
}

.dw-thumb:hover { border-color: color-mix(in oklch, var(--color-primary) 40%, var(--acrylic-border)); }
.dw-thumb.active { border-color: var(--color-primary); }

.dw-thumb img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}

.dw-thumb-remove {
  position: absolute;
  top: 2px;
  right: 2px;
  width: 20px;
  height: 20px;
  display: grid;
  place-items: center;
  border-radius: var(--radius-full);
  background: var(--acrylic-bg-strong);
  color: var(--color-text);
  opacity: 0;
  transition: opacity 0.15s;
}

.dw-thumb:hover .dw-thumb-remove { opacity: 1; }
.dw-thumb-remove .material-symbols-outlined { font-size: 14px; }

.dw-actions {
  display: flex;
  gap: 10px;
  justify-content: flex-end;
}
</style>
