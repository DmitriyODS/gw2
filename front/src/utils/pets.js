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
  foxy: { emoji: '🦊', title: 'Лисёнок' },
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
  // Эксклюзивы престиж-поколений (≡ domain.PrestigeSpecies) — не продаются,
  // разблокируются перерождением.
  phoenix: { emoji: '🐦‍🔥', title: 'Феникс' },
  alien: { emoji: '👾', title: 'Космогрувик' },
  robot: { emoji: '🤖', title: 'Робогрувик' },
}

// Поколение → эксклюзивный вид (≡ domain.PrestigeSpecies в petsvc).
export const PRESTIGE_SPECIES = { 2: 'phoenix', 3: 'alien', 5: 'robot' }

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
  // Аксессуары-награды сезонного трека (в магазине не продаются).
  star: { emoji: '⭐', title: 'Звезда сезона' },
  comet: { emoji: '☄️', title: 'Комета' },
}

// Каталог кормов (≡ domain.Foods): цена и прибавка сытости совпадают с бэком,
// эмодзи/название — только на фронте. У каждого вида грувика свой любимый корм
// (pet.favorite_food приходит с бэка), он даёт бонус к сытости и XP.
export const FOODS = [
  { key: 'berry', emoji: '🫐', title: 'Ягодка', price: 6, satiety: 20 },
  { key: 'apple', emoji: '🍎', title: 'Яблоко', price: 8, satiety: 30 },
  { key: 'carrot', emoji: '🥕', title: 'Морковка', price: 10, satiety: 40 },
  { key: 'cookie', emoji: '🍪', title: 'Печенька', price: 12, satiety: 25 },
  { key: 'salad', emoji: '🥗', title: 'Салат', price: 15, satiety: 45 },
  { key: 'fish', emoji: '🐟', title: 'Рыбка', price: 18, satiety: 55 },
  { key: 'cake', emoji: '🍰', title: 'Тортик', price: 25, satiety: 70 },
  { key: 'steak', emoji: '🥩', title: 'Стейк', price: 30, satiety: 85 },
]

export const foodMeta = (key) => FOODS.find((f) => f.key === key) || FOODS[2]

// Каталог отображения декора домика (эмодзи/название) — цены и владение
// приезжают с бэка (dto.HouseDecorDTO, ≡ domain.HouseDecor).
export const DECOR_ITEMS = {
  chair: { emoji: '🪑', title: 'Стульчик' },
  plant: { emoji: '🪴', title: 'Растение' },
  picture: { emoji: '🖼️', title: 'Картина' },
  books: { emoji: '📚', title: 'Книжная полка' },
  bed: { emoji: '🛏️', title: 'Кроватка' },
  sofa: { emoji: '🛋️', title: 'Диванчик' },
  teddy: { emoji: '🧸', title: 'Плюшевый друг' },
  console: { emoji: '🎮', title: 'Приставка' },
  piano: { emoji: '🎹', title: 'Пианино' },
  fountain: { emoji: '⛲', title: 'Фонтан' },
  disco: { emoji: '🪩', title: 'Диско-шар' },
  garland: { emoji: '🎊', title: 'Сезонная гирлянда' },
  goldfish: { emoji: '🐠', title: 'Аквариум' },
  fireplace: { emoji: '🔥', title: 'Камин' },
}

export function decorTitle(key) { return DECOR_ITEMS[key]?.title || key }
export function decorEmoji(key) { return DECOR_ITEMS[key]?.emoji || '📦' }

// Награда сезонного трека → человекочитаемые название/эмодзи.
export function seasonRewardMeta(reward) {
  if (!reward) return { emoji: '🎁', title: '' }
  if (reward.kind === 'kudos') return { emoji: '💰', title: `+${reward.amount} кудосов` }
  if (reward.kind === 'decor') return { emoji: decorEmoji(reward.key), title: decorTitle(reward.key) }
  return {
    emoji: shopItemEmoji({ kind: 'accessory', key: reward.key }),
    title: shopItemTitle({ kind: 'accessory', key: reward.key }),
  }
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

// Потребности грувика (≡ domain.Needs в petsvc): шкалы 0..100 тают со
// временем, пустая ведёт в свою болезнь (кроме общения — оно только кормит
// настроение). Порядок массива = порядок показа шкал.
export const NEEDS = [
  { key: 'satiety', emoji: '🍖', title: 'Сытость', hint: 'Тает за сутки. Кормите — иначе истощение.' },
  { key: 'energy', emoji: '⚡', title: 'Энергия', hint: 'Работа и прогулки утомляют. Восполняет сон.' },
  { key: 'hygiene', emoji: '🫧', title: 'Чистота', hint: 'Прогулки пачкают. Восполняет купание.' },
  { key: 'social', emoji: '💬', title: 'Общение', hint: 'Растёт, когда грувика гладят коллеги.' },
]

// Болезни (≡ domain.Ailments в petsvc): у каждой своя причина и рецепт.
// Неверное лечение почти не помогает — cure перечисляет, что реально лечит.
export const AILMENTS = {
  blues: { emoji: '😔', title: 'Хандра', cure: 'Юниты, прогулка, аптечка' },
  hunger: { emoji: '🥣', title: 'Истощение', cure: 'Еда (бульон), сон' },
  cold: { emoji: '🤒', title: 'Простуда', cure: 'Сон, аптечка' },
  grime: { emoji: '🧼', title: 'Грязнуля', cure: 'Купание' },
}

export function ailmentMeta(key) {
  return AILMENTS[key] || { emoji: '🤒', title: 'Болезнь', cure: 'Забота' }
}

// Подпись настроения (≡ domain.MoodTitle) — бэк присылает mood_title, но
// зоопарку коллег он приходит только числом.
export function moodTitle(mood) {
  if (mood >= 80) return 'Отличное'
  if (mood >= 60) return 'Хорошее'
  if (mood >= 40) return 'Обычное'
  if (mood >= 20) return 'Так себе'
  return 'Плохое'
}

// Эмодзи-настроение для карточек: лицо грувика в зависимости от состояния.
export function moodEmoji(pet) {
  if (!pet) return '🙂'
  if (pet.sick) return ailmentMeta(pet.ailment).emoji
  const mood = pet.mood ?? 100
  if (mood >= 80) return '😄'
  if (mood >= 60) return '🙂'
  if (mood >= 40) return '😐'
  if (mood >= 20) return '😕'
  return '😢'
}

// Самая запущенная потребность питомца — ей и посвящена подсказка на
// карточке коллеги («его давно не гладили»).
export function worstNeed(pet) {
  if (!pet?.needs) return null
  let worst = null
  for (const need of NEEDS) {
    const value = pet.needs[need.key] ?? 100
    if (!worst || value < worst.value) worst = { ...need, value }
  }
  return worst
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
  healed: { icon: 'healing', text: () => 'Лечение аптечкой' },
  slept: { icon: 'bedtime', text: () => 'Выспался' },
  bathed: { icon: 'shower', text: () => 'Купание' },
  evolved: { icon: 'auto_awesome', text: (p) => `Эволюция: ${PET_STAGES[p?.stage] || ''}`.trim() },
  sickness_started: {
    icon: 'sick',
    text: (p) => `Заболел: ${ailmentMeta(p?.ailment).title.toLowerCase()}`,
  },
  recovered: { icon: 'favorite', text: () => 'Выздоровел' },
  ran_away: {
    icon: 'directions_run',
    text: (p) => `${p?.name || 'Грувик'} сбежал — болел ${p?.days ?? ''} дней без лечения`,
  },
  item_bought: {
    icon: 'shopping_bag',
    text: (p) => `Куплено: ${shopItemTitle(p)}${p?.mystery ? ' (сюрприз дня)' : ''}`,
  },
  item_equipped: { icon: 'styler', text: (p) => `Надето: ${shopItemTitle(p)}` },
  stroked_by: {
    icon: 'pets',
    text: (p) => `Коллега погладил питомца${p?.kudos ? ` — +${p.kudos} кудосов` : ''}`,
  },
  adventure_started: {
    icon: 'explore',
    text: (p) => `Отправился в приключение${p?.place ? ` ${p.place}` : ''}`,
  },
  adventure_returned: {
    icon: 'explore',
    text: (p) => `Вернулся из приключения: +${p?.kudos ?? 0} кудосов, +${p?.xp ?? 0} XP`,
  },
  prestige: {
    icon: 'auto_awesome',
    text: (p) => `Перерождение — поколение ${p?.generation ?? '?'}`
      + (p?.unlocked ? `, открыт вид «${PET_SPECIES[p.unlocked]?.title || p.unlocked}»` : ''),
  },
  season_reward: {
    icon: 'military_tech',
    text: (p) => `Награда сезона за ${p?.threshold ?? '?'} кудосов: ${seasonRewardMeta(p).title}`,
  },
  house_bought: {
    icon: 'chair',
    text: (p) => `Обновка для домика: ${decorTitle(p?.key)}`,
  },
  kudos_sent: {
    icon: 'send_money',
    text: (p) => `Перевод коллеге: −${p?.amount ?? 0} кудосов`,
  },
  kudos_received: {
    icon: 'redeem',
    text: (p) => `Перевод от коллеги: +${p?.amount ?? 0} кудосов${p?.comment ? ` — «${p.comment}»` : ''}`,
  },
  bank_deposit: { icon: 'savings', text: (p) => `Пополнение вклада: ${p?.amount ?? 0} кудосов` },
  bank_withdraw: { icon: 'savings', text: (p) => `Снятие с вклада: ${p?.amount ?? 0} кудосов` },
  bank_interest: { icon: 'trending_up', text: (p) => `Проценты по вкладу: +${p?.amount ?? 0} кудосов` },
  loan_taken: { icon: 'credit_score', text: (p) => `Кредит: +${p?.amount ?? 0} кудосов (долг ${p?.debt ?? '?'})` },
  loan_repaid: { icon: 'credit_score', text: (p) => `Погашение кредита: −${p?.amount ?? 0} кудосов` },
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
