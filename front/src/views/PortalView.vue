<template>
  <!-- Общий каркас admin-page — та же геометрия, что у вкладки «Сотрудники»:
       переключение вкладок хаба не должно сдвигать интерфейс. -->
  <div class="admin-page portal" :class="{ 'has-panel': !isMobile, 'has-bg': feedBgOn, 'bg-mobile': feedBgOn && isMobile }">
    <!-- Стеклянная панель-хаб (как правая панель ежедневника): тулбар + лента
         внутри. На десктопе — панель, на мобиле обёртка растворяется. -->
    <div class="hub-panel">
      <!-- Обои ленты внутри панели (клипаются скруглением); только десктоп. -->
      <ChatBackgroundLayer v-if="feedBgOn" :recipe="store.background" />
      <header class="admin-sticky">
      <div class="portal-toolbar">
        <PortalHubTabs class="portal-hub-tabs" />
        <SearchField
          v-model="searchInput"
          placeholder="Поиск по постам…"
          hotkey
          :collapsible="false"
          @update:model-value="onSearch"
          @clear="clearSearch"
        />

        <button
          class="btn-glass portal-manage-btn"
          title="Оформление ленты"
          aria-label="Оформление ленты"
          @click="bgDialogOpen = true"
        >
          <span class="material-symbols-outlined">palette</span>
          <span class="portal-btn-label">Оформление</span>
        </button>
        <button
          v-if="isAdmin()"
          class="btn-glass portal-manage-btn"
          title="Управление разделами"
          aria-label="Управление разделами"
          @click="topicsDialogOpen = true"
        >
          <span class="material-symbols-outlined">tune</span>
          <span class="portal-btn-label">Разделы</span>
        </button>
        <button class="btn-grad" @click="openComposer(null)">
          <span class="material-symbols-outlined">edit</span>
          <span class="portal-btn-label">Написать пост</span>
        </button>
      </div>

      <!-- Единая строка фильтров: разделы + популярные хештеги (тренды, как в
           соцсетях) в одном горизонтальном скролле — на мобильных не отъедает
           вторую строку. Разделитель отделяет теги от разделов. -->
      <div class="portal-topics">
        <button
          class="portal-topic-chip"
          :class="{ active: store.filters.topicId == null && !store.filters.tag }"
          @click="store.setTopic(null)"
        >Все</button>
        <button
          v-for="t in store.topics"
          :key="t.id"
          class="portal-topic-chip"
          :class="{ active: store.filters.topicId === t.id }"
          :style="chipStyle(t, store.filters.topicId === t.id)"
          @click="store.setTopic(t.id)"
        >
          <span v-if="store.filters.topicId === t.id" class="material-symbols-outlined portal-chip-check">check</span>
          {{ t.name }}
        </button>

        <template v-if="store.popularTags.length && !store.filters.search">
          <span class="portal-filter-sep" aria-hidden="true" />
          <button
            v-for="t in store.popularTags"
            :key="t.tag"
            class="portal-tag-chip"
            :class="{ active: store.filters.tag === t.tag }"
            @click="store.setTag(store.filters.tag === t.tag ? null : t.tag)"
          >
            #{{ t.tag }}
            <span class="portal-tag-count">{{ t.count }}</span>
          </button>
        </template>
      </div>
    </header>

    <div class="admin-body">
      <div class="portal-feed">
      <div v-if="store.filters.tag" class="portal-tag-banner">
        <span class="material-symbols-outlined">tag</span>
        <span class="portal-tag-banner-text">Посты с тегом <strong>#{{ store.filters.tag }}</strong></span>
        <button class="portal-tag-clear" type="button" @click="store.setTag(null)">
          <span class="material-symbols-outlined">close</span>
          Сбросить
        </button>
      </div>
      <div v-if="store.loadingPosts" class="portal-status">
        <BrandLoader :size="64" />
      </div>

      <template v-else>
        <section v-if="highlightPost" class="portal-section">
          <div class="portal-section-title">
            <span class="material-symbols-outlined">open_in_new</span>
            Пост по ссылке
          </div>
          <PostCard :post="highlightPost" @edit="openComposer" @delete="confirmDelete" @forward="openForward" />
        </section>

        <section v-if="store.pinnedPosts.length" class="portal-section">
          <div class="portal-section-title">
            <span class="material-symbols-outlined">keep</span>
            Закреплено
          </div>
          <div class="portal-posts">
            <PostCard
              v-for="p in store.pinnedPosts"
              :key="p.id"
              :post="p"
              @edit="openComposer"
              @delete="confirmDelete"
              @forward="openForward"
            />
          </div>
        </section>

        <section class="portal-section">
          <EmptyState
            v-if="!store.posts.length && !store.pinnedPosts.length"
            icon="campaign"
            tone="soft"
            title="Пока пусто"
            subtitle="Станьте первым, кто поделится новостью в компании"
          />
          <div v-else-if="store.posts.length" class="portal-posts">
            <PostCard
              v-for="p in store.posts"
              :key="p.id"
              :post="p"
              @edit="openComposer"
              @delete="confirmDelete"
              @forward="openForward"
            />
          </div>
          <button
            v-if="store.nextCursor"
            class="portal-load-more"
            :disabled="store.loadingMore"
            @click="store.fetchMore()"
          >
            <ProgressSpinner v-if="store.loadingMore" style="width:16px;height:16px" />
            <template v-else>Показать ещё</template>
          </button>
        </section>
      </template>
      </div>
    </div>
    </div>

    <AppFab
      :visible="isMobile && fabVisible"
      icon="edit"
      aria-label="Написать пост"
      @click="openComposer(null)"
    />

    <PostComposer v-model="composerOpen" :post="editingPost" @saved="onSaved" />
    <ForwardPostDialog v-model="forwardOpen" :post="forwardingPost" @confirm="onForwardConfirm" />
    <TopicManageDialog v-model="topicsDialogOpen" />
    <PortalBackgroundDialog v-model="bgDialogOpen" />

    <AppDialog
      v-model="deleteConfirmOpen"
      tone="danger"
      icon="delete"
      size="sm"
      title="Удалить пост?"
      subtitle="Комментарии и вложения будут удалены безвозвратно."
      :actions="[{ kind: 'cancel', label: 'Отмена' }, { kind: 'confirm', label: 'Удалить', icon: 'delete' }]"
      @confirm="doDelete"
    />
  </div>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import ProgressSpinner from 'primevue/progressspinner'
import BrandLoader from '@/components/common/BrandLoader.vue'
import { useAuthStore } from '@/stores/auth.js'
import { usePortalStore } from '@/stores/portal.js'
import { usePermission } from '@/composables/usePermission.js'
import { useBreakpoint } from '@/composables/useBreakpoint.js'
import { useFabOnScroll } from '@/composables/useFabOnScroll.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import EmptyState from '@/components/common/EmptyState.vue'
import SearchField from '@/components/common/SearchField.vue'
import AppDialog from '@/components/common/AppDialog.vue'
import AppFab from '@/components/common/AppFab.vue'
import PortalHubTabs from '@/components/portal/PortalHubTabs.vue'
import PostCard from '@/components/portal/PostCard.vue'
import PostComposer from '@/components/portal/PostComposer.vue'
import ForwardPostDialog from '@/components/portal/ForwardPostDialog.vue'
import TopicManageDialog from '@/components/portal/TopicManageDialog.vue'
import PortalBackgroundDialog from '@/components/portal/PortalBackgroundDialog.vue'
import ChatBackgroundLayer from '@/components/messenger/ChatBackgroundLayer.vue'
import { isBlankRecipe } from '@/utils/chatBackgrounds.js'

const store = usePortalStore()
const { isAdmin } = usePermission()
const route = useRoute()
const { isMobile } = useBreakpoint()
// Мобильный FAB «Написать пост»: прячется/появляется по прокрутке ленты.
const { fabVisible } = useFabOnScroll()

// Обои ленты активны при заданном НЕпустом фоне (и на десктопе внутри панели,
// и на мобиле full-bleed — панели там нет, а фон показываем).
const feedBgOn = computed(() =>
  !!store.background && !isBlankRecipe(store.background))

// ── Поиск (debounce, серверный ?search=) ──
const searchInput = ref('')
let searchTimer = null
function onSearch() {
  clearTimeout(searchTimer)
  searchTimer = setTimeout(() => store.setSearch(searchInput.value), 300)
}
function clearSearch() {
  clearTimeout(searchTimer)
  searchInput.value = ''
  store.setSearch('')
}

// Инлайн-стиль перекрывает CSS-класс .active, поэтому активное состояние
// цветного чипа задаём здесь же: акцентная рамка (двойная через box-shadow,
// без сдвига макета) + акцентный текст + галочка в шаблоне.
function chipStyle(t, active = false) {
  if (!t.color) return {}
  if (!active) {
    return { background: `var(--tag-${t.color}-surface)`, borderColor: `var(--tag-${t.color}-border)`, color: 'var(--color-text)' }
  }
  return {
    background: `var(--tag-${t.color}-surface)`,
    borderColor: `var(--tag-${t.color}-accent)`,
    boxShadow: `inset 0 0 0 1px var(--tag-${t.color}-accent)`,
    color: `var(--tag-${t.color}-accent)`,
    fontWeight: 700,
  }
}

// Лента с серверной keyset-пагинацией: «Показать ещё» — store.fetchMore()
// по курсору, кнопка видна пока сервер отдаёт next_cursor.

// ── Композер (создание/редактирование) ──
const composerOpen = ref(false)
const editingPost = ref(null)
function openComposer(post) {
  editingPost.value = post || null
  composerOpen.value = true
}
function onSaved() { composerOpen.value = false }

// ── Пересылка ──
const forwardOpen = ref(false)
const forwardingPost = ref(null)
function openForward(post) {
  forwardingPost.value = post
  forwardOpen.value = true
}
async function onForwardConfirm({ userIds }) {
  const notif = useNotificationsStore()
  try {
    await store.forwardPost(forwardingPost.value.id, { userIds })
    notif.success('Пост переслан')
  } catch (e) {
    notif.error(e?.message || 'Не удалось переслать пост')
  } finally {
    forwardOpen.value = false
  }
}

// ── Удаление ──
const deleteConfirmOpen = ref(false)
const deletingPost = ref(null)
function confirmDelete(post) {
  deletingPost.value = post
  deleteConfirmOpen.value = true
}
async function doDelete() {
  try {
    await store.deletePost(deletingPost.value.id)
  } catch (e) {
    useNotificationsStore().error(e?.message || 'Не удалось удалить пост')
  } finally {
    deleteConfirmOpen.value = false
  }
}

const topicsDialogOpen = ref(false)
const bgDialogOpen = ref(false)

// ── Пост по прямой ссылке /portal/:id (в т.ч. клик по пересланной плашке
// в мессенджере). Живёт в сторе (реакции/комментарии/сокеты работают как в
// ленте); если пост уже виден в общей ленте — отдельной секцией не дублируем. ──
const highlightPost = computed(() => {
  const h = store.highlight
  if (!h) return null
  if (store.posts.some((p) => p.id === h.id) || store.pinnedPosts.some((p) => p.id === h.id)) return null
  return h
})
watch(() => route.params.id, (id) => store.loadHighlight(id))

async function loadAll() {
  await Promise.all([store.fetchTopics(), store.fetchPopularTags(), store.fetchPosts(), store.loadAuthors()])
  store.loadHighlight(route.params.id)
}
onMounted(() => {
  // Открытый портал гасит бейдж непрочитанных: пока экран виден, свежие
  // post:new тоже сразу подтверждаются просмотренными (см. applyPostSocket).
  store.viewingFeed = true
  store.markSeen()
  loadAll()
})
onBeforeUnmount(() => { store.viewingFeed = false })

// Смена активной компании при открытом портале — полная перезагрузка данных
// (стор к этому моменту сброшен глобальным watch в App.vue).
watch(() => useAuthStore().companyId, (id, prev) => {
  if (id != null && prev != null && id !== prev) loadAll()
})
</script>

<style scoped>
/* Тулбар без подложки — прозрачная «плавающая» шапка как в «Задачах»
   (контент скроллится в .admin-body ниже, не под шапкой). */
.admin-sticky { background: transparent; -webkit-backdrop-filter: none; backdrop-filter: none; }
.admin-sticky::after { display: none; }

.portal-hub-tabs { flex-shrink: 0; }
/* Поиск не сжимается меньше комфортной ширины — при нехватке места первыми
   переносятся кнопки/вкладки, а не поле ввода. */
.portal-toolbar :deep(.search-field) { min-width: 240px; }

@keyframes portal-fade {
  from { opacity: 0; transform: translateY(4px); }
  to { opacity: 1; transform: translateY(0); }
}
@media (prefers-reduced-motion: reduce) { .portal-feed { animation: none; } }

.portal-toolbar {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

.portal-topics {
  display: flex;
  gap: 8px;
  overflow-x: auto;
  padding-bottom: 2px;
  scrollbar-width: none;
}
.portal-topics::-webkit-scrollbar { display: none; }

.portal-topic-chip {
  flex-shrink: 0;
  padding: 7px 16px;
  border-radius: var(--radius-full);
  border: 1px solid var(--acrylic-border);
  background: var(--acrylic-card-bg);
  color: var(--color-text-dim);
  font: inherit;
  font-size: 13.5px;
  font-weight: 600;
  cursor: pointer;
}
.portal-topic-chip.active {
  background: var(--color-primary-container);
  border-color: var(--color-primary);
  color: var(--color-on-primary-container);
  font-weight: 700;
}
.portal-topic-chip { display: inline-flex; align-items: center; gap: 4px; }
.portal-chip-check { font-size: 15px; }

/* Разделитель разделов и хештегов в общей строке фильтров. */
.portal-filter-sep {
  flex-shrink: 0;
  align-self: center;
  width: 1px;
  height: 20px;
  margin: 0 2px;
  background: var(--color-outline-dim);
}

/* Популярные хештеги (тренды) — чипы в той же строке, что и разделы. */
.portal-tag-chip {
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 7px 14px;
  border-radius: var(--radius-full);
  border: 1px solid var(--acrylic-border);
  background: var(--acrylic-card-bg);
  color: var(--color-primary);
  font: inherit;
  font-size: 13.5px;
  font-weight: 600;
  cursor: pointer;
}
.portal-tag-chip:hover { border-color: color-mix(in oklch, var(--color-primary) 30%, var(--acrylic-border)); }
.portal-tag-chip.active {
  background: var(--color-primary-container);
  border-color: var(--color-primary);
  color: var(--color-on-primary-container);
}
.portal-tag-count {
  font-size: 11px;
  font-weight: 700;
  color: var(--color-text-dim);
}
.portal-tag-chip.active .portal-tag-count { color: var(--color-on-primary-container); }

/* Баннер активного фильтра по тегу над лентой. */
.portal-tag-banner {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 14px;
  border-radius: var(--radius-md);
  border: 1px solid var(--acrylic-border);
  background: var(--acrylic-card-bg);
  font-size: 13.5px;
  color: var(--color-text);
}
.portal-tag-banner > .material-symbols-outlined { color: var(--color-primary); font-size: 20px; }
.portal-tag-banner-text { flex: 1; min-width: 0; }
.portal-tag-banner strong { color: var(--color-primary); }
.portal-tag-clear {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 5px 12px;
  border: none;
  border-radius: var(--radius-full);
  background: var(--glass-bg);
  box-shadow: var(--glass-edge);
  color: var(--color-text);
  font: inherit;
  font-size: 12.5px;
  font-weight: 600;
  cursor: pointer;
}
.portal-tag-clear:hover { background: var(--glass-hover-bg); }
.portal-tag-clear .material-symbols-outlined { font-size: 16px; }

/* Контент ленты — узкая читабельная колонка внутри общего каркаса. */
.portal-feed {
  display: flex;
  flex-direction: column;
  gap: 20px;
  max-width: 760px;
  margin: 0 auto;
  width: 100%;
  animation: portal-fade 0.2s ease;
}

.portal-status {
  display: flex;
  justify-content: center;
  padding: 40px 0;
}

.portal-section { display: flex; flex-direction: column; gap: 10px; }

.portal-section-title {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  font-weight: 700;
  color: var(--color-tertiary);
  text-transform: uppercase;
  letter-spacing: 0.03em;
}
.portal-section-title .material-symbols-outlined { font-size: 18px; font-variation-settings: 'FILL' 1; }

.portal-posts {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.portal-load-more {
  align-self: center;
  margin-top: 4px;
  padding: 9px 20px;
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-full);
  background: var(--acrylic-card-bg);
  background: var(--glass-bg);
  box-shadow: var(--glass-edge);
  color: var(--color-text);
  font: inherit;
  font-size: 13.5px;
  font-weight: 600;
  cursor: pointer;
  transition: border-color 0.15s;
}
.portal-load-more:hover { border-color: color-mix(in oklch, var(--color-primary) 30%, var(--acrylic-border)); }

@media (max-width: 640px) {
  .portal-btn-label { display: none; }
  /* Без подписи кнопки тулбара сжимаются в квадратные иконки — язык .btn-icon
     из «Задач», а не растянутая пилюля. */
  .portal-toolbar .btn-glass,
  .portal-toolbar .btn-grad { padding: 0; width: 42px; height: 42px; justify-content: center; }
  /* Поиск переносится на всю ширину под вкладками/кнопками. */
  .portal-toolbar :deep(.search-field) { min-width: 0; flex: 1 1 100%; order: 1; }
}

@media (max-width: 768px) {
  /* Создание поста на мобильном — плавающий FAB; вкладки хаба растягиваются
     на освободившуюся ширину (full-width включает сам PortalHubTabs). */
  .portal-toolbar .btn-grad { display: none; }
  .portal-hub-tabs { flex: 1; min-width: 0; }
}
</style>
