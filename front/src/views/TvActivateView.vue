<template>
  <AuthShell
    title="активация ТВ-режима"
    subtitle="Подтвердите код в приложении — «Авторизовать ТВ-киоск»."
    size="sm"
    back="/login"
  >
    <DeviceLinkInitiator kind="tv" @session="onSession" />
  </AuthShell>
</template>

<script setup>
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth.js'
import { connectSocket } from '@/socket/index.js'
import AuthShell from '@/components/auth/AuthShell.vue'
import DeviceLinkInitiator from '@/components/auth/DeviceLinkInitiator.vue'

const router = useRouter()
const authStore = useAuthStore()

// ТВ-киоск авторизован (сессия уже привязана к компании) — уходим в ТВ-режим.
function onSession(session) {
  authStore.applyLinkSession(session)
  connectSocket()
  router.push('/tv')
}
</script>
