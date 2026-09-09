/**
 * Размеры каркаса рабочего стола. Держим числа в одном месте: их читают и CSS
 * (через переменную --taskbar-height), и расчёт рабочей области окон, и
 * плавающие виджеты (питомец), чтобы не залезать под панель.
 *
 * Каркаса три — настольный (окна), мобильный (экраны разделов) и «Виджеты»
 * (боковая панель плюс один раздел). Панель каркаса есть у всех, но разной
 * толщины и с разной стороны: на телефоне каждый пиксель высоты на счету, а у
 * «Виджетов» панель вертикальная, и её «толщина» — это ширина. Толщину и
 * сторону ставит смонтированный каркас, остальные считают от них.
 */
import { ref } from 'vue'

export const TASKBAR_HEIGHT = 68
export const MOBILE_TASKBAR_HEIGHT = 52
// Планшет: настольная панель в компактном виде — крупнее телефонной (цель для
// пальца), но заметно ниже настольной.
export const TABLET_TASKBAR_HEIGHT = 56
export const MOBILE_STATUSBAR_HEIGHT = 44
export const TASKBAR_MARGIN = 12

/* Каркас «Виджеты»: панель прижата к левой кромке во всю высоту экрана.
   Ширина — пятая часть экрана, но в границах читаемости: уже WIDGETS_PANEL_MIN
   широкая плитка не помещается, шире WIDGETS_PANEL_MAX панель отбирает место у
   раздела. Свёрнутая панель — полоска одних значков. */
export const WIDGETS_PANEL_RATIO = 0.2
export const WIDGETS_PANEL_MIN = 240
export const WIDGETS_PANEL_MAX = 420
export const WIDGETS_RAIL_WIDTH = 64

/**
 * Ширина панели «Виджетов»: личное значение пользователя (0 — не задано),
 * зажатое границами и половиной экрана — на узком ноутбуке пятая часть уже
 * минимума, и панель не должна съедать раздел.
 */
export function widgetsPanelWidth(saved, screenW = window.innerWidth) {
  const base = saved > 0 ? saved : Math.round(screenW * WIDGETS_PANEL_RATIO)
  const max = Math.min(WIDGETS_PANEL_MAX, Math.max(WIDGETS_PANEL_MIN, Math.round(screenW / 2)))
  return Math.min(max, Math.max(WIDGETS_PANEL_MIN, base))
}

// Толщина панели задач текущего каркаса.
export const taskbarHeight = ref(TASKBAR_HEIGHT)

// Отступ панели от кромки экрана: рабочий стол держит её плавающей, мобильный
// каркас прижимает вплотную (там каждый пиксель на счету).
export const taskbarMargin = ref(TASKBAR_MARGIN)

/** Панель задач вместе с отступами от края экрана и зазором до окон. */
export function taskbarReserve() {
  return taskbarHeight.value + taskbarMargin.value * 2
}

/** Сколько ВЫСОТЫ экрана съедает панель каркаса (боковая — нисколько). */
export function taskbarHeightReserve() {
  return taskbarSide.value === 'bottom' || taskbarSide.value === 'top' ? taskbarReserve() : 0
}

// Сторона панели задач хранится в личных настройках; здесь — как она влияет
// на геометрию. Сама панель по-прежнему одной толщины, меняется только край,
// к которому она прижата. Мобильный каркас держит её только снизу.
export const taskbarSide = ref('bottom')

/** Отступы рабочей области под панель задач для текущей стороны. */
export function areaInsets(side = taskbarSide.value) {
  const zero = { top: 0, right: 0, bottom: 0, left: 0 }
  return { ...zero, [side]: taskbarReserve() }
}

// Включён ли режим каркаса-«ОС»: настольный стол либо мобильный стартовый
// экран у авторизованного пользователя. Ставит смонтированный каркас;
// плавающие виджеты и переходы Hola смотрят сюда — в этом режиме раздел
// открывается своим окном/экраном, а не сменой всего экрана роутером.
export const shellActive = ref(false)

// Плавающие виджеты (питомец) не должны залезать под панель каркаса: снизу
// мешает горизонтальная панель, слева — вертикальная (панель задач сбоку и
// панель «Виджетов»), а угол по умолчанию у питомца как раз нижний левый.
export function floatingBottomInset() {
  return shellActive.value && taskbarSide.value === 'bottom' ? taskbarReserve() : 0
}

export function floatingLeftInset() {
  return shellActive.value && taskbarSide.value === 'left' ? taskbarReserve() : 0
}
