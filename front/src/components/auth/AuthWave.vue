<template>
  <!--
    Фирменный фон экранов входа: светлый верх и три волны внизу — тот же мотив,
    что и в логотипе. Каждая волна — свой полупрозрачный тон из токенов темы,
    слои просвечивают друг сквозь друга и дают глубину.

    Форма задана маской-синусоидой в один период: маска повторяется по
    горизонтали, а сдвиг СЛОЯ ровно на период делает ход волны бесшовным.
    Размытия под волнами нет намеренно: под ними лежит плавный градиент, от
    его размытия картинка не менялась, а пересчёт стекла шёл каждый кадр
    движения — на экране входа это впустую грузило видеопроцесс.
  -->
  <div class="aw" aria-hidden="true">
    <span class="aw-wave aw-far" />
    <span class="aw-wave aw-mid" />
    <span class="aw-wave aw-near" />
  </div>
</template>

<script setup></script>

<style scoped>
.aw {
  position: fixed;
  inset: 0;
  z-index: 0;
  pointer-events: none;
  overflow: hidden;

  /* Тона волн — из палитры темы, поэтому фон следует за темой и тёмным
     режимом. Объявлены переменными: ниже они ещё разбавляются прозрачностью. */
  --aw-far: color-mix(in oklch, var(--color-primary) 68%, black);
  --aw-mid: var(--color-primary);
  --aw-near: color-mix(in oklch, var(--color-secondary) 46%, white);

  background: var(--color-bg);
  background:
    radial-gradient(120% 78% at 12% -10%,
      color-mix(in oklch, var(--color-primary-container) 75%, transparent), transparent 62%),
    radial-gradient(95% 70% at 100% 4%,
      color-mix(in oklch, var(--color-tertiary-container) 55%, transparent), transparent 66%),
    var(--color-bg);
}

[data-dark='true'] .aw {
  --aw-far: color-mix(in oklch, var(--color-primary) 40%, black);
  --aw-mid: color-mix(in oklch, var(--color-primary) 62%, black);
  --aw-near: color-mix(in oklch, var(--color-secondary) 44%, black);
}

/* ── Общий каркас волны ─────────────────────────────────────────
   -webkit-* объявляем ПЕРВЫМИ: минификатор выбрасывает стандартное
   свойство, если оно стоит раньше префиксного. */
.aw-wave {
  position: absolute;
  left: 0;
  bottom: 0;
  display: block;
  height: clamp(300px, 56vh, 660px);
  -webkit-mask-repeat: repeat-x;
  mask-repeat: repeat-x;
  /* Двигается СЛОЙ, а не маска: mask-position не композиторское свойство, и
     его анимация заставляла браузер каждый кадр заново растрировать три
     полноэкранных слоя — на экране входа это грузило видеопроцесс клиента
     вхолостую. Слой шире экрана ровно на период маски, поэтому сдвиг на
     период по-прежнему бесшовен (как у волн лендинга). */
  will-change: transform;
}

/* Дальняя волна — самая тёмная и самая медленная (глубина). */
.aw-far {
  background: color-mix(in oklch, var(--aw-far) 46%, transparent);
  -webkit-mask-image: url("data:image/svg+xml,%3Csvg%20xmlns='http://www.w3.org/2000/svg'%20viewBox='0%200%20620%20300'%20preserveAspectRatio='none'%3E%3Cpath%20d='M0%2072%20q155%20-58%20310%200%20t310%200%20V300%20H0%20Z'%20fill='black'/%3E%3C/svg%3E");
  mask-image: url("data:image/svg+xml,%3Csvg%20xmlns='http://www.w3.org/2000/svg'%20viewBox='0%200%20620%20300'%20preserveAspectRatio='none'%3E%3Cpath%20d='M0%2072%20q155%20-58%20310%200%20t310%200%20V300%20H0%20Z'%20fill='black'/%3E%3C/svg%3E");
  -webkit-mask-size: 620px 100%;
  mask-size: 620px 100%;
  width: calc(100% + 620px);
  animation: aw-roll-far 21s linear infinite;
}

/* Средняя волна — фирменный цвет. */
.aw-mid {
  background: color-mix(in oklch, var(--aw-mid) 40%, transparent);
  -webkit-mask-image: url("data:image/svg+xml,%3Csvg%20xmlns='http://www.w3.org/2000/svg'%20viewBox='0%200%20420%20300'%20preserveAspectRatio='none'%3E%3Cpath%20d='M0%20136%20q105%20-46%20210%200%20t210%200%20V300%20H0%20Z'%20fill='black'/%3E%3C/svg%3E");
  mask-image: url("data:image/svg+xml,%3Csvg%20xmlns='http://www.w3.org/2000/svg'%20viewBox='0%200%20420%20300'%20preserveAspectRatio='none'%3E%3Cpath%20d='M0%20136%20q105%20-46%20210%200%20t210%200%20V300%20H0%20Z'%20fill='black'/%3E%3C/svg%3E");
  -webkit-mask-size: 420px 100%;
  mask-size: 420px 100%;
  width: calc(100% + 420px);
  animation: aw-roll-mid 14s linear infinite reverse;
}

/* Ближняя волна — светлая, почти прозрачная плёнка на переднем плане. */
.aw-near {
  background: color-mix(in oklch, var(--aw-near) 38%, transparent);
  -webkit-mask-image: url("data:image/svg+xml,%3Csvg%20xmlns='http://www.w3.org/2000/svg'%20viewBox='0%200%20840%20300'%20preserveAspectRatio='none'%3E%3Cpath%20d='M0%20212%20q210%20-62%20420%200%20t420%200%20V300%20H0%20Z'%20fill='black'/%3E%3C/svg%3E");
  mask-image: url("data:image/svg+xml,%3Csvg%20xmlns='http://www.w3.org/2000/svg'%20viewBox='0%200%20840%20300'%20preserveAspectRatio='none'%3E%3Cpath%20d='M0%20212%20q210%20-62%20420%200%20t420%200%20V300%20H0%20Z'%20fill='black'/%3E%3C/svg%3E");
  -webkit-mask-size: 840px 100%;
  mask-size: 840px 100%;
  width: calc(100% + 840px);
  animation: aw-roll-near 27s linear infinite;
}

/* Ход волны: сдвиг слоя ровно на период маски — кадр в конце цикла совпадает
   с начальным, стыка не видно. */
@keyframes aw-roll-far {
  from { transform: translate3d(0, 0, 0); }
  to { transform: translate3d(-620px, 0, 0); }
}

@keyframes aw-roll-mid {
  from { transform: translate3d(0, 0, 0); }
  to { transform: translate3d(-420px, 0, 0); }
}

@keyframes aw-roll-near {
  from { transform: translate3d(0, 0, 0); }
  to { transform: translate3d(-840px, 0, 0); }
}

@media (prefers-reduced-motion: reduce) {
  .aw-wave { animation: none; }
}
</style>
