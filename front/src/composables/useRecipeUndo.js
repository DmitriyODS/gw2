import { computed, ref } from 'vue'

/* «Вернуть как было» для оформления, которое применяется сразу: снимок того,
   что стояло на момент открытия редактора, и откат к нему одним действием.
   Подстраховка вместо кнопки «Применить» — правка видна мгновенно, а
   случайно потерянная настройка возвращается. `capture()` снимает новый
   снимок (открыли диалог заново, пришли настройки с сервера). */
export function useRecipeUndo(read, write) {
  const snapshot = ref('null')
  const capture = () => { snapshot.value = JSON.stringify(read() ?? null) }
  capture()

  const changed = computed(() => JSON.stringify(read() ?? null) !== snapshot.value)

  function undo() {
    write(JSON.parse(snapshot.value))
  }

  return { changed, undo, capture }
}
