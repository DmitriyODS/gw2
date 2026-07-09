/* Варианты сортировки списка задач — общие для рейки фильтров (десктоп)
   и шторки SortSheet (мобильный). */
export const TASK_SORTS = [
  { label: 'Последняя активность', value: 'last_activity', icon: 'history' },
  { label: 'Дата создания', value: 'created_at', icon: 'calendar_today' },
  { label: 'Дата поступления', value: 'received_at', icon: 'inbox' },
  { label: 'Срок исполнения', value: 'deadline', icon: 'event' },
]
