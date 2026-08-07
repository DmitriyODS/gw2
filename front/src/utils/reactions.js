import { computed, ref } from 'vue'
import { pickScore, rememberPick } from '@/utils/recentPicks.js'

/**
 * Набор быстрых реакций — общий для сообщений мессенджера и публикаций портала:
 * позитивные, нейтральные и негативные (как в Telegram/Slack).
 */
export const QUICK_REACTIONS = [
  '👍', '👎', '❤️', '🔥', '😂', '🎉', '👏', '🙏', '😮', '🤔',
  '😢', '😡', '🥰', '😍', '🤯', '💯', '🤝', '👀', '💩', '🤣',
]

// Ветка личного счётчика выбора (utils/recentPicks.js): вес растёт от частоты
// и свежести, поэтому вчерашняя привычка обгоняет прошлогоднюю.
const KIND = 'reaction'
// Сколько частых поднимается вперёд. Остальные сохраняют исходный порядок:
// перетасовывать весь набор после каждой реакции — значит терять мышечную
// память на те эмодзи, что человек и так находит глазами.
const PROMOTED = 8

// Счётчик лежит в localStorage и реактивным быть не может — отметка о новом
// выборе пересобирает порядок во всех открытых пикерах сразу.
const picked = ref(0)

/** Запомнить ПОСТАВЛЕННУЮ реакцию (снятие о предпочтении не говорит). */
export function rememberReaction(emoji) {
  if (!emoji) return
  rememberPick(KIND, 0, emoji)
  picked.value++
}

/** Набор для показа: частые впереди, следом остальные в исходном порядке. */
export const suggestedReactions = computed(() => {
  void picked.value
  const top = QUICK_REACTIONS
    .map((emoji) => ({ emoji, score: pickScore(KIND, 0, emoji) }))
    .filter((r) => r.score > 0)
    .sort((a, b) => b.score - a.score)
    .slice(0, PROMOTED)
    .map((r) => r.emoji)
  return [...top, ...QUICK_REACTIONS.filter((e) => !top.includes(e))]
})
