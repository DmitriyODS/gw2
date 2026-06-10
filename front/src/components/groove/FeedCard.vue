<template>
  <article class="feed-card" :class="[`kind-${event.kind}`, { hero: !!heroEmoji }]">
    <span v-if="heroEmoji" class="fcard-hero-emoji" aria-hidden="true">{{ heroEmoji }}</span>
    <div class="fcard-head">
      <span v-if="isSystem" class="fcard-avatar bot" aria-hidden="true">👾</span>
      <img v-else class="fcard-avatar" :src="avatarUrl(event.user)" :alt="event.user?.fio || ''" />
      <div class="fcard-who">
        <span class="fcard-name">{{ authorName }}</span>
        <span class="fcard-time">{{ timeOf(event.created_at) }}</span>
      </div>
      <span class="fcard-kind" :class="`tone-${meta.tone}`">
        <span class="material-symbols-outlined">{{ meta.icon }}</span>
      </span>
    </div>

    <!-- Кудос: адресат + цитата -->
    <div v-if="event.kind === 'kudos'" class="fcard-kudos">
      <p class="fcard-text">поблагодарил(а) <strong>{{ event.payload?.to_fio }}</strong></p>
      <blockquote class="fcard-quote">«{{ event.payload?.text }}»</blockquote>
    </div>

    <!-- AI-дайджест: текст от Грувика -->
    <p v-else-if="event.kind === 'ai_digest'" class="fcard-text digest">
      {{ event.payload?.text }}
    </p>

    <p v-else class="fcard-text">
      {{ sentence }}
      <router-link
        v-if="taskLink"
        :to="taskLink"
        class="fcard-task-link"
      >{{ taskLinkLabel }}</router-link>
    </p>

    <div class="fcard-foot">
      <ReactionBar
        :reactions="event.reactions"
        :my-reactions="event.my_reactions"
        @toggle="emoji => groove.toggleReaction(event, emoji)"
      />
      <button class="fcard-comments-btn" type="button" @click="showComments = !showComments">
        <span class="material-symbols-outlined">chat_bubble</span>
        <span v-if="event.comments_count">{{ event.comments_count }}</span>
      </button>
    </div>

    <FeedComments v-if="showComments" :event-id="event.id" />
  </article>
</template>

<script setup>
import { computed, ref } from 'vue'
import { useGrooveStore } from '@/stores/groove.js'
import { avatarUrl, formatMinutes, PET_STAGES, PET_SPECIES, BOSS_EMOJI, SHOP_ITEMS, FEED_HERO_KINDS } from '@/utils/groove.js'
import ReactionBar from './ReactionBar.vue'
import FeedComments from './FeedComments.vue'

const props = defineProps({
  event: { type: Object, required: true },
})

const groove = useGrooveStore()
const showComments = ref(false)

const KIND_META = {
  unit_started: { icon: 'play_arrow', tone: 'primary' },
  unit_stopped: { icon: 'timer', tone: 'secondary' },
  task_closed: { icon: 'task_alt', tone: 'success' },
  streak: { icon: 'local_fire_department', tone: 'warning' },
  pet_evolved: { icon: 'auto_awesome', tone: 'tertiary' },
  kudos: { icon: 'volunteer_activism', tone: 'success' },
  ai_digest: { icon: 'wb_sunny', tone: 'tertiary' },
  raid_started: { icon: 'sports_mma', tone: 'error' },
  raid_won: { icon: 'emoji_events', tone: 'warning' },
  pet_sick: { icon: 'sick', tone: 'error' },
  pet_recovered: { icon: 'healing', tone: 'success' },
  wrapped: { icon: 'auto_awesome', tone: 'tertiary' },
  quest_done: { icon: 'rocket_launch', tone: 'tertiary' },
}

const meta = computed(() => KIND_META[props.event.kind] || { icon: 'bolt', tone: 'primary' })

const heroEmoji = computed(() => FEED_HERO_KINDS[props.event.kind] || null)

const isSystem = computed(() => !props.event.user)
const authorName = computed(() => props.event.user?.fio || 'Грувик')

const sentence = computed(() => {
  const p = props.event.payload || {}
  switch (props.event.kind) {
    case 'unit_started':
      return `взял(а) в работу «${p.unit_name}»`
    case 'unit_stopped':
      return `поработал(а) «${p.unit_name}» — ${formatMinutes(p.minutes)}`
    case 'task_closed':
      return 'закрыл(а) задачу'
    case 'streak':
      return `кормит «${p.pet_name || 'Грувика'}» ${p.days} дней подряд!`
    case 'pet_evolved': {
      const species = PET_SPECIES[p.species]?.title
      const stage = PET_STAGES[p.stage] || `стадия ${p.stage}`
      return `«${p.pet_name || 'Грувик'}» эволюционировал: ${stage}${species ? ` · ${species}` : ''} 🎉`
    }
    case 'raid_started':
      return `Новый рейд недели: ${BOSS_EMOJI[p.boss] || '👾'} «${p.boss}». Цель команды — ${p.target} закрытых задач!`
    case 'raid_won':
      return `${BOSS_EMOJI[p.boss] || '👾'} «${p.boss}» повержен! Всем Грувикам — ${SHOP_ITEMS[p.reward]?.title || 'награда'} и +${p.beans} грувов`
    case 'pet_sick':
      return `«${p.pet_name || 'Грувик'}» приболел 🤒 — хозяину пора вернуться в строй. Поглаживания тоже лечат!`
    case 'pet_recovered':
      return `«${p.pet_name || 'Грувик'}» выздоровел и снова сияет! 💚`
    case 'quest_done':
      return `выполнил(а) квест Грувика «${p.title || 'квест дня'}» и забрал(а) +${p.reward || 20} грувов 🚀`
    case 'wrapped': {
      const parts = []
      if (p.units) parts.push(`${p.units} юнитов`)
      if (p.minutes) parts.push(formatMinutes(p.minutes))
      if (p.closed) parts.push(`закрыто задач: ${p.closed}`)
      const best = p.best_day ? ` Самый мощный день — ${p.best_day}.` : ''
      return `итоги недели: ${parts.join(' · ') || 'тихая неделя'}.${best}`
    }
    default:
      return ''
  }
})

const taskLink = computed(() => {
  const p = props.event.payload || {}
  if (!p.task_id) return null
  if (!['unit_started', 'unit_stopped', 'task_closed'].includes(props.event.kind)) return null
  return `/tasks/${p.task_id}`
})

const taskLinkLabel = computed(() => {
  const p = props.event.payload || {}
  if (props.event.kind === 'task_closed') return `«${p.task_name}»`
  return p.task_name && p.task_name !== p.unit_name ? `· ${p.task_name}` : ''
})

function timeOf(iso) {
  return new Date(iso).toLocaleTimeString('ru-RU', { hour: '2-digit', minute: '2-digit' })
}
</script>

<style scoped>
.feed-card {
  background: var(--color-surface);
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-lg, 16px);
  padding: 12px 14px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.kind-ai_digest { border-color: color-mix(in oklch, var(--color-tertiary) 40%, transparent); }
.kind-raid_won { border-color: color-mix(in oklch, var(--color-warning) 50%, transparent); }

/* ── Hero-вехи: на всю ширину грида, градиент в тон события ───── */
.feed-card.hero {
  grid-column: 1 / -1;
  position: relative;
  overflow: hidden;
  padding: 16px 18px;
  border-color: color-mix(in oklch, var(--hero-color, var(--color-primary)) 38%, transparent);
  background: linear-gradient(135deg,
    color-mix(in oklch, var(--hero-color, var(--color-primary)) 14%, var(--color-surface)),
    var(--color-surface) 65%);
}
.hero.kind-streak { --hero-color: var(--color-warning); }
.hero.kind-pet_evolved { --hero-color: var(--color-tertiary); }
.hero.kind-raid_won { --hero-color: var(--color-warning); }
.hero.kind-wrapped { --hero-color: var(--color-tertiary); }
.hero .fcard-text { font-size: 15px; font-weight: 600; max-width: calc(100% - 64px); }
.fcard-hero-emoji {
  position: absolute;
  right: 10px;
  top: 50%;
  transform: translateY(-50%) rotate(10deg);
  font-size: 58px;
  line-height: 1;
  opacity: 0.18;
  pointer-events: none;
  user-select: none;
}

.fcard-head { display: flex; align-items: center; gap: 10px; }
.fcard-avatar {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  object-fit: cover;
  flex-shrink: 0;
}
.fcard-avatar.bot {
  display: grid;
  place-items: center;
  font-size: 19px;
  background: var(--color-tertiary-container);
}
.fcard-who { min-width: 0; flex: 1; display: flex; flex-direction: column; }
.fcard-name {
  font-size: 13.5px;
  font-weight: 600;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.fcard-time { font-size: 11.5px; color: var(--color-text-dim); }
.fcard-kind {
  width: 30px;
  height: 30px;
  border-radius: 10px;
  display: grid;
  place-items: center;
  flex-shrink: 0;
}
.fcard-kind .material-symbols-outlined { font-size: 18px; }
.tone-primary { background: var(--color-primary-container); color: var(--color-on-primary-container); }
.tone-secondary { background: var(--color-secondary-container); color: var(--color-on-secondary-container); }
.tone-tertiary { background: var(--color-tertiary-container); color: var(--color-on-tertiary-container); }
.tone-success { background: color-mix(in oklch, var(--color-success) 22%, transparent); color: var(--color-success); }
.tone-warning { background: color-mix(in oklch, var(--color-warning) 24%, transparent); color: var(--color-warning); }
.tone-error { background: color-mix(in oklch, var(--color-error) 18%, transparent); color: var(--color-error); }

.fcard-text {
  margin: 0;
  font-size: 14px;
  line-height: 1.45;
  word-break: break-word;
}
.fcard-text.digest { font-style: italic; }
.fcard-task-link {
  color: var(--color-primary);
  text-decoration: none;
  font-weight: 500;
}
.fcard-task-link:hover { text-decoration: underline; }
.fcard-kudos { display: flex; flex-direction: column; gap: 6px; }
.fcard-quote {
  margin: 0;
  padding: 8px 12px;
  border-left: 3px solid var(--color-success);
  background: color-mix(in oklch, var(--color-success) 10%, transparent);
  border-radius: 0 10px 10px 0;
  font-size: 13.5px;
  line-height: 1.4;
}
.fcard-foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}
.fcard-comments-btn {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  border: none;
  background: none;
  cursor: pointer;
  color: var(--color-text-dim);
  font-size: 12.5px;
  padding: 4px 6px;
  border-radius: var(--radius-full);
}
.fcard-comments-btn:hover { background: var(--color-surface-high); }
.fcard-comments-btn .material-symbols-outlined { font-size: 17px; }
</style>
