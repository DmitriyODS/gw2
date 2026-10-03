<template>
  <AuthShell :title="title" :subtitle="subtitle" size="sm">
    <div class="jg">
      <BrandLoader v-if="loading" :size="64" />

      <template v-else-if="preview">
        <div class="jg-avatar">
          <img v-if="preview.avatar_path" loading="lazy" decoding="async" :src="`/uploads/${preview.avatar_path}`" alt="" />
          <span v-else class="material-symbols-outlined">groups</span>
        </div>
        <AppButton variant="filled" size="lg" block :loading="joining" @click="join">Вступить в группу</AppButton>
        <AppButton tag="router-link" to="/messenger" variant="text" label="Не сейчас" />
      </template>

      <template v-else>
        <AppButton
          tag="router-link"
          to="/messenger"
          variant="filled"
          label="К сообщениям"
          size="lg" block
        />
      </template>
    </div>
  </AuthShell>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import AppButton from '@/components/ui/AppButton.vue'
import { useRoute, useRouter } from 'vue-router'
import { useMessengerStore } from '@/stores/messenger.js'
import { groupInvitePreview } from '@/api/messenger.js'
import AuthShell from '@/components/auth/AuthShell.vue'
import BrandLoader from '@/components/common/BrandLoader.vue'

const route = useRoute()
const router = useRouter()
const messenger = useMessengerStore()

const loading = ref(true)
const joining = ref(false)
const preview = ref(null)
const error = ref('')

function plural(n) {
  const d = n % 10, dd = n % 100
  if (d === 1 && dd !== 11) return ''
  if (d >= 2 && d <= 4 && (dd < 10 || dd >= 20)) return 'а'
  return 'ов'
}

const title = computed(() => {
  if (loading.value) return 'Загружаем группу'
  return preview.value ? preview.value.title : 'Ссылка не работает'
})

const subtitle = computed(() => {
  if (loading.value) return ''
  if (!preview.value) return error.value || 'Ссылка недействительна или отозвана'
  return `${preview.value.member_count} участник${plural(preview.value.member_count)}`
})

onMounted(async () => {
  try {
    preview.value = await groupInvitePreview(route.params.code)
  } catch (e) {
    error.value = e?.message || 'Ссылка недействительна или отозвана'
  } finally {
    loading.value = false
  }
})

async function join() {
  joining.value = true
  try {
    const id = await messenger.joinGroupByCode(route.params.code)
    router.replace(`/messenger/${id}`)
  } catch (e) {
    error.value = e?.message || 'Не удалось вступить'
    preview.value = null
  } finally {
    joining.value = false
  }
}
</script>

<style scoped>
.jg {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 14px;
}

.jg-avatar {
  width: 80px;
  height: 80px;
  border-radius: 50%;
  overflow: hidden;
  display: grid;
  place-items: center;
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
}

.jg-avatar img { width: 100%; height: 100%; object-fit: cover; }
.jg-avatar .material-symbols-outlined { font-size: 40px; font-variation-settings: 'FILL' 1; }



</style>
