import { usePortalStore } from '@/stores/portal.js'

// Канал gw2:portal:events, доставляется gatewaysvc в комнату all (см.
// back-go/portal/internal/service/*.go — s.bus.Publish).
export function registerPortalSocketHandlers(socket) {
  socket.on('topic:created', (p) => usePortalStore().applyTopicSocket('created', p))
  socket.on('topic:updated', (p) => usePortalStore().applyTopicSocket('updated', p))
  socket.on('topic:deleted', (p) => usePortalStore().applyTopicSocket('deleted', p))

  socket.on('post:new', (p) => usePortalStore().applyPostSocket('new', p))
  socket.on('post:updated', (p) => usePortalStore().applyPostSocket('updated', p))
  socket.on('post:deleted', (p) => usePortalStore().applyPostSocket('deleted', p))
  socket.on('post:pinned', (p) => usePortalStore().applyPostSocket('pinned', p))
  socket.on('post:unpinned', (p) => usePortalStore().applyPostSocket('unpinned', p))

  socket.on('comment:new', (p) => usePortalStore().applyCommentSocket('new', p))
  socket.on('comment:deleted', (p) => usePortalStore().applyCommentSocket('deleted', p))
  socket.on('comment:liked', (p) => usePortalStore().applyCommentSocket('liked', p))

  socket.on('reaction:added', (p) => usePortalStore().applyReactionSocket('added', p))
  socket.on('reaction:removed', (p) => usePortalStore().applyReactionSocket('removed', p))
}
