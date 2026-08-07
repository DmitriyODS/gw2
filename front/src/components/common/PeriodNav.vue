<template>
  <div class="period-nav" :class="{ tight }">
    <AppButton
      variant="icon"
      size="sm"
      icon="chevron_left"
      title="Предыдущий период"
      aria-label="Предыдущий период"
      @click="$emit('step', -1)"
    />
    <AppButton
      v-if="tight"
      variant="icon"
      size="sm"
      icon="today"
      title="Сегодня"
      aria-label="Сегодня"
      @click="$emit('today')"
    />
    <AppButton v-else size="sm" label="Сегодня" @click="$emit('today')" />
    <AppButton
      variant="icon"
      size="sm"
      icon="chevron_right"
      title="Следующий период"
      aria-label="Следующий период"
      @click="$emit('step', 1)"
    />

    <h2 class="pn-label">{{ label }}</h2>

    <!-- Тесная панель переключает вид пунктом меню «ещё» (см. periodViews.js):
         строка вкладок съедала бы место, которого и так нет. -->
    <AppTabs
      v-if="!tight"
      class="pn-views"
      :model-value="view"
      :tabs="PERIOD_VIEWS"
      variant="tint"
      dense
      @update:model-value="$emit('update:view', $event)"
    />
  </div>
</template>

<script setup>
/* Навигация по периоду с переключателем вида — общая для ежедневников и
   календарей (раньше в обоих разделах лежала одна и та же разметка). */
import AppButton from '@/components/ui/AppButton.vue'
import AppTabs from '@/components/ui/AppTabs.vue'
import { PERIOD_VIEWS } from '@/utils/periodViews.js'

defineProps({
  /** Подпись текущего периода — считает раздел (у них разный формат). */
  label: { type: String, default: '' },
  /** Текущий вид: month | week | day. */
  view: { type: String, default: 'week' },
  /** Тесная панель: «Сегодня» значком, вкладки видов не показываем. */
  tight: { type: Boolean, default: false },
})

defineEmits(['step', 'today', 'update:view'])
</script>

<style scoped>
.period-nav {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  min-width: 0;
  flex: 1 1 auto;
}

.pn-label {
  flex: 1 1 auto;
  min-width: 0;
  margin: 0;
  font-size: 1.05rem;
  font-weight: 700;
  text-transform: capitalize;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.pn-views { flex: 0 0 auto; }

.period-nav.tight { gap: 6px; }
.period-nav.tight .pn-label { font-size: 0.98rem; flex: 1 1 60px; }
</style>
