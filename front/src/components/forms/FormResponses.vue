<template>
  <div class="fr">
    <EmptyState
      v-if="!responses.length && !loading"
      icon="inbox"
      tone="soft"
      title="Ответов пока нет"
      :subtitle="hint"
    />

    <AppStack v-else :gap="8">
      <AppRow
        v-for="r in responses"
        :key="r.id"
        :title="who(r)"
        :hint="when(r)"
        icon="assignment_turned_in"
        clickable
        @click="$emit('open', r)"
      >
        <AppChip
          v-if="quiz"
          size="sm"
          :tone="r.graded ? 'success' : 'neutral'"
          :label="`${r.score} / ${r.max_score}`"
          :title="r.graded ? 'Оценка открыта' : 'Оценка ещё не открыта'"
        />
        <AppButton
          v-if="canEdit"
          variant="icon" size="sm" tone="danger" icon="delete"
          title="Удалить ответ" aria-label="Удалить ответ"
          @click.stop="$emit('remove', r)"
        />
      </AppRow>
    </AppStack>
  </div>
</template>

<script setup>
/* Собранные ответы списком: кто и когда ответил, балл теста. Разбор одного
   ответа открывается карточкой — колонок у формы бывает три десятка, и таблица
   из них в окне не помещается. */
import { computed } from 'vue'
import AppButton from '@/components/ui/AppButton.vue'
import AppChip from '@/components/ui/AppChip.vue'
import AppRow from '@/components/ui/AppRow.vue'
import AppStack from '@/components/ui/AppStack.vue'
import EmptyState from '@/components/common/EmptyState.vue'

const props = defineProps({
  responses: { type: Array, default: () => [] },
  loading: { type: Boolean, default: false },
  quiz: { type: Boolean, default: false },
  canEdit: { type: Boolean, default: false },
  search: { type: String, default: '' },
})

defineEmits(['open', 'remove'])

const hint = computed(() => (props.search
  ? 'По этому запросу ничего не нашлось'
  : 'Поделитесь ссылкой или назначьте форму — ответы появятся здесь.'))

// who — кто отвечал: имя из аккаунта, представившийся гость либо «Аноним»
// (форма могла принимать ответы без входа).
function who(r) {
  return r.user_name?.trim() || r.name?.trim() || 'Аноним'
}

function when(r) {
  const d = new Date(r.created_at)
  const time = Number.isNaN(d.getTime())
    ? ''
    : d.toLocaleString('ru-RU', { day: '2-digit', month: '2-digit', year: '2-digit', hour: '2-digit', minute: '2-digit' })
  return [time, r.email].filter(Boolean).join(' · ')
}
</script>

<style scoped>
.fr { display: flex; flex-direction: column; gap: 10px; min-width: 0; }
</style>
