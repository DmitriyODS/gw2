<template>
  <!-- Личный dev-чат с командой разработки: бэк создаёт его при первом
       обращении, поэтому строка сразу ведёт в мессенджер. -->
  <SettingRow
    title="Чат с техподдержкой"
    hint="Опишите проблему или предложите улучшение — ответ придёт сюда же, в мессенджер."
    clickable
    :disabled="opening"
    @click="openSupport"
  />
</template>

<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useMessengerStore } from '@/stores/messenger.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import SettingRow from '@/components/common/SettingRow.vue'

const router = useRouter()
const messenger = useMessengerStore()
const notif = useNotificationsStore()

const opening = ref(false)

async function openSupport() {
  if (opening.value) return
  opening.value = true
  try {
    const convId = await messenger.openDevChat()
    router.push(`/messenger/${convId}`)
  } catch (e) {
    notif.error(e.message || 'Не удалось открыть чат техподдержки')
  } finally {
    opening.value = false
  }
}
</script>
