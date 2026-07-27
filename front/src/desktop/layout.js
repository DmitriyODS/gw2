/**
 * Размеры каркаса рабочего стола. Держим числа в одном месте: их читают и CSS
 * (через переменную --taskbar-height), и расчёт рабочей области окон, и
 * плавающие виджеты прежнего каркаса (мини-хаб), чтобы не залезать под панель.
 */
import { ref } from 'vue'

export const TASKBAR_HEIGHT = 68
export const TASKBAR_MARGIN = 12

// Панель задач вместе с отступами от края экрана и зазором до окон.
export const TASKBAR_RESERVE = TASKBAR_HEIGHT + TASKBAR_MARGIN * 2

// Включён ли режим рабочего стола (десктоп + авторизованный пользователь).
// Ставит DesktopShell; плавающие виджеты смотрят сюда, чтобы поднять себя
// над панелью задач.
export const shellActive = ref(false)

export function floatingBottomInset() {
  return shellActive.value ? TASKBAR_RESERVE : 0
}
