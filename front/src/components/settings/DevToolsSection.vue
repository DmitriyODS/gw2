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

    <AppCard title="Выпуск" hint="Карточка обновления показывается при первом запуске после выката.">
      <AppRow
        title="Показать уведомление о новой версии"
        hint="Спрашивает выпуск у сервера и показывает карточку — как при первом входе после обновления."
      >
        <AppButton variant="glass" icon="auto_awesome" label="Проверить" @click="checkRelease" />
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
import { announceRelease } from '@/utils/releaseNotice.js'
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

/* Карточку выпуска показывает вход, и дождаться её можно только следующим
   выкатом — здесь она вызывается принудительно: та же проверка, что делает
   каркас при запуске, но без правила «уже видели». */
async function checkRelease() {
  await announceRelease({ force: true })
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
