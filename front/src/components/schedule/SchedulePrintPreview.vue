<template>
  <AppDialog
    :model-value="modelValue"
    title="Печать расписания"
    subtitle="На листе — неделя целиком, по колонке на день"
    size="lg"
    @update:model-value="$emit('update:modelValue', $event)"
  >
    <AppStack :gap="12">
      <!-- Какую неделю печатать: у цикла их несколько, и на бумаге нужна
           готовая неделя, а не колонка «в какие недели идёт занятие». -->
      <div v-if="schedule.cycle_weeks > 1" class="pp-weeks">
        <AppChip
          v-for="w in cycleWeeks"
          :key="w"
          :label="weekLabel(schedule, w)"
          :selected="w === week"
          interactive
          @click="week = w"
        />
      </div>

      <!-- Предпросмотр: ровно то, что уйдёт на лист. -->
      <div class="pp-sheet">
        <table class="pp-table">
          <thead>
            <tr>
              <th class="pp-time">Время</th>
              <th v-for="d in days" :key="d">{{ WEEKDAYS[d].full }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="row in rows" :key="row.key">
              <td class="pp-time">{{ row.time }}</td>
              <td v-for="d in days" :key="d">
                <template v-for="it in row.byDay[d] || []" :key="it.id">
                  <b>{{ it.title }}</b>
                  <span v-if="metaOf(it)"> · {{ metaOf(it) }}</span>
                  <br />
                </template>
              </td>
            </tr>
          </tbody>
        </table>
        <p v-if="!rows.length" class="pp-empty">На этой неделе занятий нет.</p>
      </div>
    </AppStack>

    <template #footer>
      <span class="pp-spacer" />
      <AppButton label="Закрыть" @click="$emit('update:modelValue', false)" />
      <AppButton variant="filled" icon="print" label="Печать" :disabled="!rows.length" @click="print" />
    </template>
  </AppDialog>
</template>

<script setup>
import { computed, ref, watch } from 'vue'
import AppButton from '@/components/ui/AppButton.vue'
import AppChip from '@/components/ui/AppChip.vue'
import AppDialog from '@/components/ui/AppDialog.vue'
import AppStack from '@/components/ui/AppStack.vue'
import {
  WEEKDAYS, addDays, hhmm, itemsOfDay, mondayOf, weekIndex, weekLabel,
} from '@/utils/scheduleCycle.js'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  schedule: { type: Object, required: true },
  items: { type: Array, default: () => [] },
  monday: { type: [Date, String], required: true },
  days: { type: Array, default: () => [0, 1, 2, 3, 4] },
})

defineEmits(['update:modelValue'])

const week = ref(1)

const cycleWeeks = computed(() =>
  Array.from({ length: props.schedule?.cycle_weeks || 1 }, (_, i) => i + 1))

watch(() => props.modelValue, (open) => {
  if (open) {
    week.value = weekIndex(props.schedule.cycle_anchor, props.monday, props.schedule.cycle_weeks)
  }
})

/* Понедельник ВЫБРАННОЙ недели цикла: печатаем не «эту неделю на экране», а
   ту, что человек отметил чипом. */
const printMonday = computed(() => {
  const base = mondayOf(props.monday)
  const current = weekIndex(props.schedule.cycle_anchor, base, props.schedule.cycle_weeks)
  const shift = ((week.value - current) % props.schedule.cycle_weeks + props.schedule.cycle_weeks)
    % props.schedule.cycle_weeks
  return addDays(base, shift * 7)
})

const byDay = computed(() => {
  const map = {}
  props.days.forEach((d) => {
    map[d] = itemsOfDay(props.schedule, props.items, addDays(printMonday.value, d))
  })
  return map
})

/* Строки листа — по уникальным парам «начало–конец»: сетка звонков у каждого
   расписания своя, и рисовать часовую линейку на бумаге незачем. */
const rows = computed(() => {
  const slots = new Map()
  props.days.forEach((d) => {
    (byDay.value[d] || []).forEach((it) => {
      const key = `${it.start_min}-${it.end_min}`
      if (!slots.has(key)) {
        slots.set(key, { key, start: it.start_min, time: `${hhmm(it.start_min)}–${hhmm(it.end_min)}`, byDay: {} })
      }
      const row = slots.get(key)
      ;(row.byDay[d] ||= []).push(it)
    })
  })
  return [...slots.values()].sort((a, b) => a.start - b.start)
})

const categories = computed(() => {
  const map = {}
  for (const c of props.schedule?.categories || []) map[c.id] = c
  return map
})

function metaOf(item) {
  const parts = []
  const category = categories.value[item.category_id]
  if (category) parts.push(category.name)
  ;(props.schedule?.fields || []).filter((f) => f.show_on_block).forEach((f) => {
    const value = item.data?.[String(f.id)]
    if (value != null && value !== '') parts.push(String(value))
  })
  return parts.join(' · ')
}

// Печать во ВРЕМЕННОМ iframe, а не в новом окне: popup-блокировщики окно
// глушат, а iframe печатает и в мобильном WebView обёрток.
function print() {
  const esc = (s) => String(s).replace(/[&<>"]/g, (c) => (
    { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' }[c]
  ))
  const title = `${props.schedule.name}${props.schedule.cycle_weeks > 1 ? ` — ${weekLabel(props.schedule, week.value)}` : ''}`
  const head = props.days.map((d) => `<th>${esc(WEEKDAYS[d].full)}</th>`).join('')
  const body = rows.value.map((row) => {
    const cells = props.days.map((d) => {
      const list = (row.byDay[d] || []).map((it) => {
        const meta = metaOf(it)
        return `<b>${esc(it.title)}</b>${meta ? ` · ${esc(meta)}` : ''}`
      }).join('<br>')
      return `<td>${list}</td>`
    }).join('')
    return `<tr><td class="time">${esc(row.time)}</td>${cells}</tr>`
  }).join('')

  /* Печатный лист — самостоятельный документ: токены темы здесь неприменимы,
     на бумаге нужен чёрный текст на белом. */
  const html = `<!doctype html><html lang="ru"><head><meta charset="utf-8">
<title>${esc(title)}</title>
<style>
  @page { size: A4 landscape; margin: 10mm; }
  body { margin: 0; font-family: Arial, Helvetica, sans-serif; color: #000; background: #fff; }
  h1 { font-size: 14pt; margin: 0 0 6mm; }
  table { width: 100%; border-collapse: collapse; table-layout: fixed; }
  th, td { border: 0.4mm solid #999; padding: 2mm; font-size: 9pt; vertical-align: top; word-wrap: break-word; }
  th { background: #eee; font-size: 9pt; }
  td.time, th.time { width: 22mm; white-space: nowrap; font-variant-numeric: tabular-nums; }
  tr { page-break-inside: avoid; break-inside: avoid; }
</style></head><body>
<h1>${esc(title)}</h1>
<table><thead><tr><th class="time">Время</th>${head}</tr></thead><tbody>${body}</tbody></table>
</body></html>`

  const frame = document.createElement('iframe')
  frame.setAttribute('aria-hidden', 'true')
  frame.style.cssText = 'position:fixed;right:0;bottom:0;width:0;height:0;border:0;'
  document.body.appendChild(frame)
  const doc = frame.contentDocument
  doc.open()
  doc.write(html)
  doc.close()

  const run = () => {
    frame.contentWindow.focus()
    frame.contentWindow.print()
    // Убираем iframe только после диалога печати — иначе задание отменится.
    setTimeout(() => frame.remove(), 60000)
  }
  if (frame.contentWindow.document.readyState === 'complete') run()
  else frame.onload = run
}
</script>

<style scoped>
.pp-weeks { display: flex; flex-wrap: wrap; gap: 8px; }
.pp-sheet {
  overflow-x: auto;
  padding: 10px;
  border-radius: var(--radius-sm);
  background: var(--color-surface-container-lowest);
}
.pp-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 12px;
}
.pp-table th, .pp-table td {
  border: 1px solid var(--color-outline-variant);
  padding: 4px 6px;
  vertical-align: top;
  text-align: left;
  overflow-wrap: anywhere;
}
.pp-table th { background: var(--color-surface-container); font-weight: 600; }
.pp-time { white-space: nowrap; font-variant-numeric: tabular-nums; }
.pp-empty { margin: 0; padding-top: 8px; font-size: 13px; color: var(--color-on-surface-variant); }
.pp-spacer { flex: 1; }
</style>
