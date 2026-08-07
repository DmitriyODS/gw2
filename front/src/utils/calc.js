/**
 * Вычисление арифметических выражений: строка Hola считает прямо во время
 * ввода («2+2*2», «(1200-350)/3», «15% от 2000», «10^3»), калькулятор-окно
 * тем же движком считает и научные формы («sqrt(9)», «sin(30)», «5!»).
 *
 * Свой разбор, а не eval: строка приходит от пользователя и не должна
 * исполняться как код. Рекурсивный спуск по грамматике
 *   expr    := term (('+' | '-') term)*
 *   term    := power (('*' | '/' | '×' | '÷' | '%') power)*
 *   power   := unary ('^' power)?          — правоассоциативная степень
 *   unary   := ('+' | '-')* postfix
 *   postfix := atom '!'*                   — факториал
 *   atom    := number | const | func '(' expr ')' | '(' expr ')'
 * Проценты: «a % b» — остаток от деления, «a% от b» и «a + b%» — доля.
 */

const NUM = /^\d+(?:[.,]\d+)?/
const NAME = /^[a-zA-Zπ]+/

const CONSTANTS = { pi: Math.PI, 'π': Math.PI, e: Math.E }

/* Тригонометрия зависит от режима углов калькулятора (град/рад), поэтому
   функции получают его вторым параметром. */
const FUNCTIONS = {
  sqrt: (x) => Math.sqrt(x),
  cbrt: (x) => Math.cbrt(x),
  abs: (x) => Math.abs(x),
  ln: (x) => Math.log(x),
  log: (x) => Math.log10(x),
  exp: (x) => Math.exp(x),
  sin: (x, a) => Math.sin(toRad(x, a)),
  cos: (x, a) => Math.cos(toRad(x, a)),
  tan: (x, a) => Math.tan(toRad(x, a)),
  asin: (x, a) => fromRad(Math.asin(x), a),
  acos: (x, a) => fromRad(Math.acos(x), a),
  atan: (x, a) => fromRad(Math.atan(x), a),
  sinh: (x) => Math.sinh(x),
  cosh: (x) => Math.cosh(x),
  tanh: (x) => Math.tanh(x),
  round: (x) => Math.round(x),
  floor: (x) => Math.floor(x),
  ceil: (x) => Math.ceil(x),
}

const toRad = (x, angle) => (angle === 'deg' ? (x * Math.PI) / 180 : x)
const fromRad = (x, angle) => (angle === 'deg' ? (x * 180) / Math.PI : x)

// Факториал считаем только для неотрицательных целых: дробный требует
// гамма-функции, а калькулятору она не нужна.
function factorial(x) {
  if (!Number.isInteger(x) || x < 0 || x > 170) throw new Error('bad factorial')
  let out = 1
  for (let i = 2; i <= x; i += 1) out *= i
  return out
}

function tokenize(input) {
  const src = input
    .replace(/\s+/g, ' ')
    .replace(/×/g, '*')
    .replace(/÷/g, '/')
    .replace(/−/g, '-')
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
    // Имена — функции и константы; проверяем после «от», иначе кириллица
    // сюда и не попадёт.
    const name = NAME.exec(rest)
    if (name) {
      const key = name[0].toLowerCase()
      if (!FUNCTIONS[key] && !(key in CONSTANTS)) return null
      tokens.push({ t: 'name', v: key })
      i += name[0].length
      continue
    }
    if ('+-*/^()%!'.includes(ch)) {
      tokens.push({ t: ch })
      i += 1
      continue
    }
    return null // незнакомый символ — это не выражение, а обычный запрос
  }
  return tokens
}

function parse(tokens, angle) {
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
    return postfix()
  }

  function postfix() {
    let value = atom()
    while (eat('!')) value = factorial(value)
    return value
  }

  function atom() {
    const tok = peek()
    if (!tok) throw new Error('unexpected end')
    if (tok.t === 'num') { pos += 1; return tok.v }
    if (tok.t === 'name') {
      pos += 1
      const fn = FUNCTIONS[tok.v]
      if (!fn) return CONSTANTS[tok.v]
      // Аргумент функции — всегда в скобках: «sin 30» слишком неоднозначно
      // рядом с умножением подстановкой.
      if (!eat('(')) throw new Error('function needs (')
      const arg = expr()
      if (!eat(')')) throw new Error('unbalanced')
      return fn(arg, angle)
    }
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
 * Считает любое выражение, включая научные формы («sqrt(9)», «5!», «pi»).
 * Возвращает число или null, если выражение неполное либо бессмысленное.
 * Это путь калькулятора-окна: там пользователь заведомо считает.
 */
export function evaluate(input, { angle = 'rad' } = {}) {
  const text = String(input || '').trim()
  if (!text) return null

  const tokens = tokenize(text)
  if (!tokens || !tokens.length) return null

  try {
    const value = parse(tokens, angle)
    return Number.isFinite(value) ? value : null
  } catch {
    return null
  }
}

/**
 * Считает выражение в строке поиска. Возвращает число или null, если это не
 * арифметика (обычный поисковый запрос) — потому и требует знак действия:
 * слово «log» или число «2026» пользователь ищет, а не вычисляет.
 */
export function calculate(input) {
  const text = String(input || '').trim()
  if (!text || !/[+\-*/^%×÷!]/.test(text) || !/\d/.test(text)) return null
  return evaluate(text)
}

/** Человекочитаемый результат: без «хвоста» плавающей точки и с разрядами. */
export function formatResult(value) {
  const rounded = Math.round(value * 1e10) / 1e10
  return rounded.toLocaleString('ru-RU', { maximumFractionDigits: 10 })
}
