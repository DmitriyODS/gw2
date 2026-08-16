<template>
  <AppStack :gap="14">
    <AppCard :gap="10">
      <div class="fa-head">
        <div class="fa-counts">
          <span class="fa-value">{{ progress?.responded ?? 0 }}</span>
          <span class="fa-total">из {{ progress?.assigned ?? 0 }}</span>
          <span class="fa-label">ответили</span>
        </div>
        <span class="fa-spacer" />
        <AppButton
          v-if="canEdit"
          variant="filled"
          icon="person_add"
          label="Назначить"
          @click="$emit('assign')"
        />
      </div>
      <div class="fa-track">
        <span class="fa-fill" :style="{ width: percent }" />
      </div>
    </AppCard>

    <EmptyState
      v-if="!people.length"
      icon="how_to_reg"
      tone="soft"
      title="Форма никому не назначена"
      subtitle="Назначьте её людям или компании — здесь будет видно, кто ответил."
    />

    <AppCard v-else :gap="4">
      <AppRow
        v-for="p in people"
        :key="p.user_id"
        :title="p.name"
        :hint="hintOf(p)"
        :tone="toneOf(p)"
      >
        <template #lead>
          <img class="fa-avatar" :src="avatarUrl(p)" :alt="p.name" />
        </template>
        <AppChip
          size="sm"
          :tone="p.answered_at ? 'success' : 'neutral'"
          :icon="p.answered_at ? 'check' : 'schedule'"
          :label="p.answered_at ? 'Ответил' : 'Ждём'"
        />
      </AppRow>
    </AppCard>
  </AppStack>
</template>

<script setup>
/* Контроль исполнения: кто из назначенных ответил, а кто нет.

   Список приходит поимённо, включая участников компаний, которым форма
   назначена целиком, — считает его сервер (клиенту членства компаний не
   видны). Просроченные подсвечиваются рамкой строки. */
import { computed } from 'vue'
import AppButton from '@/components/ui/AppButton.vue'
import AppCard from '@/components/ui/AppCard.vue'
import AppChip from '@/components/ui/AppChip.vue'
import AppRow from '@/components/ui/AppRow.vue'
import AppStack from '@/components/ui/AppStack.vue'
import EmptyState from '@/components/common/EmptyState.vue'

const props = defineProps({
  progress: { type: Object, default: null },
  canEdit: { type: Boolean, default: false },
})

defineEmits(['assign'])

const people = computed(() => props.progress?.people || [])

const percent = computed(() => {
  const total = props.progress?.assigned || 0
  if (!total) return '0%'
  return `${Math.round(((props.progress?.responded || 0) / total) * 100)}%`
})

function hintOf(p) {
  const parts = []
  if (p.via) parts.push(p.via)
  if (p.answered_at) parts.push(`ответил ${dateText(p.answered_at)}`)
  else if (p.due_at) parts.push(`срок ${dateText(p.due_at)}`)
  return parts.join(' · ')
}

// Просрочка — только у тех, кто ещё не ответил: у ответившего срок уже неважен.
function toneOf(p) {
  if (p.answered_at) return 'neutral'
  if (p.due_at && new Date(p.due_at) < new Date()) return 'danger'
  return 'neutral'
}

function dateText(value) {
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return ''
  return d.toLocaleDateString('ru-RU', { day: '2-digit', month: '2-digit', year: '2-digit' })
}

// Аватар: загруженное фото либо автоматический identicon — тот же порядок, что
// и всюду на платформе.
function avatarUrl(p) {
  return p.avatar_path ? `/uploads/${p.avatar_path}` : `/api/users/${p.user_id}/identicon`
}
</script>

<style scoped>
.fa-head { display: flex; align-items: center; gap: 10px; flex-wrap: wrap; }
.fa-counts { display: flex; align-items: baseline; gap: 6px; }
.fa-value { font-size: 26px; font-weight: 700; color: var(--color-primary); }
.fa-total { font-size: 15px; color: var(--color-text-dim); }
.fa-label { font-size: 13px; color: var(--color-text-dim); }
.fa-spacer { flex: 1; }

.fa-track { height: 8px; border-radius: 4px; background: var(--color-surface-low); overflow: hidden; }
.fa-fill { display: block; height: 100%; background: var(--color-primary); transition: width 0.2s; }

.fa-avatar { width: 28px; height: 28px; border-radius: 50%; object-fit: cover; }
</style>
