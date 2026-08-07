/* Вид файла по MIME и расширению: значок и способ показа.

   MIME приходит от браузера при загрузке и бывает пустым или врущим
   (application/octet-stream на что угодно), поэтому расширение — равноправный
   источник, а не запасной. */

const BY_EXT = {
  image: ['png', 'jpg', 'jpeg', 'gif', 'webp', 'svg', 'bmp', 'avif', 'heic'],
  video: ['mp4', 'webm', 'mov', 'mkv', 'avi', 'm4v'],
  audio: ['mp3', 'wav', 'ogg', 'm4a', 'flac', 'aac'],
  pdf: ['pdf'],
  doc: ['doc', 'docx', 'odt', 'rtf'],
  sheet: ['xls', 'xlsx', 'ods', 'csv'],
  slides: ['ppt', 'pptx', 'odp'],
  archive: ['zip', 'rar', '7z', 'tar', 'gz', 'bz2'],
  code: ['js', 'ts', 'vue', 'go', 'py', 'java', 'json', 'xml', 'yml', 'yaml', 'sql', 'sh', 'css', 'html'],
  text: ['txt', 'md', 'log'],
}

const ICONS = {
  folder: 'folder',
  image: 'image',
  video: 'movie',
  audio: 'audiotrack',
  pdf: 'picture_as_pdf',
  doc: 'description',
  sheet: 'table',
  slides: 'slideshow',
  archive: 'folder_zip',
  code: 'code',
  text: 'article',
  file: 'draft',
}

/** Вид файла: image | video | audio | pdf | doc | … | file. */
export function fileKind(mime = '', name = '') {
  const type = String(mime || '').toLowerCase()
  if (type.startsWith('image/')) return 'image'
  if (type.startsWith('video/')) return 'video'
  if (type.startsWith('audio/')) return 'audio'
  if (type === 'application/pdf') return 'pdf'

  const ext = String(name || '').split('.').pop()?.toLowerCase() || ''
  for (const [kind, list] of Object.entries(BY_EXT)) {
    if (list.includes(ext)) return kind
  }
  if (type.startsWith('text/')) return 'text'
  return 'file'
}

/** Значок material-symbols для вида файла. */
export function fileIcon(mime = '', name = '') {
  return ICONS[fileKind(mime, name)] || ICONS.file
}

/** Показывается ли файл прямо в приложении (без скачивания). */
export function isPreviewable(mime = '', name = '') {
  return ['image', 'video', 'audio', 'pdf', 'text', 'code'].includes(fileKind(mime, name))
}
