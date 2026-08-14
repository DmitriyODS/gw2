<template>
  <AppPage
    embedded
    title="Реестры"
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

    <div class="rl">
      <div class="rl-scroll">
        <EmptyState
          v-if="!registries.length"
          size="sm"
          :icon="emptyIcon"
          :title="emptyTitle"
          :subtitle="emptySubtitle"
        />
        <AppStack v-else :gap="6">
          <template v-for="r in registries" :key="r.id">
            <!-- Правка названия занимает место пункта: так видно, что именно
                 переименовывают, и список не прыгает. -->
            <AppInlineEdit
              v-if="renamingId === r.id"
              :model-value="r.name"
              placeholder="Название реестра"
              @save="$emit('rename', r, $event)"
              @cancel="$emit('rename-cancel')"
            />
            <AppRow
              v-else
              :title="r.name"
              :hint="hintOf(r)"
              icon="list_alt"
              dense
              clickable
              :selected="r.id === selectedId"
              @click="$emit('select', r.id)"
              @contextmenu.prevent="$emit('context', r, $event)"
            />
          </template>
        </AppStack>
      </div>

      <!-- Кнопка прижата к низу панели и видна всегда: заводить реестры — самое
           частое действие списка, а прокручивать до конца ради неё незачем. -->
      <div class="rl-foot">
        <AppButton
          variant="filled"
          icon="add"
          label="Создать новый"
          full-width
          @click="$emit('create')"
        />
      </div>
    </div>
  </AppPage>
</template>

<script setup>
/* Левая панель раздела: области, список реестров и создание.

   Своих запросов не делает — всё приходит пропсами и уходит событиями: тот же
   список нужен и разделу, и (в будущем) выбору реестра из других мест. */
import { computed } from 'vue'
import AppButton from '@/components/ui/AppButton.vue'
import AppInlineEdit from '@/components/ui/AppInlineEdit.vue'
import AppPage from '@/components/ui/AppPage.vue'
import AppRow from '@/components/ui/AppRow.vue'
import AppStack from '@/components/ui/AppStack.vue'
import AppTabs from '@/components/ui/AppTabs.vue'
import EmptyState from '@/components/common/EmptyState.vue'

const props = defineProps({
  registries: { type: Array, default: () => [] },
  selectedId: { type: [Number, null], default: null },
  scope: { type: String, default: 'all' },
  /** id реестра, название которого сейчас правят ('' — никакого). */
  renamingId: { type: [Number, null], default: null },
  narrow: { type: Boolean, default: false },
})

defineEmits(['select', 'update:scope', 'create', 'context', 'rename', 'rename-cancel', 'toggle'])

const SCOPES = [
  { value: 'all', label: 'Все' },
  { value: 'mine', label: 'Мои' },
  { value: 'shared', label: 'Поделились' },
  { value: 'company', label: 'Компания' },
]

/* Подпись под названием — чей это реестр. У своих её нет: «мой» и так понятно,
   а лишняя строка у каждого пункта только съедает высоту списка. */
function hintOf(r) {
  return r.my_access === 'owner' ? '' : (r.owner_name || '')
}

const EMPTY = {
  mine: {
    icon: 'list_alt',
    title: 'Своих реестров нет',
    subtitle: 'Создайте первый — кнопка внизу панели.',
  },
  shared: {
    icon: 'group',
    title: 'С вами не делились',
    subtitle: 'Здесь появятся реестры, к которым вам открыли доступ.',
  },
  company: {
    icon: 'domain',
    title: 'Реестров компании нет',
    subtitle: 'Здесь появятся реестры, открытые всей компании.',
  },
  all: {
    icon: 'list_alt',
    title: 'Реестров нет',
    subtitle: 'Создайте первый — кнопка внизу панели.',
  },
}

const emptyIcon = computed(() => EMPTY[props.scope].icon)
const emptyTitle = computed(() => EMPTY[props.scope].title)
const emptySubtitle = computed(() => EMPTY[props.scope].subtitle)
</script>

<style scoped>
/* Тело панели прокручивается само, а подвал с кнопкой остаётся на месте —
   поэтому AppPage идёт без своего скролла (:scroll="false"). */
.rl {
  display: flex;
  flex: 1;
  flex-direction: column;
  min-height: 0;
}

.rl-scroll {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding-bottom: 8px;
}

.rl-foot {
  padding-top: 10px;
  border-top: 1px solid var(--acrylic-border);
}
</style>
