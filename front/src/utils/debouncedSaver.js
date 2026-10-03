/* Отложенная запись по ключу: оформление применяется на каждом движении
   ползунка, а на сервер уходит одно последнее значение после паузы.
   Промис каждого вызова отражает судьбу ИМЕННО его записи: вытесненный более
   свежим вызовом разрешается сразу и пусто, поэтому ошибку сети вызывающий
   увидит один раз, а не по разу на каждый шаг ползунка. */
export function createDebouncedSaver(delay = 500) {
  const pending = new Map()

  function settle(key) {
    const p = pending.get(key)
    if (!p) return
    clearTimeout(p.timer)
    p.resolve()
    pending.delete(key)
  }

  function save(key, run) {
    settle(key)
    return new Promise((resolve, reject) => {
      const timer = setTimeout(() => {
        pending.delete(key)
        Promise.resolve().then(run).then(resolve, reject)
      }, delay)
      pending.set(key, { timer, resolve })
    })
  }

  // Выход из аккаунта: недописанное чужой сессии не отправляем.
  const clear = () => [...pending.keys()].forEach(settle)

  return { save, cancel: settle, clear }
}
