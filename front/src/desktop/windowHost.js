/**
 * Хост оверлеев окна — цель <Teleport> для модалок раздела.
 *
 * Модалка по умолчанию уходит в <body> и накрывает весь экран, включая соседние
 * окна рабочего стола: диалог «Настроек» гасил бы открытый рядом мессенджер.
 * Окно раздаёт поддереву своё тело как цель телепорта, поэтому модалка остаётся
 * внутри своего окна, уезжает вместе с ним и исчезает при его закрытии.
 *
 * Позиционирование при этом менять не нужно: у окна есть transform, а значит оно
 * само служит containing block для `position: fixed` потомков — маска `inset: 0`
 * накрывает окно, а не экран. Единственное, что не перенастраивается само, —
 * единицы вьюпорта (vh/dvh/vw): их пересчитывают на проценты те компоненты,
 * которые ими меряются (см. AppDialog).
 *
 * Вне рабочего стола (мобильный каркас, вход, ТВ-режим) инжекта нет — цель
 * остаётся прежней, `body`, и всё работает как раньше. Поэтому один и тот же
 * компонент (PetDetailModal, ImageLightbox) корректен и в окне раздела, и у
 * глобального плавающего виджета.
 */
import { computed, inject, provide } from 'vue'

const WINDOW_HOST = Symbol('gw-window-host')
const FLOAT_HOST = Symbol('gw-float-host')

/**
 * Отдать поддереву тело окна как цель телепорта.
 * @param {import('vue').Ref<HTMLElement|null>} el — ref на элемент-хост.
 */
export function provideWindowHost(el) {
  provide(WINDOW_HOST, el)
}

/**
 * Цель телепорта для модалки раздела.
 * @returns {{ host: import('vue').ComputedRef<HTMLElement|string>,
 *             inWindow: import('vue').ComputedRef<boolean> }}
 */
export function useModalHost() {
  const el = inject(WINDOW_HOST, null)
  const host = computed(() => el?.value || 'body')
  const inWindow = computed(() => host.value !== 'body')
  return { host, inWindow }
}

/**
 * Отдельный хост для ПЛАВАЮЩИХ виджетов раздела (FAB).
 *
 * Модалку показывают по действию пользователя, а плавающая кнопка висит всё
 * время, пока раздел открыт, — и, улетев в `body`, продолжает висеть над чужим
 * разделом и над стартовым экраном. Поэтому мобильный каркас раздаёт экран
 * разделу именно как float-хост (модалки у него по-прежнему уходят в `body`:
 * на экране всё равно один раздел).
 */
export function provideFloatHost(el) {
  provide(FLOAT_HOST, el)
}

/** Цель телепорта плавающего виджета: свой хост → окно → body. */
export function useFloatHost() {
  const float = inject(FLOAT_HOST, null)
  const win = inject(WINDOW_HOST, null)
  return computed(() => float?.value || win?.value || 'body')
}

/**
 * Раздел примыкает к кромкам своего экрана — сенсорный каркас (телефон и
 * планшет).
 *
 * У окна рабочего стола есть рамка и тень, поэтому раздел внутри отбивается
 * полями. У экрана каркаса рамки нет: поля превращаются в пустую полосу вдоль
 * кромки, а на планшете, где рядом стоят ДВЕ зоны, — ещё и в двойной зазор
 * посередине. Каркас говорит об этом один раз, разделы читают через AppPage и
 * AppListDetail и сами ничего про каркас не знают.
 */
const FLUSH_SHELL = Symbol('gw-flush-shell')

export function provideFlushShell() {
  provide(FLUSH_SHELL, true)
}

/** @returns {boolean} */
export function useFlushShell() {
  return inject(FLUSH_SHELL, false)
}
