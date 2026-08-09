/**
 * Какой каркас показывать: окна, планшет или телефон.
 *
 * Окна хороши мышью и плохи пальцем: попасть в кромку окна, потянуть заголовок
 * и прицелиться в кнопку 32×32 на сенсорном экране трудно. Поэтому у большого
 * СЕНСОРНОГО экрана свой каркас — планшетный: раздел открывается во весь экран,
 * рядом можно поставить второй, и всё управление сводится к касанию плитки.
 *
 * Выбор — про УСТРОЙСТВО, а не про человека, поэтому он не едет на сервер
 * вместе с остальными настройками рабочего стола: у одного и того же аккаунта
 * планшет и рабочий ноутбук должны выглядеть по-разному. Хранится локально.
 *
 * `auto` смотрит на грубый указатель (`pointer: coarse` — палец, а не мышь) и
 * ширину: сенсорный ноутбук с мышью остаётся столом, планшет и обёртка на нём
 * получают планшетный каркас. Промах автоопределения человек чинит сам —
 * «Настройки → Рабочий стол → Раскладка».
 */
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { storageGet, storageSet } from '@/utils/storage.js'

const KEY = 'gw_shell_mode'

/** Что можно выбрать руками. */
export const SHELL_MODES = ['auto', 'windows', 'tablet']

// Ниже этого — телефон при любой настройке: две зоны рядом там бессмысленны.
const PHONE_AT = 768
// Планшетом считаем экран, на котором две зоны по ~450px реально помещаются.
const TABLET_AT = 900

const stored = SHELL_MODES.includes(storageGet(KEY)) ? storageGet(KEY) : 'auto'
// Модульное состояние: каркас и настройки читают один и тот же выбор.
export const shellModeSetting = ref(stored)

export function setShellMode(mode) {
  if (!SHELL_MODES.includes(mode)) return
  shellModeSetting.value = mode
  storageSet(KEY, mode)
}

/**
 * @returns {{ shell: import('vue').ComputedRef<'windows'|'tablet'|'phone'>,
 *             touchLarge: import('vue').ComputedRef<boolean> }}
 */
export function useShellMode() {
  const width = ref(typeof window === 'undefined' ? 1280 : window.innerWidth)
  const coarse = ref(typeof window !== 'undefined'
    && window.matchMedia?.('(pointer: coarse)').matches === true)

  function onResize() { width.value = window.innerWidth }

  let mq = null
  function onPointer(e) { coarse.value = e.matches }

  onMounted(() => {
    window.addEventListener('resize', onResize, { passive: true })
    // Указатель меняется на ходу: планшет-трансформер с пристёгнутой клавиатурой.
    mq = window.matchMedia?.('(pointer: coarse)')
    mq?.addEventListener?.('change', onPointer)
  })

  onBeforeUnmount(() => {
    window.removeEventListener('resize', onResize)
    mq?.removeEventListener?.('change', onPointer)
  })

  const touchLarge = computed(() => coarse.value && width.value >= TABLET_AT)

  const shell = computed(() => {
    if (width.value <= PHONE_AT) return 'phone'
    if (shellModeSetting.value === 'tablet') return 'tablet'
    if (shellModeSetting.value === 'windows') return 'windows'
    return touchLarge.value ? 'tablet' : 'windows'
  })

  return { shell, touchLarge }
}
