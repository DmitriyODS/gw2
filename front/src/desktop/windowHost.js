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
