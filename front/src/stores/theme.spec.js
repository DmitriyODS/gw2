import { describe, it, expect, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useThemeStore } from './theme.js'

/* Личная тема читается из localStorage при СОЗДАНИИ стора — поэтому ключи
   ставим до freshStore(). init() не зовём: он вешает слушателя системной
   темы и тик расписания, а палитру всё равно применяют setAuthPreview/setMode. */
function freshStore() {
  setActivePinia(createPinia())
  return useThemeStore()
}

describe('оформление экранов входа и примерка на регистрации', () => {
  beforeEach(() => { localStorage.clear() })

  it('экран входа встречает классикой, личная тема его не подписывает', () => {
    localStorage.setItem('gw_theme', 'violet')
    const theme = freshStore()
    expect(theme.activePreset).toBe('violet')
    theme.setAuthPreview(true)
    expect(theme.activePreset).toBe('classic')
  })

  it('выбор на регистрации не пишется в localStorage и откатывается при уходе', () => {
    localStorage.setItem('gw_theme', 'violet')
    localStorage.setItem('gw_theme_mode', 'light')
    const theme = freshStore()
    theme.setAuthPreview(true)
    theme.startThemeTrial()

    theme.applyTheme('mint')
    theme.setMode('dark')
    expect(theme.activePreset).toBe('mint')
    expect(theme.dark).toBe(true)
    expect(localStorage.getItem('gw_theme')).toBe('violet')
    expect(localStorage.getItem('gw_theme_mode')).toBe('light')

    theme.cancelThemeTrial()
    expect(theme.activePreset).toBe('classic')  // на экране входа снова классика
    expect(theme.currentPreset).toBe('violet')  // личная тема хозяина устройства
    expect(theme.mode).toBe('light')
    expect(theme.dark).toBe(false)              // светлый/тёмный опять от системы
  })

  it('созданный аккаунт закрепляет примерку, размонтирование её уже не откатывает', () => {
    localStorage.setItem('gw_theme', 'violet')
    const theme = freshStore()
    theme.setAuthPreview(true)
    theme.startThemeTrial()
    theme.applyTheme('mint')
    theme.setMode('dark')

    theme.commitThemeTrial()
    expect(localStorage.getItem('gw_theme')).toBe('mint')
    expect(localStorage.getItem('gw_theme_mode')).toBe('dark')

    theme.cancelThemeTrial()
    expect(theme.currentPreset).toBe('mint')
    theme.setAuthPreview(false)  // вход в приложение
    expect(theme.activePreset).toBe('mint')
    expect(theme.dark).toBe(true)
  })

  it('вне регистрации выбор темы и режима сохраняется сразу', () => {
    const theme = freshStore()
    theme.applyTheme('ocean')
    theme.setMode('dark')
    expect(localStorage.getItem('gw_theme')).toBe('ocean')
    expect(localStorage.getItem('gw_theme_mode')).toBe('dark')
  })
})
