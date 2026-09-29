/* Адрес аватара. Автоматический аватар и выбранный значок отдаёт ОДНА ручка,
   поэтому при смене значка адрес не меняется — и браузер продолжал бы
   показывать прежнюю картинку. Добавляем к нему отпечаток выбора: он меняется
   вместе со значком и заставляет перезапросить. */
export function avatarUrl(user) {
  if (!user) return null
  if (user.avatar_path) return `/uploads/${user.avatar_path}`
  const stamp = user.avatar_emoji ? `?v=${encodeURIComponent(user.avatar_emoji)}` : ''
  return `/api/users/${user.id}/identicon${stamp}`
}
