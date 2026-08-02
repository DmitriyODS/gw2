import { describe, it, expect } from 'vitest'
import { globSync, readFileSync } from 'node:fs'

/**
 * Каждый компонент, использованный в шаблоне, должен быть импортирован.
 *
 * Сборка такое не ловит: `<script setup>` резолвит компоненты в рантайме, и
 * забытый импорт превращается в предупреждение «Failed to resolve component»
 * прямо у пользователя, а вместо кнопок остаётся пустое место. Проверка нужна
 * при массовых правках разметки — например, когда разделы переводят на общее
 * ядро компонентов.
 */

// Глобально зарегистрированные (vue-router) — их импортировать не нужно.
const GLOBAL = new Set(['RouterLink', 'RouterView'])

const BUILTIN = new Set([
  'Template', 'Transition', 'TransitionGroup', 'KeepAlive', 'Teleport',
  'Suspense', 'Component', 'Slot',
])

function stripComments(html) {
  return html.replace(/<!--[\s\S]*?-->/g, '')
}

function templateOf(source) {
  // Верхнеуровневый <template> компонента: до последнего закрывающего тега.
  const start = source.indexOf('<template>')
  const end = source.lastIndexOf('</template>')
  return start === -1 || end === -1 ? '' : source.slice(start + 10, end)
}

function scriptOf(source) {
  const m = source.match(/<script setup[^>]*>([\s\S]*?)<\/script>/)
  return m ? m[1] : ''
}

describe('резолвинг компонентов в шаблонах', () => {
  // Тесты запускаются из корня фронта (см. vitest.config.js).
  const root = `${process.cwd()}/src`
  const files = globSync(['components/**/*.vue', 'views/*.vue'], { cwd: root })

  it('находит файлы проекта', () => {
    expect(files.length).toBeGreaterThan(100)
  })

  it.each(files)('%s — все компоненты импортированы', (file) => {
    const source = readFileSync(`${root}/${file}`, 'utf8')
    const script = scriptOf(source)
    if (!script) return

    const template = stripComments(templateOf(source))
    const used = new Set(template.match(/<([A-Z][A-Za-z0-9]*)/g)?.map((t) => t.slice(1)) || [])

    const missing = [...used].filter(
      (name) => !GLOBAL.has(name) && !BUILTIN.has(name) && !new RegExp(`\\b${name}\\b`).test(script),
    )

    expect(missing, `не импортированы: ${missing.join(', ')}`).toEqual([])
  })
})
