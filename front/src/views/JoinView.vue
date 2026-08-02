<template>
  <AuthShell
    :title="loading ? 'подключаем к компании' : 'приглашение недоступно'"
    :subtitle="loading ? 'Секунду, оформляем членство.' : message"
    size="sm"
  >
    <div class="jn">
      <BrandLoader v-if="loading" :size="64" />
      <RouterLink v-else to="/home" class="btn-grad jn-wide">на главную</RouterLink>
    </div>
  </AuthShell>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth.js'
import AuthShell from '@/components/auth/AuthShell.vue'
import BrandLoader from '@/components/common/BrandLoader.vue'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()

const loading = ref(true)
const message = ref('')

onMounted(async () => {
  try {
    await auth.joinCompany(route.params.code)
    // Токен/активная компания уже переключены — уходим в приложение.
    router.replace('/tasks')
  } catch (e) {
    message.value = e?.message || 'Ссылка-приглашение недействительна или истекла'
  } finally {
    loading.value = false
  }
})
</script>

<style scoped>
.jn {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
}

.jn-wide {
  width: 100%;
  justify-content: center;
  height: 44px;
  text-decoration: none;
}
</style>
