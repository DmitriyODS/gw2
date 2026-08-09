<template>
  <!-- Раздел собран ТРЕМЯ группами по смыслу: как экран устроен, как он
       выглядит и настройки самой установки. Плоским списком карточек «панель
       задач», «обои», «фон», «живые плитки» и «десктоп-клиент» читались как
       случайный набор — рядом стояли вещи из разных миров. -->
  <div class="dsec">
    <section class="dsec-group">
      <h3 class="dsec-title">Раскладка</h3>
      <p class="dsec-hint">Как приложение раскладывает разделы и чем ими управлять.</p>
      <AppStack>
        <DesktopLayoutCard />
        <DesktopShellCard v-if="windowsShell" />
        <DesktopTilesCard />
      </AppStack>
    </section>

    <section class="dsec-group">
      <h3 class="dsec-title">Оформление</h3>
      <p class="dsec-hint">Что видно под разделами. Синхронизируется на всех ваших устройствах.</p>
      <AppStack>
        <DesktopWallpaperCard />
        <AppGradientCard />
      </AppStack>
    </section>

    <!-- Группа появляется только внутри десктоп-обёртки: в браузере настраивать
         нечего, и пустой заголовок был бы враньём. -->
    <section v-if="hasDesktopApp" class="dsec-group">
      <h3 class="dsec-title">Приложение для компьютера</h3>
      <p class="dsec-hint">Поведение этой установки: окно, трей и автозапуск.</p>
      <AppStack>
        <DesktopAppCard />
      </AppStack>
    </section>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import AppStack from '@/components/ui/AppStack.vue'
import DesktopLayoutCard from './DesktopLayoutCard.vue'
import DesktopShellCard from './DesktopShellCard.vue'
import DesktopTilesCard from './DesktopTilesCard.vue'
import DesktopWallpaperCard from './DesktopWallpaperCard.vue'
import AppGradientCard from './AppGradientCard.vue'
import DesktopAppCard from './DesktopAppCard.vue'
import { useShellMode } from '@/composables/useShellMode.js'

const { shell } = useShellMode()

/* Сторона панели задач и режим меню «Пуск» — про ОКОННЫЙ каркас: у телефона и
   планшета панель всегда снизу, а «Пуск» и так во весь экран. */
const windowsShell = computed(() => shell.value === 'windows')

const hasDesktopApp = computed(() => !!window.GrooveDesktop)
</script>

<style scoped>
.dsec { display: flex; flex-direction: column; gap: 26px; }

.dsec-group { display: flex; flex-direction: column; gap: 4px; }

.dsec-title {
  margin: 0;
  padding: 0 2px;
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0.4px;
  text-transform: uppercase;
  color: var(--color-primary);
}

.dsec-hint {
  margin: 0 0 8px;
  padding: 0 2px;
  font-size: 12.5px;
  line-height: 1.45;
  color: var(--color-text-dim);
}
</style>
