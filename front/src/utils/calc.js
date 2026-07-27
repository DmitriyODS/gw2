/**
 * Калькулятор строки поиска: считает арифметические выражения прямо во время
 * ввода («2+2*2», «(1200-350)/3», «15% от 2000», «10^3»).
 *
 * Свой разбор, а не eval: строка приходит от пользователя и не должна
 * исполняться как код. Рекурсивный спуск по грамматике
 *   expr   := term (('+' | '-') term)*
 *   term   := power (('*' | '/' | '×' | '÷' | '%') power)*
 *   power  := unary ('^' power)?          — правоассоциативная степень
 *   unary  := ('+' | '-')* atom
 *   atom   := number | '(' expr ')'
 * Проценты: «a % b» — остаток от деления, «a% от b» и «a + b%» — доля.
 */

const NUM = /^\d+(?:[.,]\d+)?/

function tokenize(input) {
  const src = input
    .replace(/\s+/g, ' ')
    .replace(/×/g, '*')
    .replace(/÷/g, '/')
    .replace(/,(?=\d)/g, '.') // десятичная запятая
  const tokens = []
  let i = 0

  while (i < src.length) {
    const rest = src.slice(i)
    const ch = rest[0]

    if (ch === ' ') { i += 1; continue }

    const num = NUM.exec(rest)
    if (num) {
      tokens.push({ t: 'num', v: parseFloat(num[0]) })
      i += num[0].length
      continue
    }
    // «процентов от» / «от» — словесная форма доли: 15% от 2000.
    const of = /^(?:процентов\s+от|процента\s+от|от)(?=\s|$)/i.exec(rest)
    if (of) {
      tokens.push({ t: 'of' })
      i += of[0].length
      continue
    }
    if ('+-*/^()%'.includes(ch)) {
      tokens.push({ t: ch })
      i += 1
      continue
    }
    return null // незнакомый символ — это не выражение, а обычный запрос
  }
  return tokens
}

function parse(tokens) {
  let pos = 0
  const peek = () => tokens[pos]
  const eat = (t) => (peek()?.t === t ? (pos += 1, true) : false)

  function expr() {
    let left = term()
    for (;;) {
      if (eat('+')) {
        const right = term()
        // «a + b%» — прибавить процент от a (как в калькуляторах).
        left = eat('%') ? left + (left * right) / 100 : left + right
      } else if (eat('-')) {
        const right = term()
        left = eat('%') ? left - (left * right) / 100 : left - right
      } else return left
    }
  }

  function term() {
    let left = power()
    for (;;) {
      if (eat('*')) left *= power()
      else if (eat('/')) left /= power()
      else if (peek()?.t === '%') {
        // «15% от 2000» — доля, «7 % 3» — остаток. Процент без операнда справа
        // («2000 + 15%») оставляем сложению — там он считается от левой части.
        const next = tokens[pos + 1]
        if (next?.t === 'of') { pos += 2; left = (left * power()) / 100 }
        else if (next?.t === 'num' || next?.t === '(') { pos += 1; left %= power() }
        else return left
      } else return left
    }
  }

  function power() {
    const base = unary()
    if (eat('^')) return base ** power()
    return base
  }

  function unary() {
    if (eat('-')) return -unary()
    if (eat('+')) return unary()
    return atom()
  }

  function atom() {
    const tok = peek()
    if (!tok) throw new Error('unexpected end')
    if (tok.t === 'num') { pos += 1; return tok.v }
    if (eat('(')) {
      const value = expr()
      if (!eat(')')) throw new Error('unbalanced')
      return value
    }
    throw new Error('unexpected token')
  }

  const value = expr()
  if (pos !== tokens.length) throw new Error('trailing input')
  return value
}

/**
 * Считает выражение. Возвращает число или null, если это не арифметика
 * (обычный поисковый запрос).
 */
export function calculate(input) {
  const text = String(input || '').trim()
  // Без хотя бы одного знака действия это просто число или слово — не считаем.
  if (!text || !/[+\-*/^%×÷]/.test(text) || !/\d/.test(text)) return null

  const tokens = tokenize(text)
  if (!tokens || !tokens.length) return null

  try {
    const value = parse(tokens)
    return Number.isFinite(value) ? value : null
  } catch {
    return null
  }
}

/** Человекочитаемый результат: без «хвоста» плавающей точки и с разрядами. */
export function formatResult(value) {
  const rounded = Math.round(value * 1e10) / 1e10
  return rounded.toLocaleString('ru-RU', { maximumFractionDigits: 10 })
}
