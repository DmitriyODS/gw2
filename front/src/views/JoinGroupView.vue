<template>
  <div class="jg-page">
    <div class="jg-card">
      <Logo :size="56" class="jg-logo" />
      <template v-if="loading">
        <div class="jg-spinner" />
        <p class="jg-text">Загружаем группу…</p>
      </template>
      <template v-else-if="preview">
        <div class="jg-avatar">
          <img v-if="preview.avatar_path" :src="`/uploads/${preview.avatar_path}`" alt="" />
          <span v-else class="material-symbols-outlined">groups</span>
        </div>
        <h1 class="jg-title">{{ preview.title }}</h1>
        <p class="jg-sub">{{ preview.member_count }} участник{{ plural(preview.member_count) }}</p>
        <button class="jg-btn" :disabled="joining" @click="join">
          <span class="material-symbols-outlined">login</span>
          {{ joining ? 'Вступаем…' : 'Вступить в группу' }}
        </button>
        <router-link to="/messenger" class="jg-link">Не сейчас</router-link>
      </template>
      <template v-else>
        <span class="material-symbols-outlined jg-icon">error</span>
        <p class="jg-text">{{ error }}</p>
        <router-link to="/messenger" class="jg-btn">К сообщениям</router-link>
      </template>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useMessengerStore } from '@/stores/messenger.js'
import { groupInvitePreview } from '@/api/messenger.js'
import Logo from '@/components/common/Logo.vue'

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
.jg-page { position: fixed; inset: 0; display: grid; place-items: center; background: var(--color-surface-low, var(--color-surface)); padding: 24px; }
.jg-card {
  display: flex; flex-direction: column; align-items: center; gap: 14px; padding: 40px 32px;
  border-radius: var(--radius-xl, 20px); background: var(--acrylic-card-bg);
  -webkit-backdrop-filter: var(--acrylic-blur); backdrop-filter: var(--acrylic-blur);
  border: 1px solid var(--acrylic-border); box-shadow: var(--shadow-lg); max-width: 360px; text-align: center;
}
.jg-logo { opacity: 0.9; }
.jg-avatar {
  width: 80px; height: 80px; border-radius: 50%; overflow: hidden; display: grid; place-items: center;
  background: var(--color-primary-container); color: var(--color-on-primary-container);
}
.jg-avatar img { width: 100%; height: 100%; object-fit: cover; }
.jg-avatar .material-symbols-outlined { font-size: 40px; font-variation-settings: 'FILL' 1; }
.jg-title { margin: 0; font-size: 20px; font-weight: 700; color: var(--color-text); }
.jg-sub { margin: 0; font-size: 14px; color: var(--color-text-dim); }
.jg-text { margin: 0; font-size: 15px; color: var(--color-text); }
.jg-icon { font-size: 48px; color: var(--color-error); }
.jg-spinner { width: 36px; height: 36px; border-radius: 50%; border: 3px solid var(--color-outline-dim); border-top-color: var(--color-primary); animation: jg-spin 0.8s linear infinite; }
@keyframes jg-spin { to { transform: rotate(360deg); } }
.jg-btn {
  display: inline-flex; align-items: center; gap: 8px; padding: 12px 22px; border: none; border-radius: 999px;
  background: var(--color-primary); color: var(--color-on-primary); text-decoration: none; font-weight: 600; font-size: 15px; cursor: pointer;
}
.jg-btn:disabled { opacity: 0.6; cursor: default; }
.jg-btn .material-symbols-outlined { font-size: 20px; }
.jg-link { color: var(--color-text-dim); text-decoration: none; font-size: 14px; }
</style>
