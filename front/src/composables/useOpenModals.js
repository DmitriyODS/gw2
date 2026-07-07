// Глобальный счётчик открытых AppDialog — плавающие виджеты (FloatingPet,
// FAB мини-хаба) живут на z-index выше диалоговых масок и обязаны прятаться,
// пока открыт любой диалог (особенно на мобильном, где диалог — bottom sheet
// в той же нижней зоне экрана).
import { computed, ref } from 'vue'

const openCount = ref(0)

export function registerOpenModal() {
  openCount.value++
}

export function unregisterOpenModal() {
  openCount.value = Math.max(0, openCount.value - 1)
}

export const anyModalOpen = computed(() => openCount.value > 0)
