import { ref } from 'vue'

// Глобальный индикатор перехода между разделами. Роутер поднимает флаг на
// время навигации (включая загрузку чанка вью) — на медленном канале это
// секунды, и без индикации клик по меню выглядит как «ничего не произошло».
export const navProgress = ref(false)

// Мгновенные переходы полосой не отмечаем: на рабочем столе адрес меняется при
// каждом переключении окна (чанк раздела уже загружен), и полоса лишь мигала бы.
const SHOW_DELAY = 150
let timer = null

export function startNavProgress() {
  if (timer) return
  timer = setTimeout(() => { navProgress.value = true }, SHOW_DELAY)
}

export function stopNavProgress() {
  clearTimeout(timer)
  timer = null
  navProgress.value = false
}
