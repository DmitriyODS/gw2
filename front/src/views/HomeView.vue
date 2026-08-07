<template>
  <div class="home-redirect">
    <BrandLoader />
  </div>
</template>

<script setup>
/* Корневой адрес. На рабочем столе (десктоп) это «пустой стол»: каркас
   рисует DesktopShell, а router-view не монтируется вовсе. В мобильном
   каркасе полноценного стола нет — уводим пользователя в его стартовый
   раздел. */
import { onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth.js'
import { landingFor } from '@/router/index.js'
import BrandLoader from '@/components/common/BrandLoader.vue'

const router = useRouter()
const auth = useAuthStore()

onMounted(() => { router.replace(landingFor(auth)) })
</script>

<style scoped>
.home-redirect {
  display: grid;
  place-items: center;
  height: 100%;
  min-height: 60dvh;
}
</style>
