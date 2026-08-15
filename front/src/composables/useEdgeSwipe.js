import { onBeforeUnmount, onMounted } from 'vue'

/* Жест «потянуть от кромки экрана» — приём сенсорных ОС (шторка уведомлений,
   пункт управления): панель вызывается движением от края, а не кнопкой.
   Слушаем документ, как pull-to-refresh: разделы про жест ничего не знают.

   Стартовая зона — узкая полоса у самого края. Свайп через весь экран брать
   нельзя: горизонтальные жесты уже заняты — холст доски панорамируется пальцем,
   ленты и таблицы прокручиваются вбок.

   На Android с жестовой навигацией эта же полоса — системное «назад». Обёртка
   отдаёт нам её середину (GrooveWebView, setSystemGestureExclusionRects); выше
   и ниже сработает система, и это ожидаемо: назад — тоже осмысленный ответ. */

const EDGE = 24        // ширина полосы у кромки, где жест начинается
const DISTANCE = 56    // сколько пройти внутрь экрана, чтобы жест засчитался
const SLOPE = 36       // допустимый увод по вертикали до отмены

/** edge — от какой кромки тянут ('right' | 'left'); onSwipe — что открыть. */
export function useEdgeSwipe({ edge = 'right', onSwipe, enabled = () => true }) {
  let startX = 0
  let startY = 0
  let tracking = false

  function onTouchStart(e) {
    if (!enabled() || e.touches.length !== 1) return
    const t = e.touches[0]
    const fromEdge = edge === 'right' ? window.innerWidth - t.clientX : t.clientX
    if (fromEdge > EDGE) return
    startX = t.clientX
    startY = t.clientY
    tracking = true
  }

  function onTouchMove(e) {
    if (!tracking) return
    const t = e.touches[0]
    const dx = t.clientX - startX
    if (Math.abs(t.clientY - startY) > SLOPE) { tracking = false; return }
    // Внутрь экрана: от правой кромки — влево, от левой — вправо.
    const inward = edge === 'right' ? -dx : dx
    if (inward < DISTANCE) return
    tracking = false
    onSwipe()
  }

  function stop() {
    tracking = false
  }

  onMounted(() => {
    document.addEventListener('touchstart', onTouchStart, { passive: true })
    document.addEventListener('touchmove', onTouchMove, { passive: true })
    document.addEventListener('touchend', stop, { passive: true })
    document.addEventListener('touchcancel', stop, { passive: true })
  })

  onBeforeUnmount(() => {
    document.removeEventListener('touchstart', onTouchStart)
    document.removeEventListener('touchmove', onTouchMove)
    document.removeEventListener('touchend', stop)
    document.removeEventListener('touchcancel', stop)
  })
}
