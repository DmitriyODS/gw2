import { useGrooveStore } from '@/stores/groove.js'
import { useNotificationsStore } from '@/stores/notifications.js'

export function registerGrooveSocketHandlers(socket) {
  socket.on('feed:new', (data) => {
    try { useGrooveStore().applyNewEvent(data) } catch {}
  })

  // Опорные точки блока «Сейчас в эфире» — сокет-события юнитов tasksvc
  // (машинных событий ленты unit_started/unit_stopped больше нет).
  const refreshLive = () => {
    try {
      const groove = useGrooveStore()
      if (groove.liveLoaded) groove.fetchLive().catch(() => {})
    } catch {}
  }
  socket.on('unit:started', refreshLive)
  socket.on('unit:stopped', refreshLive)

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
