<template>
  <AuthShell
    :title="loading ? 'Подключаем к компании' : 'Приглашение недоступно'"
    :subtitle="loading ? 'Секунду, оформляем членство.' : message"
    size="sm"
  >
    <div class="jn">
      <BrandLoader v-if="loading" :size="64" />
      <AppButton
        v-else
        tag="router-link"
        to="/home"
        variant="filled"
        label="На главную"
        size="lg" block
      />
    </div>
  </AuthShell>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import AppButton from '@/components/ui/AppButton.vue'
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

</style>
