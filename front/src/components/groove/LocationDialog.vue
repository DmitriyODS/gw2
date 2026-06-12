<template>
  <AppDialog
    :model-value="modelValue"
    title="Погода за окном"
    subtitle="Подскажи Грувику, где ты — он будет знать, что у тебя за окном"
    icon="partly_cloudy_day"
    tone="primary"
    size="sm"
    @update:model-value="$emit('update:modelValue', $event)"
  >
    <div v-if="groove.location" class="loc-current">
      <span class="loc-current-chip">
        <span class="material-symbols-outlined">location_on</span>
        {{ groove.location.city || 'По координатам' }}
      </span>
      <span v-if="groove.weather" class="loc-current-weather">
        {{ groove.weather.emoji }} {{ formatTemp(groove.weather.temp_c) }} · {{ groove.weather.description }}
      </span>
      <button class="loc-remove" type="button" :disabled="busy" @click="remove">
        <span class="material-symbols-outlined">delete</span>
        Удалить
      </button>
    </div>

    <button class="loc-geo-btn" type="button" :disabled="busy" @click="useGeolocation">
      <span class="material-symbols-outlined">my_location</span>
      {{ geoLoading ? 'Определяем…' : 'Определить автоматически' }}
    </button>

    <div class="loc-search">
      <span class="material-symbols-outlined loc-search-ico">search</span>
      <input
        v-model.trim="query"
        class="loc-search-input"
        type="text"
        placeholder="Или найди свой город…"
        maxlength="80"
        @input="onQueryInput"
      />
    </div>

    <div v-if="searching" class="loc-hint">Ищем…</div>
    <div v-else-if="query.length >= 2 && !results.length && searched" class="loc-hint">
      Ничего не нашлось — попробуй иначе
    </div>
    <ul v-else-if="results.length" class="loc-results">
      <li v-for="(place, i) in results" :key="i">
        <button class="loc-result" type="button" :disabled="busy" @click="pick(place)">
          <span class="loc-result-name">{{ place.name }}</span>
          <span class="loc-result-sub">{{ placeSub(place) }}</span>
        </button>
      </li>
    </ul>

    <p class="loc-note">
      Локация видна только Грувику: он подмечает дождь, снег и жару в чате
      и брифингах. Удалить её можно в любой момент.
    </p>
  </AppDialog>
</template>

<script setup>
import { ref, watch } from 'vue'
import AppDialog from '@/components/common/AppDialog.vue'
import { useGrooveStore } from '@/stores/groove.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { searchCities } from '@/api/groove.js'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
})
const emit = defineEmits(['update:modelValue'])

const groove = useGrooveStore()
const notify = useNotificationsStore()

const query = ref('')
const results = ref([])
const searching = ref(false)
const searched = ref(false)
const busy = ref(false)
const geoLoading = ref(false)
let searchTimer = null
let searchSeq = 0

watch(() => props.modelValue, (open) => {
  if (open) {
    query.value = ''
    results.value = []
    searched.value = false
  }
})

function formatTemp(t) {
  const n = Math.round(t)
  return (n > 0 ? `+${n}` : `${n}`) + '°C'
}

function placeSub(place) {
  return [place.region, place.country].filter(Boolean).join(', ')
}

function onQueryInput() {
  clearTimeout(searchTimer)
  searched.value = false
  if (query.value.length < 2) {
    results.value = []
    return
  }
  searchTimer = setTimeout(runSearch, 350)
}

async function runSearch() {
  const seq = ++searchSeq
  searching.value = true
  try {
    const res = await searchCities(query.value)
    if (seq !== searchSeq) return
    results.value = res.items || []
    searched.value = true
  } catch {
    if (seq === searchSeq) results.value = []
  } finally {
    if (seq === searchSeq) searching.value = false
  }
}

async function save(payload, successText) {
  busy.value = true
  try {
    await groove.saveLocation(payload)
    notify.success(successText)
    emit('update:modelValue', false)
  } catch (e) {
    notify.error(e?.message || 'Не удалось сохранить локацию')
  } finally {
    busy.value = false
  }
}

function pick(place) {
  const city = [place.name, place.country].filter(Boolean).join(', ')
  save(
    { latitude: place.lat ?? place.latitude, longitude: place.lon ?? place.longitude, city },
    'Грувик теперь следит за погодой 🌦️'
  )
}

function useGeolocation() {
  if (!navigator.geolocation) {
    notify.warn('Браузер не поддерживает геолокацию — найди город вручную')
    return
  }
  geoLoading.value = true
  navigator.geolocation.getCurrentPosition(
    (pos) => {
      geoLoading.value = false
      save(
        { latitude: pos.coords.latitude, longitude: pos.coords.longitude, city: null },
        'Грувик теперь следит за погодой 🌦️'
      )
    },
    () => {
      geoLoading.value = false
      notify.warn('Не удалось определить локацию — найди город вручную')
    },
    { timeout: 10000, maximumAge: 600000 }
  )
}

async function remove() {
  busy.value = true
  try {
    await groove.removeLocation()
    notify.success('Локация удалена')
  } catch (e) {
    notify.error(e?.message || 'Не удалось удалить локацию')
  } finally {
    busy.value = false
  }
}
</script>

<style scoped>
.loc-current {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  padding: 10px 12px;
  margin-bottom: 12px;
  border-radius: var(--radius-md);
  background: var(--color-surface-low);
}

.loc-current-chip {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-weight: 600;
  font-size: 13.5px;
  color: var(--color-text);
}

.loc-current-chip .material-symbols-outlined {
  font-size: 17px;
  color: var(--color-primary);
}

.loc-current-weather {
  font-size: 13px;
  color: var(--color-text-dim);
}

.loc-remove {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  margin-left: auto;
  padding: 4px 10px;
  border: none;
  border-radius: var(--radius-full);
  background: transparent;
  color: var(--color-error);
  font-size: 12.5px;
  font-weight: 600;
  cursor: pointer;
}

.loc-remove:hover { background: color-mix(in oklch, var(--color-error) 10%, transparent); }
.loc-remove .material-symbols-outlined { font-size: 16px; }

.loc-geo-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  width: 100%;
  padding: 10px 14px;
  border: none;
  border-radius: var(--radius-md);
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: filter 0.15s;
}

.loc-geo-btn:hover:not(:disabled) { filter: brightness(0.97); }
.loc-geo-btn:disabled { opacity: 0.6; cursor: default; }
.loc-geo-btn .material-symbols-outlined { font-size: 19px; }

.loc-search {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 12px;
  padding: 0 12px;
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-md);
  background: var(--color-surface);
}

.loc-search-ico {
  font-size: 19px;
  color: var(--color-text-dim);
}

.loc-search-input {
  flex: 1;
  padding: 10px 0;
  border: none;
  outline: none;
  background: transparent;
  color: var(--color-text);
  font-size: 14px;
}

.loc-hint {
  margin-top: 10px;
  font-size: 13px;
  color: var(--color-text-dim);
  text-align: center;
}

.loc-results {
  list-style: none;
  margin: 8px 0 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
  max-height: 240px;
  overflow-y: auto;
}

.loc-result {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 1px;
  width: 100%;
  padding: 8px 12px;
  border: none;
  border-radius: var(--radius-md);
  background: var(--color-surface-low);
  cursor: pointer;
  text-align: left;
  transition: background 0.12s;
}

.loc-result:hover { background: var(--color-surface-high); }
.loc-result-name { font-size: 14px; font-weight: 600; color: var(--color-text); }
.loc-result-sub { font-size: 12px; color: var(--color-text-dim); }

.loc-note {
  margin: 14px 0 0;
  font-size: 12px;
  line-height: 1.45;
  color: var(--color-text-dim);
}
</style>
