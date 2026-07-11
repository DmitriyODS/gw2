/* Экранные координаты выделенного фрагмента textarea: сам textarea их не
   отдаёт, поэтому меряем по зеркальному div с теми же текстовыми метриками
   (стандартный приём textarea-caret-position). Возвращает rect ПЕРВОЙ строки
   выделения в координатах viewport — над ней позиционируются контекстные
   меню (Markdown-тулбар мессенджера, меню композера портала). */
export function selectionViewportRect(el, start, end) {
  const style = getComputedStyle(el)
  const mirror = document.createElement('div')
  for (const p of [
    'boxSizing', 'width', 'fontFamily', 'fontSize', 'fontWeight', 'fontStyle',
    'letterSpacing', 'lineHeight', 'textTransform', 'textIndent',
    'paddingTop', 'paddingRight', 'paddingBottom', 'paddingLeft',
    'borderTopWidth', 'borderRightWidth', 'borderBottomWidth', 'borderLeftWidth',
  ]) mirror.style[p] = style[p]
  Object.assign(mirror.style, {
    position: 'fixed', top: '0', left: '0', visibility: 'hidden',
    whiteSpace: 'pre-wrap', overflowWrap: 'break-word',
  })
  mirror.textContent = el.value.slice(0, start)
  const marker = document.createElement('span')
  marker.textContent = el.value.slice(start, end) || '​'
  mirror.appendChild(marker)
  document.body.appendChild(mirror)
  const line = marker.getClientRects()[0] || marker.getBoundingClientRect()
  const mirrorRect = mirror.getBoundingClientRect()
  mirror.remove()
  const elRect = el.getBoundingClientRect()
  return {
    top: elRect.top + (line.top - mirrorRect.top) - el.scrollTop,
    bottom: elRect.top + (line.bottom - mirrorRect.top) - el.scrollTop,
    left: elRect.left + (line.left - mirrorRect.left),
    width: line.width,
  }
}
