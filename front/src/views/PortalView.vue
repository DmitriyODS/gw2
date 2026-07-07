<template>
  <!-- Общий каркас admin-page — та же геометрия, что у вкладки «Сотрудники»:
       переключение вкладок хаба не должно сдвигать интерфейс. -->
  <div class="admin-page portal">
    <header class="admin-sticky">
      <PortalHubTabs class="portal-hub-tabs" />

      <div class="portal-toolbar">
        <h1 class="portal-title">
          <span class="material-symbols-outlined">campaign</span>
          Портал
        </h1>

        <div class="portal-search">
          <span class="material-symbols-outlined">search</span>
          <input v-model="searchInput" type="text" placeholder="Поиск по постам…" @input="onSearch" />
          <button v-if="searchInput" class="portal-search-clear" title="Очистить" aria-label="Очистить поиск" @click="clearSearch">
            <span class="material-symbols-outlined">close</span>
          </button>
        </div>

        <div class="portal-actions">
          <button v-if="isAdmin()" class="portal-icon-btn" title="Управление разделами" aria-label="Управление разделами" @click="topicsDialogOpen = true">
            <span class="material-symbols-outlined">tune</span>
          </button>
          <button class="portal-btn-primary" @click="openComposer(null)">
            <span class="material-symbols-outlined">edit</span>
            <span class="portal-btn-label">Написать пост</span>
          </button>
        </div>
      </div>

      <div class="portal-topics">
        <button
          class="portal-topic-chip"
          :class="{ active: store.filters.topicId == null }"
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
      </div>
    </header>

    <div class="admin-body">
      <div class="portal-feed">
      <div v-if="store.loadingPosts" class="portal-status">
        <ProgressSpinner style="width:32px;height:32px" />
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

    <PostComposer v-model="composerOpen" :post="editingPost" @saved="onSaved" />
    <ForwardPostDialog v-model="forwardOpen" :post="forwardingPost" @confirm="onForwardConfirm" />
    <TopicManageDialog v-model="topicsDialogOpen" />

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
import { useAuthStore } from '@/stores/auth.js'
import { usePortalStore } from '@/stores/portal.js'
import { usePermission } from '@/composables/usePermission.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import EmptyState from '@/components/common/EmptyState.vue'
import AppDialog from '@/components/common/AppDialog.vue'
import PortalHubTabs from '@/components/portal/PortalHubTabs.vue'
import PostCard from '@/components/portal/PostCard.vue'
import PostComposer from '@/components/portal/PostComposer.vue'
import ForwardPostDialog from '@/components/portal/ForwardPostDialog.vue'
import TopicManageDialog from '@/components/portal/TopicManageDialog.vue'

const store = usePortalStore()
const { isAdmin } = usePermission()
const route = useRoute()

// ── Поиск (debounce, серверный ?search=) ──
const searchInput = ref('')
let searchTimer = null
function onSearch() {
  clearTimeout(searchTimer)
  searchTimer = setTimeout(() => store.setSearch(searchInput.value), 300)
}
function clearSearch() {
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
  await Promise.all([store.fetchTopics(), store.fetchPosts(), store.loadAuthors()])
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
.portal-hub-tabs { align-self: flex-start; }

@keyframes portal-fade {
  from { opacity: 0; transform: translateY(4px); }
  to { opacity: 1; transform: translateY(0); }
}
@media (prefers-reduced-motion: reduce) { .portal-feed { animation: none; } }

.portal-toolbar {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}

.portal-title {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 0;
  font-size: 20px;
  font-weight: 700;
  color: var(--color-text);
}
.portal-title .material-symbols-outlined { font-size: 24px; color: var(--color-primary); }

.portal-search {
  position: relative;
  flex: 1;
  min-width: 180px;
  display: flex;
  align-items: center;
}
.portal-search .material-symbols-outlined {
  position: absolute;
  left: 12px;
  font-size: 18px;
  color: var(--color-text-dim);
  pointer-events: none;
}
.portal-search input {
  width: 100%;
  padding: 9px 12px 9px 38px;
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-full);
  background: var(--color-surface);
  color: var(--color-text);
  font: inherit;
  font-size: 14px;
  outline: none;
  box-sizing: border-box;
}
.portal-search input:focus { border-color: var(--color-primary); }

.portal-search-clear {
  position: absolute;
  right: 8px;
  border: none;
  background: transparent;
  color: var(--color-text-dim);
  cursor: pointer;
  display: grid;
  place-items: center;
}

.portal-actions { display: flex; align-items: center; gap: 8px; }

.portal-icon-btn {
  width: 40px;
  height: 40px;
  border: 1px solid var(--color-outline-dim);
  border-radius: 50%;
  background: var(--color-surface);
  color: var(--color-text);
  cursor: pointer;
  display: grid;
  place-items: center;
}
.portal-icon-btn:hover { background: var(--color-surface-high); }

.portal-btn-primary {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  height: 40px;
  padding: 0 18px;
  border: none;
  border-radius: var(--radius-full);
  background: var(--color-primary);
  color: var(--color-on-primary);
  font: inherit;
  font-size: 14px;
  font-weight: 700;
  cursor: pointer;
}
.portal-btn-primary:hover { box-shadow: var(--shadow-sm); }

.portal-topics {
  display: flex;
  gap: 8px;
  overflow-x: auto;
  padding-bottom: 2px;
}

.portal-topic-chip {
  flex-shrink: 0;
  padding: 7px 16px;
  border-radius: var(--radius-full);
  border: 1px solid var(--color-outline-dim);
  background: var(--color-surface);
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
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-full);
  background: var(--color-surface);
  color: var(--color-text);
  font: inherit;
  font-size: 13.5px;
  font-weight: 600;
  cursor: pointer;
}
.portal-load-more:hover { background: var(--color-surface-high); }

@media (max-width: 640px) {
  .portal-btn-label { display: none; }
  .portal-btn-primary { padding: 0 12px; }
}
</style>
