<template>
  <!-- AI-факт дня, если он есть; иначе — брендовый слайд с цитатой -->
  <div v-if="aiFact" class="tv-ai-fact-stage">
    <div class="tv-ai-fact-glow"></div>
    <div class="tv-ai-fact-eyebrow">
      <span class="material-symbols-outlined">lightbulb_2</span>
      Факт дня
    </div>
    <div class="tv-ai-fact-text">{{ aiFact.text }}</div>
    <div class="tv-ai-fact-foot">{{ dateLabel }} · Groove Work</div>
  </div>
  <div v-else class="tv-brand-stage">
    <div class="tv-brand-glow"></div>
    <img class="tv-brand-big-logo" src="/logo.svg" alt="" />
    <div class="tv-brand-big-name">Groove Work</div>
    <div class="tv-brand-quote">«{{ quote }}»</div>
    <div class="tv-brand-date">{{ dateLabel }}</div>
  </div>
</template>

<script setup>
// Брендовый слайд. Цитата выбирается при каждом монтировании — сцена
// пересоздаётся по :key на каждом показе слайда, так что цитаты ротируются.
import { ref } from 'vue'

defineProps({
  slide: { type: Object, required: true },
  aiFact: { type: Object, default: null }, // {text, generated_at, ...} | null
  dateLabel: { type: String, default: '' },
})

// Шуточные и тёплые цитаты в разном настроении.
const BRAND_QUOTES = [
  'Команда — это сила',
  'Лучшая задача — закрытая задача',
  'Сегодня было неплохо. А завтра будет ещё лучше.',
  'Каждый юнит — кусочек большого дела',
  'Кофе допит, дедлайны побеждены',
  'Закрывайте задачи, как двери — с уверенностью',
  'Делаем — значит делаем хорошо',
  'Если задача не двигается, значит она копит энергию',
  'Один за всех — и все на одном Groove',
  'Сегодня выложились — завтра выложимся ещё',
  'Пусть бэклог тает, как снег весной',
  'Кто рано встал — тот рано закрыл',
  'Не бывает маленьких задач — бывают большие закрытия',
  'Считаем не часы, а сделанное. Но часы тоже считаем.',
  'Время — деньги. У нас в платформе и то и другое под учётом.',
  'Лучший юнит — тот, который начат',
  'Помните: даже Эйнштейн делал ошибки в дедлайнах',
  'Релиз ближе, чем кажется',
  'Хорошего дня, хорошей команды и хорошего кофе',
  'Дисциплина — это когда ты закрываешь юнит до обеда',
]

const quote = ref(BRAND_QUOTES[Math.floor(Math.random() * BRAND_QUOTES.length)])
</script>

<style scoped>
.tv-brand-stage {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: clamp(12px, 1.6vmin, 22px);
  position: relative;
  text-align: center;
}

.tv-brand-glow {
  position: absolute;
  inset: 0;
  background: radial-gradient(circle at 50% 50%,
    color-mix(in oklch, var(--color-primary) 22%, transparent),
    transparent 60%);
  filter: blur(20px);
  pointer-events: none;
}

.tv-brand-big-logo {
  position: relative;
  width: clamp(96px, 14vmin, 220px);
  height: clamp(96px, 14vmin, 220px);
  border-radius: 50%;
  animation: tv-brand-pulse 3.6s ease-in-out infinite;
}

@keyframes tv-brand-pulse {
  0%, 100% { transform: scale(1); filter: drop-shadow(0 0 18px color-mix(in oklch, var(--color-primary) 35%, transparent)); }
  50%      { transform: scale(1.04); filter: drop-shadow(0 0 32px color-mix(in oklch, var(--color-primary) 55%, transparent)); }
}

.tv-brand-big-name {
  position: relative;
  font-size: clamp(36px, 6.4vmin, 96px);
  font-weight: 900;
  letter-spacing: 0.02em;
  color: var(--color-text);
}

.tv-brand-quote {
  position: relative;
  font-size: clamp(16px, 2.2vmin, 28px);
  color: var(--color-text-dim);
  font-style: italic;
}

.tv-brand-date {
  position: relative;
  font-size: clamp(14px, 1.8vmin, 22px);
  color: var(--color-text-dim);
  text-transform: capitalize;
  font-weight: 600;
}

/* ── AI-факт дня ── */
.tv-ai-fact-stage {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: clamp(20px, 2.6vmin, 36px);
  padding: clamp(24px, 3vmin, 56px);
  position: relative;
  text-align: center;
}

.tv-ai-fact-glow {
  position: absolute;
  inset: 0;
  background: radial-gradient(ellipse at 50% 50%,
    color-mix(in oklch, var(--color-tertiary) 28%, transparent),
    transparent 65%);
  filter: blur(28px);
  pointer-events: none;
}

.tv-ai-fact-eyebrow {
  position: relative;
  display: inline-flex;
  align-items: center;
  gap: 10px;
  font-size: clamp(14px, 1.8vmin, 20px);
  font-weight: 700;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: var(--color-text);
}
.tv-ai-fact-eyebrow .material-symbols-outlined {
  font-size: clamp(22px, 2.6vmin, 30px);
  font-variation-settings: 'FILL' 1;
  color: var(--color-tertiary);
  animation: tv-ai-fact-pulse 2.4s ease-in-out infinite;
}

@keyframes tv-ai-fact-pulse {
  0%, 100% { transform: scale(1); filter: drop-shadow(0 0 6px color-mix(in oklch, var(--color-tertiary) 45%, transparent)); }
  50%      { transform: scale(1.06); filter: drop-shadow(0 0 12px color-mix(in oklch, var(--color-tertiary) 65%, transparent)); }
}

.tv-ai-fact-text {
  position: relative;
  font-size: clamp(20px, 2.8vmin, 40px);
  line-height: 1.4;
  font-weight: 500;
  color: var(--color-text);
  max-width: 36ch;
  text-wrap: balance;
  animation: tv-ai-fact-rise 0.7s cubic-bezier(0.16, 1, 0.3, 1);
}

@keyframes tv-ai-fact-rise {
  from { opacity: 0; transform: translateY(14px); }
  to   { opacity: 1; transform: translateY(0); }
}

.tv-ai-fact-foot {
  position: relative;
  font-size: clamp(13px, 1.6vmin, 20px);
  color: var(--color-text-dim);
  text-transform: capitalize;
  font-weight: 600;
}
</style>
