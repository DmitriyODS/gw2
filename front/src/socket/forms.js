import { useFormsStore } from '@/stores/forms.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { pushNotification } from '@/composables/useDesktopNotifications.js'
import { isSourceEnabled } from '@/utils/notifySettings.js'
import { showSystemNotification } from '@/utils/systemNotify.js'

/* События форм приходят поимённо аудитории формы (владелец, соавторы,
   назначенные), поэтому фильтровать их на клиенте не нужно.

   Особняком стоят form:assigned и form:due: это не обновление списка, а
   поручение человеку. Оно больше нигде не хранится, поэтому кладётся в центр
   уведомлений — переживает перезагрузку страницы. Тем, кто офлайн, то же
   событие ушло FCM-пушем из pushsvc, дубля не будет. */
export function registerFormsSocketHandlers(socket) {
  socket.on('form:created', (p) => useFormsStore().applyFormSocket('created', p))
  socket.on('form:updated', (p) => useFormsStore().applyFormSocket('updated', p))
  socket.on('form:structure', (p) => useFormsStore().applyFormSocket('updated', p))
  socket.on('form:deleted', (p) => useFormsStore().applyFormSocket('deleted', p))
  socket.on('form:shared', (p) => useFormsStore().applyFormSocket('shared', p))
  socket.on('form:grades', (p) => useFormsStore().applyFormSocket('updated', p))

  socket.on('response:new', (p) => {
    useFormsStore().applyResponseSocket(p)
    if (!isSourceEnabled('forms')) return
    useNotificationsStore().notify({
      severity: 'info',
      summary: 'Новый ответ на форму',
      detail: p?.form_title || '',
      source: 'forms',
    })
  })
  socket.on('response:updated', (p) => useFormsStore().applyResponseSocket(p))
  socket.on('response:deleted', (p) => useFormsStore().applyResponseSocket(p))
  socket.on('response:bulk-deleted', (p) => useFormsStore().applyResponseSocket(p))

  socket.on('form:assigned', (p) => announce(p, 'assigned'))
  socket.on('form:due', (p) => announce(p, 'due'))
}

// announce — поручение заполнить форму: тост, строка в центре уведомлений и
// системное уведомление. Раздел выключен в настройках — молчим целиком.
function announce(payload, kind) {
  useFormsStore().fetchForms()
  if (!isSourceEnabled('forms')) return

  const title = kind === 'due' ? 'Скоро срок ответа' : 'Вам назначили форму'
  const text = payload?.title || 'Откройте раздел «Формы и опросы»'
  const path = payload?.form_id ? `/forms?form=${payload.form_id}` : '/forms'

  pushNotification({
    key: `form-${kind}-${payload?.form_id}`,
    icon: kind === 'due' ? 'schedule' : 'assignment',
    tone: kind === 'due' ? 'alert' : 'primary',
    title,
    text,
    path,
    source: 'forms',
  })
  useNotificationsStore().notify({
    severity: kind === 'due' ? 'warn' : 'info',
    summary: title,
    detail: text,
    life: 10_000,
    source: 'forms',
  })
  showSystemNotification(title, text, {
    data: { type: 'form', form_id: payload?.form_id },
    onClick: () => { window.location.href = path },
  })
}
