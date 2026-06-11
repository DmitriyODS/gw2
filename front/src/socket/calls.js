import { useCallStore } from '@/stores/call.js'
import {
  showCallNotification, closeCallNotification, playNotifySound,
} from '@/utils/systemNotify.js'

/**
 * Ринг-сигналинг звонков. Медиа-события (кто в комнате, треки, mute) сюда
 * больше не ходят — их доставляет сам LiveKit через services/livekit.js.
 */
export function registerCallSocketHandlers(socket) {
  socket.on('call:incoming', (call) => {
    console.info('[gw2 call] incoming', call)
    const callStore = useCallStore()
    if (callStore.phase !== 'idle') {
      callStore.handleIncoming(call)
      return
    }

    callStore.handleIncoming(call)
    const initiator = call.participants?.find(p => p.role === 'initiator')
    const initiatorName = initiator?.fio || 'Сотрудник'
    const mediaText = call.media === 'audio' ? 'аудиозвонок' : 'видеозвонок'
    showCallNotification(
      `Входящий ${mediaText}`,
      `${initiatorName} звонит вам`,
      {
        callId: call.id,
        onClick: () => window.focus?.(),
      },
    )
    playNotifySound()
  })

  socket.on('call:started', (data) => {
    useCallStore().handleStarted(data)
  })

  socket.on('call:accepted', (data) => {
    closeCallNotification()
    useCallStore().handleAccepted(data)
  })

  socket.on('call:invited', (data) => {
    useCallStore().handleInvited(data)
  })

  socket.on('call:participant-declined', (data) => {
    useCallStore().handleParticipantDeclined(data)
  })

  socket.on('call:ended', (data) => {
    closeCallNotification()
    useCallStore().handleEnded(data)
  })

  socket.on('call:error', (data) => {
    closeCallNotification()
    useCallStore().handleError(data)
  })
}
