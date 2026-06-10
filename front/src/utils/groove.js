// Константы раздела «Мой Groove». Набор реакций продублирован на бэке
// в schemas/groove.py (FEED_REACTIONS), магазин — в services/pet_service.py.

export const FEED_REACTIONS = ['🔥', '💪', '👏', '🎉', '😮', '❤️']

export const PET_STAGES = [
  'Яйцо', 'Малыш', 'Непоседа', 'Подросток', 'Взрослый', 'Герой', 'Легенда',
]

export const PET_SPECIES = {
  egg: { emoji: '🥚', title: 'Ещё не вылупился' },
  owl: { emoji: '🦉', title: 'Сова' },
  lark: { emoji: '🐤', title: 'Жаворонок' },
  sprinter: { emoji: '🐆', title: 'Спринтер' },
  marathoner: { emoji: '🐢', title: 'Марафонец' },
  fox: { emoji: '🦊', title: 'Универсал' },
  // Покупные виды (магазин обликов).
  cat: { emoji: '🐱', title: 'Котёнок' },
  dog: { emoji: '🐶', title: 'Щенок' },
  rabbit: { emoji: '🐰', title: 'Крольчонок' },
  frog: { emoji: '🐸', title: 'Лягушонок' },
  chick: { emoji: '🐥', title: 'Цыплёнок' },
  monkey: { emoji: '🐵', title: 'Обезьянка' },
  panda: { emoji: '🐼', title: 'Панда' },
  tiger: { emoji: '🐯', title: 'Тигрёнок' },
  bear: { emoji: '🐻', title: 'Медвежонок' },
  penguin: { emoji: '🐧', title: 'Пингвинёнок' },
  unicorn: { emoji: '🦄', title: 'Единорог' },
  dragon: { emoji: '🐲', title: 'Дракон' },
}

// Природные виды (определяются эволюцией). Покупные не входят сюда —
// они доступны на любой стадии после разблокировки в магазине.
export const NATURAL_SPECIES = new Set(['owl', 'lark', 'sprinter', 'marathoner', 'fox'])

// Малыш всех видов выглядит одинаково — вид проявляется со 2-й стадии.
// Исключение: купленный покупной вид показывается сразу (хозяин же не
// зря платил), но всё равно после вылупления.
export function petEmoji(pet) {
  if (!pet || pet.stage === 0) return '🥚'
  const species = pet?.species
  const isBought = species && !NATURAL_SPECIES.has(species) && species !== 'egg'
  if (pet.stage === 1 && !isBought) return '🐣'
  return PET_SPECIES[species]?.emoji || '🦊'
}

export const SHOP_ITEMS = {
  party: { emoji: '🥳', title: 'Колпак на праздник' },
  cap: { emoji: '🧢', title: 'Кепка' },
  bow: { emoji: '🎀', title: 'Бантик' },
  scarf: { emoji: '🧣', title: 'Шарф' },
  glasses: { emoji: '🕶️', title: 'Очки' },
  headphones: { emoji: '🎧', title: 'Наушники' },
  tophat: { emoji: '🎩', title: 'Цилиндр' },
  crown: { emoji: '👑', title: 'Корона' },
  helmet: { emoji: '⛑️', title: 'Каска дедлайнщика' },
  flower: { emoji: '🌸', title: 'Весенний цветок' },
  icecream: { emoji: '🍦', title: 'Летнее мороженое' },
  pumpkin: { emoji: '🎃', title: 'Осенняя тыква' },
  santa: { emoji: '🎅', title: 'Новогодний колпак' },
}

// Характеры (≡ PERSONALITIES в pet_service.py).
export const PERSONALITIES = {
  lazy: { emoji: '🦥', title: 'Ленивец-мечтатель' },
  night: { emoji: '🌙', title: 'Ночной активист' },
  early: { emoji: '🌅', title: 'Ранняя пташка' },
  energizer: { emoji: '⚡', title: 'Бодрячок-энерджайзер' },
  zen: { emoji: '🧘', title: 'Дзен-марафонец' },
  steady: { emoji: '🪨', title: 'Уравновешенный трудяга' },
}

export const BOSS_EMOJI = {
  'Дедлайнозавр': '🦖',
  'Багоблин': '👺',
  'Прокрастинатор': '🦥',
  'Совещаниус': '🐙',
  'Хаос-гоблин': '👹',
  'Технодолг': '🤖',
}

export function avatarUrl(user) {
  if (!user) return null
  return user.avatar_path ? `/uploads/${user.avatar_path}` : `/api/users/${user.id}/identicon`
}

export function formatMinutes(min) {
  if (!min || min <= 0) return 'меньше минуты'
  const h = Math.floor(min / 60)
  const m = min % 60
  if (h === 0) return `${m} мин`
  if (m === 0) return `${h} ч`
  return `${h} ч ${m} мин`
}

// День для группировки ленты (локальное время пользователя).
export function dayKey(iso) {
  const d = new Date(iso)
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
}

export function dayTitle(key) {
  const [y, m, d] = key.split('-').map(Number)
  const date = new Date(y, m - 1, d)
  const today = new Date()
  const yesterday = new Date(today.getFullYear(), today.getMonth(), today.getDate() - 1)
  const isSame = (a, b) => a.getFullYear() === b.getFullYear()
    && a.getMonth() === b.getMonth() && a.getDate() === b.getDate()
  if (isSame(date, today)) return 'Сегодня'
  if (isSame(date, yesterday)) return 'Вчера'
  return date.toLocaleDateString('ru-RU', { day: 'numeric', month: 'long', weekday: 'short' })
}

// Временная зона дня — опорные точки таймлайна.
export function timeZoneOf(iso) {
  const h = new Date(iso).getHours()
  if (h < 12) return { key: 'morning', title: 'Утро', icon: 'wb_twilight' }
  if (h < 18) return { key: 'day', title: 'День', icon: 'light_mode' }
  return { key: 'evening', title: 'Вечер', icon: 'nights_stay' }
}
