<template>
  <AppPage
    embedded
    title="Формы и опросы"
    show-title
    :menu="!narrow"
    menu-icon="left_panel_close"
    menu-label="Свернуть список"
    :scroll="false"
    @menu="$emit('toggle')"
  >
    <!-- Области идут строкой управления: вкладок четыре, и в узкой панели они
         прокручиваются, а не переносятся (AppTabs скроллит их сам). -->
    <template #subhead>
      <AppTabs
        :model-value="scope"
        :tabs="SCOPES"
        variant="tint"
        dense
        full-width
        @update:model-value="$emit('update:scope', $event)"
      />
    </template>

    <div class="fl">
      <div class="fl-scroll">
        <EmptyState
          v-if="!forms.length"
          size="sm"
          :icon="empty.icon"
          :title="empty.title"
          :subtitle="empty.subtitle"
        />
        <AppStack v-else :gap="6">
          <template v-for="f in forms" :key="f.id">
            <AppInlineEdit
              v-if="renamingId === f.id"
              :model-value="f.title"
              placeholder="Название формы"
              maxlength="200"
              @save="$emit('rename', f, $event)"
              @cancel="$emit('rename-cancel')"
            />
            <AppRow
              v-else
              :title="f.title"
              :hint="hintOf(f)"
              :icon="f.quiz ? 'quiz' : 'assignment'"
              dense
              clickable
              :selected="f.id === selectedId"
              @click="$emit('select', f.id)"
              @contextmenu.prevent="$emit('context', f, $event)"
            >
              <AppChip
                size="sm"
                :tone="statusMeta(f.status).tone"
                :label="statusMeta(f.status).label"
              />
              <AppChip
                v-if="f.responses"
                size="sm"
                icon="forum"
                :label="String(f.responses)"
                title="Собрано ответов"
              />
            </AppRow>
          </template>
        </AppStack>
      </div>

      <!-- Кнопка прижата к низу панели и видна всегда: заводить формы — самое
           частое действие списка, а прокручивать до конца ради неё незачем. -->
      <div class="fl-foot">
        <AppButton
          variant="filled"
          icon="add"
          label="Создать форму"
          full-width
          @click="$emit('create')"
        />
      </div>
    </div>
  </AppPage>
</template>

<script setup>
/* Левая панель раздела: области, список форм и создание.

   Своих запросов не делает — всё приходит пропсами и уходит событиями. */
import { computed } from 'vue'
import AppButton from '@/components/ui/AppButton.vue'
import AppChip from '@/components/ui/AppChip.vue'
import AppInlineEdit from '@/components/ui/AppInlineEdit.vue'
import AppPage from '@/components/ui/AppPage.vue'
import AppRow from '@/components/ui/AppRow.vue'
import AppStack from '@/components/ui/AppStack.vue'
import AppTabs from '@/components/ui/AppTabs.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import { statusMeta } from '@/utils/formFields.js'

const props = defineProps({
  forms: { type: Array, default: () => [] },
  selectedId: { type: [Number, null], default: null },
  scope: { type: String, default: 'all' },
  /** id формы, название которой сейчас правят (null — никакой). */
  renamingId: { type: [Number, null], default: null },
  narrow: { type: Boolean, default: false },
})

defineEmits(['select', 'update:scope', 'create', 'context', 'rename', 'rename-cancel', 'toggle'])

const SCOPES = [
  { value: 'all', label: 'Все' },
  { value: 'mine', label: 'Мои' },
  { value: 'assigned', label: 'Назначены' },
  { value: 'shared', label: 'Совместные' },
]

/* Подпись под названием: у чужой формы — хозяин, у назначенной — срок ответа
   и отметка «отвечено». Своя форма подписи не требует. */
function hintOf(f) {
  const parts = []
  if (f.my_access !== 'owner' && f.owner_name) parts.push(f.owner_name)
  if (f.my_due_at) parts.push(f.my_responded ? 'Вы ответили' : `Ответить до ${dueText(f.my_due_at)}`)
  else if (f.my_responded) parts.push('Вы ответили')
  return parts.join(' · ')
}

function dueText(value) {
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return ''
  return d.toLocaleDateString('ru-RU', { day: '2-digit', month: '2-digit' })
}

const EMPTY = {
  mine: {
    icon: 'assignment',
    title: 'Своих форм нет',
    subtitle: 'Создайте первую — кнопка внизу панели.',
  },
  assigned: {
    icon: 'assignment_turned_in',
    title: 'Вам ничего не назначили',
    subtitle: 'Здесь появятся формы, которые нужно заполнить.',
  },
  shared: {
    icon: 'group',
    title: 'Совместных форм нет',
    subtitle: 'Здесь появятся формы, где вам открыли ответы или правку.',
  },
  all: {
    icon: 'assignment',
    title: 'Форм пока нет',
    subtitle: 'Создайте первую — кнопка внизу панели.',
  },
}

const empty = computed(() => EMPTY[props.scope] || EMPTY.all)
</script>

<style scoped>
/* Тело панели прокручивается само, а подвал с кнопкой остаётся на месте —
   поэтому AppPage идёт без своего скролла (:scroll="false"). */
.fl {
  display: flex;
  flex: 1;
  flex-direction: column;
  min-height: 0;
}

.fl-scroll {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding-bottom: 8px;
}

.fl-foot {
  padding-top: 10px;
  border-top: 1px solid var(--acrylic-border);
}
</style>
