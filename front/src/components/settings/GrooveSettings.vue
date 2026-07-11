<template>
  <div class="groove-settings">
    <div v-if="!companyId" class="settings-card groove-empty">
      <div class="hero-icon" data-tone="tertiary">
        <span class="material-symbols-outlined">domain</span>
      </div>
      <div class="card-text">
        <h3>Сначала выберите компанию</h3>
        <p>Используйте селектор компании в шапке, чтобы перейти к её настройкам Groove.</p>
      </div>
    </div>

    <template v-else>
      <section class="settings-card groove-card">
        <header class="groove-card-head">
          <div class="hero-icon" data-tone="tertiary">
            <span class="material-symbols-outlined">celebration</span>
          </div>
          <div class="card-text">
            <h3>Питомцы-Грувики</h3>
            <p>
              Игровая механика компании: у каждого сотрудника — питомец,
              растущий за работу (XP, кудосы, магазин, прогулки, лечение,
              поглаживание питомцев коллег, рейтинг недели).
            </p>
          </div>
        </header>

        <label class="switch-row">
          <span class="switch-text">
            <span class="material-symbols-outlined">pets</span>
            <span>
              <strong>Включить питомцев для компании</strong>
              <small>
                Когда выключено — раздел и плавающий питомец скрыты у всех
                сотрудников компании.
              </small>
            </span>
          </span>
          <input type="checkbox" v-model="enabled" class="switch" />
        </label>

        <p class="hint">{{ summary }}</p>

        <footer class="groove-actions">
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
import { useAuthStore } from '@/stores/auth.js'
import { useCompaniesStore } from '@/stores/companies.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { getGrooveSettings, updateGrooveSettings } from '@/api/companies.js'

const props = defineProps({ companyId: { type: Number, default: null } })
const auth = useAuthStore()
const companies = useCompaniesStore()
const notif = useNotificationsStore()
const { effectiveCompanyId } = storeToRefs(companies)
// Явный companyId (страница управления компанией) приоритетнее активной компании.
const companyId = computed(() => props.companyId ?? effectiveCompanyId.value)

const enabled = ref(true)
const initial = ref(true)
const saving = ref(false)

const dirty = computed(() => enabled.value !== initial.value)

const summary = computed(() =>
  enabled.value
    ? 'Грувики на месте: сотрудники получают питомцев, магазин и рейтинг недели.'
    : 'Режим выключен — раздел «Грувики» скрыт у всех сотрудников компании.',
)

async function load() {
  if (!companyId.value) return
  try {
    const data = await getGrooveSettings(companyId.value)
    enabled.value = data.enabled !== false
    initial.value = enabled.value
  } catch (e) {
    notif.error(e.message || 'Не удалось загрузить настройки Groove')
  }
}

async function onSave() {
  if (!companyId.value) return
  saving.value = true
  try {
    const data = await updateGrooveSettings(companyId.value, enabled.value)
    enabled.value = data.enabled !== false
    initial.value = enabled.value
    // Сразу отражаем смену в источниках, из которых useCompanySettings берёт
    // настройки (меню/гард раздела не ждут рефреша токена): клеймы своей сессии
    // (Руководитель) и список компаний (Администратор системы).
    if (auth.companyId === companyId.value) {
      auth.patchCompanySettings({ uses_groove: enabled.value })
    }
    companies.patchSettings(companyId.value, { uses_groove: enabled.value })
    notif.success(enabled.value ? 'Питомцы включены' : 'Питомцы выключены')
  } catch (e) {
    notif.error(e.message || 'Не удалось сохранить настройки Groove')
  } finally {
    saving.value = false
  }
}

onMounted(load)
watch(companyId, load)
</script>

<style scoped>
/* Тот же визуальный язык, что и в WeekendSettings/AiSettings — дублируем в
   scoped, потому что stylesheet родителя не доезжает в дочерний компонент. */
.groove-settings {
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
.hero-icon[data-tone="tertiary"] { --tone-bg: var(--color-tertiary-container); --tone-fg: var(--color-on-tertiary-container); }
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

.groove-card {
  flex-direction: column;
  align-items: stretch;
  gap: 18px;
}
.groove-card-head {
  display: flex;
  align-items: flex-start;
  gap: 18px;
}

.groove-empty { align-items: center; }

/* ── Switch-row (M3, синхронизирован с AiSettings/CompanyFormDialog) ── */
.switch-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 12px 14px;
  background: var(--color-surface-container, var(--color-surface-low));
  border-radius: var(--radius-md, 14px);
  cursor: pointer;
}

.switch-text {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
}
.switch-text > .material-symbols-outlined {
  flex: none;
  width: 40px;
  height: 40px;
  border-radius: var(--radius-md, 12px);
  display: grid;
  place-items: center;
  background: var(--color-tertiary-container);
  color: var(--color-on-tertiary-container);
  font-size: 22px;
}
.switch-text strong { display: block; font-size: 14px; color: var(--color-text); }
.switch-text small { display: block; font-size: 12px; line-height: 1.4; color: var(--color-text-dim); margin-top: 2px; }

.switch {
  appearance: none;
  flex: none;
  width: 52px;
  height: 32px;
  border-radius: 999px;
  background: var(--color-surface-highest, var(--color-surface-high));
  border: 2px solid var(--color-outline, var(--color-outline-variant));
  cursor: pointer;
  outline: none;
  position: relative;
  transition: background 0.18s, border-color 0.18s;
}
.switch::after {
  content: '';
  position: absolute;
  top: 50%;
  left: 5px;
  width: 18px;
  height: 18px;
  border-radius: 50%;
  background: var(--color-outline, var(--color-outline-variant));
  transform: translateY(-50%);
  transition: transform 0.18s, background 0.18s, width 0.18s, height 0.18s;
}
.switch:checked {
  background: var(--color-primary);
  border-color: var(--color-primary);
}
.switch:checked::after {
  background: var(--color-on-primary);
  width: 22px;
  height: 22px;
  transform: translate(18px, -50%);
}

.hint {
  margin: 0;
  font-size: 12px;
  color: var(--color-on-surface-variant);
  line-height: 1.4;
}

.groove-actions {
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
  .groove-settings { gap: 12px; }

  .settings-card {
    padding: 16px;
    gap: 12px;
    border-radius: 18px;
  }

  .groove-card { gap: 14px; }
  .groove-card-head {
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

  .groove-actions {
    flex-direction: column;
    align-items: stretch;
  }
  .groove-actions .btn-filled { width: 100%; }

  .groove-empty {
    flex-direction: column;
    text-align: center;
    align-items: center;
  }
}
</style>
