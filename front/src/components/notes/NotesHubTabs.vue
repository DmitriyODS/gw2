<template>
  <SegmentedTabs
    :model-value="current"
    :tabs="tabs"
    :full-width="fullWidth"
    dense
    @update:model-value="go"
  />
</template>

<script setup>
// Единый раздел «Заметки»: заметки (/notes) и ежедневник (/diaries) — две
// вкладки одного хаба; переключение — навигация между вьюхами
// (по образцу PortalHubTabs).
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import SegmentedTabs from '@/components/common/SegmentedTabs.vue'

defineProps({ fullWidth: { type: Boolean, default: false } })

const route = useRoute()
const router = useRouter()

const current = computed(() => (route.path.startsWith('/diaries') ? 'diary' : 'notes'))

const tabs = [
  { value: 'notes', label: 'Заметки', icon: 'note_stack' },
  { value: 'diary', label: 'Ежедневник', icon: 'event_note' },
]

function go(value) {
  if (value === current.value) return
  router.push(value === 'diary' ? '/diaries' : '/notes')
}
</script>
