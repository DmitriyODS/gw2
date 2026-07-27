/**
 * Собственный «маршрут» окна.
 *
 * useRoute()/useRouter() — это inject(routeLocationKey)/inject(routerKey), и оба
 * ключа vue-router экспортирует публично. Подменяя их в поддереве окна, мы даём
 * каждому окну свой маршрут: разделы читают route.params/query и зовут
 * router.push как обычно, не зная, что живут в окне, — переписывать их не нужно.
 */
import { computed, provide, shallowReactive, markRaw, watch } from 'vue'
import { routeLocationKey, routerKey, routerViewLocationKey } from 'vue-router'
import router from '@/router/index.js'
import { appForPath } from '@/desktop/apps.js'

export function resolveWindowRoute(path) {
  const r = router.resolve(path)
  return {
    path: r.path,
    fullPath: r.fullPath,
    name: r.name,
    params: r.params,
    query: r.query,
    hash: r.hash,
    meta: r.meta,
    // matched — сырые записи маршрутов: проксировать их вглубь незачем.
    matched: markRaw(r.matched),
    redirectedFrom: undefined,
  }
}

/**
 * Готовит маршрут и роутер окна и раздаёт их поддереву.
 * @param {object} win — окно из стора рабочего стола.
 * @param {object} desktop — стор рабочего стола (переходы окна).
 */
export function provideWindowRoute(win, desktop) {
  const winRoute = shallowReactive(resolveWindowRoute(win.path))
  watch(() => win.path, (path) => Object.assign(winRoute, resolveWindowRoute(path)))

  /* Роутер окна: всё как у настоящего (resolve, getRoutes, …), но переходы
     остаются внутри рабочего стола. Свой раздел — переход в этом же окне
     (с историей окна), чужой — открывается своим окном, а внешние экраны
     (вход, промо, ТВ-режим) уходят настоящему роутеру и меняют весь экран. */
  const winRouter = Object.create(router)
  Object.defineProperty(winRouter, 'currentRoute', { value: computed(() => winRoute) })

  function go(to, replace) {
    const resolved = router.resolve(to)
    const app = appForPath(resolved.path)
    if (!app) return router[replace ? 'replace' : 'push'](to)
    if (app.id === win.appId) desktop.navigate(win.id, resolved.fullPath, { replace })
    else desktop.open(resolved.fullPath)
    return Promise.resolve()
  }

  winRouter.push = (to) => go(to, false)
  winRouter.replace = (to) => go(to, true)
  winRouter.back = () => { desktop.back(win.id); return Promise.resolve() }
  winRouter.forward = () => Promise.resolve()
  winRouter.go = (n) => (n < 0 ? winRouter.back() : Promise.resolve())

  provide(routeLocationKey, winRoute)
  provide(routerViewLocationKey, computed(() => winRoute))
  provide(routerKey, winRouter)

  return { winRoute, winRouter }
}
