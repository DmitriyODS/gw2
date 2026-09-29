<script setup>
/* Тикающий счётчик идущего юнита. Отдельный компонент намеренно: секундный
   тик перерисовывает только этот текст, а не панель, в которой он стоит. */
import { useElapsed } from '@/composables/useElapsed.js'

const props = defineProps({
  start: { type: [String, Date], default: null },
  // clock — «01:45:32», display — «1 ч 45 мин 32 сек».
  format: { type: String, default: 'clock' },
})

const { clock, display } = useElapsed(() => props.start)
</script>

<template>
  <span>{{ format === 'display' ? display : clock }}</span>
</template>
