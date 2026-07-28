/* Лимит тарифа исчерпан. Такая ошибка (HTTP 402) приходит из любого раздела —
   создание компании, сотрудник, доска, файл, токены ИИ, — и означает одно и то
   же: нужно в магазин. Поэтому апсейл показывается в ОДНОМ месте (api/client),
   а не расписывается в каждом экране.

   Стор уведомлений импортируется лениво: utils не должен тянуть pinia в
   момент загрузки модуля (иначе ломается порядок инициализации приложения). */

// Одно и то же ограничение за пару секунд показываем один раз: сохранение
// формы часто дёргает несколько запросов подряд.
const shown = new Map()
const REPEAT_MS = 5000

export async function notifyPlanLimit(err) {
  const key = `${err?.error}:${err?.limit_kind || err?.feature || ''}`
  const now = Date.now()
  if (shown.get(key) > now - REPEAT_MS) return
  shown.set(key, now)

  const { useNotificationsStore } = await import('@/stores/notifications.js')
  useNotificationsStore().notify({
    severity: 'warn',
    summary: summaryFor(err),
    detail: err?.message || 'Оформите подписку в магазине, чтобы продолжить.',
    life: 8000,
  })
}

function summaryFor(err) {
  if (err?.error === 'AI_NO_TOKENS') return 'Закончились токены ИИ'
  if (err?.error === 'PLAN_FEATURE_REQUIRED') return 'Возможность платного тарифа'
  return 'Достигнут предел тарифа'
}
