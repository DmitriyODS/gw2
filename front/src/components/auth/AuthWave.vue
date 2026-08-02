<template>
  <!--
    Фирменный фон экранов входа: светлый верх и три волны внизу — тот же мотив,
    что и в логотипе. КАЖДАЯ волна — самостоятельное матовое стекло: свой
    полупрозрачный тон из токенов темы поверх backdrop-размытия того, что уже
    нарисовано под ней (фон и предыдущие волны), поэтому слои просвечивают друг
    сквозь друга и дают глубину.

    Форма задана маской-синусоидой в один период: маска повторяется по
    горизонтали, а её сдвиг ровно на период делает ход волны бесшовным.
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
  right: 0;
  bottom: 0;
  display: block;
  height: clamp(300px, 56vh, 660px);
  -webkit-mask-repeat: repeat-x;
  mask-repeat: repeat-x;
  will-change: mask-position;
}

/* Дальняя волна — самая тёмная и самая размытая (глубина). */
.aw-far {
  background: color-mix(in oklch, var(--aw-far) 46%, transparent);
  -webkit-backdrop-filter: blur(18px) saturate(1.25);
  backdrop-filter: blur(18px) saturate(1.25);
  -webkit-mask-image: url("data:image/svg+xml,%3Csvg%20xmlns='http://www.w3.org/2000/svg'%20viewBox='0%200%20620%20300'%20preserveAspectRatio='none'%3E%3Cpath%20d='M0%2072%20q155%20-58%20310%200%20t310%200%20V300%20H0%20Z'%20fill='black'/%3E%3C/svg%3E");
  mask-image: url("data:image/svg+xml,%3Csvg%20xmlns='http://www.w3.org/2000/svg'%20viewBox='0%200%20620%20300'%20preserveAspectRatio='none'%3E%3Cpath%20d='M0%2072%20q155%20-58%20310%200%20t310%200%20V300%20H0%20Z'%20fill='black'/%3E%3C/svg%3E");
  -webkit-mask-size: 620px 100%;
  mask-size: 620px 100%;
  animation: aw-roll-far 21s linear infinite;
}

/* Средняя волна — фирменный цвет. */
.aw-mid {
  background: color-mix(in oklch, var(--aw-mid) 40%, transparent);
  -webkit-backdrop-filter: blur(12px) saturate(1.2);
  backdrop-filter: blur(12px) saturate(1.2);
  -webkit-mask-image: url("data:image/svg+xml,%3Csvg%20xmlns='http://www.w3.org/2000/svg'%20viewBox='0%200%20420%20300'%20preserveAspectRatio='none'%3E%3Cpath%20d='M0%20136%20q105%20-46%20210%200%20t210%200%20V300%20H0%20Z'%20fill='black'/%3E%3C/svg%3E");
  mask-image: url("data:image/svg+xml,%3Csvg%20xmlns='http://www.w3.org/2000/svg'%20viewBox='0%200%20420%20300'%20preserveAspectRatio='none'%3E%3Cpath%20d='M0%20136%20q105%20-46%20210%200%20t210%200%20V300%20H0%20Z'%20fill='black'/%3E%3C/svg%3E");
  -webkit-mask-size: 420px 100%;
  mask-size: 420px 100%;
  animation: aw-roll-mid 14s linear infinite reverse;
}

/* Ближняя волна — светлая, почти прозрачная плёнка на переднем плане. */
.aw-near {
  background: color-mix(in oklch, var(--aw-near) 38%, transparent);
  -webkit-backdrop-filter: blur(8px) saturate(1.15);
  backdrop-filter: blur(8px) saturate(1.15);
  -webkit-mask-image: url("data:image/svg+xml,%3Csvg%20xmlns='http://www.w3.org/2000/svg'%20viewBox='0%200%20840%20300'%20preserveAspectRatio='none'%3E%3Cpath%20d='M0%20212%20q210%20-62%20420%200%20t420%200%20V300%20H0%20Z'%20fill='black'/%3E%3C/svg%3E");
  mask-image: url("data:image/svg+xml,%3Csvg%20xmlns='http://www.w3.org/2000/svg'%20viewBox='0%200%20840%20300'%20preserveAspectRatio='none'%3E%3Cpath%20d='M0%20212%20q210%20-62%20420%200%20t420%200%20V300%20H0%20Z'%20fill='black'/%3E%3C/svg%3E");
  -webkit-mask-size: 840px 100%;
  mask-size: 840px 100%;
  animation: aw-roll-near 27s linear infinite;
}

/* Ход волны: сдвиг маски ровно на её период — кадр в конце цикла совпадает
   с начальным, стыка не видно. */
@keyframes aw-roll-far {
  from { -webkit-mask-position: 0 0; mask-position: 0 0; }
  to { -webkit-mask-position: -620px 0; mask-position: -620px 0; }
}

@keyframes aw-roll-mid {
  from { -webkit-mask-position: 0 0; mask-position: 0 0; }
  to { -webkit-mask-position: -420px 0; mask-position: -420px 0; }
}

@keyframes aw-roll-near {
  from { -webkit-mask-position: 0 0; mask-position: 0 0; }
  to { -webkit-mask-position: -840px 0; mask-position: -840px 0; }
}

@media (prefers-reduced-motion: reduce) {
  .aw-wave { animation: none; }
}

/* Без backdrop-filter (заводской WebView старых Android) стекло имитируем
   плотной заливкой — иначе волны выглядели бы выцветшими. */
@supports not ((backdrop-filter: blur(1px)) or (-webkit-backdrop-filter: blur(1px))) {
  .aw-far { background: color-mix(in oklch, var(--aw-far) 82%, transparent); }
  .aw-mid { background: color-mix(in oklch, var(--aw-mid) 78%, transparent); }
  .aw-near { background: color-mix(in oklch, var(--aw-near) 76%, transparent); }
}
</style>
