// Словарь операций выписки кудо-банка: иконка + человекочитаемый текст.
// Единственное место маппинга kind → представление (страница банка, фильтры).

export const LEDGER_META = {
  unit: { icon: 'timer', group: 'earn', text: () => 'Работа: завершённые юниты' },
  task_closed: { icon: 'task_alt', group: 'earn', text: () => 'Работа: закрытая задача' },
  quest: { icon: 'flag', group: 'earn', text: () => 'Награда дневного квеста' },
  adventure: { icon: 'explore', group: 'earn', text: (e) => `Приключение${e.comment ? ` ${e.comment}` : ''}` },
  adventure_recall: { icon: 'undo', group: 'spend', text: (e) => `Досрочный возврат из приключения${e.comment ? ` (${e.comment})` : ''}` },
  season: { icon: 'military_tech', group: 'earn', text: () => 'Награда сезонного трека' },
  feed: { icon: 'restaurant', group: 'spend', text: (e) => (e.comment === 'бульон' ? 'Лечебный бульон' : 'Кормление питомца') },
  walk: { icon: 'directions_walk', group: 'spend', text: () => 'Прогулка с питомцем' },
  heal: { icon: 'healing', group: 'spend', text: () => 'Лечение питомца' },
  stroke: { icon: 'pets', group: 'spend', text: (e) => `Поглаживание питомца${e.counterparty ? ` — ${e.counterparty.fio}` : ''}` },
  shop: { icon: 'shopping_bag', group: 'spend', text: () => 'Покупка в магазине' },
  house: { icon: 'chair', group: 'spend', text: () => 'Декор для домика' },
  transfer_in: { icon: 'call_received', group: 'social', text: (e) => `Перевод от ${e.counterparty?.fio || 'коллеги'}${e.comment ? ` — «${e.comment}»` : ''}` },
  transfer_out: { icon: 'call_made', group: 'social', text: (e) => `Перевод: ${e.counterparty?.fio || 'коллеге'}${e.comment ? ` — «${e.comment}»` : ''}` },
  charity: { icon: 'volunteer_activism', group: 'social', text: (e) => `Взнос в сбор${e.comment ? ` ${e.comment}` : ''}` },
  bank_deposit: { icon: 'savings', group: 'bank', text: () => 'Пополнение вклада' },
  bank_withdraw: { icon: 'savings', group: 'bank', text: () => 'Снятие с вклада' },
  bank_interest: { icon: 'trending_up', group: 'bank', text: () => 'Проценты по вкладу' },
  loan_taken: { icon: 'credit_score', group: 'bank', text: (e) => `Кредит${e.comment ? ` (${e.comment})` : ''}` },
  loan_repaid: { icon: 'credit_score', group: 'bank', text: () => 'Погашение кредита' },
  goal_deposit: { icon: 'target', group: 'bank', text: (e) => `В копилку${e.comment ? ` ${e.comment}` : ''}` },
  goal_withdraw: { icon: 'target', group: 'bank', text: (e) => `Из копилки${e.comment ? ` ${e.comment}` : ''}` },
}

export const ledgerIcon = (e) => LEDGER_META[e.kind]?.icon || 'receipt_long'

export const ledgerText = (e) => {
  try { return LEDGER_META[e.kind]?.text(e) || e.kind } catch { return e.kind }
}

// Группа для фильтров истории: earn | spend | social | bank.
export const ledgerGroup = (e) => LEDGER_META[e.kind]?.group || (e.delta > 0 ? 'earn' : 'spend')

// Короткие подписи категорий — структура трат/приходов в аналитике.
export const KIND_TITLES = {
  unit: 'Юниты', task_closed: 'Задачи', quest: 'Квесты', adventure: 'Приключения',
  adventure_recall: 'Возврат из похода', season: 'Сезонный трек',
  feed: 'Кормление', walk: 'Прогулки', heal: 'Лечение', stroke: 'Поглаживания',
  shop: 'Магазин', house: 'Домик', transfer_in: 'Переводы (вход)',
  transfer_out: 'Переводы (исход)', charity: 'Благотворительность',
  bank_deposit: 'Вклад', bank_withdraw: 'Вклад (снятие)', bank_interest: 'Проценты',
  loan_taken: 'Кредит', loan_repaid: 'Кредит (погашение)',
  goal_deposit: 'Копилки', goal_withdraw: 'Копилки (снятие)',
}

export const kindTitle = (kind) => KIND_TITLES[kind] || kind
