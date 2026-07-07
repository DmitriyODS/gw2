// Константы раздела «Питомцы» (бывший «Мой Groove», лента/рейд/wrapped
// убраны). Каталоги видов/аксессуаров/характеров продублированы на бэке
// в petsvc (domain/consts.go) — держать в синхроне при правках.

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
  hamster: { emoji: '🐹', title: 'Хомячок' },
  hedgehog: { emoji: '🦔', title: 'Ёжик' },
  koala: { emoji: '🐨', title: 'Коала' },
  deer: { emoji: '🦌', title: 'Оленёнок' },
  bee: { emoji: '🐝', title: 'Пчёлка' },
  octopus: { emoji: '🐙', title: 'Осьминожка' },
  wolf: { emoji: '🐺', title: 'Волчонок' },
  lion: { emoji: '🦁', title: 'Львёнок' },
  dolphin: { emoji: '🐬', title: 'Дельфин' },
  whale: { emoji: '🐋', title: 'Китёнок' },
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

// Каталог отображения товаров магазина (эмодзи/название) — сам магазин
// (цена/редкость/лимиты/ротация) приезжает с бэка (dto.ShopItemDTO).
export const SHOP_ITEMS = {
  party: { emoji: '🥳', title: 'Колпак на праздник' },
  cap: { emoji: '🧢', title: 'Кепка' },
  bow: { emoji: '🎀', title: 'Бантик' },
  scarf: { emoji: '🧣', title: 'Шарф' },
  tie: { emoji: '👔', title: 'Галстук' },
  glasses: { emoji: '🕶️', title: 'Очки' },
  headphones: { emoji: '🎧', title: 'Наушники' },
  mask: { emoji: '🥸', title: 'Инкогнито' },
  tophat: { emoji: '🎩', title: 'Цилиндр' },
  medal: { emoji: '🏅', title: 'Медаль' },
  crown: { emoji: '👑', title: 'Корона' },
  helmet: { emoji: '⛑️', title: 'Каска дедлайнщика' },
  santa: { emoji: '🎅', title: 'Новогодний колпак' },
  snowman: { emoji: '☃️', title: 'Снеговик' },
  mittens: { emoji: '🧤', title: 'Тёплые варежки' },
  flower: { emoji: '🌸', title: 'Весенний цветок' },
  butterfly: { emoji: '🦋', title: 'Бабочка' },
  rainbow: { emoji: '🌈', title: 'Радуга' },
  icecream: { emoji: '🍦', title: 'Летнее мороженое' },
  sunhat: { emoji: '👒', title: 'Летняя шляпка' },
  watermelon: { emoji: '🍉', title: 'Долька арбуза' },
  pumpkin: { emoji: '🎃', title: 'Осенняя тыква' },
  leaf: { emoji: '🍁', title: 'Осенний лист' },
  mushroom: { emoji: '🍄', title: 'Грибочек' },
  fireworks: { emoji: '🎆', title: 'Праздничный салют' },
  love: { emoji: '💝', title: 'Подарок ко Дню влюблённых' },
  shamrock: { emoji: '🍀', title: 'Клевер на удачу' },
  rocket: { emoji: '🚀', title: 'Космическая ракета' },
  graduation: { emoji: '🎓', title: 'Выпускной колпак' },
}

export function shopItemTitle(item) {
  if (!item) return ''
  if (item.kind === 'species') return PET_SPECIES[item.key]?.title || item.key
  return SHOP_ITEMS[item.key]?.title || item.key
}

export function shopItemEmoji(item) {
  if (!item) return '🎁'
  if (item.kind === 'species') return PET_SPECIES[item.key]?.emoji || '🐾'
  return SHOP_ITEMS[item.key]?.emoji || '🎁'
}

// Редкость → готовый тег цвета проекта (--tag-<name>-*, tokens.css). Ничего
// нового не заводим — переиспользуем существующую палитру тегов задач.
export const RARITY_TAG = {
  common: 'teal',
  rare: 'blue',
  epic: 'violet',
  legendary: 'amber',
}

export const RARITY_TITLE = {
  common: 'Обычный',
  rare: 'Редкий',
  epic: 'Эпический',
  legendary: 'Легендарный',
}

// Характеры (≡ domain.Personalities в petsvc). Характер пересчитывается
// по юнитам за 21 день; desc — критерий, по которому он был присвоен.
export const PERSONALITIES = {
  lazy: {
    emoji: '🦥', title: 'Ленивец-мечтатель',
    desc: 'За последние 3 недели — не больше 3 юнитов в неделю',
  },
  night: {
    emoji: '🌙', title: 'Ночной активист',
    desc: 'Юниты чаще всего стартуют после 19:00',
  },
  early: {
    emoji: '🌅', title: 'Ранняя пташка',
    desc: 'Юниты чаще всего стартуют до 10 утра',
  },
  energizer: {
    emoji: '⚡', title: 'Бодрячок-энерджайзер',
    desc: 'От 12 юнитов в неделю, в среднем до часа каждый',
  },
  zen: {
    emoji: '🧘', title: 'Дзен-марафонец',
    desc: 'Длинные сессии — в среднем от 110 минут',
  },
  steady: {
    emoji: '🪨', title: 'Уравновешенный трудяга',
    desc: 'Ровный ритм без перекосов по времени и длине сессий',
  },
}

// Приватная история активности питомца (pet_activity_log) — человекочитаемые
// формулировки по kind (≡ domain.ActivityLogEntry.Kind в petsvc).
export const ACTIVITY_META = {
  fed: { icon: 'restaurant', text: (p) => `Покормили${p?.streak ? ` — стрик ${p.streak} дн.` : ''}` },
  walked: { icon: 'directions_walk', text: () => 'Прогулка' },
  healed: { icon: 'healing', text: () => 'Лечение' },
  evolved: { icon: 'auto_awesome', text: (p) => `Эволюция: ${PET_STAGES[p?.stage] || ''}`.trim() },
  sickness_started: { icon: 'sick', text: () => 'Заболел — давно не было юнитов' },
  recovered: { icon: 'favorite', text: () => 'Выздоровел' },
  item_bought: {
    icon: 'shopping_bag',
    text: (p) => `Куплено: ${shopItemTitle(p)}${p?.mystery ? ' (сюрприз дня)' : ''}`,
  },
  item_equipped: { icon: 'styler', text: (p) => `Надето: ${shopItemTitle(p)}` },
  stroked_by: { icon: 'pets', text: () => 'Коллега погладил питомца' },
  adventure_started: {
    icon: 'explore',
    text: (p) => `Отправился в приключение${p?.place ? ` ${p.place}` : ''}`,
  },
  adventure_returned: {
    icon: 'explore',
    text: (p) => `Вернулся из приключения: +${p?.kudos ?? 0} кудосов, +${p?.xp ?? 0} XP`,
  },
}

export function activityText(entry) {
  const meta = ACTIVITY_META[entry?.kind]
  if (!meta) return entry?.kind || ''
  try { return meta.text(entry.payload || {}) } catch { return entry.kind }
}

export function activityIcon(entry) {
  return ACTIVITY_META[entry?.kind]?.icon || 'info'
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
