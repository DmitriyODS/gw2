import { describe, it, expect } from 'vitest'
import {
  WIDGETS_PANEL_MAX, WIDGETS_PANEL_MIN, widgetsPanelWidth,
} from './layout.js'

describe('widgetsPanelWidth', () => {
  it('без своей настройки — пятая часть экрана', () => {
    expect(widgetsPanelWidth(0, 1600)).toBe(320)
  })

  it('на узком экране не опускается ниже минимума, на широком — не растёт бесконечно', () => {
    expect(widgetsPanelWidth(0, 900)).toBe(WIDGETS_PANEL_MIN)
    expect(widgetsPanelWidth(0, 4000)).toBe(WIDGETS_PANEL_MAX)
  })

  it('своё значение зажимается границами', () => {
    expect(widgetsPanelWidth(360, 1600)).toBe(360)
    expect(widgetsPanelWidth(120, 1600)).toBe(WIDGETS_PANEL_MIN)
    expect(widgetsPanelWidth(900, 1600)).toBe(WIDGETS_PANEL_MAX)
  })

  it('панель не съедает больше половины экрана', () => {
    // 800px: своя ширина 600 упирается в половину экрана, а не в общий максимум.
    expect(widgetsPanelWidth(600, 800)).toBe(400)
    // Половина уже минимума — читаемость важнее: панель остаётся минимальной.
    expect(widgetsPanelWidth(400, 460)).toBe(WIDGETS_PANEL_MIN)
  })
})
