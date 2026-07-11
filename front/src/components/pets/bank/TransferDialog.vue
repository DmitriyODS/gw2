<template>
  <AppDialog
    :model-value="modelValue"
    title="Перевести кудосы"
    subtitle="Признание коллегам — комментарий увидит получатель"
    icon="send_money"
    tone="primary"
    size="lg"
    @update:model-value="$emit('update:modelValue', $event)"
  >
    <div class="td">
      <!-- Получатель: карточки с радио-отметкой -->
      <div class="td-recipients">
        <button
          v-for="p in colleagues"
          :key="p.user_id"
          class="td-recipient glass-hover"
          :class="{ active: recipientId === p.user_id }"
          type="button"
          @click="recipientId = p.user_id"
        >
          <img class="td-recipient-avatar" :src="avatarUrl(p.user)" :alt="p.user?.fio" />
          <span class="td-recipient-name">{{ firstName(p.user?.fio) }}</span>
          <span class="td-recipient-radio" :class="{ checked: recipientId === p.user_id }">
            <span v-if="recipientId === p.user_id" class="material-symbols-outlined">check</span>
          </span>
        </button>
        <p v-if="!colleagues.length" class="td-hint">В компании пока нет коллег с питомцами.</p>
      </div>

      <!-- Сумма -->
      <div class="td-amounts">
        <button
          v-for="a in quickAmounts"
          :key="a"
          class="td-chip"
          :class="{ active: amount === a }"
          type="button"
          @click="amount = a"
        ><KudosCoin /> {{ a }}</button>
        <AmountInput v-model="amount" :max="transferMax" class="td-amount" />
      </div>
      <p class="td-hint">
        За один раз — до {{ transferMax }} кудосов; сегодня осталось {{ leftToday }}.
      </p>

      <!-- Теги «за что» — быстрые благодарности в комментарий -->
      <div class="td-tags">
        <button
          v-for="t in THANK_TAGS"
          :key="t"
          class="td-tag"
          :class="{ active: comment === t }"
          type="button"
          @click="comment = comment === t ? '' : t"
        >{{ t }}</button>
      </div>

      <div class="td-comment">
        <textarea
          v-model="comment"
          rows="2" maxlength="120"
          class="td-comment-input"
          placeholder="Спасибо за… (комментарий увидит получатель)"
        ></textarea>
        <span class="td-comment-count">{{ comment.length }} / 120</span>
      </div>
    </div>

    <template #footer>
      <div class="td-footer">
        <button
          class="btn-grad td-send"
          :disabled="busy || !recipientId || !validAmount || amount > transferMax"
          @click="send"
        >
          <span class="material-symbols-outlined">send</span>
          Перевести
          <KudosCoin />
        </button>
      </div>
    </template>
  </AppDialog>
</template>

<script setup>
import { computed, ref, watch } from 'vue'
import AppDialog from '@/components/common/AppDialog.vue'
import KudosCoin from '@/components/pets/KudosCoin.vue'
import AmountInput from '@/components/pets/bank/AmountInput.vue'
import { usePetsStore } from '@/stores/pets.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { avatarUrl } from '@/utils/pets.js'
import { playKudosSent } from '@/utils/kudosSound.js'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  // Предвыбранный получатель (кнопка «Отблагодарить» из топа щедрости).
  presetUserId: { type: Number, default: null },
})
const emit = defineEmits(['update:modelValue', 'sent'])

const THANK_TAGS = ['🙏 Спасибо!', '🚀 За скорость', '💎 За качество', '🤝 Выручил(а)', '🔥 Топ работа']

const pets = usePetsStore()
const notify = useNotificationsStore()

const busy = ref(false)
const recipientId = ref(null)
const amount = ref(null)
const comment = ref('')

const bank = computed(() => pets.bank)
const transferMax = computed(() => bank.value?.tier.transfer_max ?? 20)
const leftToday = computed(() => bank.value?.transfer_left_today ?? 0)

const colleagues = computed(() =>
  (pets.zoo || []).filter((p) => p.user_id !== pets.myId && p.user))

const quickAmounts = computed(() =>
  [5, 10, 20, 30, 50, 75, 100].filter((a) => a <= transferMax.value))

const validAmount = computed(() => Number.isFinite(amount.value) && amount.value >= 1)

watch(() => props.modelValue, (open) => {
  if (!open) return
  recipientId.value = props.presetUserId
  amount.value = null
  comment.value = ''
  if (!pets.zoo.length) pets.fetchZoo().catch(() => {})
})

async function send() {
  busy.value = true
  try {
    await pets.transferKudos(recipientId.value, amount.value, comment.value)
    playKudosSent()
    notify.success('Перевод отправлен')
    emit('sent')
    emit('update:modelValue', false)
  } catch (e) {
    notify.error(e?.message || 'Перевод не удался')
  } finally {
    busy.value = false
  }
}

function firstName(fio = '') {
  const parts = fio.trim().split(/\s+/)
  return parts.length > 1 ? parts[1] : parts[0] || ''
}
</script>

<style scoped>
.td { display: flex; flex-direction: column; gap: 14px; }
.td-hint { margin: -6px 0 0; font-size: 12.5px; color: var(--color-text-dim); line-height: 1.4; }

/* Карточки получателей: аватар + имя + радио-отметка снизу. */
/* Лента получателей: горизонтальный скролл навылет к краям модалки —
   отрицательные маргины съедают паддинг тела AppDialog (24px / 20px на мобильных),
   padding-inline возвращает отступ первой/последней карточке внутри скролла. */
.td-recipients {
  --td-bleed: 24px;
  display: flex;
  gap: 12px;
  overflow-x: auto;
  overflow-y: hidden;
  margin-inline: calc(-1 * var(--td-bleed));
  padding-inline: var(--td-bleed);
  /* Вертикальный запас: overflow-y hidden иначе обрезает hover-подъём карточек. */
  padding-block: 8px;
  scroll-snap-type: x proximity;
  /* Иначе snap-align: start прижимает первую карточку к краю, съедая padding-inline. */
  scroll-padding-inline: var(--td-bleed);
  scrollbar-width: none;
}
@media (max-width: 768px) {
  .td-recipients { --td-bleed: 20px; }
}
.td-recipients::-webkit-scrollbar { display: none; }
.td-recipient {
  border: 1.5px solid var(--acrylic-border);
  background: var(--acrylic-bg);
  background: var(--glass-bg);
  box-shadow: var(--glass-edge);
  border-radius: var(--radius-lg, 18px);
  padding: 14px 8px 12px;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  cursor: pointer;
  font: inherit;
  color: var(--color-text);
  flex: 0 0 116px;
  scroll-snap-align: start;
  overflow: hidden;
}
.td-recipient.active {
  border-color: var(--color-primary);
  background: color-mix(in oklch, var(--color-primary-container) 45%, transparent);
}
.td-recipient-avatar { width: 52px; height: 52px; border-radius: 50%; object-fit: cover; }
.td-recipient-name {
  font-size: 13px; font-weight: 700;
  max-width: 100%;
  overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
}
.td-recipient-radio {
  width: 22px; height: 22px;
  border-radius: 50%;
  border: 1.5px solid var(--color-outline-dim);
  background: transparent;
  display: grid; place-items: center;
  transition: background 0.12s, border-color 0.12s;
}
.td-recipient-radio.checked {
  border-color: transparent;
  background: var(--color-primary);
  color: var(--color-on-primary);
}
.td-recipient-radio .material-symbols-outlined { font-size: 15px; font-weight: 700; }

/* Сумма: чипы + инпут в одном ряду. */
.td-amounts { display: flex; gap: 10px; flex-wrap: wrap; align-items: stretch; }
.td-chip {
  border: 1px solid var(--color-outline-dim);
  background: var(--color-surface);
  background: var(--glass-bg);
  box-shadow: var(--glass-edge);
  color: var(--color-text);
  border-radius: var(--radius-md);
  font: inherit; font-size: 14px; font-weight: 700;
  padding: 10px 18px; cursor: pointer;
  display: inline-flex; align-items: center; gap: 6px;
}
.td-chip.active {
  border-color: var(--color-primary);
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
}
.td-amount { flex: 1 1 100%; width: 100%; }
/* Акцентная рамка поля суммы — как в референсе. */
.td-amount :deep(input) { border: 1.5px solid var(--color-primary); font-size: 14px; padding-top: 10px; padding-bottom: 10px; }

.td-tags { display: flex; gap: 8px; flex-wrap: wrap; }
.td-tag {
  border: none;
  background: var(--color-surface);
  background: var(--glass-bg);
  color: var(--color-text);
  border-radius: var(--radius-full);
  font: inherit; font-size: 13px; font-weight: 600;
  padding: 9px 16px; cursor: pointer;
  box-shadow: var(--glass-edge), inset 0 0 0 1px var(--color-outline-dim);
}
.td-tag.active {
  box-shadow: inset 0 0 0 1.5px var(--color-primary);
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
}

/* Комментарий: округлое поле со счётчиком в углу. */
.td-comment { position: relative; }
.td-comment-input {
  width: 100%;
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-lg, 16px);
  background: var(--color-surface);
  background: var(--glass-bg);
  box-shadow: var(--glass-edge);
  color: var(--color-text);
  font: inherit; font-size: 13.5px;
  padding: 12px 14px 22px;
  resize: none;
}
.td-comment-input:focus { outline: none; border-color: var(--color-primary); }
.td-comment-count {
  position: absolute;
  right: 12px; bottom: 8px;
  font-size: 11px; color: var(--color-text-dim);
  pointer-events: none;
}

.td-footer { display: flex; width: 100%; }
.td-send {
  flex: 1;
  display: inline-flex; align-items: center; justify-content: center; gap: 8px;
  font-size: 14.5px;
  padding: 12px 26px;
  border-radius: var(--radius-full);
}
</style>
