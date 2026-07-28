import { useBillingStore } from '@/stores/billing.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { pushNotification } from '@/composables/useDesktopNotifications.js'

/* События биллинга приходят только в комнату владельца: покупка, продление,
   начисление токенов, продажа своего товара и решение модерации. Любое из них
   меняет лимиты, поэтому витрина перечитывается — карточки места и токенов
   должны показывать актуальные цифры без перезагрузки страницы. */
export function registerBillingSocketHandlers(socket) {
  const refresh = () => useBillingStore().applyEvent()

  socket.on('billing:subscription', refresh)
  socket.on('billing:addon', refresh)
  socket.on('billing:tokens', refresh)
  socket.on('billing:product', refresh)

  // Счёт на автопродление больше нигде не всплывает — кладём его в центр
  // уведомлений, иначе подписка молча закончится.
  socket.on('billing:renewal', (p) => {
    refresh()
    pushNotification({
      key: `billing-renewal-${p?.order_id}`,
      icon: 'autorenew',
      tone: 'warn',
      title: 'Подписку пора продлить',
      text: 'Счёт на продление ждёт оплаты в разделе «Магазин».',
      path: '/store?tab=orders',
    })
  })

  socket.on('billing:expired', () => {
    refresh()
    pushNotification({
      key: 'billing-expired',
      icon: 'card_membership',
      tone: 'warn',
      title: 'Тариф закончился',
      text: 'Вернулись лимиты бесплатного тарифа — оформить подписку можно в магазине.',
      path: '/store?tab=subs',
    })
  })

  // Продажа своего товара и вердикт модерации — их автор ждёт.
  socket.on('billing:sale', (p) => {
    useNotificationsStore().notify({
      severity: 'success',
      summary: 'Ваш товар купили',
      detail: 'Выручка зачислена в раздел «Мои товары».',
      life: 6000,
    })
    pushNotification({
      key: `billing-sale-${p?.product_id}-${Date.now()}`,
      icon: 'sell',
      tone: 'success',
      title: 'Ваш товар купили',
      text: 'Выручка зачислена на кошелёк автора.',
      path: '/store?tab=mine',
    })
  })

  socket.on('billing:product_review', (p) => {
    const approved = p?.status === 'published'
    useNotificationsStore().notify({
      severity: approved ? 'success' : 'warn',
      summary: approved ? 'Товар опубликован' : 'Товар отклонён',
      detail: approved ? 'Он появился на витрине магазина.' : (p?.reason || 'Проверьте замечания в разделе «Мои товары».'),
      life: 7000,
    })
  })
}
