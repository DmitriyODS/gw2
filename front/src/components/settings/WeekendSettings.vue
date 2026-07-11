<template>
  <div class="weekend-settings">
    <div v-if="!companyId" class="settings-card weekend-empty">
      <div class="hero-icon" data-tone="secondary">
        <span class="material-symbols-outlined">domain</span>
      </div>
      <div class="card-text">
        <h3>Сначала выберите компанию</h3>
        <p>Используйте селектор компании в шапке, чтобы перейти к её выходным дням.</p>
      </div>
    </div>

    <template v-else>
      <section class="settings-card weekend-card">
        <header class="weekend-card-head">
          <div class="hero-icon" data-tone="secondary">
            <span class="material-symbols-outlined">weekend</span>
          </div>
          <div class="card-text">
            <h3>Выходные дни компании</h3>
            <p>
              В отмеченные дни Грувик не считает простой, не заболевает от
              отсутствия работы и вместо призывов к задачам предлагает
              отдых и активности.
            </p>
          </div>
        </header>

        <div class="day-grid" role="group" aria-label="Выходные дни недели">
          <button
            v-for="day in DAYS"
            :key="day.idx"
            type="button"
            class="day-chip"
            :class="{ selected: selected.has(day.idx) }"
            :aria-pressed="selected.has(day.idx)"
            @click="toggleDay(day.idx)"
          >
            <span class="material-symbols-outlined day-check">
              {{ selected.has(day.idx) ? 'check' : 'add' }}
            </span>
            {{ day.label }}
          </button>
        </div>

        <p class="hint">
          {{ selectedSummary }}
        </p>

        <footer class="weekend-actions">
          <button class="btn-filled" :disabled="saving || !dirty" @click="onSave">
            <span class="material-symbols-outlined">save</span>
            {{ saving ? 'Сохраняю…' : 'Сохранить' }}
          </button>
        </footer>
      </section>
    </template>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import { storeToRefs } from 'pinia'
import { useCompaniesStore } from '@/stores/companies.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { getWeekendSettings, updateWeekendSettings } from '@/api/companies.js'

// Индексы совпадают с бэком: Python date.weekday(), 0=Пн … 6=Вс.
const DAYS = [
  { idx: 0, label: 'Пн' },
  { idx: 1, label: 'Вт' },
  { idx: 2, label: 'Ср' },
  { idx: 3, label: 'Чт' },
  { idx: 4, label: 'Пт' },
  { idx: 5, label: 'Сб' },
  { idx: 6, label: 'Вс' },
]
const DAY_TITLES = ['понедельник', 'вторник', 'среда', 'четверг', 'пятница', 'суббота', 'воскресенье']

const props = defineProps({ companyId: { type: Number, default: null } })
const companies = useCompaniesStore()
const notif = useNotificationsStore()
const { effectiveCompanyId } = storeToRefs(companies)
// Явный companyId (страница управления компанией) приоритетнее активной компании.
const companyId = computed(() => props.companyId ?? effectiveCompanyId.value)

const selected = ref(new Set([5, 6]))
const initial = ref([5, 6])
const saving = ref(false)

const current = computed(() => [...selected.value].sort((a, b) => a - b))
const dirty = computed(() => current.value.join(',') !== initial.value.join(','))

const selectedSummary = computed(() => {
  if (!current.value.length) return 'Выходных нет — Грувик будет ждать работу каждый день.'
  if (current.value.length === 7) return 'Отмечены все дни — Грувик никогда не позовёт работать.'
  return 'Выходные: ' + current.value.map(i => DAY_TITLES[i]).join(', ') + '.'
})

function toggleDay(idx) {
  const next = new Set(selected.value)
  if (next.has(idx)) next.delete(idx)
  else next.add(idx)
  selected.value = next
}

async function load() {
  if (!companyId.value) return
  try {
    const data = await getWeekendSettings(companyId.value)
    const days = (data.weekend_days || []).filter(d => d >= 0 && d <= 6)
    selected.value = new Set(days)
    initial.value = [...days].sort((a, b) => a - b)
  } catch (e) {
    notif.error(e.message || 'Не удалось загрузить выходные дни')
  }
}

async function onSave() {
  if (!companyId.value) return
  saving.value = true
  try {
    const data = await updateWeekendSettings(companyId.value, current.value)
    const days = (data.weekend_days || []).filter(d => d >= 0 && d <= 6)
    selected.value = new Set(days)
    initial.value = [...days].sort((a, b) => a - b)
    notif.success('Выходные дни сохранены')
  } catch (e) {
    notif.error(e.message || 'Не удалось сохранить выходные дни')
  } finally {
    saving.value = false
  }
}

onMounted(load)
watch(companyId, load)
</script>

<style scoped>
/* Тот же визуальный язык, что и в SettingsView/AiSettings — дублируем в
   scoped, потому что stylesheet родителя не доезжает в дочерний компонент. */
.weekend-settings {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.settings-card {
  display: flex;
  align-items: flex-start;
  gap: 18px;
  padding: 20px 22px;
  background: var(--acrylic-card-bg);
  background: var(--glass-bg);
  box-shadow: var(--glass-edge);
  border: 1px solid var(--acrylic-border);
  border-radius: 20px;
  transition: border-color 0.15s, box-shadow 0.15s;
}
.settings-card:hover {
  border-color: color-mix(in oklch, var(--color-primary) 30%, var(--color-outline-dim));
}

.hero-icon {
  flex-shrink: 0;
  width: 56px;
  height: 56px;
  border-radius: 16px;
  display: grid;
  place-items: center;
  background: var(--tone-bg, var(--color-primary-container));
  color: var(--tone-fg, var(--color-on-primary-container));
}
.hero-icon[data-tone="secondary"] { --tone-bg: var(--color-secondary-container); --tone-fg: var(--color-on-secondary-container); }
.hero-icon .material-symbols-outlined { font-size: 28px; }

.card-text { flex: 1; min-width: 0; }
.card-text h3 {
  margin: 0 0 4px;
  font-size: 16px;
  font-weight: 700;
  color: var(--color-text);
}
.card-text p {
  margin: 0;
  font-size: 13px;
  line-height: 1.5;
  color: var(--color-text-dim);
}

.weekend-card {
  flex-direction: column;
  align-items: stretch;
  gap: 18px;
}
.weekend-card-head {
  display: flex;
  align-items: flex-start;
  gap: 18px;
}

.weekend-empty { align-items: center; }

/* ── Чипы дней недели (M3 filter chips) ─────────────────────── */
.day-grid {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.day-chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 10px 16px;
  border-radius: 999px;
  border: 1px solid var(--color-outline-variant);
  background: var(--acrylic-card-bg);
  color: var(--color-text);
  font: inherit;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: background 0.15s, border-color 0.15s, color 0.15s;
}
.day-chip:hover {
  background: color-mix(in oklch, var(--color-primary) 8%, transparent);
  border-color: var(--color-primary);
}
.day-chip.selected {
  background: var(--color-secondary-container);
  border-color: transparent;
  color: var(--color-on-secondary-container);
}
.day-check { font-size: 18px; }

.hint {
  margin: 0;
  font-size: 12px;
  color: var(--color-on-surface-variant);
  line-height: 1.4;
}

.weekend-actions {
  display: flex;
  gap: 10px;
  justify-content: flex-end;
  flex-wrap: wrap;
}

.btn-filled {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 10px 18px;
  border-radius: 999px;
  border: none;
  font: inherit;
  font-weight: 600;
  cursor: pointer;
  background: var(--color-primary);
  color: var(--color-on-primary);
  transition: background 0.15s, box-shadow 0.15s;
}
.btn-filled:hover:not(:disabled) {
  background: color-mix(in oklch, var(--color-primary) 88%, var(--color-on-primary) 12%);
}
.btn-filled:disabled { opacity: 0.55; cursor: not-allowed; }
.btn-filled .material-symbols-outlined { font-size: 18px; }

/* ── Adaptive: мобильный ≤768 ───────────────────────────────── */
@media (max-width: 768px) {
  .weekend-settings { gap: 12px; }

  .settings-card {
    padding: 16px;
    gap: 12px;
    border-radius: 18px;
  }

  .weekend-card { gap: 14px; }
  .weekend-card-head {
    flex-direction: column;
    align-items: flex-start;
    gap: 12px;
  }

  .hero-icon {
    width: 48px;
    height: 48px;
    border-radius: 14px;
  }
  .hero-icon .material-symbols-outlined { font-size: 24px; }

  .card-text h3 { font-size: 15px; }
  .card-text p { font-size: 12px; }

  .day-chip {
    flex: 1 0 calc(25% - 8px);
    justify-content: center;
    padding: 10px 8px;
  }

  .weekend-actions {
    flex-direction: column;
    align-items: stretch;
  }
  .weekend-actions .btn-filled { width: 100%; }

  .weekend-empty {
    flex-direction: column;
    text-align: center;
    align-items: center;
  }
}
</style>
