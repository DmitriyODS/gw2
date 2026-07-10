// Единый рендер эмодзи грувиков: системные наборы эмодзи у всех разные
// (а старые устройства часть глифов вообще не знают), поэтому каталог
// питомцев/аксессуаров/декора рисуем забандленными Twemoji SVG (CC-BY 4.0).
// Набор в assets/twemoji ограничен каталогом раздела питомцев — новый эмодзи
// в каталоге = докачать его SVG (имя файла — кодпоинты через дефис, без fe0f).

const files = import.meta.glob('@/assets/twemoji/*.svg', {
  eager: true, query: '?url', import: 'default',
})

const byCode = {}
for (const [path, url] of Object.entries(files)) {
  byCode[path.split('/').pop().replace('.svg', '')] = url
}

function codes(char, stripVariation) {
  return [...char]
    .map((c) => c.codePointAt(0).toString(16))
    .filter((c) => !(stripVariation && c === 'fe0f'))
    .join('-')
}

// URL забандленного SVG для эмодзи; null — глифа в наборе нет (рисуем текстом).
export function emojiAsset(char) {
  if (!char) return null
  return byCode[codes(char, true)] || byCode[codes(char, false)] || null
}
