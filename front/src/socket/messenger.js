import { useAuthStore } from '@/stores/auth.js'
import { useMessengerStore } from '@/stores/messenger.js'
import { showSystemNotification, playNotifySound } from '@/utils/systemNotify.js'
import { stripMarkdown } from '@/utils/markdown.js'

export function registerMessengerSocketHandlers(socket) {
  socket.on('message:new', ({ conversation_id, message, from_user_id }) => {
    const messenger = useMessengerStore()
    const fromMe = from_user_id === useAuthStore().user?.id
    messenger.applyIncomingMessage(conversation_id, message, fromMe)

    if (fromMe) return
    const isActive = messenger.activeConversationId === conversation_id
      && document.visibilityState === 'visible'
      && document.hasFocus()

    if (isActive) return
    // Системные плашки группы (вошёл/вышел/переименовал) не уведомляем.
    if (message.kind === 'system') return
    const conv = messenger.conversations.find(c => c.id === conversation_id)
    if (conv?.muted) return
    const heading = conv?.is_group ? (conv.title || 'Группа') : (conv?.other_user?.fio || 'Сотрудник')
    // Разметка в уведомлении не рендерится — показываем чистый текст.
    const body = stripMarkdown(message.text || '')
      || (message.attachments?.length ? 'Прислал(а) вложение' : 'Новое сообщение')
    showSystemNotification(heading, body, {
      data: { conversation_id },
      onClick: () => {
        window.focus()
        window.dispatchEvent(new CustomEvent('messenger:open-conversation', { detail: { conversation_id } }))
      },
    })
    playNotifySound()
  })

  socket.on('message:read', ({ conversation_id, reader_id, last_read_id }) => {
    useMessengerStore().applyReadReceipt(conversation_id, reader_id, last_read_id ?? null)
  })

  socket.on('group:updated', ({ conversation_id }) => {
    useMessengerStore().applyGroupUpdated(conversation_id)
  })

  socket.on('message:updated', ({ conversation_id, message }) => {
    useMessengerStore().applyMessageUpdated(conversation_id, message)
  })

  socket.on('message:deleted', ({ conversation_id, message_id }) => {
    useMessengerStore().applyMessageDeleted(conversation_id, message_id)
  })

  socket.on('conversation:deleted', ({ conversation_id }) => {
    useMessengerStore().applyConversationDeleted(conversation_id)
  })

  socket.on('conversation:pin', ({ conversation_id, is_pinned }) => {
    useMessengerStore().applyPinChange(conversation_id, is_pinned)
  })

  socket.on('chat_bg:updated', (payload) => {
    useMessengerStore().applyChatBgUpdated(payload)
  })

  socket.on('folders:changed', () => {
    useMessengerStore().applyFoldersChanged()
  })

  socket.on('message:pin', ({ conversation_id, message_id, pinned, message }) => {
    useMessengerStore().applyMessagePin(conversation_id, message_id, pinned, message)
  })

  socket.on('presence:update', (payload) => {
    useMessengerStore().applyPresence(payload)
  })

  // Эфемерный «печатает…» — релей gateway, мимо БД и REST.
  socket.on('typing', (payload) => {
    useMessengerStore().applyTyping(payload)
  })
}
