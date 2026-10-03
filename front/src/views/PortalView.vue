<template>
  <AppPage
    class="portal"
    title="Портал"
    :commands="commands"
    flush
    :scroll="false"
    @command="onCommand"
  >
    <!-- Свои обои ленты: слой уходит под содержимое панели раздела. -->
    <template v-if="feedBgOn" #background>
      <ChatBackgroundLayer :recipe="store.background" />
    </template>

    <template #subhead>
      <SearchField
        v-model="searchInput"
        placeholder="Поиск по постам…"
        hotkey
        :collapsible="false"
        @update:model-value="onSearch"
        @clear="clearSearch"
      />
    </template>

    <!-- Раскладка корпоративной ленты (по образцу Viva Engage / LinkedIn):
         слева разделы, по центру лента читаемой ширины, справа закреплённое и
         популярные теги. Ширину меряем у САМОЙ раскладки — портал живёт окном;
         в узком окне боковые колонки сворачиваются в ряд чипов над лентой. -->
    <div ref="layoutEl" class="portal-layout" :class="{ wide }">
      <nav v-if="wide" class="portal-rail" aria-label="Разделы портала">
        <h3 class="portal-rail-title">Разделы</h3>
        <AppRow
          plain
          dense
          clickable
          title="Все публикации"
          :selected="store.filters.topicId == null && !store.filters.tag"
          @click="store.setTopic(null)"
        >
          <template #lead><span class="material-symbols-outlined portal-rail-icon">dynamic_feed</span></template>
        </AppRow>
        <AppRow
          v-for="t in store.topics"
          :key="t.id"
          plain
          dense
          clickable
          :title="t.name"
          :selected="store.filters.topicId === t.id"
          @click="store.setTopic(t.id)"
        >
          <template #lead><TopicIcon :icon="t.icon" :color="t.color" /></template>
        </AppRow>
        <AppButton
          v-if="isAdmin()"
          class="portal-rail-manage"
          variant="text"
          size="sm"
          icon="tune"
          label="Управление разделами"
          @click="topicsDialogOpen = true"
        />
      </nav>

      <!-- Прокрутка своя, а не тела страницы: по ней виртуальная лента считает,
           какие посты у экрана. -->
      <div ref="scrollEl" class="portal-scroll" @scroll="postsList.onScroll">
        <div class="portal-feed">
          <!-- Узкое окно: разделы и популярные теги — одним рядом чипов. -->
          <div v-if="!wide" class="portal-chips">
            <AppChip
              interactive
              :selected="store.filters.topicId == null && !store.filters.tag"
              label="Все"
              @click="store.setTopic(null)"
            />
            <AppChip
              v-for="t in store.topics"
              :key="t.id"
              interactive
              :selected="store.filters.topicId === t.id"
              :label="t.name"
              @click="store.setTopic(t.id)"
            />
            <AppChip
              v-for="t in popularTags"
              :key="`#${t.tag}`"
              interactive
              tone="primary"
              :selected="store.filters.tag === t.tag"
              :label="`#${t.tag}`"
              :count="t.count"
              @click="toggleTag(t.tag)"
            />
          </div>

          <AppInfoBar v-if="store.filters.tag" tone="info" icon="tag">
            Посты с тегом <strong>#{{ store.filters.tag }}</strong>
            <template #actions>
              <AppButton size="sm" icon="close" label="Сбросить" @click="store.setTag(null)" />
            </template>
          </AppInfoBar>

          <!-- Приглашение написать — первым в ленте, как в соцсетях. -->
          <AppCard clickable class="portal-prompt" :gap="0" @click="openComposer(null)">
            <img v-if="myAvatar" class="portal-prompt-avatar" :src="myAvatar" alt="" />
            <span class="portal-prompt-field">Что нового{{ myName ? `, ${myName}` : '' }}?</span>
            <span class="material-symbols-outlined portal-prompt-icon">image</span>
            <span class="material-symbols-outlined portal-prompt-icon">tag</span>
          </AppCard>

          <BrandLoader v-if="store.loadingPosts" block :size="64" :min-height="200" />

          <template v-else>
            <section v-if="highlightPost" class="portal-section">
              <h4 class="portal-section-title">
                <span class="material-symbols-outlined">open_in_new</span>
                Пост по ссылке
              </h4>
              <PostCard :post="highlightPost" @edit="openComposer" @delete="confirmDelete" @forward="openForward" />
            </section>

            <!-- В узком окне закреплённое — прямо в ленте; в широком — справа. -->
            <section v-if="!wide && store.pinnedPosts.length" class="portal-section">
              <h4 class="portal-section-title">
                <span class="material-symbols-outlined">keep</span>
                Закреплено
              </h4>
              <PostCard
                v-for="p in store.pinnedPosts"
                :key="p.id"
                :post="p"
                @edit="openComposer"
                @delete="confirmDelete"
                @forward="openForward"
              />
            </section>

            <EmptyState
              v-if="!store.posts.length && !store.pinnedPosts.length"
              icon="campaign"
              tone="soft"
              title="Пока пусто"
              subtitle="Станьте первым, кто поделится новостью"
            />
            <!-- Лента виртуальная (useVirtualList): «Показать ещё» копит посты
                 без предела, а в DOM остаются только те, что у экрана. -->
            <div
              v-else-if="store.posts.length"
              ref="postsEl"
              class="portal-posts-virtual"
              :style="{ paddingTop: `${postsTop}px`, paddingBottom: `${postsBottom}px` }"
            >
              <div v-for="p in visiblePosts" :key="p.id" :ref="postsList.measureRef(p.id)" class="portal-vrow">
                <PostCard :post="p" @edit="openComposer" @delete="confirmDelete" @forward="openForward" />
              </div>
            </div>
            <AppButton
              v-if="store.nextCursor"
              class="portal-load-more"
              label="Показать ещё"
              :loading="store.loadingMore"
              @click="store.fetchMore()"
            />
          </template>
        </div>
      </div>

      <aside v-if="wide" class="portal-side">
        <AppCard v-if="store.pinnedPosts.length" title="Закреплено" :gap="4">
          <AppRow
            v-for="p in store.pinnedPosts"
            :key="p.id"
            plain
            dense
            clickable
            :title="postTitle(p)"
            :hint="authorName(p)"
            @click="openPinned(p)"
          >
            <template #lead><span class="material-symbols-outlined portal-rail-icon">keep</span></template>
          </AppRow>
        </AppCard>

        <AppCard v-if="popularTags.length" title="Популярное" :gap="10">
          <div class="portal-side-tags">
            <AppChip
              v-for="t in popularTags"
              :key="t.tag"
              interactive
              size="sm"
              :selected="store.filters.tag === t.tag"
              :label="`#${t.tag}`"
              :count="t.count"
              @click="toggleTag(t.tag)"
            />
          </div>
        </AppCard>
      </aside>
    </div>

    <PostComposer v-model="composerOpen" :post="editingPost" @saved="onSaved" />
    <ForwardPostDialog v-model="forwardOpen" :post="forwardingPost" @confirm="onForwardConfirm" />
    <TopicManageDialog v-model="topicsDialogOpen" />
    <PortalBackgroundDialog v-model="bgDialogOpen" />

    <AppDialog
      v-model="deleteConfirmOpen"
      tone="danger"
      size="sm"
      title="Удалить пост?"
      subtitle="Комментарии и вложения будут удалены безвозвратно."
      :actions="[{ kind: 'cancel', label: 'Отмена' }, { kind: 'confirm', label: 'Удалить', icon: 'delete' }]"
      @confirm="doDelete"
    />
  </AppPage>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import BrandLoader from '@/components/common/BrandLoader.vue'
import { useAuthStore } from '@/stores/auth.js'
import { usePortalStore } from '@/stores/portal.js'
import { usePermission } from '@/composables/usePermission.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import EmptyState from '@/components/common/EmptyState.vue'
import AppButton from '@/components/ui/AppButton.vue'
import AppCard from '@/components/ui/AppCard.vue'
import AppChip from '@/components/ui/AppChip.vue'
import AppRow from '@/components/ui/AppRow.vue'
import TopicIcon from '@/components/portal/TopicIcon.vue'
import { avatarUrl } from '@/utils/avatar.js'
import AppInfoBar from '@/components/ui/AppInfoBar.vue'
import AppPage from '@/components/ui/AppPage.vue'
import SearchField from '@/components/common/SearchField.vue'
import AppDialog from '@/components/ui/AppDialog.vue'
import PostCard from '@/components/portal/PostCard.vue'
import PostComposer from '@/components/portal/PostComposer.vue'
import ForwardPostDialog from '@/components/portal/ForwardPostDialog.vue'
import TopicManageDialog from '@/components/portal/TopicManageDialog.vue'
import PortalBackgroundDialog from '@/components/portal/PortalBackgroundDialog.vue'
import ChatBackgroundLayer from '@/components/common/ChatBackgroundLayer.vue'
import { isBlankRecipe } from '@/utils/chatBackgrounds.js'
import { useVirtualList } from '@/composables/useVirtualList.js'

const store = usePortalStore()
const { isAdmin } = usePermission()
const route = useRoute()
const router = useRouter()
const auth = useAuthStore()

/* ── Раскладка: три колонки, пока хватает ширины САМОЙ ленты-окна ── */
const WIDE_MIN = 1060
const layoutEl = ref(null)
const wide = ref(false)
let layoutRo = null
onMounted(() => {
  if (typeof ResizeObserver !== 'function') return
  layoutRo = new ResizeObserver(([e]) => { wide.value = e.contentRect.width >= WIDE_MIN })
  layoutRo.observe(layoutEl.value)
})
onBeforeUnmount(() => layoutRo?.disconnect())

// Приглашение «Что нового, Имя?»: имя — второе слово ФИО («Фамилия Имя …»).
const myAvatar = computed(() => (auth.user ? avatarUrl(auth.user) : ''))
const myName = computed(() => (auth.user?.fio || '').trim().split(/\s+/)[1] || '')

// Тренды прячутся, пока идёт поиск: они про всю ленту, а не про выдачу.
const popularTags = computed(() => (store.filters.search ? [] : store.popularTags))
const toggleTag = (tag) => store.setTag(store.filters.tag === tag ? null : tag)

/* Строка закреплённого справа: заголовок поста или первая строка текста. */
function postTitle(p) {
  const raw = (p.title || p.body || '').replace(/[#*_>`~[\]]/g, '').trim()
  const line = raw.split('\n')[0] || 'Публикация'
  return line.length > 70 ? `${line.slice(0, 69)}…` : line
}
const authorName = (p) => store.resolveAuthor(p.author_id).fio

// Закреплённое справа открывается в ленте тем же путём, что пост по ссылке.
function openPinned(p) {
  router.push(`/portal/${p.id}`)
  scrollEl.value?.scrollTo({ top: 0, behavior: 'smooth' })
}
// Обои ленты активны при заданном НЕпустом фоне: слой рисуется внутри панели
// раздела и клипается её скруглением.
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

// Лента с серверной keyset-пагинацией: «Показать ещё» — store.fetchMore()
// по курсору, кнопка видна пока сервер отдаёт next_cursor.
const scrollEl = ref(null)
const postsEl = ref(null)
const postsList = useVirtualList({
  container: scrollEl,
  list: postsEl,
  keys: computed(() => store.posts.map((p) => p.id)),
  estimate: () => 360,
})
const visiblePosts = computed(() => store.posts.slice(postsList.start.value, postsList.end.value))
const postsTop = computed(() => postsList.offsetOf(postsList.start.value))
const postsBottom = computed(() => postsList.total.value - postsList.offsetOf(postsList.end.value))

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

const commands = computed(() => [
  { key: 'post', label: 'Написать пост', icon: 'edit', variant: 'filled', primary: true, fab: true },
  { key: 'background', label: 'Оформление ленты', icon: 'palette' },
  // В широком окне управление разделами стоит под их списком.
  ...(isAdmin() && !wide.value ? [{ key: 'topics', label: 'Управление разделами', icon: 'tune' }] : []),
])

function onCommand(key) {
  if (key === 'post') openComposer(null)
  else if (key === 'background') bgDialogOpen.value = true
  else if (key === 'topics') topicsDialogOpen.value = true
}

// ── Пост по прямой ссылке /portal/:id (в т.ч. клик по пересланной плашке
// в мессенджере). Живёт в сторе (реакции/комментарии/сокеты работают как в
// ленте); если пост уже виден в общей ленте — отдельной секцией не дублируем. ──
const highlightPost = computed(() => {
  const h = store.highlight
  if (!h) return null
  if (store.posts.some((p) => p.id === h.id)) return null
  // В узком окне закреплённое и так стоит в ленте; в широком оно справа
  // строкой — и открывается сюда.
  if (!wide.value && store.pinnedPosts.some((p) => p.id === h.id)) return null
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
watch(() => auth.companyId, (id, prev) => {
  if (id != null && prev != null && id !== prev) loadAll()
})
</script>

<style scoped>
.portal-layout {
  flex: 1;
  min-height: 0;
  display: grid;
  grid-template-columns: minmax(0, 1fr);
}

.portal-layout.wide {
  grid-template-columns: 220px minmax(0, 1fr) 280px;
  gap: 8px;
}

/* ── Левая колонка: разделы ── */
.portal-rail,
.portal-side {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-height: 0;
  overflow-y: auto;
  padding: 4px 4px 18px 14px;
}

.portal-side { gap: 14px; padding: 4px 14px 18px 4px; }

.portal-rail-title {
  margin: 4px 10px 8px;
  font-size: 12.5px;
  font-weight: 700;
  letter-spacing: 0.04em;
  text-transform: uppercase;
  color: var(--color-text-dim);
}

.portal-rail-icon { font-size: 20px; color: var(--color-text-dim); }
.portal-rail-manage { align-self: flex-start; margin-top: 8px; }
.portal-rail :deep(.ti) { width: 28px; height: 28px; }

.portal-side-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

/* ── Центр: лента читаемой ширины ── */
/* Своя прокрутка ленты (AppPage :scroll="false" — содержимое скроллится само).
   Замер строк держит позицию сам, браузерная поправка сложилась бы дважды. */
.portal-scroll {
  min-height: 0;
  overflow-y: auto;
  overflow-anchor: none;
}

.portal-feed {
  display: flex;
  flex-direction: column;
  gap: 16px;
  max-width: 680px;
  margin: 0 auto;
  padding: 4px 16px 18px;
  animation: portal-fade 0.2s ease;
}

@keyframes portal-fade {
  from { opacity: 0; transform: translateY(4px); }
  to { opacity: 1; transform: translateY(0); }
}
@media (prefers-reduced-motion: reduce) { .portal-feed { animation: none; } }

.portal-chips {
  display: flex;
  gap: 8px;
  overflow-x: auto;
  padding: 2px 0 4px;
  scrollbar-width: none;
}
.portal-chips::-webkit-scrollbar { display: none; }
.portal-chips > * { flex-shrink: 0; }

/* Приглашение написать пост — карточка с «полем», как в соцсетях. */
.portal-prompt {
  flex-direction: row;
  align-items: center;
  gap: 12px;
  padding: 12px 14px;
}

.portal-prompt-avatar {
  width: 40px;
  height: 40px;
  flex-shrink: 0;
  border-radius: 50%;
  object-fit: cover;
}

.portal-prompt-field {
  flex: 1;
  min-width: 0;
  padding: 10px 16px;
  border: 1px solid var(--sk-edge);
  border-radius: var(--radius-full);
  background: var(--sk-well-bg);
  box-shadow: var(--sk-well-shadow);
  color: var(--color-text-dim);
  font-size: 14px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.portal-prompt-icon { font-size: 22px; color: var(--color-primary); }

.portal-section { display: flex; flex-direction: column; gap: 10px; }

.portal-section-title {
  display: flex;
  align-items: center;
  gap: 6px;
  margin: 0;
  font-size: 12.5px;
  font-weight: 700;
  letter-spacing: 0.04em;
  text-transform: uppercase;
  color: var(--color-text-dim);
}
.portal-section-title .material-symbols-outlined { font-size: 18px; font-variation-settings: 'FILL' 1; }

/* Промежуток между постами — отступом строки, а не gap: он должен попасть в
   замер высоты, иначе распорки ленты расходятся с реальной раскладкой. */
.portal-vrow {
  display: flow-root;
  padding-bottom: 14px;
}

.portal-load-more { align-self: center; }

@media (max-width: 768px) {
  .portal-feed { padding: 4px 12px 18px; }
  /* На телефоне главное действие — плавающая кнопка, приглашение не нужно. */
  .portal-prompt { display: none; }
}
</style>
