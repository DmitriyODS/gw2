<template>
  <AppCard
    class="dw-card"
    title="Обои рабочего стола"
    hint="Готовый комплект, своя картинка, градиент или узор под окнами. Оформление личное и синхронизируется на всех ваших устройствах."
  >
    <BackgroundEditor :recipe="recipe" :upload-fn="uploadFn" preview="desktop" :presets="WALLPAPERS" />

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
      <AppButton icon="restart_alt" label="Сбросить" :disabled="!prefs.wallpaper" @click="reset" />
      <AppButton variant="filled" icon="check" label="Применить" @click="apply" />
    </div>
  </AppCard>
</template>

<script setup>
import { onMounted, reactive } from 'vue'
import AppButton from '@/components/ui/AppButton.vue'
import BackgroundEditor from '@/components/common/BackgroundEditor.vue'
import AppCard from '@/components/ui/AppCard.vue'
import { useDesktopPrefsStore } from '@/stores/desktopPrefs.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { uploadAttachment } from '@/api/messenger.js'
import { normalizeRecipe, cloneRecipe } from '@/utils/chatBackgrounds.js'
import { WALLPAPERS, defaultWallpaperRecipe } from '@/utils/wallpapers.js'

const prefs = useDesktopPrefsStore()
const notif = useNotificationsStore()

const recipe = reactive(defaultWallpaperRecipe())

// Картинка обоев — личный ассет пользователя; грузим через общий uploads
// мессенджера (тот же путь, что у фонов чатов и ленты портала).
const uploadFn = (file) => uploadAttachment(file)

onMounted(() => {
  const saved = normalizeRecipe(prefs.wallpaper)
  Object.assign(recipe, saved ? cloneRecipe(saved) : defaultWallpaperRecipe())
})

// Возврат к ранее загруженной картинке: размытие оставляем текущее.
function useImage(url) {
  recipe.image = { url, blur: recipe.image?.blur ?? 0 }
}

function apply() {
  prefs.setWallpaper(cloneRecipe(recipe))
  notif.success('Обои рабочего стола обновлены')
}

// Сброс — к заводским обоям: своя настройка снимается, стол возвращается к
// встроенному комплекту.
function reset() {
  prefs.setWallpaper(null)
  Object.assign(recipe, defaultWallpaperRecipe())
}
</script>

<style scoped>
/* Общая карточка настроек, но содержимое дышит чуть свободнее: внутри —
   редактор фона со своими превью. */
.dw-card { gap: 18px; }

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
