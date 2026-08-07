// Встроенные обои рабочего стола: пары «светлая/тёмная» одной картины —
// комплект следует режиму оформления, как токены темы. Файлы лежат статикой
// (`front/public/bg`), поэтому в рецепте живут обычными путями `/bg/...` и
// ничем не отличаются от загруженной пользователем картинки.
//
// В рецепте у такой картинки есть `key` — по нему обои узнаются в выборе и
// пересобираются при чтении настроек, так что переименование файла не оставит
// пользователя с битой ссылкой.

/* КАК ДОБАВИТЬ ОБОИ В КОМПЛЕКТ (без правок сборки и сервера):
   1) положить пару WebP в `front/public/bg/` с именами `<ключ>-light.webp` и
      `<ключ>-dark.webp` — каталог `public` уезжает в сборку как есть и
      раздаётся nginx по тем же путям `/bg/...`;
   2) добавить сюда одну строку `wp('<ключ>', 'Название')`.
   Файлы с другими именами тоже можно — тогда пути передаются третьим
   аргументом (так живут исторические обои ниже). */
const wp = (key, label, files) => ({
  key,
  label,
  light: files?.light || `/bg/${key}-light.webp`,
  dark: files?.dark || `/bg/${key}-dark.webp`,
})

export const WALLPAPERS = [
  wp('wave', 'Волна'),
  wp('star', 'Звезда'),
  wp('gw7', 'Groove Work 7'),
]

// Обои «из коробки»: их видит каждый, кто своих не выбирал.
export const DEFAULT_WALLPAPER_KEY = 'gw7'

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
