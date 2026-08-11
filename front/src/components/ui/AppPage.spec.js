import { describe, it, expect } from 'vitest'
import { defineComponent, h, nextTick } from 'vue'
import { mount } from '@vue/test-utils'
import AppPage from './AppPage.vue'

/* Раздел объявляет слоты условно — «поиск появляется, когда приехали данные»
   (`<template v-if="registry" #search>`). Объект слотов НЕ реактивен, поэтому
   любая проверка их наличия обязана считаться на отрисовке: вычисляемое
   значение поверх слотов кэшировалось навсегда, и строка поиска на публичной
   странице реестра не появлялась вовсе. */
const Host = defineComponent({
  props: { ready: { type: Boolean, default: false } },
  setup(props) {
    return () => h(
      AppPage,
      { title: 'Реестр оборудования', showTitle: true },
      {
        default: () => h('div', { class: 'body' }, 'записи'),
        ...(props.ready ? { search: () => h('input', { class: 'probe-search' }) } : {}),
        ...(props.ready ? { subhead: () => h('div', { class: 'probe-sub' }) } : {}),
      },
    )
  },
})

describe('AppPage', () => {
  it('показывает строку поиска, появившуюся ПОСЛЕ первой отрисовки', async () => {
    const w = mount(Host, { props: { ready: false } })
    expect(w.find('.head-line').exists()).toBe(false)

    await w.setProps({ ready: true })
    await nextTick()

    expect(w.find('.head-line').exists()).toBe(true)
    expect(w.find('.head-line .head-search .probe-search').exists()).toBe(true)
    expect(w.find('.head-sub .probe-sub').exists()).toBe(true)
  })

  it('снимает строку управления, когда слоты исчезли', async () => {
    const w = mount(Host, { props: { ready: true } })
    expect(w.find('.head-line').exists()).toBe(true)

    await w.setProps({ ready: false })
    await nextTick()

    expect(w.find('.head-line').exists()).toBe(false)
    // Шапка с названием при этом остаётся — раздел не должен «терять» имя.
    expect(w.find('.page-title').text()).toBe('Реестр оборудования')
  })
})
