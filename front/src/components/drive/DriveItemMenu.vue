<template>
  <!-- Меню элемента диска. Позиционируется по курсору, поэтому живёт в body:
       координаты клика — экранные (то же решение, что у ContextMenu). -->
  <ContextMenu
    :visible="visible"
    :x="x"
    :y="y"
    :items="items"
    @select="$emit('action', $event)"
    @close="$emit('close')"
  />
</template>

<script setup>
import { computed } from 'vue'
import ContextMenu from '@/components/common/ContextMenu.vue'

const props = defineProps({
  visible: { type: Boolean, default: false },
  x: { type: Number, default: 0 },
  y: { type: Number, default: 0 },
  item: { type: Object, default: null },
  kind: { type: String, default: 'file' },
  trash: { type: Boolean, default: false },
})

defineEmits(['action', 'close'])

const items = computed(() => {
  if (!props.item) return []
  // В корзине действий два: вернуть или стереть насовсем.
  if (props.trash) {
    return [
      { action: 'restore', label: 'Восстановить', icon: 'restore' },
      { action: 'purge', label: 'Удалить навсегда', icon: 'delete_forever', danger: true },
    ]
  }
  const isFile = props.kind === 'file'
  return [
    { action: 'open', label: isFile ? 'Открыть' : 'Перейти', icon: isFile ? 'visibility' : 'folder_open' },
    ...(isFile ? [{ action: 'download', label: 'Скачать', icon: 'download' }] : []),
    { action: 'rename', label: 'Переименовать', icon: 'edit' },
    ...(isFile ? [{
      action: 'star',
      label: props.item.starred ? 'Убрать из избранного' : 'В избранное',
      icon: props.item.starred ? 'star_border' : 'star',
    }] : []),
    { action: 'move', label: 'Переместить', icon: 'drive_file_move' },
    { action: 'share', label: 'Поделиться', icon: 'share' },
    { action: 'trash', label: 'В корзину', icon: 'delete', danger: true },
  ]
})
</script>
