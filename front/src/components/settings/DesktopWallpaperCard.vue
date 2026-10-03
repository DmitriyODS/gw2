<template>
  <AppCard
    title="Обои рабочего стола"
    hint="Применяются сразу. Оформление личное и синхронизируется на всех ваших устройствах."
  >
    <BackgroundEditor
      :recipe="recipe"
      :upload-fn="uploadFn"
      preview="desktop"
      :presets="WALLPAPERS"
      :recent="prefs.wallpapers"
      @update:recipe="prefs.setWallpaper"
      @uploaded="prefs.rememberWallpaper"
      @forget-recent="prefs.forgetWallpaper"
    >
      <template #actions>
        <AppButton variant="text" icon="undo" label="Вернуть как было" :disabled="!changed" @click="undo" />
        <AppButton variant="text" icon="restart_alt" label="Стандартные" :disabled="!prefs.wallpaper" @click="prefs.setWallpaper(null)" />
      </template>
    </BackgroundEditor>
  </AppCard>
</template>

<script setup>
import { computed, watch } from 'vue'
import AppButton from '@/components/ui/AppButton.vue'
import AppCard from '@/components/ui/AppCard.vue'
import BackgroundEditor from '@/components/common/BackgroundEditor.vue'
import { useDesktopPrefsStore } from '@/stores/desktopPrefs.js'
import { uploadAttachment } from '@/api/messenger.js'
import { useRecipeUndo } from '@/composables/useRecipeUndo.js'
import { normalizeRecipe } from '@/utils/chatBackgrounds.js'
import { WALLPAPERS, defaultWallpaperRecipe } from '@/utils/wallpapers.js'

const prefs = useDesktopPrefsStore()

// Картинка обоев — личный ассет пользователя; грузим через общий uploads
// мессенджера (тот же путь, что у фонов чатов и ленты портала).
const uploadFn = (file) => uploadAttachment(file)

const recipe = computed(() => normalizeRecipe(prefs.wallpaper) || defaultWallpaperRecipe())

const { changed, undo, capture } = useRecipeUndo(() => prefs.wallpaper, prefs.setWallpaper)
// Первый кадр рисуется из локального кэша; «как было» — то, что пришло с сервера.
watch(() => prefs.loaded, (loaded) => { if (loaded) capture() })
</script>
