import { ref } from 'vue'

/* Прячет плавающие виджеты (питомец, FAB хаба, FAB раздела), пока пользователь
   скроллит ВНИЗ на мобильном: поверх контента их слишком много, и при
   прокрутке они загораживают карточки. Скролл вверх или пауза — виджеты сразу
   возвращаются (пружинное появление — класс .float-spring в main.css).
   Слушатель — capture на document, поэтому ловит скролл ЛЮБОГО внутреннего
   контейнера (у разделов свои scroll-области, не window).

   Использование: :class="{ 'float-hidden': floatingHidden }" на корне виджета
   с базовым классом .float-fade (+ .float-spring на «кружке»). */

const IDLE_MS = 800
const DELTA = 3 // px — отсев дрожания тачпада/резинки iOS

export const floatingHidden = ref(false)

let timer = null
let installed = false
const lastTops = new WeakMap() // контейнер → последний scrollTop

function onAnyScroll(e) {
  if (window.innerWidth > 768) return
  const el = e.target === document
    ? (document.scrollingElement || document.documentElement)
    : e.target
  if (!el || typeof el.scrollTop !== 'number') return
  const prev = lastTops.get(el) ?? el.scrollTop
  const delta = el.scrollTop - prev
  lastTops.set(el, el.scrollTop)

  if (delta > DELTA) floatingHidden.value = true
  else if (delta < -DELTA) floatingHidden.value = false // вверх — показать сразу

  clearTimeout(timer)
  timer = setTimeout(() => { floatingHidden.value = false }, IDLE_MS)
}

/* Идемпотентная установка глобального слушателя (зовут все потребители). */
export function installFloatingHide() {
  if (installed || typeof document === 'undefined') return
  installed = true
  document.addEventListener('scroll', onAnyScroll, { capture: true, passive: true })
}
