<template>
  <Teleport to="body">
    <div class="pdm-overlay" @click.self="close">
      <div class="pdm-panel">
        <div class="pdm-cover" aria-hidden="true"></div>
        <button class="pdm-close" type="button" aria-label="Закрыть" @click="close">
          <span class="material-symbols-outlined">close</span>
        </button>

        <div class="pdm-scroll">
        <div class="pdm-hero">
          <div class="pdm-figure" :class="{ sick: pet?.sick, bounce: justActed }">
            <span class="pdm-emoji"><EmojiGlyph :char="petEmoji(pet)" /></span>
            <span v-if="hatEmoji" class="pdm-hat"><EmojiGlyph :char="hatEmoji" /></span>
            <span v-if="pet?.sick" class="pdm-sick-badge" :title="ailment.title">
              <EmojiGlyph :char="ailment.emoji" />
            </span>
          </div>

          <div class="pdm-name-row">
            <template v-if="renaming">
              <input
                ref="nameInput"
                v-model.trim="newName"
                class="pdm-name-input"
                maxlength="50"
                @keyup.enter="saveName"
                @keyup.esc="renaming = false"
              />
              <button class="pdm-icon-btn" type="button" @click="saveName" aria-label="Сохранить имя">
                <span class="material-symbols-outlined">check</span>
              </button>
            </template>
            <template v-else>
              <h2 class="pdm-name">{{ pet?.name }}</h2>
              <button class="pdm-icon-btn" type="button" @click="startRename" aria-label="Переименовать">
                <span class="material-symbols-outlined">edit</span>
              </button>
            </template>
          </div>
          <p class="pdm-subtitle">
            {{ stageTitle }}<template v-if="speciesTitle"> · {{ speciesTitle }}</template>
          </p>
          <p v-if="personality" class="pdm-personality">{{ personality.emoji }} {{ personality.title }}</p>
          <p class="pdm-mood" :title="`Настроение множит XP за работу ×${pet?.mood_factor ?? 1}`">
            {{ moodEmoji(pet) }} {{ pet?.mood_title || moodTitle(pet?.mood ?? 100) }}
            <span class="pdm-mood-factor">XP ×{{ pet?.mood_factor ?? 1 }}</span>
          </p>
        </div>

        <div class="pdm-chips">
          <span class="pdm-chip kudos"><KudosCoin class="pdm-chip-emoji" /> {{ pet?.kudos ?? 0 }}</span>
          <span class="pdm-chip">
            <span class="material-symbols-outlined">local_fire_department</span>
            {{ pet?.feed_streak ?? 0 }} дн.
          </span>
          <span v-if="generation >= 2" class="pdm-chip gen" title="Поколение питомца">
            🌟 {{ generation }}-е поколение
          </span>
        </div>

        <!-- Потребности: пустая шкала укладывает питомца в свою болезнь,
             поэтому они всегда на виду — до, а не после беды. -->
        <div class="pdm-needs">
          <NeedBars :needs="pet?.needs" />
        </div>

        <div v-if="pet?.sick" class="pdm-sick-block" :class="{ urgent: runawaySoon }">
          <div class="pdm-sick-head">
            <span class="pdm-sick-emoji"><EmojiGlyph :char="ailment.emoji" /></span>
            {{ pet.ailment_title || ailment.title }}
          </div>
          <p class="pdm-sick-cause">{{ pet.ailment_hint }}</p>
          <div class="pdm-sick-progress">
            <span
              v-for="i in pet.recovery_target"
              :key="i"
              class="pdm-sick-dot"
              :class="{ filled: i <= pet.recovery }"
            ></span>
            <span class="pdm-sick-count">{{ pet.recovery }}/{{ pet.recovery_target }}</span>
          </div>
          <p class="pdm-sick-hint">Лечит: {{ ailment.cure }}</p>
          <!-- Заброшенный питомец уходит: предупреждение — последний шанс. -->
          <p v-if="runawaySoon" class="pdm-runaway-warn">
            <span class="material-symbols-outlined">directions_run</span>
            {{ runawayText }}
          </p>
        </div>

        <div v-else class="pdm-xp">
          <div class="pdm-xp-meta">
            <span>{{ atMaxStage ? 'Максимальная форма — доступно перерождение' : 'До эволюции' }}</span>
            <span v-if="pet?.next_stage_xp">{{ pet.xp }} / {{ pet.next_stage_xp }} XP</span>
          </div>
          <div class="pdm-xp-bar"><div class="pdm-xp-fill" :style="{ width: xpPercent + '%' }"></div></div>
        </div>

        <div v-if="quest" class="pdm-quest" :class="{ done: quest.done, claimed: quest.claimed }">
          <div class="pdm-quest-head">
            <span class="material-symbols-outlined">
              {{ quest.claimed ? 'check_circle' : (quest.done ? 'rocket_launch' : 'flag') }}
            </span>
            <span class="pdm-quest-title">{{ quest.title }}</span>
            <span class="pdm-quest-reward">+{{ quest.reward }} <KudosCoin /></span>
          </div>
          <div class="pdm-quest-bar"><div class="pdm-quest-fill" :style="{ width: questPercent + '%' }"></div></div>
          <div class="pdm-quest-meta">
            <span>{{ quest.progress }} / {{ quest.target }} {{ quest.unit }}</span>
            <button
              v-if="quest.done && !quest.claimed"
              class="pdm-quest-claim"
              type="button"
              :disabled="claiming"
              @click="claim"
            >Забрать награду</button>
            <span v-else-if="quest.claimed" class="pdm-quest-claimed">Награда получена</span>
            <span v-else class="pdm-quest-hint">{{ quest.hint }}</span>
          </div>
        </div>

        <div class="pdm-tabs" role="tablist">
          <button class="pdm-tab" :class="{ active: tab === 'actions' }" type="button" @click="tab = 'actions'">
            Действия
          </button>
          <button class="pdm-tab" :class="{ active: tab === 'history' }" type="button" @click="openHistory">
            История
          </button>
        </div>

        <div v-if="tab === 'actions'" class="pdm-actions">
          <!-- Отпуск хозяина: питомец отдыхает, уход недоступен, показатели заморожены -->
          <div v-if="pet?.on_vacation" class="pdm-adventure">
            <span class="pdm-adventure-emoji">🏖️</span>
            <div class="pdm-adventure-text">
              <strong>В отпуске вместе с хозяином</strong>
              <span>показатели заморожены и не тают — вернётесь, и всё продолжится</span>
            </div>
          </div>
          <div v-else-if="onAdventure" class="pdm-adventure">
            <span class="pdm-adventure-emoji">🧭</span>
            <div class="pdm-adventure-text">
              <strong>Гуляет {{ pet.adventure_place }}</strong>
              <span>вернётся ~в {{ adventureReturnTime }}</span>
            </div>
            <button
              class="pdm-recall-btn"
              type="button"
              :disabled="recallSending || (pet?.kudos ?? 0) < RECALL_COST"
              :title="(pet?.kudos ?? 0) < RECALL_COST ? `Нужно ${RECALL_COST} кудосов` : 'Питомец вернётся сразу, но без награды за поход'"
              @click="recallPet"
            >
              Вернуть сейчас
              <span class="pdm-recall-cost"><KudosCoin /> {{ RECALL_COST }}</span>
            </button>
          </div>
          <template v-else>
            <button class="pdm-action-btn" type="button" :disabled="!canFeed" @click="activeGame = 'feed'">
              <span class="pdm-action-emoji">{{ pet?.sick ? '🍲' : '🥕' }}</span>
              <span>{{ pet?.sick ? 'Дать бульон' : 'Покормить' }}</span>
              <span class="pdm-action-cost"><KudosCoin /> {{ feedCost }}</span>
            </button>
            <button class="pdm-action-btn" type="button" :disabled="!canWalk" @click="activeGame = 'walk'">
              <span class="pdm-action-emoji">🚶</span>
              <span>Погулять</span>
              <span class="pdm-action-cost"><KudosCoin /> {{ WALK_COST }}</span>
            </button>
            <!-- Сон — единственное бесплатное действие ухода: энергия не
                 должна упираться в кошелёк. -->
            <button class="pdm-action-btn" type="button" :disabled="!sleepsLeft" @click="doSleep">
              <span class="pdm-action-emoji">😴</span>
              <span>Уложить спать</span>
              <span class="pdm-action-cost">{{ sleepsLeft ? 'бесплатно' : 'выспался' }}</span>
            </button>
            <button class="pdm-action-btn" type="button" :disabled="!canBath" @click="activeGame = 'bath'">
              <span class="pdm-action-emoji">🛁</span>
              <span>Искупать</span>
              <span class="pdm-action-cost"><KudosCoin /> {{ BATH_COST }}</span>
            </button>
            <button v-if="pet?.sick" class="pdm-action-btn" type="button" :disabled="!canHeal" @click="activeGame = 'heal'">
              <span class="pdm-action-emoji">💊</span>
              <span>Полечить</span>
              <span class="pdm-action-cost"><KudosCoin /> {{ HEAL_COST }}</span>
            </button>
            <button
              v-if="!pet?.sick"
              class="pdm-action-btn"
              type="button"
              :disabled="adventureSending"
              @click="sendAdventure"
            >
              <span class="pdm-action-emoji">🧭</span>
              <span>Отправить в приключение</span>
              <span class="pdm-action-cost">бесплатно</span>
            </button>
            <!-- Перерождение — только «Легенде»: поколение +1, рост заново,
                 богатство (кудосы/гардероб/виды/домик) остаётся. -->
            <button
              v-if="atMaxStage && !pet?.sick"
              class="pdm-action-btn prestige"
              type="button"
              @click="confirmPrestige = true"
            >
              <span class="pdm-action-emoji">🌟</span>
              <span>Переродиться — поколение {{ generation + 1 }}</span>
              <span class="pdm-action-cost">{{ nextPrestigeSpecies ? 'новый вид' : 'бесплатно' }}</span>
            </button>
          </template>

          <button class="pdm-action-btn" type="button" @click="houseOpen = true">
            <span class="pdm-action-emoji">🏠</span>
            <span>Домик</span>
            <span v-if="placedCount" class="pdm-action-cost">{{ placedCount }} предм.</span>
          </button>

          <div v-if="ownedItems.length" class="pdm-closet">
            <button
              v-for="item in ownedItems"
              :key="item"
              class="pdm-closet-item"
              :class="{ active: pet.hat === item }"
              type="button"
              :title="shopItemTitle({ kind: 'accessory', key: item })"
              @click="toggleEquip(item)"
            ><EmojiGlyph :char="shopItemEmoji({ kind: 'accessory', key: item })" /></button>
          </div>
        </div>

        <div v-else class="pdm-history">
          <div v-if="historyLoading" class="pdm-history-loading">Загружаем историю…</div>
          <EmptyState
            v-else-if="!pets.activityLog.length"
            icon="history"
            title="Пока пусто"
            subtitle="Действия с питомцем появятся здесь"
          />
          <ul v-else class="pdm-history-list">
            <li v-for="(e, i) in pets.activityLog" :key="e.created_at + '-' + i" class="pdm-history-item">
              <span class="material-symbols-outlined">{{ activityIcon(e) }}</span>
              <span class="pdm-history-text">{{ activityText(e) }}</span>
              <span class="pdm-history-time">{{ fmtTime(e.created_at) }}</span>
            </li>
          </ul>
        </div>

        </div>

        <!-- Закреплённый низ: /pets теперь единая страница (магазин, коллеги,
             рейтинг) — одна ссылка вместо трёх. -->
        <div class="pdm-links">
          <button class="pdm-link-btn" type="button" @click="gotoPets">
            <span class="material-symbols-outlined">pets</span> Перейти к грувикам
          </button>
        </div>
      </div>
    </div>

    <FeedMiniGame v-if="activeGame === 'feed'" :pet="pet" @success="onFeedSuccess" @close="activeGame = null" />
    <WalkMiniGame v-if="activeGame === 'walk'" :pet="pet" @success="onWalkSuccess" @close="activeGame = null" />
    <HealMiniGame v-if="activeGame === 'heal'" @success="onHealSuccess" @close="activeGame = null" />
    <BathMiniGame
      v-if="activeGame === 'bath'"
      :pet="pet"
      :cost="BATH_COST"
      @success="onBathSuccess"
      @close="activeGame = null"
    />

    <PetHouseDialog v-model="houseOpen" />
    <!-- above-pet-modal: конфирм должен всплыть поверх этой модалки (10700). -->
    <ConfirmDialog
      :visible="confirmPrestige"
      header="Переродиться?"
      :message="prestigeMessage"
      confirm-label="Переродиться"
      mask-class="above-pet-modal"
      dialog-class="above-pet-modal"
      @confirm="doPrestige"
      @cancel="confirmPrestige = false"
    />
  </Teleport>
</template>

<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import EmojiGlyph from '@/components/common/EmojiGlyph.vue'
import { useRouter } from 'vue-router'
import EmptyState from '@/components/common/EmptyState.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import KudosCoin from '@/components/pets/KudosCoin.vue'
import FeedMiniGame from '@/components/pets/FeedMiniGame.vue'
import WalkMiniGame from '@/components/pets/WalkMiniGame.vue'
import HealMiniGame from '@/components/pets/HealMiniGame.vue'
import BathMiniGame from '@/components/pets/BathMiniGame.vue'
import NeedBars from '@/components/pets/NeedBars.vue'
import PetHouseDialog from '@/components/pets/PetHouseDialog.vue'
import { usePetsStore } from '@/stores/pets.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import {
  petEmoji, PET_STAGES, PET_SPECIES, PERSONALITIES, PRESTIGE_SPECIES,
  shopItemEmoji, shopItemTitle, activityIcon, activityText,
  ailmentMeta, moodEmoji, moodTitle,
} from '@/utils/pets.js'

const props = defineProps({
  // 'feed' | 'walk' | 'heal' | 'bath' | null
  initialAction: { type: String, default: null },
})
const emit = defineEmits(['close'])

const FEED_COST_NORMAL = 10
const FEED_COST_SICK = 1 // «бульон» больному — зеркало domain.SickFeedCost
const WALK_COST = 15
const HEAL_COST = 25
const BATH_COST = 12 // = domain.BathCost

const pets = usePetsStore()
const notify = useNotificationsStore()
const router = useRouter()

const pet = computed(() => pets.pet)
const maxStage = PET_STAGES.length - 1
const atMaxStage = computed(() => (pet.value?.stage ?? 0) >= maxStage)

const tab = ref('actions')
const activeGame = ref(null)
const historyLoading = ref(false)
const justActed = ref(false)
const claiming = ref(false)
const renaming = ref(false)
const newName = ref('')
const nameInput = ref(null)

const stageTitle = computed(() => PET_STAGES[pet.value?.stage] || '')
const speciesTitle = computed(() => (pet.value?.stage >= 2 ? PET_SPECIES[pet.value.species]?.title : ''))
const hatEmoji = computed(() => (pet.value?.hat ? shopItemEmoji({ kind: 'accessory', key: pet.value.hat }) : null))
const personality = computed(() => (pet.value?.personality ? PERSONALITIES[pet.value.personality] : null))
const ownedItems = computed(() => pet.value?.accessories || [])

const xpPercent = computed(() => {
  if (!pet.value?.next_stage_xp) return 100
  return Math.min(100, Math.round((pet.value.xp / pet.value.next_stage_xp) * 100))
})

const quest = computed(() => pet.value?.quest || null)
const questPercent = computed(() => {
  const q = quest.value
  if (!q || !q.target) return 0
  return Math.min(100, Math.round((q.progress / q.target) * 100))
})

const feedCost = computed(() => (pet.value?.sick ? FEED_COST_SICK : FEED_COST_NORMAL))
const canFeed = computed(() => (pet.value?.kudos ?? 0) >= feedCost.value)
const canWalk = computed(() => (pet.value?.kudos ?? 0) >= WALK_COST)
const canHeal = computed(() => (pet.value?.kudos ?? 0) >= HEAL_COST)
// Остатки дневных лимитов приходят с бэка (feeds_left/sleeps_left/baths_left):
// кнопка гаснет до отказа, а не после него.
const sleepsLeft = computed(() => pet.value?.sleeps_left ?? 1)
const bathsLeft = computed(() => pet.value?.baths_left ?? 1)
const canBath = computed(() => (pet.value?.kudos ?? 0) >= BATH_COST && bathsLeft.value > 0)

// Болезнь: вид, подпись, рецепт (бэк присылает свои тексты, каталог — фолбэк
// для зоопарка, где их нет).
const ailment = computed(() => ailmentMeta(pet.value?.ailment))

// Предупреждение о побеге: сколько дней болезни осталось до ухода питомца.
const runawaySoon = computed(() => pet.value?.runaway_in_days != null)
const runawayText = computed(() => {
  const days = pet.value?.runaway_in_days ?? 0
  if (days <= 0) return `${pet.value?.name || 'Грувик'} вот-вот сбежит — лечите немедленно!`
  return `Если не вылечить, ${pet.value?.name || 'грувик'} сбежит через ${days} ${plural(days)}: `
    + 'прогресс обнулится, дома останется яйцо.'
})

function plural(n) {
  const mod10 = n % 10
  const mod100 = n % 100
  if (mod10 === 1 && mod100 !== 11) return 'день'
  if (mod10 >= 2 && mod10 <= 4 && (mod100 < 12 || mod100 > 14)) return 'дня'
  return 'дней'
}

const onAdventure = computed(() => {
  const until = pet.value?.adventure_until
  return !!until && new Date(until) > new Date()
})
const adventureReturnTime = computed(() => {
  const until = pet.value?.adventure_until
  if (!until) return ''
  return new Date(until).toLocaleTimeString('ru-RU', { hour: '2-digit', minute: '2-digit' })
})
const adventureSending = ref(false)

// ── Престиж и домик ─────────────────────────────────────────────
const generation = computed(() => pet.value?.generation || 1)
const placedCount = computed(() => pet.value?.house_placed?.length || 0)
const houseOpen = ref(false)
const confirmPrestige = ref(false)
const nextPrestigeSpecies = computed(() => {
  const key = PRESTIGE_SPECIES[generation.value + 1]
  return key ? PET_SPECIES[key] : null
})
const prestigeMessage = computed(() => {
  const base = `${pet.value?.name || 'Питомец'} начнёт рост заново яйцом ${generation.value + 1}-го поколения. ` +
    'Кудосы, гардероб, купленные виды и домик сохранятся.'
  return nextPrestigeSpecies.value
    ? `${base} Откроется эксклюзивный вид «${nextPrestigeSpecies.value.title}» ${nextPrestigeSpecies.value.emoji}!`
    : base
})

async function doPrestige() {
  confirmPrestige.value = false
  try {
    const res = await pets.prestigePet()
    pulse()
    notify.success(`Перерождение! Поколение ${res?.generation ?? generation.value}`)
  } catch (e) {
    notify.warn(e?.message || 'Перерождение не удалось')
  }
}

function close() {
  emit('close')
}

function onKeydown(e) {
  if (e.key === 'Escape' && !activeGame.value) close()
}

onMounted(async () => {
  document.addEventListener('keydown', onKeydown)
  if (!pets.pet) await pets.fetchPet().catch(() => {})
  if (['feed', 'walk', 'heal', 'bath'].includes(props.initialAction)) {
    activeGame.value = props.initialAction
  }
})
onBeforeUnmount(() => document.removeEventListener('keydown', onKeydown))

function pulse() {
  justActed.value = true
  setTimeout(() => { justActed.value = false }, 650)
}

async function onFeedSuccess() {
  activeGame.value = null
  try {
    await pets.feedPet()
    pulse()
    notify.success('Питомец покормлен')
  } catch (e) {
    notify.warn(e?.message || 'Покормить не получилось')
  }
}

async function onWalkSuccess() {
  activeGame.value = null
  try {
    const res = await pets.walkPet()
    pulse()
    notify.success(res?.recovered ? 'Прогулка помогла — питомцу лучше' : 'Прогулка удалась')
  } catch (e) {
    notify.warn(e?.message || 'Прогулка не получилась')
  }
}

async function onHealSuccess() {
  activeGame.value = null
  try {
    const res = await pets.healPet()
    pulse()
    notify.success(res?.recovered ? 'Питомец полностью выздоровел!' : 'Лечение подействовало')
  } catch (e) {
    notify.warn(e?.message || 'Лечение не подействовало')
  }
}

// Сон без мини-игры: он бесплатный и по смыслу пассивный — питомец просто
// отсыпается (мини-игры остаются платным действиям).
const sleeping = ref(false)

async function doSleep() {
  if (sleeping.value) return
  sleeping.value = true
  try {
    const res = await pets.sleepPet()
    pulse()
    notify.success(res?.recovered ? 'Выспался и поправился!' : 'Питомец выспался — энергия восстановлена')
  } catch (e) {
    notify.warn(e?.message || 'Уложить спать не получилось')
  } finally {
    sleeping.value = false
  }
}

async function onBathSuccess() {
  activeGame.value = null
  try {
    const res = await pets.bathPet()
    pulse()
    notify.success(res?.recovered ? 'Отмыт и здоров!' : 'Питомец чист и доволен')
  } catch (e) {
    notify.warn(e?.message || 'Купание не получилось')
  }
}

async function sendAdventure() {
  if (adventureSending.value) return
  adventureSending.value = true
  try {
    await pets.startAdventure()
    pulse()
    notify.success(`Питомец отправился ${pets.pet?.adventure_place || 'в приключение'}`)
  } catch (e) {
    notify.warn(e?.message || 'Не получилось отправить в приключение')
  } finally {
    adventureSending.value = false
  }
}

// Досрочный возврат: питомец дома сразу, но без награды за поход.
const RECALL_COST = 100 // = domain.AdventureRecallCost
const recallSending = ref(false)

async function recallPet() {
  if (recallSending.value) return
  recallSending.value = true
  try {
    const res = await pets.recallAdventure()
    pulse()
    if (!res?.adventure_reward) notify.success('Питомец вернулся домой')
  } catch (e) {
    notify.warn(e?.message || 'Не получилось вернуть питомца')
  } finally {
    recallSending.value = false
  }
}

async function claim() {
  if (claiming.value) return
  claiming.value = true
  try {
    await pets.claimQuest()
    notify.success(`+${quest.value?.reward || 100} кудосов за квест`)
  } catch (e) {
    notify.warn(e?.message || 'Не удалось забрать награду')
  } finally {
    claiming.value = false
  }
}

function startRename() {
  newName.value = pet.value?.name || ''
  renaming.value = true
  nextTick(() => nameInput.value?.focus())
}

async function saveName() {
  if (!newName.value) { renaming.value = false; return }
  try {
    await pets.renamePet(newName.value)
  } catch (e) {
    notify.error(e?.message || 'Не удалось переименовать')
  }
  renaming.value = false
}

async function toggleEquip(item) {
  try {
    await pets.equipItem(pet.value.hat === item ? null : item)
  } catch (e) {
    notify.error(e?.message || 'Не получилось')
  }
}

function openHistory() {
  tab.value = 'history'
  if (!pets.activityLoaded) {
    historyLoading.value = true
    pets.fetchActivityLog().catch(() => {}).finally(() => { historyLoading.value = false })
  }
}

function fmtTime(iso) {
  if (!iso) return ''
  return new Date(iso).toLocaleString('ru-RU', { day: '2-digit', month: '2-digit', hour: '2-digit', minute: '2-digit' })
}

function gotoPets() {
  close()
  router.push('/pets')
}
</script>

<style scoped>
.pdm-overlay {
  position: fixed;
  inset: 0;
  z-index: 10700;
  background: color-mix(in oklch, var(--color-scrim, var(--color-text)) 32%, transparent);
  display: grid;
  place-items: center;
  padding: 16px;
}
/* Акриловая панель — как у остальных модалок (AppDialog): стекло + blur. */
.pdm-panel {
  position: relative;
  width: min(420px, 100%);
  max-height: min(88dvh, 780px);
  overflow: hidden;
  background: var(--acrylic-bg);
  -webkit-backdrop-filter: var(--acrylic-blur);
  backdrop-filter: var(--acrylic-blur);
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-xl);
  box-shadow: var(--shadow-lg);
  display: flex;
  flex-direction: column;
}

/* Скроллится только середина; дети не сжимаются (иначе история вытекала бы
   под закреплённые кнопки). */
.pdm-scroll {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding: 0 20px 16px;
  display: flex;
  flex-direction: column;
  align-items: center;
}
.pdm-scroll > * { flex-shrink: 0; }

/* На мобильном — нижний sheet во всю ширину. */
@media (max-width: 600px) {
  .pdm-overlay { place-items: end center; padding: 0; }
  .pdm-panel {
    width: 100%;
    max-height: 92dvh;
    border-radius: var(--radius-xl) var(--radius-xl) 0 0;
  }
}

/* Градиентная «обложка» — паттерн профиля сотрудника (EmployeesView/ProfileView). */
.pdm-cover {
  position: absolute;
  inset: 0 0 auto;
  height: 150px;
  background:
    radial-gradient(120% 140% at 85% 0%,
      color-mix(in oklch, var(--color-tertiary-container) 40%, transparent) 0%,
      transparent 60%),
    linear-gradient(120deg,
      color-mix(in oklch, var(--color-primary-container) 55%, var(--color-surface)),
      color-mix(in oklch, var(--color-secondary-container) 55%, var(--color-surface)));
  -webkit-mask-image: linear-gradient(to bottom, black 30%, transparent 100%);
  mask-image: linear-gradient(to bottom, black 30%, transparent 100%);
  pointer-events: none;
  border-radius: var(--radius-xl) var(--radius-xl) 0 0;
}
.pdm-close {
  position: absolute;
  top: 12px;
  right: 12px;
  /* Выше .pdm-hero (position:relative, идёт позже в DOM и растянут на всю
     ширину) — иначе перекрывает крестик и он не кликается. */
  z-index: 2;
  width: 44px;
  height: 44px;
  border-radius: 50%;
  border: none;
  background: color-mix(in oklch, var(--color-surface) 70%, transparent);
  color: var(--color-text);
  display: grid;
  place-items: center;
  cursor: pointer;
  -webkit-backdrop-filter: blur(8px);
  backdrop-filter: blur(8px);
}
.pdm-close:hover { background: var(--color-surface); }

.pdm-hero {
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
  padding-top: 28px;
  width: 100%;
}
.pdm-figure {
  position: relative;
  width: 116px;
  height: 116px;
  border-radius: 50%;
  background: var(--color-surface);
  display: grid;
  place-items: center;
  box-shadow: 0 8px 24px color-mix(in oklch, var(--color-primary) 24%, transparent);
  animation: pdm-idle 3.8s ease-in-out infinite;
}
.pdm-figure.bounce { animation: pdm-bounce 0.65s cubic-bezier(0.34, 1.56, 0.64, 1); }
.pdm-figure.sick { animation: none; }
@keyframes pdm-idle {
  0%, 100% { transform: translateY(0) scale(1); }
  50% { transform: translateY(-4px) scale(1.015); }
}
@keyframes pdm-bounce {
  0% { transform: scale(1); }
  35% { transform: scale(1.16) rotate(-4deg); }
  70% { transform: scale(0.95) rotate(2deg); }
  100% { transform: scale(1); }
}
@media (prefers-reduced-motion: reduce) { .pdm-figure { animation: none; } }
.pdm-figure.sick .pdm-emoji { filter: grayscale(0.55) brightness(0.92); }
.pdm-emoji { font-size: 62px; line-height: 1; }
.pdm-hat { position: absolute; top: -14px; right: 4px; font-size: 28px; transform: rotate(12deg); }
.pdm-sick-badge { position: absolute; bottom: -4px; left: -4px; font-size: 24px; }

.pdm-name-row { display: flex; align-items: center; gap: 4px; margin-top: 12px; }
.pdm-name { margin: 0; font-size: 20px; font-weight: 800; }
.pdm-name-input {
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-full);
  padding: 5px 12px;
  font-size: 16px;
  width: 170px;
  background: var(--color-surface);
  color: var(--color-text);
  outline: none;
}
.pdm-icon-btn {
  width: 30px; height: 30px; min-height: 0; border-radius: 50%; border: none; background: none;
  display: grid; place-items: center; cursor: pointer; color: var(--color-text-dim);
}
.pdm-icon-btn:hover { background: var(--color-surface-high); }
.pdm-icon-btn .material-symbols-outlined { font-size: 18px; }
.pdm-subtitle { margin: 2px 0 0; font-size: 13px; color: var(--color-text-dim); }
.pdm-personality {
  margin: 8px 0 0;
  font-size: 12px;
  font-weight: 600;
  padding: 4px 12px;
  border-radius: var(--radius-full);
  background: var(--color-tertiary-container);
  color: var(--color-on-tertiary-container);
}

.pdm-chips { display: flex; gap: 8px; margin-top: 14px; flex-wrap: wrap; justify-content: center; }
.pdm-chip {
  display: inline-flex; align-items: center; gap: 5px;
  padding: 5px 12px; border-radius: var(--radius-full);
  background: var(--color-surface-high); font-size: 13px; font-weight: 600;
}
.pdm-chip.kudos { background: color-mix(in oklch, var(--color-success) 18%, transparent); }
.pdm-chip.gen {
  background: linear-gradient(120deg,
    color-mix(in oklch, var(--color-tertiary-container) 80%, transparent),
    color-mix(in oklch, var(--color-primary-container) 80%, transparent));
  color: var(--color-on-tertiary-container);
}
.pdm-chip .material-symbols-outlined { font-size: 16px; }
.pdm-chip-emoji { font-size: 14px; }

.pdm-needs {
  width: 100%;
  margin-top: 14px;
  padding: 10px 12px;
  border-radius: 14px;
  background: var(--color-surface-low);
}

.pdm-mood {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  margin: 8px 0 0;
  font-size: 12px;
  font-weight: 600;
  color: var(--color-text-dim);
}
.pdm-mood-factor {
  padding: 2px 8px;
  border-radius: var(--radius-full);
  background: var(--color-surface-high);
  font-size: 11px;
  font-variant-numeric: tabular-nums;
}

.pdm-sick-block {
  width: 100%; margin-top: 14px;
  border: 1px dashed color-mix(in oklch, var(--color-error) 45%, transparent);
  border-radius: 14px; padding: 10px 12px;
  display: flex; flex-direction: column; gap: 6px;
}
/* Осталось несколько дней до побега — блок перестаёт быть «фоновым». */
.pdm-sick-block.urgent {
  border-style: solid;
  background: color-mix(in oklch, var(--color-error) 8%, transparent);
}
.pdm-sick-head { display: flex; align-items: center; gap: 6px; font-size: 12.5px; font-weight: 600; color: var(--color-error); }
.pdm-sick-head .material-symbols-outlined { font-size: 17px; }
.pdm-sick-emoji { font-size: 15px; line-height: 1; }
.pdm-sick-cause { margin: 0; font-size: 12px; line-height: 1.4; }
.pdm-runaway-warn {
  display: flex; align-items: flex-start; gap: 6px;
  margin: 2px 0 0; font-size: 12px; font-weight: 600; line-height: 1.4;
  color: var(--color-error);
}
.pdm-runaway-warn .material-symbols-outlined { font-size: 16px; flex-shrink: 0; }
.pdm-sick-progress { display: flex; align-items: center; gap: 6px; }
.pdm-sick-dot { width: 14px; height: 14px; border-radius: 50%; background: var(--color-surface-high); border: 1.5px solid var(--color-outline-dim); }
.pdm-sick-dot.filled { background: var(--color-success); border-color: var(--color-success); }
.pdm-sick-count { font-size: 12px; font-weight: 700; margin-left: 2px; }
.pdm-sick-hint { margin: 0; font-size: 11.5px; color: var(--color-text-dim); line-height: 1.4; }

.pdm-xp { width: 100%; margin-top: 14px; }
.pdm-xp-meta { display: flex; justify-content: space-between; font-size: 11.5px; color: var(--color-text-dim); margin-bottom: 4px; }
.pdm-xp-bar { height: 8px; border-radius: var(--radius-full); background: var(--color-surface-high); overflow: hidden; }
.pdm-xp-fill { height: 100%; border-radius: inherit; background: var(--color-primary); transition: width 0.6s cubic-bezier(0.34, 1.56, 0.64, 1); }

.pdm-quest {
  width: 100%; margin-top: 14px;
  border: 1px solid color-mix(in oklch, var(--color-tertiary) 35%, var(--color-outline-dim));
  border-radius: 14px; padding: 10px 12px;
  background: color-mix(in oklch, var(--color-tertiary-container) 35%, transparent);
  display: flex; flex-direction: column; gap: 6px;
}
.pdm-quest.done { border-color: var(--color-success); background: color-mix(in oklch, var(--color-success) 14%, transparent); }
.pdm-quest.claimed { border-color: var(--color-outline-dim); background: var(--color-surface-high); opacity: 0.85; }
.pdm-quest-head { display: flex; align-items: center; gap: 6px; font-size: 13px; font-weight: 700; }
.pdm-quest-title { flex: 1; min-width: 0; }
.pdm-quest-reward { font-size: 12px; font-weight: 700; color: var(--color-text-dim); white-space: nowrap; }
.pdm-quest-bar { height: 6px; border-radius: var(--radius-full); background: var(--color-surface); overflow: hidden; }
.pdm-quest-fill { height: 100%; border-radius: inherit; background: var(--color-tertiary); transition: width 0.5s cubic-bezier(0.34, 1.56, 0.64, 1); }
.pdm-quest.done .pdm-quest-fill { background: var(--color-success); }
.pdm-quest-meta { display: flex; justify-content: space-between; align-items: center; gap: 8px; font-size: 11.5px; color: var(--color-text-dim); }
.pdm-quest-claim {
  border: none; border-radius: var(--radius-full); background: var(--color-success); color: var(--color-on-success);
  font-size: 12px; font-weight: 700; padding: 5px 12px; cursor: pointer;
}
.pdm-quest-claimed { font-weight: 700; color: var(--color-success); }

.pdm-tabs {
  display: flex; gap: 6px; margin-top: 16px; width: 100%;
  background: var(--color-surface-high);
  background: var(--glass-bg);
  box-shadow: var(--glass-edge);
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-full); padding: 4px;
}
.pdm-tab {
  flex: 1; border: none; background: none; font-size: 13px; font-weight: 600;
  padding: 8px 12px; border-radius: var(--radius-full); cursor: pointer; color: var(--color-text-dim);
}
.pdm-tab.active { background: var(--grad-primary); color: var(--color-on-primary); }

.pdm-actions { width: 100%; display: flex; flex-direction: column; gap: 8px; margin-top: 14px; }
.pdm-action-btn {
  display: flex; align-items: center; gap: 10px;
  border: 1px solid var(--acrylic-border); border-radius: 16px;
  background: var(--color-surface);
  background: var(--glass-bg);
  box-shadow: var(--glass-edge); padding: 12px 14px; cursor: pointer;
  font-size: 14px; font-weight: 600; text-align: left; color: var(--color-text);
  transition: background 0.15s, transform 0.1s;
}
.pdm-action-btn:hover:not(:disabled) { background: var(--glass-hover-bg); }
.pdm-action-btn:active:not(:disabled) { transform: scale(0.99); }
.pdm-action-btn:disabled { opacity: 0.45; cursor: default; }
.pdm-action-btn.prestige {
  border-color: color-mix(in oklch, var(--color-tertiary) 55%, transparent);
  background: linear-gradient(120deg,
    color-mix(in oklch, var(--color-tertiary-container) 55%, var(--color-surface)),
    color-mix(in oklch, var(--color-primary-container) 55%, var(--color-surface)));
}
.pdm-action-emoji { font-size: 22px; }
.pdm-action-btn span:nth-child(2) { flex: 1; }
.pdm-action-cost {
  display: inline-flex; align-items: center; gap: 4px;
  font-size: 12.5px; font-weight: 700; color: var(--color-text-dim);
}

.pdm-adventure {
  display: flex; align-items: center; gap: 12px;
  border: 1px dashed color-mix(in oklch, var(--color-tertiary) 45%, transparent);
  border-radius: 14px; padding: 12px 14px;
  background: color-mix(in oklch, var(--color-tertiary-container) 35%, transparent);
}
.pdm-adventure-emoji { font-size: 26px; }
.pdm-adventure-text { display: flex; flex-direction: column; gap: 2px; flex: 1; min-width: 0; }
.pdm-adventure-text strong { font-size: 13.5px; }
.pdm-adventure-text span { font-size: 12px; color: var(--color-text-dim); }
.pdm-recall-btn {
  display: inline-flex; align-items: center; gap: 6px;
  border: none; border-radius: var(--radius-full);
  background: var(--color-secondary-container);
  color: var(--color-on-secondary-container);
  font: inherit; font-size: 12px; font-weight: 700;
  padding: 8px 14px;
  cursor: pointer;
  flex-shrink: 0;
}
.pdm-recall-btn:disabled { opacity: 0.5; cursor: default; }
.pdm-recall-cost { display: inline-flex; align-items: center; gap: 3px; opacity: 0.85; }

.pdm-closet { display: flex; gap: 6px; margin-top: 6px; flex-wrap: wrap; justify-content: center; }
.pdm-closet-item {
  width: 38px; height: 38px; border-radius: 12px;
  border: 1.5px solid var(--color-outline-dim); background: var(--color-surface);
  font-size: 19px; cursor: pointer; display: grid; place-items: center;
  transition: border-color 0.15s, background 0.15s;
}
.pdm-closet-item:hover { background: var(--glass-hover-bg); }
.pdm-closet-item.active { border-color: var(--color-primary); background: var(--color-primary-container); }

.pdm-history { width: 100%; margin-top: 14px; min-height: 80px; }
.pdm-history-loading { text-align: center; font-size: 13px; color: var(--color-text-dim); padding: 20px 0; }
.pdm-history-list { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: 6px; }
.pdm-history-item {
  display: flex; align-items: center; gap: 8px;
  padding: 8px 10px; border-radius: 12px; background: var(--color-surface-high);
  font-size: 12.5px;
}
.pdm-history-item .material-symbols-outlined { font-size: 17px; color: var(--color-text-dim); }
.pdm-history-text { flex: 1; min-width: 0; }
.pdm-history-time { font-size: 11px; color: var(--color-text-dim); white-space: nowrap; }

.pdm-links {
  display: flex;
  gap: 8px;
  width: 100%;
  flex-shrink: 0;
  padding: 12px 20px calc(14px + env(safe-area-inset-bottom, 0px));
  border-top: 1px solid var(--color-outline-dim);
  background: var(--color-surface);
}
.pdm-link-btn {
  flex: 1; min-width: 110px;
  display: inline-flex; align-items: center; justify-content: center; gap: 6px;
  border: none; border-radius: var(--radius-full);
  background: var(--color-secondary-container); color: var(--color-on-secondary-container);
  font-size: 12.5px; font-weight: 600; padding: 9px 10px; cursor: pointer;
}
.pdm-link-btn .material-symbols-outlined { font-size: 16px; }
</style>
