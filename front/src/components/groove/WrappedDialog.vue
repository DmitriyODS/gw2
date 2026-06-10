<template>
  <AppDialog
    :model-value="modelValue"
    title="Моя неделя"
    subtitle="Личный итог последних 7 дней"
    icon="auto_awesome"
    tone="tertiary"
    size="md"
    @update:model-value="$emit('update:modelValue', $event)"
  >
    <div v-if="groove.wrappedLoading" class="wr-loading">Собираем вашу неделю…</div>

    <div v-else-if="w" class="wr-story" @click="onStoryClick">
      <div class="wr-progress">
        <span
          v-for="(s, i) in slides"
          :key="s.key"
          class="wr-progress-seg"
          :class="{ done: i < idx, active: i === idx }"
        ></span>
      </div>

      <Transition name="wr-slide" mode="out-in">
        <div :key="slide.key" class="wr-card" :class="`tone-${slide.tone}`">
          <span class="wr-emoji">{{ slide.emoji }}</span>
          <h3 class="wr-title">{{ slide.title }}</h3>
          <div v-if="slide.big" class="wr-big">{{ slide.big }}</div>
          <p v-if="slide.sub" class="wr-sub">{{ slide.sub }}</p>
          <p v-if="slide.extra" class="wr-extra">{{ slide.extra }}</p>

          <div v-if="slide.key === 'social' && w.soulmate" class="wr-soulmate">
            <img :src="avatarUrl(w.soulmate.user)" :alt="w.soulmate.user.fio" />
            <span>{{ w.soulmate.user.fio }}</span>
          </div>

          <button
            v-if="slide.key === 'final'"
            class="wr-share"
            type="button"
            :disabled="sharing || shared"
            @click.stop="share"
          >
            <span class="material-symbols-outlined">campaign</span>
            {{ shared ? 'Опубликовано!' : 'Поделиться в ленте' }}
          </button>
        </div>
      </Transition>

      <div class="wr-nav-hint">
        <span class="material-symbols-outlined">touch_app</span>
        ткните слева/справа, чтобы листать · {{ idx + 1 }}/{{ slides.length }}
      </div>
    </div>
  </AppDialog>
</template>

<script setup>
import { computed, ref, watch } from 'vue'
import AppDialog from '@/components/common/AppDialog.vue'
import { useGrooveStore } from '@/stores/groove.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { avatarUrl, formatMinutes, PET_STAGES, petEmoji } from '@/utils/groove.js'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
})
defineEmits(['update:modelValue'])

const groove = useGrooveStore()
const notify = useNotificationsStore()

const idx = ref(0)
const sharing = ref(false)
const shared = ref(false)

const w = computed(() => groove.wrapped)

watch(() => props.modelValue, (open) => {
  if (!open) return
  idx.value = 0
  shared.value = false
  groove.fetchWrapped().catch(() => notify.warn('Не удалось собрать итоги недели'))
})

const slides = computed(() => {
  if (!w.value) return []
  const v = w.value
  const out = []

  out.push({
    key: 'intro', tone: 'tertiary', emoji: '✨',
    title: 'Ваша неделя в Groove',
    big: null,
    sub: v.ai_phrase || 'Семь дней — полёт нормальный. Листайте!',
  })

  out.push({
    key: 'time', tone: 'primary', emoji: '⏱️',
    title: 'В потоке',
    big: formatMinutes(v.minutes),
    sub: v.units
      ? `за ${v.units} ${plural(v.units, 'юнит', 'юнита', 'юнитов')}`
      : 'юнитов на этой неделе не было — бывает!',
    extra: v.longest ? `Самый длинный заход — «${v.longest.name}», ${formatMinutes(v.longest.minutes)}` : null,
  })

  if (v.best_day || v.peak_hour != null) {
    out.push({
      key: 'rhythm', tone: 'secondary', emoji: '📈',
      title: 'Ваш ритм',
      big: v.best_day ? v.best_day.label : '—',
      sub: v.best_day ? `самый продуктивный день · ${formatMinutes(v.best_day.minutes)}` : '',
      extra: v.peak_hour != null ? `Пик формы — около ${v.peak_hour}:00` : null,
    })
  }

  out.push({
    key: 'tasks', tone: 'success', emoji: '✅',
    title: 'Доведено до конца',
    big: String(v.closed),
    sub: plural(v.closed, 'задача закрыта', 'задачи закрыто', 'задач закрыто'),
  })

  out.push({
    key: 'social', tone: 'warning', emoji: '🤝',
    title: 'Команда вас видит',
    big: String(v.reactions + v.kudos),
    sub: `${v.reactions} ${plural(v.reactions, 'реакция', 'реакции', 'реакций')} и ${v.kudos} ${plural(v.kudos, 'кудос', 'кудоса', 'кудосов')}`,
    extra: v.soulmate ? `Соулмейт недели — вместе над одними задачами:` : null,
  })

  out.push({
    key: 'final', tone: 'tertiary', emoji: petEmoji(v.pet),
    title: `«${v.pet.name}» гордится вами`,
    big: PET_STAGES[v.pet.stage],
    sub: v.pet.sick
      ? 'Правда, он приболел — на следующей неделе подлечим!'
      : `Стрик кормления — ${v.pet.feed_streak} дн. Так держать!`,
  })

  return out
})

const slide = computed(() => slides.value[idx.value] || slides.value[0])

function plural(n, one, few, many) {
  const m10 = n % 10, m100 = n % 100
  if (m10 === 1 && m100 !== 11) return one
  if (m10 >= 2 && m10 <= 4 && (m100 < 12 || m100 > 14)) return few
  return many
}

function onStoryClick(e) {
  const rect = e.currentTarget.getBoundingClientRect()
  const left = (e.clientX - rect.left) < rect.width / 2
  if (left) idx.value = Math.max(0, idx.value - 1)
  else idx.value = Math.min(slides.value.length - 1, idx.value + 1)
}

async function share() {
  sharing.value = true
  try {
    await groove.shareWrapped()
    shared.value = true
    notify.success('Итоги недели улетели в ленту ✨')
  } catch (e) {
    notify.warn(e?.message || 'Не получилось опубликовать')
  } finally {
    sharing.value = false
  }
}
</script>

<style scoped>
.wr-loading {
  text-align: center;
  padding: 40px 0;
  color: var(--color-text-dim);
}
.wr-story { cursor: pointer; user-select: none; }
.wr-progress { display: flex; gap: 4px; margin-bottom: 12px; }
.wr-progress-seg {
  flex: 1;
  height: 4px;
  border-radius: 2px;
  background: var(--color-surface-high);
}
.wr-progress-seg.done { background: var(--color-primary); }
.wr-progress-seg.active { background: color-mix(in oklch, var(--color-primary) 55%, transparent); }

.wr-card {
  border-radius: 20px;
  padding: 36px 24px;
  min-height: 270px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  text-align: center;
  gap: 8px;
}
.tone-primary { background: var(--color-primary-container); color: var(--color-on-primary-container); }
.tone-secondary { background: var(--color-secondary-container); color: var(--color-on-secondary-container); }
.tone-tertiary { background: var(--color-tertiary-container); color: var(--color-on-tertiary-container); }
.tone-success { background: color-mix(in oklch, var(--color-success) 18%, var(--color-surface)); color: var(--color-text); }
.tone-warning { background: color-mix(in oklch, var(--color-warning) 20%, var(--color-surface)); color: var(--color-text); }

.wr-emoji { font-size: 44px; line-height: 1; }
.wr-title { margin: 0; font-size: 16px; font-weight: 700; }
.wr-big { font-size: 38px; font-weight: 800; line-height: 1.1; }
.wr-sub { margin: 0; font-size: 14px; opacity: 0.85; line-height: 1.45; max-width: 320px; }
.wr-extra { margin: 6px 0 0; font-size: 12.5px; opacity: 0.7; line-height: 1.4; max-width: 320px; }

.wr-soulmate {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 8px;
  background: var(--color-surface);
  color: var(--color-text);
  border-radius: var(--radius-full);
  padding: 6px 14px 6px 6px;
  font-size: 13.5px;
  font-weight: 600;
}
.wr-soulmate img { width: 30px; height: 30px; border-radius: 50%; object-fit: cover; }

.wr-share {
  margin-top: 14px;
  display: inline-flex;
  align-items: center;
  gap: 8px;
  border: none;
  border-radius: var(--radius-full);
  background: var(--color-primary);
  color: var(--color-on-primary);
  font-size: 14px;
  font-weight: 600;
  padding: 11px 22px;
  cursor: pointer;
}
.wr-share:disabled { opacity: 0.55; cursor: default; }
.wr-share .material-symbols-outlined { font-size: 18px; }

.wr-nav-hint {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  margin-top: 10px;
  font-size: 11.5px;
  color: var(--color-text-dim);
}
.wr-nav-hint .material-symbols-outlined { font-size: 14px; }

.wr-slide-enter-active, .wr-slide-leave-active { transition: opacity 0.18s, transform 0.18s; }
.wr-slide-enter-from { opacity: 0; transform: translateX(14px); }
.wr-slide-leave-to { opacity: 0; transform: translateX(-14px); }
</style>
