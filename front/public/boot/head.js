/* Неблокирующие шрифты и счётчик Яндекс.Метрики. Отдельным файлом, а не
   инлайном в index.html: CSP страницы запрещает встроенные скрипты и
   обработчики-атрибуты (прежний onload="this.media='all'"). */
(function () {
  // Шрифты вставляются скриптом: блокирующий stylesheet с недоступного
  // fonts.googleapis.com держал белый экран до сетевого таймаута.
  var fonts = document.createElement('link')
  fonts.rel = 'stylesheet'
  fonts.href = 'https://fonts.googleapis.com/css2?family=Roboto+Flex:opsz,wght@8..144,100..1000' +
    '&family=Material+Symbols+Outlined:opsz,wght,FILL,GRAD@20..48,100..700,0..1,-50..200'
  document.head.appendChild(fonts)

  /* Метрика. SPA: defer:true — первый просмотр не шлётся автоматически, все
     хиты (включая первый) отправляет router.afterEach (utils/metrika.js).
     Dev-стенд не считаем: localhost, голый IP, имя вида host.local и порт
     Vite — стенд открывают и с телефона по имени машины. */
  if (/^(localhost$|\d+\.\d+\.\d+\.\d+$|.+\.local$)/.test(location.hostname) || location.port === '5173') return
  (function (m, e, t, r, i, k, a) {
    m[i] = m[i] || function () { (m[i].a = m[i].a || []).push(arguments) }
    m[i].l = 1 * new Date()
    for (var j = 0; j < document.scripts.length; j++) { if (document.scripts[j].src === r) { return } }
    k = e.createElement(t), a = e.getElementsByTagName(t)[0], k.async = 1, k.src = r, a.parentNode.insertBefore(k, a)
  })(window, document, 'script', 'https://mc.yandex.ru/metrika/tag.js?id=110869595', 'ym')

  window.ym(110869595, 'init', {
    ssr: true, defer: true, webvisor: true, clickmap: true, ecommerce: 'dataLayer',
    referrer: document.referrer, url: location.href, accurateTrackBounce: true, trackLinks: true,
  })
})()
