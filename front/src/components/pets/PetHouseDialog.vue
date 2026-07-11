<template>
  <!-- above-pet-modal: открывается ПОВЕРХ PetDetailModal (его overlay 10700);
       класс из z-index-лестницы main.css — base-z-index PrimeVue ненадёжен. -->
  <AppDialog
    :model-value="modelValue"
    :title="readonly ? `Домик «${guestPet?.name || ''}»` : 'Домик питомца'"
    :subtitle="readonly
      ? `Хозяин — ${guestPet?.user?.fio || 'коллега'}`
      : 'Расставляйте декор как хочется — домик видят коллеги'"
    icon="cottage"
    size="md"
    mask-class="above-pet-modal"
    dialog-class="above-pet-modal"
    @update:model-value="(v) => emit('update:modelValue', v)"
  >
    <div class="phd">
      <!-- Сцена: питомец + декор со свободными координатами (drag). -->
      <div ref="sceneEl" class="phd-scene" :style="{ background: sceneBackground }">
        <span
          class="phd-scene-pet"
          :class="{ movable: !readonly, dragging: petDragging }"
          :style="petPosStyle"
          @pointerdown="readonly ? null : startPetDrag($event)"
        ><EmojiGlyph :char="petEmoji(scenePet)" /></span>
        <div
          v-for="item in localPlaced"
          :key="item.key"
          class="phd-scene-item"
          :class="{ readonly, dragging: dragKey === item.key }"
          :style="{ left: item.x + '%', top: item.y + '%' }"
          :title="decorTitle(item.key)"
          @pointerdown="readonly ? null : startDrag(item, $event)"
        >
          <span class="phd-scene-emoji"><EmojiGlyph :char="decorEmoji(item.key)" /></span>
          <button
            v-if="!readonly"
            class="phd-scene-remove"
            type="button"
            :aria-label="`Убрать ${decorTitle(item.key)}`"
            @pointerdown.stop
            @click.stop="removePlaced(item.key)"
          >✕</button>
        </div>
        <p v-if="!localPlaced.length" class="phd-scene-empty">
          {{ readonly ? 'В домике пока пусто' : 'Пока пусто — купите декор и поставьте' }}
        </p>
      </div>

      <!-- Обустройство и витрина — только у своего домика. -->
      <template v-if="!readonly">
        <p class="phd-slots">
          Перетаскивайте предметы (и самого грувика!) по комнате ·
          занято {{ localPlaced.length }} / {{ house?.placed_max ?? 6 }}
        </p>

        <!-- Тема комнаты: градиентные пресеты. -->
        <div class="phd-themes">
          <button
            v-for="t in HOUSE_THEMES"
            :key="t.key"
            class="phd-theme"
            :class="{ active: currentTheme === t.key }"
            type="button"
            :title="t.title"
            :style="{ background: t.background }"
            @click="pickTheme(t.key)"
          ></button>
        </div>

        <div v-if="ownedIdle.length" class="phd-owned">
          <button
            v-for="d in ownedIdle"
            :key="d.key"
            class="phd-owned-item"
            type="button"
            :disabled="localPlaced.length >= (house?.placed_max ?? 6)"
            :title="`${decorTitle(d.key)} — поставить`"
            @click="addPlaced(d.key)"
          ><EmojiGlyph :char="decorEmoji(d.key)" /></button>
        </div>

        <h4 class="phd-shop-title">
          Купить декор
          <span class="phd-balance"><KudosCoin /> {{ house?.kudos ?? pets.pet?.kudos ?? 0 }}</span>
        </h4>
        <div class="phd-shop">
          <div v-for="d in buyable" :key="d.key" class="phd-shop-item" :class="{ owned: d.owned }">
            <span class="phd-shop-emoji"><EmojiGlyph :char="decorEmoji(d.key)" /></span>
            <span class="phd-shop-name">{{ decorTitle(d.key) }}</span>
            <button
              v-if="!d.owned"
              class="phd-shop-buy"
              type="button"
              :disabled="buying || (house?.kudos ?? 0) < d.price"
              @click="buy(d)"
            ><KudosCoin /> {{ d.price }}</button>
            <span v-else class="phd-shop-owned material-symbols-outlined">check</span>
          </div>
        </div>
        <p v-if="seasonOnly.length" class="phd-season-hint">
          <span v-for="d in seasonOnly" :key="d.key" class="phd-season-item">
            <EmojiGlyph :char="decorEmoji(d.key)" /> {{ decorTitle(d.key) }}
          </span>
          — только за награды сезонного трека
        </p>
      </template>
    </div>
  </AppDialog>
</template>

<script setup>
import { computed, ref, watch } from 'vue'
import EmojiGlyph from '@/components/common/EmojiGlyph.vue'
import AppDialog from '@/components/common/AppDialog.vue'
import KudosCoin from '@/components/pets/KudosCoin.vue'
import { usePetsStore } from '@/stores/pets.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { decorEmoji, decorTitle, petEmoji } from '@/utils/pets.js'
import { HOUSE_THEMES, houseThemeBackground } from '@/utils/houseThemes.js'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  // Питомец коллеги → режим просмотра (сцена из его house_placed, без витрины).
  guestPet: { type: Object, default: null },
})
const emit = defineEmits(['update:modelValue'])

const pets = usePetsStore()
const notify = useNotificationsStore()
const buying = ref(false)

const readonly = computed(() => !!props.guestPet)
const scenePet = computed(() => (readonly.value ? props.guestPet : pets.pet))

const house = computed(() => pets.house)

// Тема комнаты: у гостя — из его питомца, у себя — из house (fallback pet).
const currentTheme = computed(() => (readonly.value
  ? props.guestPet?.house_theme
  : (house.value?.theme || pets.pet?.house_theme)) || 'cozy')
const sceneBackground = computed(() => houseThemeBackground(currentTheme.value))

async function pickTheme(key) {
  if (key === currentTheme.value) return
  try {
    await pets.setHouseTheme(key)
  } catch (e) {
    notify.warn(e?.message || 'Не получилось сменить тему')
  }
}

// Позиция самого грувика (проценты сцены; NULL — место по умолчанию).
const petDragging = ref(false)
const localPetPos = ref(null)
const petPosStyle = computed(() => {
  const src = readonly.value
    ? { x: props.guestPet?.house_pet_x, y: props.guestPet?.house_pet_y }
    : (localPetPos.value || { x: house.value?.pet_x ?? pets.pet?.house_pet_x, y: house.value?.pet_y ?? pets.pet?.house_pet_y })
  if (src?.x == null || src?.y == null) return {}
  return { left: src.x + '%', top: src.y + '%', bottom: 'auto', transform: 'translate(-50%, -50%)' }
})

function startPetDrag(e) {
  e.preventDefault()
  petDragging.value = true
  const move = (ev) => {
    const rect = sceneEl.value?.getBoundingClientRect()
    if (!rect) return
    localPetPos.value = {
      x: clampPct(((ev.clientX - rect.left) / rect.width) * 100),
      y: clampPct(((ev.clientY - rect.top) / rect.height) * 100),
    }
  }
  const up = async () => {
    document.removeEventListener('pointermove', move)
    document.removeEventListener('pointerup', up)
    petDragging.value = false
    if (!localPetPos.value) return
    try {
      await pets.setHousePetPos(localPetPos.value.x, localPetPos.value.y)
    } catch (err) {
      notify.warn(err?.message || 'Не получилось сохранить позицию грувика')
      localPetPos.value = null
    }
  }
  document.addEventListener('pointermove', move)
  document.addEventListener('pointerup', up)
}

/* Локальная копия расстановки: во время drag двигаем её (плавно, без
   запросов), на pointerup сохраняем всю раскладку одним arrangeHouse. */
const dragKey = ref(null)
const localPlaced = ref([])
const serverPlaced = computed(() =>
  readonly.value ? (props.guestPet?.house_placed || []) : (house.value?.placed || []))
/* Строка вместо объекта — данные старого формата (до свободной расстановки)
   или ответ ещё не перезапущенного petsvc: даём дефолтные координаты. */
function normalizeItem(i, idx) {
  return typeof i === 'string'
    ? { key: i, x: 16 + (idx % 5) * 17, y: 78 }
    : { ...i }
}

watch(serverPlaced, (items) => {
  if (!dragKey.value) localPlaced.value = items.map(normalizeItem)
}, { immediate: true, deep: true })

const ownedIdle = computed(() =>
  (house.value?.catalog || []).filter((d) =>
    d.owned && !localPlaced.value.some((i) => i.key === d.key)))
const buyable = computed(() => (house.value?.catalog || []).filter((d) => d.price > 0))
const seasonOnly = computed(() => (house.value?.catalog || []).filter((d) => d.price === 0))

watch(() => props.modelValue, (open) => {
  if (!open) return
  localPetPos.value = null
  if (!readonly.value) pets.fetchHouse().catch(() => {})
})

// ── Drag: свободные координаты в процентах сцены ────────────────
const sceneEl = ref(null)

function startDrag(item, e) {
  e.preventDefault()
  dragKey.value = item.key
  const move = (ev) => {
    const rect = sceneEl.value?.getBoundingClientRect()
    if (!rect) return
    const target = localPlaced.value.find((i) => i.key === item.key)
    if (!target) return
    target.x = clampPct(((ev.clientX - rect.left) / rect.width) * 100)
    target.y = clampPct(((ev.clientY - rect.top) / rect.height) * 100)
  }
  const up = async () => {
    document.removeEventListener('pointermove', move)
    document.removeEventListener('pointerup', up)
    dragKey.value = null
    await persist()
  }
  document.addEventListener('pointermove', move)
  document.addEventListener('pointerup', up)
}

// 4..96 — чтобы эмодзи не уезжал за границы сцены наполовину.
function clampPct(v) {
  return Math.min(96, Math.max(4, Math.round(v * 10) / 10))
}

async function persist() {
  try {
    await pets.arrangeHouse(localPlaced.value.map((i) => ({ key: i.key, x: i.x, y: i.y })))
  } catch (e) {
    notify.warn(e?.message || 'Не получилось сохранить расстановку')
    localPlaced.value = serverPlaced.value.map((i) => ({ ...i }))
  }
}

// Новый предмет встаёт в свободное место нижнего ряда — дальше перетащат.
function addPlaced(key) {
  const n = localPlaced.value.length
  localPlaced.value.push({ key, x: clampPct(16 + (n % 5) * 17), y: 78 })
  persist()
}

function removePlaced(key) {
  localPlaced.value = localPlaced.value.filter((i) => i.key !== key)
  persist()
}

async function buy(d) {
  if (buying.value) return
  buying.value = true
  try {
    await pets.buyHouseDecor(d.key)
    notify.success(`Куплено: ${decorTitle(d.key)}`)
  } catch (e) {
    notify.warn(e?.message || 'Не получилось купить')
  } finally {
    buying.value = false
  }
}
</script>

<style scoped>
.phd { display: flex; flex-direction: column; gap: 10px; }

/* Сцена — «комната»: градиент выбранной темы, питомец и декор двигаются
   свободно (координаты в процентах). Фон приходит инлайном из houseThemes. */
.phd-scene {
  position: relative;
  height: 210px;
  border-radius: var(--radius-lg, 16px);
  border: 1px solid var(--color-outline-dim);
  overflow: hidden;
}
.phd-scene-pet {
  position: absolute;
  left: 50%; bottom: 14px;
  transform: translateX(-50%);
  font-size: 52px;
  line-height: 1;
}
.phd-scene-pet.movable { cursor: grab; touch-action: none; user-select: none; }
.phd-scene-pet.dragging { cursor: grabbing; z-index: 2; }

/* Свотчи тем — во всю ширину модалки, поровну; не влезли — переносятся. */
.phd-themes { display: flex; gap: 8px; width: 100%; flex-wrap: wrap; }
.phd-theme {
  flex: 1 1 56px;
  min-width: 56px;
  height: 32px;
  border-radius: 10px;
  border: 2px solid var(--color-outline-dim);
  cursor: pointer;
  padding: 0;
  transition: transform 0.12s, border-color 0.12s;
}
.phd-theme:hover { border-color: color-mix(in oklch, var(--color-primary) 30%, var(--acrylic-border)); }
.phd-theme.active { border-color: var(--color-primary); box-shadow: 0 0 0 2px var(--color-primary-container); }
.phd-scene-item {
  position: absolute;
  transform: translate(-50%, -50%);
  font-size: 30px;
  line-height: 1;
  cursor: grab;
  touch-action: none; /* иначе тач скроллит диалог вместо drag */
  user-select: none;
}
.phd-scene-item.dragging { cursor: grabbing; z-index: 2; }
.phd-scene-item.dragging .phd-scene-emoji { transform: scale(1.18); }
.phd-scene-item.readonly { cursor: default; }
.phd-scene-emoji { display: block; transition: transform 0.12s; pointer-events: none; }
.phd-scene-item:not(.readonly):hover .phd-scene-emoji { transform: scale(1.12); }
.phd-scene-remove {
  position: absolute;
  top: -10px; right: -12px;
  width: 18px; height: 18px;
  /* min-height: 0 — глобальный мобильный min-height у button (36px,
     main.css) растягивал кружок в овал. */
  min-height: 0;
  padding: 0;
  border: none; border-radius: 50%;
  background: var(--color-surface);
  color: var(--color-text-dim);
  font-size: 10px; line-height: 1;
  display: none; place-items: center;
  cursor: pointer;
  box-shadow: var(--shadow-sm);
}
.phd-scene-item:hover .phd-scene-remove { display: grid; }
/* На таче hover нет — крестик виден всегда и чуть крупнее (tap target). */
@media (hover: none) {
  .phd-scene-remove {
    display: grid;
    width: 24px; height: 24px;
    font-size: 12px;
  }
}
.phd-scene-empty {
  position: absolute;
  top: 14px; left: 0; right: 0;
  margin: 0; text-align: center;
  font-size: 12.5px; color: var(--color-text-dim);
}
.phd-slots { margin: 0; font-size: 11.5px; color: var(--color-text-dim); text-align: right; }

.phd-owned { display: flex; gap: 6px; flex-wrap: wrap; }
.phd-owned-item {
  width: 42px; height: 42px; border-radius: 12px;
  border: 1.5px dashed var(--color-outline-dim);
  background: var(--color-surface);
  font-size: 21px; cursor: pointer;
  display: grid; place-items: center;
}
.phd-owned-item:hover:not(:disabled) { border-color: var(--color-primary); background: var(--color-primary-container); }
.phd-owned-item:disabled { opacity: 0.45; cursor: default; }

.phd-shop-title {
  margin: 8px 0 0;
  display: flex; align-items: center; justify-content: space-between;
  font-size: 13.5px; font-weight: 700;
}
.phd-balance {
  display: inline-flex; align-items: center; gap: 4px;
  font-size: 12.5px; font-weight: 700;
  padding: 3px 10px; border-radius: var(--radius-full);
  background: color-mix(in oklch, var(--color-success) 18%, transparent);
}
.phd-shop { display: flex; flex-direction: column; gap: 6px; max-height: 220px; overflow-y: auto; }
.phd-shop-item {
  display: flex; align-items: center; gap: 10px;
  padding: 8px 10px; border-radius: 12px;
  background: var(--color-surface-high);
  font-size: 13px;
}
.phd-shop-item.owned { opacity: 0.7; }
.phd-shop-emoji { font-size: 20px; }
.phd-shop-name { flex: 1; min-width: 0; font-weight: 600; }
.phd-shop-buy {
  display: inline-flex; align-items: center; gap: 4px;
  border: none; border-radius: var(--radius-full);
  background: var(--color-primary); color: var(--color-on-primary);
  font-size: 12px; font-weight: 700; padding: 6px 12px; cursor: pointer;
}
.phd-shop-buy:disabled { opacity: 0.45; cursor: default; }
.phd-shop-owned { color: var(--color-success); font-size: 20px; }
.phd-season-hint { margin: 2px 0 0; font-size: 11.5px; color: var(--color-text-dim); line-height: 1.6; }
.phd-season-item { white-space: nowrap; margin-right: 6px; }
</style>
