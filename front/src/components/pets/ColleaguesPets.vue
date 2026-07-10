<template>
  <section class="cp-wrap">
    <header class="cp-head">
      <h3 class="cp-title">
        <span class="material-symbols-outlined">pets</span>
        Питомцы коллег
      </h3>
      <span class="cp-hint">Тап по питомцу — погладить ладошкой (<KudosCoin class="cp-hint-coin" /> 1 за поглаживание, до 3 в день)</span>
    </header>

    <div v-if="pets.zoo.length" class="cp-grid">
      <div v-for="p in pets.zoo" :key="p.user_id" class="cp-cell">
        <component
          :is="p.user_id === pets.myId ? 'div' : 'button'"
          class="cp-card"
          :class="{
            mine: p.user_id === pets.myId,
            stroked: isStrokedOut(p),
            disabled: p.user_id !== pets.myId && (isAway(p) || (!canAfford && !isStrokedOut(p))),
          }"
          :type="p.user_id === pets.myId ? undefined : 'button'"
          :disabled="p.user_id === pets.myId ? undefined : (isAway(p) || !canAfford || isStrokedOut(p))"
          :aria-label="p.user_id === pets.myId ? undefined : `Погладить питомца «${p.name}»`"
          @click="stroke(p)"
        >
          <div class="cp-figure" :class="{ sick: p.sick, pulse: pulsing[p.user_id] }">
            <span class="cp-emoji"><EmojiGlyph :char="petEmoji(p)" /></span>
            <span v-if="p.hat" class="cp-hat"><EmojiGlyph :char="shopItemEmoji({ kind: 'accessory', key: p.hat })" /></span>
            <span v-if="p.sick" class="cp-sick" title="Болеет">🤒</span>
            <span v-else-if="isAway(p)" class="cp-sick" title="В приключении">🧭</span>
            <span
              v-if="(p.generation || 1) >= 2"
              class="cp-gen"
              :title="`${p.generation}-е поколение`"
            >🌟{{ p.generation }}</span>
          </div>
          <span class="cp-name">{{ p.name }}</span>
          <span class="cp-owner">{{ firstName(p.user?.fio) }}</span>
          <span class="cp-stage">{{ PET_STAGES[p.stage] || '' }}</span>

          <span v-if="p.user_id === pets.myId" class="cp-tag">Ваш питомец</span>
          <span v-else-if="isAway(p)" class="cp-tag">🧭 В приключении</span>
          <span v-else-if="isStrokedOut(p)" class="cp-tag done">
            <span class="material-symbols-outlined">check</span> Поглажен
          </span>
          <span v-else class="cp-tag action">
            <span class="material-symbols-outlined">volunteer_activism</span>
            Погладить&nbsp;<KudosCoin class="cp-tag-coin" />&nbsp;1
          </span>
        </component>
        <!-- Домик коллеги (просмотр) — сиблинг карточки, как и удаление. -->
        <button
          v-if="p.user_id !== pets.myId && p.house_placed?.length"
          class="cp-house"
          type="button"
          :aria-label="`Посмотреть домик «${p.name}»`"
          title="Посмотреть домик"
          @click.stop="houseTarget = p"
        >🏠</button>
        <!-- Удаление питомца сотрудника — только администратор компании;
             кнопка — сиблинг карточки (карточка сама <button>). -->
        <button
          v-if="isAdmin() && p.user_id !== pets.myId"
          class="cp-delete"
          type="button"
          :aria-label="`Удалить питомца «${p.name}»`"
          title="Удалить питомца"
          @click.stop="deleteTarget = p"
        >
          <span class="material-symbols-outlined">delete</span>
        </button>
      </div>
    </div>

    <EmptyState
      v-else
      icon="pets"
      title="Пока пусто"
      subtitle="Питомцы коллег появятся здесь, как только кто-то войдёт в раздел"
    />

    <StrokeMiniGame
      v-if="strokeTarget"
      :pet="strokeTarget"
      :initial-strokes="strokeTarget.strokes_today || 0"
      @exhausted="strokedOut[strokeTarget.user_id] = true"
      @close="strokeTarget = null"
    />

    <ConfirmDialog
      :visible="!!deleteTarget"
      header="Удалить питомца?"
      :message="deleteMessage"
      confirm-label="Удалить"
      danger-confirm
      @confirm="confirmDelete"
      @cancel="deleteTarget = null"
    />

    <!-- Домик коллеги — режим просмотра (guestPet). -->
    <PetHouseDialog
      :model-value="!!houseTarget"
      :guest-pet="houseTarget"
      @update:model-value="(v) => { if (!v) houseTarget = null }"
    />
  </section>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import EmojiGlyph from '@/components/common/EmojiGlyph.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import KudosCoin from '@/components/pets/KudosCoin.vue'
import StrokeMiniGame from '@/components/pets/StrokeMiniGame.vue'
import PetHouseDialog from '@/components/pets/PetHouseDialog.vue'
import { usePermission } from '@/composables/usePermission.js'
import { usePetsStore } from '@/stores/pets.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { petEmoji, shopItemEmoji, PET_STAGES } from '@/utils/pets.js'

const STROKE_COST = 2 // = domain.StrokeCost в petsvc

const pets = usePetsStore()
const notify = useNotificationsStore()
const { isAdmin } = usePermission()

const canAfford = computed(() => (pets.pet?.kudos ?? 0) >= STROKE_COST)

// Тап по карточке открывает мини-игру «трение ладошкой» (StrokeMiniGame) —
// она сама списывает кудосы за каждый цикл и сообщает об исчерпании лимита.
const strokeTarget = ref(null)
const houseTarget = ref(null)
const pulsing = reactive({})
const strokedOut = reactive({})

// «Наглажен до завтра»: серверный счётчик strokes_today (переживает
// перезагрузку) ИЛИ исчерпание в текущей сессии.
const STROKE_DAILY_MAX = 3 // = domain.StrokeDailyMaxPerPet
function isStrokedOut(p) {
  return !!strokedOut[p.user_id] || (p.strokes_today ?? 0) >= STROKE_DAILY_MAX
}

// Питомец коллеги в приключении — поглаживание недоступно, пока не вернётся.
function isAway(p) {
  return !!(p?.adventure_until && new Date(p.adventure_until) > new Date())
}

function firstName(fio) {
  if (!fio) return ''
  const parts = fio.split(' ')
  return parts.length > 1 ? `${parts[0]} ${parts[1]}` : fio
}

function stroke(p) {
  if (p.user_id === pets.myId || isStrokedOut(p) || isAway(p)) return
  if (!canAfford.value) return
  pulsing[p.user_id] = true
  setTimeout(() => { pulsing[p.user_id] = false }, 450)
  strokeTarget.value = p
}

// ── Удаление питомца сотрудника (администратор) ────────────────────

const deleteTarget = ref(null)
const deleteMessage = computed(() => {
  const p = deleteTarget.value
  if (!p) return ''
  return `Питомец «${p.name}» (${firstName(p.user?.fio)}) будет удалён вместе с прогрессом. У сотрудника появится новый — с нуля.`
})

async function confirmDelete() {
  const p = deleteTarget.value
  deleteTarget.value = null
  if (!p) return
  try {
    await pets.deleteColleaguePet(p.user_id)
    notify.success(`Питомец «${p.name}» удалён`)
  } catch (e) {
    notify.warn(e?.message || 'Не удалось удалить питомца')
  }
}

onMounted(() => {
  if (!pets.zoo.length) pets.fetchZoo().catch(() => {})
})
</script>

<style scoped>
.cp-wrap { display: flex; flex-direction: column; gap: 12px; }
.cp-head { display: flex; flex-direction: column; gap: 2px; }
.cp-title {
  margin: 0;
  font-size: 16px;
  font-weight: 700;
  display: flex;
  align-items: center;
  gap: 8px;
}
.cp-title .material-symbols-outlined { font-size: 20px; color: var(--color-primary); }
.cp-hint { font-size: 12.5px; color: var(--color-text-dim); }
.cp-hint-coin { font-size: 12px; }

.cp-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(140px, 1fr));
  gap: 14px;
}
.cp-cell { position: relative; display: flex; }
.cp-card {
  flex: 1;
  width: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  padding: 14px 10px;
  border-radius: var(--radius-lg);
  background: var(--acrylic-card-bg);
  border: 1px solid var(--acrylic-border);
  font: inherit;
  color: var(--color-text);
  cursor: default;
  text-align: center;
}

/* Кнопка домика — сиблинг карточки, слева сверху (справа — удаление). */
.cp-house {
  position: absolute;
  top: 6px;
  left: 6px;
  width: 28px;
  height: 28px;
  border: none;
  border-radius: 50%;
  background: none;
  font-size: 14px;
  display: grid;
  place-items: center;
  cursor: pointer;
  transition: background 0.15s, transform 0.1s;
}
.cp-house:hover { background: var(--color-secondary-container); transform: scale(1.1); }

/* Бейдж поколения — на кружке питомца. */
.cp-gen {
  position: absolute;
  top: -6px;
  left: -8px;
  font-size: 11px;
  font-weight: 800;
  padding: 1px 6px;
  border-radius: var(--radius-full);
  background: linear-gradient(120deg,
    color-mix(in oklch, var(--color-tertiary-container) 90%, transparent),
    color-mix(in oklch, var(--color-primary-container) 90%, transparent));
  color: var(--color-on-tertiary-container);
  box-shadow: var(--shadow-sm);
}

/* Кнопка удаления (администратор) — поверх карточки, сиблинг, не вложена. */
.cp-delete {
  position: absolute;
  top: 6px;
  right: 6px;
  width: 28px;
  height: 28px;
  border: none;
  border-radius: 50%;
  background: none;
  color: var(--color-text-dim);
  display: grid;
  place-items: center;
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
}
.cp-delete .material-symbols-outlined { font-size: 17px; }
.cp-delete:hover { background: var(--color-error-container); color: var(--color-error); }
button.cp-card { cursor: pointer; transition: transform 0.1s, border-color 0.15s; }
button.cp-card:hover:not(:disabled) { border-color: color-mix(in oklch, var(--color-primary) 45%, var(--color-outline-dim)); }
button.cp-card:active:not(:disabled) { transform: scale(0.97); }
button.cp-card:disabled { cursor: default; }
.cp-card.disabled { opacity: 0.6; }
.cp-card.mine { border-color: color-mix(in oklch, var(--color-primary) 45%, var(--color-outline-dim)); }

.cp-figure {
  position: relative;
  width: 64px;
  height: 64px;
  border-radius: 50%;
  background: var(--color-primary-container);
  display: grid;
  place-items: center;
  margin-bottom: 4px;
}
.cp-figure.pulse { animation: cp-pulse 0.45s cubic-bezier(0.34, 1.56, 0.64, 1); }
@keyframes cp-pulse {
  0% { transform: scale(1); }
  40% { transform: scale(1.14) rotate(-3deg); }
  100% { transform: scale(1); }
}
@media (prefers-reduced-motion: reduce) { .cp-figure.pulse { animation: none; } }
.cp-emoji { font-size: 32px; line-height: 1; }
.cp-figure.sick .cp-emoji { filter: grayscale(0.55) brightness(0.92); }
.cp-hat { position: absolute; top: -8px; right: -2px; font-size: 18px; transform: rotate(12deg); }
.cp-sick { position: absolute; bottom: -4px; left: -4px; font-size: 16px; }

.cp-name {
  font-size: 13px;
  font-weight: 700;
  max-width: 100%;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.cp-owner {
  font-size: 11px;
  color: var(--color-text-dim);
  max-width: 100%;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.cp-stage {
  font-size: 10.5px;
  font-weight: 700;
  padding: 1px 8px;
  border-radius: var(--radius-full);
  background: var(--color-tertiary-container);
  color: var(--color-on-tertiary-container);
  margin-top: 2px;
}

.cp-tag {
  margin-top: 8px;
  display: inline-flex;
  align-items: center;
  gap: 5px;
  font-size: 11.5px;
  font-weight: 700;
  color: var(--color-text-dim);
  min-height: 28px;
}
.cp-tag .material-symbols-outlined { font-size: 14px; }
.cp-tag-coin { font-size: 11px; }
.cp-tag.action {
  border-radius: var(--radius-full);
  background: var(--color-secondary-container);
  color: var(--color-on-secondary-container);
  padding: 6px 12px;
}
.cp-tag.done { color: var(--color-success); }
</style>
