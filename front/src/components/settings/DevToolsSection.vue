<template>
  <div class="dts">
    <AppInfoBar
      icon="code"
      message="Служебный раздел для проверки самого приложения. Открывается пятью быстрыми нажатиями по номеру сборки в «О приложении»."
    />

    <AppCard title="Уведомления" hint="Проверить, как выглядит и ведёт себя стопка карточек.">
      <AppRow title="Одно уведомление" hint="Обычная карточка со своим сроком жизни из настроек.">
        <AppButton variant="glass" icon="notifications" label="Показать" @click="showOne" />
      </AppRow>

      <AppRow
        title="Сразу несколько"
        hint="Пять карточек разного вида: видно каскад и то, как стопка сдвигает соседей."
      >
        <AppButton variant="glass" icon="notifications_active" label="Показать 5" @click="showMany" />
      </AppRow>
    </AppCard>

    <AppCard title="Раздел">
      <AppRow
        title="Скрыть DevTools"
        hint="Раздел исчезнет из настроек. Вернуть — снова пять быстрых нажатий по номеру сборки."
      >
        <AppButton variant="glass" tone="danger" icon="visibility_off" label="Скрыть" @click="hide" />
      </AppRow>
    </AppCard>
  </div>
</template>

<script setup>
/* Скрытый раздел разработчика. Пока в нём только проверка уведомлений — она
   нужна чаще всего: карточки зависят от угла, срока жизни и настроек
   источников, а дождаться настоящего события ради проверки вёрстки сложно. */
import AppButton from '@/components/ui/AppButton.vue'
import AppCard from '@/components/ui/AppCard.vue'
import AppInfoBar from '@/components/ui/AppInfoBar.vue'
import AppRow from '@/components/ui/AppRow.vue'
import { hideDevTools } from '@/utils/devTools.js'
import { useNotificationsStore } from '@/stores/notifications.js'

const emit = defineEmits(['close'])

const notif = useNotificationsStore()

// Тестовые карточки идут БЕЗ источника: их не должны глушить настройки
// разделов — иначе проверка молчала бы, и было бы непонятно почему.
const SAMPLES = [
  { severity: 'info', summary: 'Тестовое уведомление', detail: 'Так выглядит обычное сообщение приложения.' },
  { severity: 'success', summary: 'Готово', detail: 'Действие завершилось успешно.' },
  { severity: 'warn', summary: 'Внимание', detail: 'Что-то требует вашего решения.' },
  { severity: 'error', summary: 'Ошибка', detail: 'Действие не удалось — вот подробность подлиннее, чтобы проверить перенос строк.' },
  { severity: 'info', summary: 'Ещё одно', detail: 'Пятая карточка — стопка заполнена.' },
]

function showOne() {
  notif.notify({ ...SAMPLES[0], sound: false })
}

function showMany() {
  // Со звуком у каждой был бы залп сигналов — проверяем вид, а не голос.
  SAMPLES.forEach((s) => notif.notify({ ...s, sound: false }))
}

/* Раздел исчезает из списка прямо под ногами — уводим туда, откуда его
   вызывали («О приложении»), иначе настройки просто схлопнулись бы на первый
   попавшийся раздел. */
function hide() {
  hideDevTools()
  notif.notify({ severity: 'info', summary: 'DevTools скрыты', detail: 'Вернуть — пять быстрых нажатий по номеру сборки.' })
  emit('close')
}
</script>

<style scoped>
.dts {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
</style>
