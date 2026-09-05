import router from '@/router/index.js'
import { useDesktopStore } from '@/stores/desktop.js'
import { shellActive } from './layout.js'

/**
 * Открыть раздел по его пути.
 *
 * В каркасе-«ОС» (рабочий стол, мобильный стартовый экран) раздел открывается
 * своим окном/экраном, вне каркаса — обычным переходом роутера. Спрашивают об
 * этом все, кто уводит в раздел «со стороны»: Hola, уведомления, ярлыки.
 */
export function openPath(path) {
  if (shellActive.value) useDesktopStore().open(path)
  else router.push(path)
}
