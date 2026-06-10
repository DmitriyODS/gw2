import { useGrooveStore } from '@/stores/groove.js'
import { useNotificationsStore } from '@/stores/notifications.js'

export function registerGrooveSocketHandlers(socket) {
  socket.on('feed:new', (data) => {
    try {
      const groove = useGrooveStore()
      groove.applyNewEvent(data)
      // Опорные точки live-блока меняются вместе с юнитами.
      if (groove.liveLoaded && groove.isMine(data.company_id)
          && (data.kind === 'unit_started' || data.kind === 'unit_stopped')) {
        groove.fetchLive().catch(() => {})
      }
    } catch {}
  })

  socket.on('feed:reaction', (data) => {
    try { useGrooveStore().applyReaction(data) } catch {}
  })

  socket.on('feed:comment', (data) => {
    try { useGrooveStore().applyComment(data) } catch {}
  })

  socket.on('feed:comment_deleted', (data) => {
    try { useGrooveStore().applyCommentDeleted(data) } catch {}
  })

  socket.on('pet:update', (data) => {
    try { useGrooveStore().applyPetUpdate(data) } catch {}
  })

  socket.on('groove:zap', (data) => {
    try {
      useNotificationsStore().notify({
        severity: 'success',
        summary: 'Заряд энергии!',
        detail: `⚡ ${data.from_fio} зарядил(а) вас энергией`,
      })
    } catch {}
  })

  socket.on('groove:zap-count', (data) => {
    try {
      const groove = useGrooveStore()
      if (groove.isMine(data.company_id)) groove.applyZapCount(data)
    } catch {}
  })

  socket.on('groove:stroke', (data) => {
    try {
      useNotificationsStore().notify({
        severity: 'info',
        summary: 'Вашего Грувика погладили',
        detail: `${data.from_fio} погладил(а) «${data.pet_name}» — вам обоим по груву`,
      })
    } catch {}
  })

  socket.on('raid:update', (data) => {
    try {
      const groove = useGrooveStore()
      groove.applyRaidUpdate(data)
      if (data.defeated_now && groove.isMine(data.company_id)) {
        useNotificationsStore().success('Босс недели повержен! Всем Грувикам — награда 🏆')
      }
    } catch {}
  })
}
