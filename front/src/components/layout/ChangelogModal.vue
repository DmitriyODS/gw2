<template>
  <AppDialog
    model-value
    tone="primary"
    icon="new_releases"
    size="xl"
    title="Что нового"
    :subtitle="currentVersion ? `Текущая версия: ${currentVersion}` : ''"
    @update:model-value="(v) => !v && $emit('close')"
  >
    <div v-if="loading" class="cl-loading">
      <span class="material-symbols-outlined spinning">progress_activity</span>
    </div>

    <div v-else-if="error" class="cl-error">
      <span class="material-symbols-outlined">error_outline</span>
      Не удалось загрузить изменения
    </div>

    <!-- Паттерн release notes: сначала — «главное» свежего релиза крупными
         карточками, полные списки — ниже; прошлые версии свёрнуты в таймлайн
         (клик раскрывает), чтобы окно не превращалось в бесконечную простыню. -->
    <template v-else>
      <section v-if="latest" class="cl-hero">
        <div class="cl-version-top">
          <span class="cl-badge">v{{ latest.version }}</span>
          <span class="cl-badge-new">Свежее</span>
          <span class="cl-date">{{ formatDate(latest.date) }}</span>
        </div>
        <h2 class="cl-title">{{ latest.title }}</h2>
        <p v-if="latest.description" class="cl-desc">{{ latest.description }}</p>

        <div v-if="highlights.length" class="cl-highlights">
          <div v-for="(text, i) in highlights" :key="i" class="cl-highlight">
            <span class="material-symbols-outlined cl-highlight-ico">{{ HIGHLIGHT_ICONS[i % HIGHLIGHT_ICONS.length] }}</span>
            <p class="cl-highlight-text">{{ text }}</p>
          </div>
        </div>

        <div class="cl-groups">
          <div v-for="group in latestGroups" :key="group.type" class="cl-group">
            <div class="cl-chip" :class="`cl-chip--${group.type}`">
              <span class="material-symbols-outlined cl-chip-icon">{{ groupMeta[group.type]?.icon }}</span>
              <span class="cl-chip-label">{{ groupMeta[group.type]?.label }}</span>
              <span class="cl-chip-count">{{ group.items.length }}</span>
            </div>
            <ul class="cl-items">
              <li
                v-for="(text, i) in group.items"
                :key="i"
                class="cl-item"
                :class="`cl-item--${group.type}`"
              >{{ text }}</li>
            </ul>
          </div>
        </div>
      </section>

      <section v-if="history.length" class="cl-history">
        <h3 class="cl-history-title">
          <span class="material-symbols-outlined">history</span>
          Предыдущие версии
        </h3>

        <article
          v-for="ver in history"
          :key="ver.version"
          class="cl-past"
          :class="{ open: opened.has(ver.version) }"
        >
          <button class="cl-past-head glass-hover" @click="toggle(ver.version)">
            <span class="cl-past-info">
              <span class="cl-past-line">
                <span class="cl-badge ghost">v{{ ver.version }}</span>
                <span class="cl-date">{{ formatDate(ver.date) }}</span>
              </span>
              <span class="cl-past-title">{{ ver.title }}</span>
              <span class="cl-past-counts">
                <span
                  v-for="g in groupsOf(ver)"
                  :key="g.type"
                  class="cl-count"
                  :class="`cl-count--${g.type}`"
                >
                  <span class="material-symbols-outlined">{{ groupMeta[g.type]?.icon }}</span>
                  {{ g.items.length }}
                </span>
              </span>
            </span>
            <span class="material-symbols-outlined cl-past-chev">
              {{ opened.has(ver.version) ? 'expand_less' : 'expand_more' }}
            </span>
          </button>

          <div v-if="opened.has(ver.version)" class="cl-past-body">
            <p v-if="ver.description" class="cl-desc">{{ ver.description }}</p>
            <div class="cl-groups">
              <div v-for="group in groupsOf(ver)" :key="group.type" class="cl-group">
                <div class="cl-chip" :class="`cl-chip--${group.type}`">
                  <span class="material-symbols-outlined cl-chip-icon">{{ groupMeta[group.type]?.icon }}</span>
                  <span class="cl-chip-label">{{ groupMeta[group.type]?.label }}</span>
                  <span class="cl-chip-count">{{ group.items.length }}</span>
                </div>
                <ul class="cl-items">
                  <li
                    v-for="(text, i) in group.items"
                    :key="i"
                    class="cl-item"
                    :class="`cl-item--${group.type}`"
                  >{{ text }}</li>
                </ul>
              </div>
            </div>
          </div>
        </article>
      </section>
    </template>
  </AppDialog>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import AppDialog from '@/components/common/AppDialog.vue'
import { changelogApi } from '@/api/changelog.js'

defineEmits(['close'])

const loading  = ref(true)
const error    = ref(false)
const versions = ref([])
const opened   = ref(new Set())

const currentVersion = computed(() => versions.value[0]?.version ?? null)
const latest  = computed(() => versions.value[0] ?? null)
const history = computed(() => versions.value.slice(1))

const HIGHLIGHTS_MAX = 3
const HIGHLIGHT_ICONS = ['rocket_launch', 'auto_awesome', 'celebration']

// Главные пункты релиза — первые «Добавили»; в общем списке не дублируются.
const highlights = computed(() => (latest.value?.added ?? []).slice(0, HIGHLIGHTS_MAX))

const latestGroups = computed(() => {
  if (!latest.value) return []
  return groupsOf(latest.value)
    .map(g => g.type === 'added' ? { ...g, items: g.items.slice(HIGHLIGHTS_MAX) } : g)
    .filter(g => g.items.length)
})

const groupMeta = {
  added:    { icon: 'add_circle', label: 'Добавили'  },
  improved: { icon: 'upgrade',    label: 'Улучшили'  },
  fixed:    { icon: 'bug_report', label: 'Исправили' },
}

const GROUP_ORDER = ['added', 'improved', 'fixed']

function groupsOf(ver) {
  return GROUP_ORDER
    .filter(t => Array.isArray(ver[t]) && ver[t].length)
    .map(t => ({ type: t, items: ver[t] }))
}

function toggle(version) {
  const next = new Set(opened.value)
  if (next.has(version)) next.delete(version)
  else next.add(version)
  opened.value = next
}

function formatDate(str) {
  if (!str) return ''
  return new Date(str).toLocaleDateString('ru-RU', { day: 'numeric', month: 'long', year: 'numeric' })
}

onMounted(async () => {
  try {
    const data = await changelogApi.get()
    versions.value = data.versions ?? []
  } catch {
    error.value = true
  } finally {
    loading.value = false
  }
})
</script>

<style scoped>
.cl-version-top {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 10px;
}

.cl-badge {
  background: var(--color-primary);
  background: var(--grad-primary);
  color: var(--color-on-primary);
  font-size: 11px;
  font-weight: 800;
  padding: 3px 12px;
  border-radius: var(--radius-full);
  letter-spacing: 0.5px;
  text-transform: uppercase;
}

.cl-badge.ghost {
  background: var(--acrylic-card-bg);
  background: var(--glass-bg);
  box-shadow: var(--glass-edge);
  border: 1px solid var(--acrylic-border);
  color: var(--color-text);
}

.cl-badge-new {
  background: var(--grad-primary, var(--color-primary));
  color: var(--color-on-primary);
  font-size: 10.5px;
  font-weight: 800;
  padding: 3px 10px;
  border-radius: var(--radius-full);
  letter-spacing: 0.6px;
  text-transform: uppercase;
}

.cl-date {
  font-size: 13px;
  color: var(--color-text-dim);
}

.cl-title {
  font-size: 24px;
  font-weight: 800;
  color: var(--color-text);
  line-height: 1.25;
  letter-spacing: -0.3px;
  margin: 0 0 12px;
}

.cl-desc {
  font-size: 14px;
  color: var(--color-text-dim);
  line-height: 1.7;
  margin: 0 0 4px;
}

/* «Главное в релизе» — крупные стеклянные карточки. */
.cl-highlights {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(210px, 1fr));
  gap: 10px;
  margin-top: 16px;
}

.cl-highlight {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 14px;
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-lg);
  background: var(--acrylic-card-bg);
  background: var(--glass-bg);
  box-shadow: var(--glass-edge);
}

.cl-highlight-ico {
  font-size: 26px;
  color: var(--color-primary);
}

.cl-highlight-text {
  margin: 0;
  font-size: 13.5px;
  line-height: 1.55;
  color: var(--color-text);
}

.cl-groups {
  display: flex;
  flex-direction: column;
  gap: 18px;
  margin-top: 18px;
}

.cl-group {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.cl-chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 5px 12px 5px 8px;
  border-radius: var(--radius-full);
  font-size: 12px;
  font-weight: 700;
  align-self: flex-start;
}
.cl-chip-icon { font-size: 16px; }
.cl-chip-label { letter-spacing: 0.3px; }
.cl-chip-count {
  font-size: 11px;
  font-weight: 800;
  background: color-mix(in oklch, currentColor 20%, transparent);
  min-width: 20px;
  height: 20px;
  padding: 0 5px;
  border-radius: var(--radius-full);
  display: flex;
  align-items: center;
  justify-content: center;
  margin-left: 2px;
}

.cl-chip--added    { background: var(--color-success-container);  color: var(--color-on-success-container);  }
.cl-chip--improved { background: var(--color-tertiary-container); color: var(--color-on-tertiary-container); }
.cl-chip--fixed    { background: var(--color-warning-container);  color: var(--color-on-warning-container);  }

/* Строки изменений — стеклянные карточки; цветная кромка слева хранит
   семантику группы (добавили/улучшили/исправили). */
.cl-items {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.cl-item {
  font-size: 14px;
  color: var(--color-text);
  line-height: 1.6;
  padding: 10px 14px 10px 16px;
  background: var(--color-surface-low);
  background: var(--glass-bg);
  box-shadow: var(--glass-edge);
  border-radius: var(--radius-md);
  overflow: hidden;
  position: relative;
}

.cl-item::before {
  content: '';
  position: absolute;
  left: 0;
  top: 0;
  bottom: 0;
  width: 3px;
  border-radius: 0 2px 2px 0;
}

.cl-item--added::before    { background: var(--color-success);  }
.cl-item--improved::before { background: var(--color-tertiary); }
.cl-item--fixed::before    { background: var(--color-warning);  }

/* ── История: таймлайн свёрнутых версий ── */
.cl-history {
  margin-top: 30px;
  padding-top: 22px;
  border-top: 1px solid var(--color-outline-dim);
}

.cl-history-title {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 0 0 14px;
  font-size: 13px;
  font-weight: 800;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  color: var(--color-text-dim);
}
.cl-history-title .material-symbols-outlined { font-size: 18px; }

.cl-past-head {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  padding: 12px 14px;
  margin-bottom: 8px;
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-lg);
  background: var(--acrylic-card-bg);
  background: var(--glass-bg);
  box-shadow: var(--glass-edge);
  font: inherit;
  color: var(--color-text);
  text-align: left;
  cursor: pointer;
}
/* Hover — глобальное «запотевание» .glass-hover (main.css). */

.cl-past-info {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 5px;
}

.cl-past-line { display: flex; align-items: center; gap: 8px; }

.cl-past-title {
  font-size: 14.5px;
  font-weight: 700;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.cl-past-counts { display: flex; gap: 10px; }

.cl-count {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  font-size: 12px;
  font-weight: 700;
}
.cl-count .material-symbols-outlined { font-size: 15px; }
.cl-count--added    { color: var(--color-success);  }
.cl-count--improved { color: var(--color-tertiary); }
.cl-count--fixed    { color: var(--color-warning);  }

.cl-past-chev {
  flex-shrink: 0;
  color: var(--color-text-dim);
}

.cl-past-body {
  padding: 2px 2px 18px;
}

.cl-loading,
.cl-error {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 60px 0;
  color: var(--color-text-dim);
  font-size: 14px;
}

.spinning { animation: spin 1s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }

@media (max-width: 600px) {
  .cl-title { font-size: 20px; }
  .cl-highlights { grid-template-columns: 1fr; }
}
</style>
