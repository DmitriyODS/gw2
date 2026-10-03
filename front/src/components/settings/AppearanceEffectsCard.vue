<template>
  <!-- Эффекты — про то, КАК выглядят стекло и подложка под обоями, и про
       нагрузку на видеокарту. Миниатюра показывает стол целиком, поэтому
       прозрачность и размытие видны на ней так же, как на экране. -->
  <AppCard title="Эффекты" hint="Применяются сразу. Действуют на этом устройстве.">
    <PreviewLayout>
      <template #preview>
        <BackgroundPreview :recipe="wallpaper" scene="desktop" />
      </template>

      <AppSwitchRow
        :model-value="themeStore.bgGradient.enabled"
        title="Сияние фона"
        :hint="glowHint"
        @update:model-value="themeStore.setBgGradientEnabled"
      />

      <Transition name="fx-reveal">
        <div v-if="themeStore.bgGradient.enabled" class="fx-actions">
          <AppButton icon="shuffle" label="Другая композиция" @click="themeStore.regenerateBgGradient()" />
          <AppButton variant="text" icon="restart_alt" label="Стандартная" @click="themeStore.resetBgGradient()" />
        </div>
      </Transition>

      <!-- Про железо, а не про вкус: стекло — самая дорогая часть интерфейса
           для видеокарты и батареи. Без прозрачности размытие теряет смысл,
           поэтому его переключатель тогда неактивен. -->
      <AppSwitchRow
        :model-value="transparencyEnabled"
        title="Прозрачность"
        hint="Выключено — окна и панели плотные, обои под ними не видны."
        @update:model-value="setTransparency"
      />
      <AppSwitchRow
        :model-value="blurEnabled"
        :disabled="!transparencyEnabled"
        title="Размытие"
        hint="Выключено — панели остаются полупрозрачными, но без размытия. Легче для видеокарты и батареи."
        @update:model-value="setBlur"
      />
      <div v-if="transparencyChoice || blurChoice" class="fx-actions">
        <AppButton variant="text" icon="settings_suggest" label="Как в системе" @click="resetGlass" />
      </div>
    </PreviewLayout>
  </AppCard>
</template>

<script setup>
import { computed } from 'vue'
import AppCard from '@/components/ui/AppCard.vue'
import AppButton from '@/components/ui/AppButton.vue'
import AppSwitchRow from '@/components/ui/AppSwitchRow.vue'
import BackgroundPreview from '@/components/common/BackgroundPreview.vue'
import PreviewLayout from '@/components/common/PreviewLayout.vue'
import { useThemeStore } from '@/stores/theme.js'
import { useDesktopPrefsStore } from '@/stores/desktopPrefs.js'
import { isBlankRecipe, normalizeRecipe } from '@/utils/chatBackgrounds.js'
import { defaultWallpaperRecipe } from '@/utils/wallpapers.js'
import {
  blurChoice, blurEnabled, setBlur, setTransparency, transparencyChoice, transparencyEnabled,
} from '@/utils/transparency.js'

const themeStore = useThemeStore()
const prefs = useDesktopPrefsStore()

// Те же обои, что рисует каркас (shellCore): своя настройка или комплект.
const wallpaper = computed(() => normalizeRecipe(prefs.wallpaper) || defaultWallpaperRecipe())

/* Сияние лежит ПОД обоями, поэтому видно только без них — честно говорим об
   этом, а не оставляем переключатель, который «ничего не делает». */
const glowHint = computed(() => (isBlankRecipe(wallpaper.value)
  ? 'Мягкие цветные пятна из цветов темы под окнами. Выключено — ровный фон темы.'
  : 'Мягкие цветные пятна из цветов темы. Видны, когда обои сняты: «Обои → Градиент → без градиента».'))

function resetGlass() {
  setTransparency(null)
  setBlur(null)
}
</script>

<style scoped>
.fx-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  margin-top: -10px;
}

.fx-reveal-enter-active, .fx-reveal-leave-active {
  transition: opacity 0.2s ease, transform 0.2s ease;
}

.fx-reveal-enter-from, .fx-reveal-leave-to {
  opacity: 0;
  transform: translateY(-6px);
}
</style>
