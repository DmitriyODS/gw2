<template>
  <SegmentedTabs :model-value="current" :tabs="tabs" dense @update:model-value="go" />
</template>

<script setup>
// Единый раздел «Портал»: лента (/portal) и сотрудники (/employees) — две
// вкладки одного хаба; переключение — навигация между существующими вьюхами.
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import SegmentedTabs from '@/components/common/SegmentedTabs.vue'
import { usePortalStore } from '@/stores/portal.js'

const route = useRoute()
const router = useRouter()
const portal = usePortalStore()

const current = computed(() => (route.path.startsWith('/employees') ? 'employees' : 'feed'))

const tabs = computed(() => [
  {
    value: 'feed', label: 'Лента', icon: 'campaign',
    badge: portal.unread ? (portal.unread > 99 ? '99+' : portal.unread) : null,
  },
  { value: 'employees', label: 'Сотрудники', icon: 'groups' },
])

function go(value) {
  if (value === current.value) return
  router.push(value === 'employees' ? '/employees' : '/portal')
}
</script>
