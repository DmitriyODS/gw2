// Иконка раздела портала живёт в одном поле topic.icon и хранит либо ключ
// material-symbols, либо эмодзи. Ключи material — только ASCII-буквы, цифры и
// подчёркивания, поэтому всё остальное трактуем как эмодзи: отдельная колонка
// «тип иконки» ради этого различия не нужна.
const MATERIAL_KEY = /^[a-z0-9_]+$/

export function isEmojiIcon(icon) {
  return !!icon && !MATERIAL_KEY.test(icon)
}
