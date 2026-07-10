<template>
  <div class="promo">
    <!-- ── Hero: гигантская типографика + парящие карточки и формы ── -->
    <section class="promo-hero">
      <!-- Декоративные формы -->
      <span class="promo-shape promo-shape--circle" aria-hidden="true"></span>
      <span class="promo-shape promo-shape--squircle" aria-hidden="true"></span>
      <span class="promo-shape promo-shape--spark" aria-hidden="true">✦</span>
      <span class="promo-shape promo-shape--cursor material-symbols-outlined" aria-hidden="true">near_me</span>

      <!-- Парящие карточки-статы -->
      <div class="promo-float promo-float--stat reveal" aria-hidden="true">
        <span class="promo-float-num">8</span>
        <span class="promo-float-text">разделов в одном окне — от задач и звонков до грувиков</span>
      </div>
      <div class="promo-float promo-float--team reveal" aria-hidden="true">
        <span class="promo-float-chip"><span class="material-symbols-outlined">group</span> Команда</span>
        <span class="promo-float-pets">
          <EmojiGlyph char="🦊" /><EmojiGlyph char="🐼" /><EmojiGlyph char="🦄" />
        </span>
        <span class="promo-float-text">у каждого сотрудника — свой грувик, который растёт от работы</span>
      </div>
      <div class="promo-float promo-float--badge reveal" aria-hidden="true">
        <span class="material-symbols-outlined">bolt</span>
        Делаем работу приятнее
      </div>

      <p class="promo-eyebrow reveal">платформа для команд</p>
      <h1 class="promo-display reveal">
        Groove<span class="promo-display-accent">Work</span>
      </h1>
      <p class="promo-sub reveal">
        Задачи и учёт времени, мессенджер и звонки, портал и совместные
        заметки — и целая команда грувиков, превращающая будни в игру.
      </p>
      <div class="promo-actions reveal">
        <RouterLink class="promo-btn promo-btn--lg" to="/register">
          Попробовать уже сейчас
          <span class="material-symbols-outlined">arrow_forward</span>
        </RouterLink>
        <span class="promo-cta-hint">Бесплатно · регистрация за минуту</span>
      </div>
    </section>

    <!-- ── Волна Groove: фирменный мотив логотипа в движении ── -->
    <div class="promo-ocean" aria-hidden="true">
      <div class="promo-wave promo-wave--soft"><svg viewBox="0 0 2880 140" preserveAspectRatio="none"><path :d="WAVE_PATH" /></svg></div>
      <div class="promo-wave promo-wave--mid"><svg viewBox="0 0 2880 140" preserveAspectRatio="none"><path :d="WAVE_PATH" /></svg></div>
      <div class="promo-wave promo-wave--deep"><svg viewBox="0 0 2880 140" preserveAspectRatio="none"><path :d="WAVE_PATH" /></svg></div>
    </div>

    <!-- ── Фичи ── -->
    <section class="promo-features">
      <article v-for="f in FEATURES" :key="f.title" class="promo-card reveal">
        <span class="promo-card-icon material-symbols-outlined">{{ f.icon }}</span>
        <h3>{{ f.title }}</h3>
        <p>{{ f.text }}</p>
      </article>
    </section>

    <!-- ── Как начать ── -->
    <section class="promo-steps">
      <h2 class="promo-h2 reveal">Начать проще, чем кажется</h2>
      <div class="promo-steps-row">
        <div v-for="(s, i) in STEPS" :key="i" class="promo-step reveal">
          <span class="promo-step-num">{{ i + 1 }}</span>
          <h4>{{ s.title }}</h4>
          <p>{{ s.text }}</p>
        </div>
      </div>
    </section>

    <!-- ── Финальный CTA ── -->
    <section class="promo-final reveal">
      <div class="promo-final-card">
        <div class="promo-final-pets" aria-hidden="true">
          <span v-for="(e, i) in FINAL_PETS" :key="i" class="promo-final-pet" :style="{ animationDelay: `${i * 0.35}s` }">
            <EmojiGlyph :char="e" />
          </span>
        </div>
        <h2 class="promo-h2">Команда уже ждёт. <span class="promo-final-tail">И грувик тоже.</span></h2>
        <p class="promo-sub">Создайте компанию за минуту — без карточек, тарифов и звонков менеджера.</p>
        <RouterLink class="promo-btn promo-btn--lg" to="/register">
          Попробовать уже сейчас
          <span class="material-symbols-outlined">arrow_forward</span>
        </RouterLink>
      </div>
    </section>

    <footer class="promo-foot">
      <span>Groove Work · сделано с 💛 и кудосами</span>
      <RouterLink to="/login">Вход для команд</RouterLink>
    </footer>
  </div>
</template>

<script setup>
import { onBeforeUnmount, onMounted } from 'vue'
import EmojiGlyph from '@/components/common/EmojiGlyph.vue'

const FINAL_PETS = ['🦊', '🐼', '🐧', '🐲', '🦄']

// Периодическая волна (6 периодов по 480 на 2880): сдвиг ровно на половину
// ширины совпадает сам с собой — бесшовный дрейф-цикл.
const WAVE_PATH = 'M0,70 q120,52 240,0 t240,0 t240,0 t240,0 t240,0 t240,0 t240,0 t240,0 t240,0 t240,0 t240,0 t240,0 L2880,140 L0,140 Z'

const FEATURES = [
  {
    icon: 'timer', title: 'Задачи и время',
    text: 'Юниты учитывают каждую минуту работы сами — статистика честная, отчёты собираются без напоминаний.',
  },
  {
    icon: 'forum', title: 'Мессенджер и звонки',
    text: 'Чаты, файлы, демонстрация экрана и видеозвонки — вся команда в одном окне, без сторонних сервисов.',
  },
  {
    icon: 'campaign', title: 'Корпоративный портал',
    text: 'Новости, посты и обсуждения с красивым оформлением — команда в курсе, даже когда все на удалёнке.',
  },
  {
    icon: 'pets', title: 'Грувики',
    text: 'У каждого — питомец, который растёт от вашей работы. Кудосы, банк с копилками, магазин обликов и домик с гостями.',
  },
  {
    icon: 'edit_note', title: 'Заметки и ежедневники',
    text: 'Личные и совместные: пишите документ вдвоём вживую, раздавайте списки дел и следите за прогрессом.',
  },
  {
    icon: 'devices', title: 'На любом устройстве',
    text: 'Браузер, приложение для компьютера с иконкой в трее и Android-приложение с пушами — всё синхронно.',
  },
]

const STEPS = [
  { title: 'Зарегистрируйтесь', text: 'Почта, пароль и подтверждение — минута, честно.' },
  { title: 'Создайте компанию', text: 'Вы сразу администратор: настройте всё под свою команду.' },
  { title: 'Позовите коллег', text: 'Ссылка-приглашение или письмо — и у каждого появится грувик.' },
]

// Плавное появление секций при скролле (с уважением к reduced-motion).
let observer = null
onMounted(() => {
  const reduced = window.matchMedia?.('(prefers-reduced-motion: reduce)')?.matches
  const els = document.querySelectorAll('.promo .reveal')
  if (reduced || !('IntersectionObserver' in window)) {
    els.forEach((el) => el.classList.add('is-visible'))
    return
  }
  observer = new IntersectionObserver((entries) => {
    for (const e of entries) {
      if (e.isIntersecting) {
        e.target.classList.add('is-visible')
        observer.unobserve(e.target)
      }
    }
  }, { threshold: 0.15 })
  els.forEach((el) => observer.observe(el))
})
onBeforeUnmount(() => observer?.disconnect())
</script>

<style scoped>
/* ── Палитра промо ─────────────────────────────────────────────────
   Витрина НАМЕРЕННО отвязана от пользовательских тем приложения:
   классические фиолетовые цвета фиксированы хексами (исключение из
   правила токенов — как email-шаблоны), а светлая/тёмная версия
   выбирается ТОЛЬКО системной prefers-color-scheme. */
.promo {
  --p-bg: #e9e4fa;
  --p-surface: #ffffff;
  --p-surface-soft: rgba(255, 255, 255, 0.72);
  --p-text: #17141f;
  --p-dim: #5c5570;
  --p-accent: #7b5cfa;
  --p-accent-deep: #5b3df5;
  --p-accent-soft: #c9b8ff;
  --p-shadow: 0 18px 44px rgba(76, 46, 178, 0.16);
}
@media (prefers-color-scheme: dark) {
  .promo {
    --p-bg: #16121f;
    --p-surface: #241e33;
    --p-surface-soft: rgba(36, 30, 51, 0.82);
    --p-text: #f1edfc;
    --p-dim: #a79fbe;
    --p-accent: #9b82ff;
    --p-accent-deep: #7b5cfa;
    --p-accent-soft: #4a3a85;
    --p-shadow: 0 18px 44px rgba(0, 0, 0, 0.45);
  }
}

.promo {
  position: relative;
  width: 100%;
  align-self: stretch;
  min-height: 100dvh;
  overflow-x: hidden;
  display: flex;
  flex-direction: column;
  align-items: stretch;
  background: var(--p-bg);
  color: var(--p-text);
}

/* ── Кнопки промо (своя палитра, не btn-grad приложения) ── */
.promo-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  border: none;
  border-radius: 999px;
  background: linear-gradient(120deg, var(--p-accent), var(--p-accent-deep));
  color: #ffffff;
  font-weight: 700;
  text-decoration: none;
  cursor: pointer;
  transition: transform 0.15s, box-shadow 0.15s;
  box-shadow: 0 10px 26px rgba(91, 61, 245, 0.35);
}
.promo-btn:hover { transform: translateY(-2px) scale(1.02); }
.promo-btn--lg { font-size: 16px; padding: 15px 32px; }

/* ── Hero ── */
.promo-hero {
  position: relative;
  z-index: 1;
  text-align: center;
  padding: clamp(72px, 14vh, 150px) 20px clamp(56px, 10vh, 110px);
  max-width: 1100px;
  margin: 0 auto;
  width: 100%;
}
/* Пилюля на подложке — не сливается с фигурами позади (круг в hero). */
.promo-eyebrow {
  position: relative;
  z-index: 2;
  display: inline-flex;
  align-self: center;
  margin: 0 0 14px;
  padding: 8px 16px;
  border-radius: 999px;
  background: var(--p-surface);
  box-shadow: 0 6px 18px rgba(76, 46, 178, 0.14);
  font-size: 12px;
  font-weight: 800;
  letter-spacing: 0.2em;
  text-transform: uppercase;
  color: var(--p-accent-deep);
}
/* Гигантское слово-логотип: чёрная гарнитура + акцентная часть. */
.promo-display {
  position: relative;
  z-index: 2;
  margin: 0 0 18px;
  font-size: clamp(52px, 12vw, 148px);
  font-weight: 900;
  line-height: 0.95;
  letter-spacing: -0.045em;
}
.promo-display-accent { color: var(--p-accent); }
.promo-sub {
  position: relative;
  z-index: 2;
  margin: 0 auto 28px;
  max-width: 560px;
  font-size: clamp(15px, 2.4vw, 17.5px);
  line-height: 1.55;
  color: var(--p-dim);
}
.promo-actions { position: relative; z-index: 2; display: flex; flex-direction: column; align-items: center; gap: 10px; }
.promo-cta-hint { font-size: 12.5px; color: var(--p-dim); }

/* ── Формы (реверанс референсу: круг, сквиркл, искра, курсор) ── */
.promo-shape { position: absolute; pointer-events: none; }
.promo-shape--circle {
  right: -12%;
  top: 4%;
  width: clamp(260px, 40vw, 560px);
  height: clamp(260px, 40vw, 560px);
  border-radius: 50%;
  background: radial-gradient(120% 120% at 30% 20%, var(--p-accent-soft) 0%, var(--p-accent) 78%);
  opacity: 0.9;
  animation: promo-drift 18s ease-in-out infinite alternate;
}
.promo-shape--squircle {
  left: -8%;
  bottom: -6%;
  width: clamp(180px, 26vw, 360px);
  height: clamp(180px, 26vw, 360px);
  border-radius: 34%;
  transform: rotate(24deg);
  background: linear-gradient(140deg, var(--p-accent) 10%, var(--p-accent-deep) 90%);
  opacity: 0.85;
  animation: promo-drift 22s ease-in-out infinite alternate-reverse;
}
/* Искра живёт между карточкой-статом (слева) и заголовком — не под ними. */
.promo-shape--spark {
  left: 28%;
  top: 4%;
  font-size: clamp(44px, 6vw, 80px);
  color: var(--p-accent);
  line-height: 1;
  animation: promo-spin 14s linear infinite;
}
.promo-shape--cursor {
  right: 20%;
  bottom: 4%;
  font-size: clamp(48px, 7vw, 96px);
  color: var(--p-surface);
  opacity: 0.85;
  transform: rotate(84deg);
  filter: drop-shadow(0 10px 22px rgba(76, 46, 178, 0.35));
  animation: promo-bob 6s ease-in-out infinite;
}
@keyframes promo-drift {
  from { transform: translate(0, 0); }
  to { transform: translate(3%, 5%); }
}
@keyframes promo-spin {
  from { transform: rotate(0); }
  to { transform: rotate(360deg); }
}
@keyframes promo-bob {
  0%, 100% { transform: translateY(0) rotate(84deg); }
  50% { transform: translateY(-16px) rotate(84deg); }
}

/* ── Парящие карточки ── */
.promo-float {
  position: absolute;
  z-index: 3;
  background: var(--p-surface);
  border-radius: 22px;
  box-shadow: var(--p-shadow);
  text-align: left;
  animation: promo-float 7s ease-in-out infinite;
}
@keyframes promo-float {
  0%, 100% { transform: translateY(0); }
  50% { transform: translateY(-12px); }
}
.promo-float--stat {
  left: 3%;
  top: 12%;
  width: 200px;
  padding: 16px 18px;
  animation-delay: 0.6s;
}
.promo-float-num { display: block; font-size: 44px; font-weight: 900; line-height: 1; }
.promo-float-text { display: block; margin-top: 6px; font-size: 12px; line-height: 1.4; color: var(--p-dim); }
.promo-float--team {
  left: 6%;
  bottom: 10%;
  width: 220px;
  padding: 14px 16px;
  animation-delay: 1.4s;
}
.promo-float-chip {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  font-size: 11.5px;
  font-weight: 700;
  color: var(--p-accent-deep);
  background: rgba(123, 92, 250, 0.12);
  border-radius: 999px;
  padding: 4px 10px;
  margin-bottom: 8px;
}
.promo-float-chip .material-symbols-outlined { font-size: 14px; }
.promo-float-pets { display: flex; gap: 6px; font-size: 24px; margin-bottom: 6px; }
.promo-float--badge {
  right: 4%;
  top: 10%;
  display: inline-flex;
  align-items: center;
  gap: 7px;
  padding: 12px 18px;
  font-size: 13.5px;
  font-weight: 700;
  border-radius: 999px;
  animation-delay: 2.2s;
}
.promo-float--badge .material-symbols-outlined { color: var(--p-accent); font-size: 19px; }

/* ── Волна Groove: фоновая текстура, а не пояс. Живёт ПОД контентом
   (отрицательные поля заводят её под hero и карточки фич) и медленно
   растворяется книзу маской — переход в следующий контент без шва. ── */
.promo-ocean {
  position: relative;
  z-index: 0;
  height: clamp(160px, 24vw, 300px);
  margin-top: clamp(-90px, -10vw, -40px);
  margin-bottom: clamp(-140px, -16vw, -70px);
  overflow: hidden;
  pointer-events: none;
  -webkit-mask-image: linear-gradient(180deg, black 0%, rgba(0, 0, 0, 0.55) 50%, transparent 96%);
  mask-image: linear-gradient(180deg, black 0%, rgba(0, 0, 0, 0.55) 50%, transparent 96%);
}
.promo-wave {
  position: absolute;
  inset: 0;
  width: 200%;
  animation: promo-wave-drift linear infinite;
}
.promo-wave svg { width: 100%; height: 100%; display: block; }
/* Слои как в логотипе, но полупрозрачные — фон, сквозь который идёт страница. */
.promo-wave--soft { animation-duration: 30s; }
.promo-wave--soft path { fill: var(--p-accent-soft); opacity: 0.22; }
.promo-wave--mid { animation-duration: 20s; animation-delay: -6s; top: 12%; }
.promo-wave--mid path { fill: var(--p-accent); opacity: 0.18; }
.promo-wave--deep { animation-duration: 13s; animation-delay: -3s; top: 26%; }
.promo-wave--deep path { fill: var(--p-accent-deep); opacity: 0.24; }
@keyframes promo-wave-drift {
  from { transform: translateX(0); }
  to { transform: translateX(-50%); }
}

/* ── Фичи ── */
.promo-features {
  position: relative;
  z-index: 1;
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 16px;
  padding: 0 clamp(16px, 5vw, 48px);
  max-width: 1120px;
  margin: 0 auto;
  width: 100%;
}
.promo-card {
  background: var(--p-surface);
  border-radius: 22px;
  padding: 22px;
  box-shadow: 0 8px 24px rgba(76, 46, 178, 0.08);
  transition: transform 0.18s, box-shadow 0.18s;
}
.promo-card:hover { transform: translateY(-4px); box-shadow: var(--p-shadow); }
.promo-card-icon {
  width: 46px;
  height: 46px;
  border-radius: 15px;
  display: grid;
  place-items: center;
  font-size: 24px;
  margin-bottom: 14px;
  background: rgba(123, 92, 250, 0.14);
  color: var(--p-accent-deep);
}
.promo-card h3 { margin: 0 0 6px; font-size: 16.5px; font-weight: 800; }
.promo-card p { margin: 0; font-size: 13.5px; line-height: 1.5; color: var(--p-dim); }

/* ── Шаги ── */
.promo-steps {
  position: relative;
  z-index: 1;
  padding: clamp(56px, 10vh, 110px) clamp(16px, 5vw, 48px) 0;
  max-width: 1120px;
  margin: 0 auto;
  width: 100%;
  text-align: center;
}
.promo-h2 {
  margin: 0 0 28px;
  font-size: clamp(24px, 4.4vw, 36px);
  font-weight: 850;
  letter-spacing: -0.015em;
}
.promo-steps-row { display: grid; grid-template-columns: repeat(3, 1fr); gap: 16px; }
.promo-step {
  background: var(--p-surface);
  border-radius: 22px;
  padding: 24px 20px;
  box-shadow: 0 8px 24px rgba(76, 46, 178, 0.08);
}
.promo-step-num {
  display: inline-grid;
  place-items: center;
  width: 38px;
  height: 38px;
  border-radius: 50%;
  background: linear-gradient(135deg, var(--p-accent), var(--p-accent-deep));
  color: #ffffff;
  font-weight: 800;
  margin-bottom: 12px;
}
.promo-step h4 { margin: 0 0 6px; font-size: 15.5px; font-weight: 800; }
.promo-step p { margin: 0; font-size: 13.5px; color: var(--p-dim); line-height: 1.5; }

/* ── Финал ── */
.promo-final {
  position: relative;
  z-index: 1;
  text-align: center;
  padding: clamp(56px, 10vh, 120px) clamp(16px, 5vw, 48px);
}
.promo-final-card {
  max-width: 720px;
  margin: 0 auto;
  padding: clamp(32px, 6vw, 56px) clamp(20px, 5vw, 48px);
  border-radius: 28px;
  background:
    radial-gradient(130% 160% at 50% -35%, rgba(123, 92, 250, 0.22) 0%, transparent 65%),
    var(--p-surface);
  box-shadow: var(--p-shadow);
}
.promo-final .promo-sub { margin-bottom: 24px; }
/* «И грувик тоже.» — всегда с новой строки, на любых экранах. */
.promo-final-tail { display: block; }
.promo-final-pets { display: flex; justify-content: center; gap: 14px; margin-bottom: 18px; }
.promo-final-pet {
  font-size: 34px;
  line-height: 1;
  display: inline-block;
  animation: promo-float 4.5s ease-in-out infinite;
}

.promo-foot {
  position: relative;
  z-index: 1;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 18px clamp(16px, 5vw, 48px) calc(18px + env(safe-area-inset-bottom, 0px));
  border-top: 1px solid rgba(123, 92, 250, 0.16);
  font-size: 12.5px;
  color: var(--p-dim);
}
.promo-foot a { color: var(--p-dim); }
.promo-foot a:hover { color: var(--p-accent-deep); }

/* ── Reveal-анимация ── */
.reveal { opacity: 0; transform: translateY(22px); transition: opacity 0.55s ease, transform 0.55s ease; }
.reveal.is-visible { opacity: 1; transform: none; }

@media (prefers-reduced-motion: reduce) {
  .promo-shape, .promo-float, .promo-final-pet, .promo-wave { animation: none !important; }
  .reveal { transition: none; transform: none; opacity: 1; }
}

@media (max-width: 900px) {
  .promo-features { grid-template-columns: repeat(2, 1fr); }
  .promo-steps-row { grid-template-columns: 1fr; }
  .promo-float--stat, .promo-float--team { display: none; }
  .promo-float--badge { position: static; margin: 0 auto 18px; animation: none; }
  .promo-hero { display: flex; flex-direction: column; align-items: center; }
}
@media (max-width: 600px) {
  .promo-features { grid-template-columns: 1fr; }
  .promo-btn--lg { width: 100%; }
  .promo-actions { width: 100%; }
  .promo-foot { flex-direction: column; text-align: center; }
  .promo-final-pets { gap: 10px; }
  .promo-final-pet { font-size: 28px; }
  .promo-shape--cursor { display: none; }
}
</style>
