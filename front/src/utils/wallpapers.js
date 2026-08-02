// Встроенные обои рабочего стола: пары «светлая/тёмная» одной картины —
// комплект следует режиму оформления, как токены темы. Файлы лежат статикой
// (`front/public/bg`), поэтому в рецепте живут обычными путями `/bg/...` и
// ничем не отличаются от загруженной пользователем картинки.
//
// В рецепте у такой картинки есть `key` — по нему обои узнаются в выборе и
// пересобираются при чтении настроек, так что переименование файла не оставит
// пользователя с битой ссылкой.

export const WALLPAPERS = [
  { key: 'wave', label: 'Волна', light: '/bg/gw_light_1.webp', dark: '/bg/gw_black_1.webp' },
  { key: 'bloom', label: 'Цветение', light: '/bg/gw_white.webp', dark: '/bg/gw_night.webp' },
]

// Обои «из коробки»: их видит каждый, кто своих не выбирал.
export const DEFAULT_WALLPAPER_KEY = 'wave'

export function wallpaperByKey(key) {
  return WALLPAPERS.find((w) => w.key === key) || null
}

/** Картинка рецепта из встроенных обоев (blur — по умолчанию без размытия). */
export function wallpaperImage(key, blur = 0) {
  const w = wallpaperByKey(key)
  return w ? { key: w.key, url: w.light, dark: w.dark, blur } : null
}

/** Рецепт обоев по умолчанию: только картинка, без градиента и узора —
    поверх фотографии они лишние. */
export function defaultWallpaperRecipe() {
  return {
    gradient: { preset: 'plain', blobs: null },
    pattern: { key: null, emoji: null, alpha: 0, size: 128 },
    image: wallpaperImage(DEFAULT_WALLPAPER_KEY),
  }
}
