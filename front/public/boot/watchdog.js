/* Бут-watchdog (ES5, классический скрипт — выполнится в любом WebView).
   Если бандл упал (SyntaxError на старом WebView, битый чанк), страница
   отдана с 200 и нативные фолбэки обёрток не срабатывают — оставался вечный
   белый экран. Показываем читаемую ошибку с кнопками перезагрузки и сброса
   кэша (SW + Cache Storage). Файл лежит отдельно от index.html: CSP страницы
   запрещает встроенные скрипты. */
(function () {
  var errs = []
  function remember(msg) {
    if (msg && errs.length < 3) errs.push(String(msg).slice(0, 300))
  }
  window.addEventListener('error', function (e) {
    if (e && e.target && (e.target.src || e.target.href)) {
      remember('Не загрузился ресурс: ' + (e.target.src || e.target.href))
    } else if (e && e.message) {
      remember(e.message)
    }
  }, true)
  window.addEventListener('unhandledrejection', function (e) {
    remember(e && e.reason && (e.reason.message || e.reason))
  })

  function show() {
    var detail = errs.length
      ? '<div style="margin-top:16px;padding:10px 12px;border-radius:10px;background:rgba(128,128,128,.15);' +
        'font:12px/1.5 monospace;word-break:break-word;text-align:left;opacity:.75">' +
        errs.join('<br>').replace(/</g, '&lt;') + '</div>'
      : ''
    document.body.innerHTML =
      '<div style="min-height:100dvh;display:flex;align-items:center;justify-content:center;' +
      'font-family:system-ui,sans-serif;background:#1a1c1e;color:#e3e2e6;text-align:center">' +
      '<div style="padding:32px;max-width:440px">' +
      '<h1 style="font-size:20px;margin:0 0 8px">Приложение не запустилось</h1>' +
      '<p style="font-size:14px;opacity:.8;margin:0 0 20px;line-height:1.45">Попробуйте перезагрузить. ' +
      'Если не помогает — сбросьте кэш приложения и обновите Android System WebView в Google Play.</p>' +
      '<button id="gw-reload" style="padding:12px 28px;border:0;border-radius:999px;font-size:15px;' +
      'font-weight:600;background:#a8c8ff;color:#0a305f;cursor:pointer">Перезагрузить</button> ' +
      '<button id="gw-reset" style="padding:12px 20px;border:1px solid rgba(227,226,230,.4);border-radius:999px;' +
      'font-size:15px;background:transparent;color:#e3e2e6;cursor:pointer;margin-left:8px">Сбросить кэш</button>' +
      detail + '</div></div>'
    document.getElementById('gw-reload').onclick = function () { location.reload() }
    document.getElementById('gw-reset').onclick = function () {
      var done = function () { location.reload() }
      try {
        var jobs = []
        if (window.caches && caches.keys) {
          jobs.push(caches.keys().then(function (ks) {
            return Promise.all(ks.map(function (k) { return caches.delete(k) }))
          }))
        }
        if (navigator.serviceWorker && navigator.serviceWorker.getRegistrations) {
          jobs.push(navigator.serviceWorker.getRegistrations().then(function (rs) {
            return Promise.all(rs.map(function (r) { return r.unregister() }))
          }))
        }
        Promise.all(jobs).then(done, done)
        setTimeout(done, 3000)
      } catch (e) { done() }
    }
  }

  function check(attempt) {
    var app = document.getElementById('app')
    if (window.__gwBooted || (app && app.children.length > 0)) return
    // Ошибок нет и это первая проверка — возможно, просто медленная сеть
    // ещё качает бандл: даём второй шанс.
    if (!errs.length && attempt === 1) {
      setTimeout(function () { check(2) }, 15000)
      return
    }
    show()
  }
  setTimeout(function () { check(1) }, 15000)
})()
