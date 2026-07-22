<template>
  <AppDialog
    :model-value="modelValue"
    :title="folder ? 'Изменить папку' : 'Новая папка'"
    icon="folder"
    tone="tertiary"
    size="md"
    :busy="saving"
    :actions="[
      { kind: 'cancel', label: 'Отмена' },
      { kind: 'confirm', label: folder ? 'Сохранить' : 'Создать', icon: 'check', disabled: !canSave },
    ]"
    @update:model-value="$emit('update:modelValue', $event)"
    @cancel="$emit('update:modelValue', false)"
    @confirm="save"
  >
    <div class="fe-body">
      <!-- Название + эмодзи -->
      <div class="fe-name-row">
        <button
          type="button"
          class="fe-emoji-btn"
          :title="emoji ? 'Сменить эмодзи' : 'Добавить эмодзи'"
          @click="emojiOpen = !emojiOpen"
        >
          <EmojiGlyph v-if="emoji" :char="emoji" class="fe-emoji-glyph" />
          <span v-else class="material-symbols-outlined">add_reaction</span>
        </button>
        <InputText
          v-model="title"
          class="fe-name-input"
          placeholder="Название папки"
          maxlength="64"
          @keydown.enter="canSave && save()"
        />
        <button v-if="emoji" type="button" class="fe-emoji-clear" title="Убрать эмодзи" @click="emoji = ''">
          <span class="material-symbols-outlined">close</span>
        </button>
      </div>

      <div v-if="emojiOpen" class="fe-emoji-pop">
        <EmojiPicker @pick="onEmoji" />
      </div>

      <!-- Авто-фильтры -->
      <div class="fe-section-label">Включать автоматически</div>
      <div class="fe-toggles">
        <button
          v-for="opt in autoOptions"
          :key="opt.key"
          type="button"
          class="fe-toggle"
          :class="{ on: auto[opt.key] }"
          role="switch"
          :aria-checked="auto[opt.key]"
          @click="auto[opt.key] = !auto[opt.key]"
        >
          <span class="material-symbols-outlined fe-toggle-ico">{{ opt.icon }}</span>
          <span class="fe-toggle-text">{{ opt.label }}</span>
          <span class="fe-switch" :class="{ on: auto[opt.key] }"><span class="fe-knob" /></span>
        </button>
      </div>

      <!-- Ручной выбор чатов -->
      <div class="fe-section-label">
        Чаты в папке
        <span v-if="selectedIds.size" class="fe-count">{{ selectedIds.size }}</span>
      </div>
      <div class="fe-search">
        <span class="material-symbols-outlined">search</span>
        <input v-model="q" class="fe-search-input" placeholder="Найти чат" />
      </div>
      <div class="fe-chats">
        <EmptyState
          v-if="!filteredChats.length"
          icon="forum"
          title="Чатов нет"
          :subtitle="q ? 'Попробуйте другой запрос.' : 'Здесь появятся ваши чаты.'"
        />
        <button
          v-for="c in filteredChats"
          :key="c.id"
          type="button"
          class="fe-chat"
          :class="{ on: selectedIds.has(c.id) }"
          @click="toggleChat(c.id)"
        >
          <div class="fe-chat-ava" :class="{ group: c.is_group, dev: c.is_dev_chat }">
            <img v-if="chatAvatar(c)" :src="chatAvatar(c)" :alt="chatName(c)" />
            <span v-else class="material-symbols-outlined">{{ c.is_group ? 'groups' : 'support_agent' }}</span>
          </div>
          <span class="fe-chat-name">{{ chatName(c) }}</span>
          <span class="fe-chat-check material-symbols-outlined">{{ selectedIds.has(c.id) ? 'check_circle' : 'radio_button_unchecked' }}</span>
        </button>
      </div>
    </div>
  </AppDialog>
</template>

<script setup>
import { ref, reactive, computed, watch } from 'vue'
import AppDialog from '@/components/common/AppDialog.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import EmojiPicker from '@/components/common/EmojiPicker.vue'
import EmojiGlyph from '@/components/common/EmojiGlyph.vue'
import InputText from 'primevue/inputtext'
import { useMessengerStore } from '@/stores/messenger.js'
import { useNotificationsStore } from '@/stores/notifications.js'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  folder: { type: Object, default: null },
})
const emit = defineEmits(['update:modelValue', 'saved'])

const messenger = useMessengerStore()

const title = ref('')
const emoji = ref('')
const emojiOpen = ref(false)
const auto = reactive({ include_personal: false, include_groups: false, include_unread: false })
const selectedIds = ref(new Set())
const q = ref('')
const saving = ref(false)

const autoOptions = [
  { key: 'include_personal', label: 'Личные чаты', icon: 'person' },
  { key: 'include_groups', label: 'Группы', icon: 'groups' },
  { key: 'include_unread', label: 'Непрочитанные', icon: 'mark_chat_unread' },
]

// Инициализация при каждом открытии.
watch(() => props.modelValue, (open) => {
  if (!open) return
  emojiOpen.value = false
  q.value = ''
  const f = props.folder
  title.value = f?.title || ''
  emoji.value = f?.emoji || ''
  auto.include_personal = !!f?.include_personal
  auto.include_groups = !!f?.include_groups
  auto.include_unread = !!f?.include_unread
  selectedIds.value = new Set(f?.conversation_ids || [])
})

const canSave = computed(() => title.value.trim().length > 0 && !saving.value)

const filteredChats = computed(() => {
  const query = q.value.trim().toLowerCase()
  if (!query) return messenger.conversations
  return messenger.conversations.filter(c => chatName(c).toLowerCase().includes(query))
})

function chatName(c) {
  if (c.is_dev_chat) return 'Техподдержка'
  if (c.is_group) return c.title || 'Группа'
  return c.other_user?.fio || 'Чат'
}

function chatAvatar(c) {
  if (c.is_dev_chat) return ''
  if (c.is_group) return c.avatar_path ? `/uploads/${c.avatar_path}` : ''
  const u = c.other_user
  if (!u) return ''
  return u.avatar_path ? `/uploads/${u.avatar_path}` : `/api/users/${u.id}/identicon`
}

function toggleChat(id) {
  const s = new Set(selectedIds.value)
  if (s.has(id)) s.delete(id)
  else s.add(id)
  selectedIds.value = s
}

function onEmoji(e) {
  emoji.value = e
  emojiOpen.value = false
}

async function save() {
  if (!canSave.value) return
  saving.value = true
  const payload = {
    title: title.value.trim(),
    emoji: emoji.value || null,
    include_personal: auto.include_personal,
    include_groups: auto.include_groups,
    include_unread: auto.include_unread,
    conversation_ids: [...selectedIds.value],
  }
  try {
    if (props.folder) await messenger.updateFolderAction(props.folder.id, payload)
    else await messenger.createFolderAction(payload)
    emit('saved')
    emit('update:modelValue', false)
  } catch (e) {
    useNotificationsStore().error(e?.message || 'Не удалось сохранить папку')
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
.fe-body { display: flex; flex-direction: column; gap: 14px; }

.fe-name-row { display: flex; align-items: center; gap: 8px; }

.fe-emoji-btn {
  width: 44px; height: 44px; min-height: 0;
  flex-shrink: 0;
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-md);
  background: var(--glass-bg);
  box-shadow: var(--glass-edge);
  color: var(--color-text-dim);
  cursor: pointer;
  display: grid;
  place-items: center;
}
.fe-emoji-btn:hover { color: var(--color-text); border-color: var(--color-primary); }
.fe-emoji-btn .material-symbols-outlined { font-size: 22px; }
.fe-emoji-glyph { font-size: 24px; line-height: 1; }

.fe-name-input { flex: 1; }

.fe-emoji-clear {
  width: 32px; height: 32px; min-height: 0;
  flex-shrink: 0;
  border: none; border-radius: 50%;
  background: transparent; color: var(--color-text-dim); cursor: pointer;
  display: grid; place-items: center;
}
.fe-emoji-clear:hover { background: var(--color-surface-high); color: var(--color-text); }
.fe-emoji-clear .material-symbols-outlined { font-size: 18px; }

.fe-emoji-pop {
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-md);
  overflow: hidden;
}

.fe-section-label {
  display: flex; align-items: center; gap: 6px;
  font-size: 12px; font-weight: 700; letter-spacing: 0.3px;
  text-transform: uppercase;
  color: var(--color-text-dim);
}

.fe-count {
  font-size: 11px; font-weight: 700;
  padding: 1px 7px; border-radius: var(--radius-full);
  background: var(--color-tertiary-container); color: var(--color-on-tertiary-container);
}

.fe-toggles { display: flex; flex-direction: column; gap: 6px; }

.fe-toggle {
  display: flex; align-items: center; gap: 12px;
  padding: 10px 12px;
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-md);
  background: var(--glass-bg);
  box-shadow: var(--glass-edge);
  color: var(--color-text);
  cursor: pointer;
  text-align: left;
  transition: border-color 0.15s;
}
.fe-toggle.on { border-color: var(--color-primary); }
.fe-toggle-ico { font-size: 20px; color: var(--color-text-dim); flex-shrink: 0; }
.fe-toggle.on .fe-toggle-ico { color: var(--color-primary); }
.fe-toggle-text { flex: 1; font-size: 14px; font-weight: 500; }

.fe-switch {
  width: 40px; height: 24px;
  flex-shrink: 0;
  border-radius: var(--radius-full);
  background: var(--color-surface-high);
  border: 1px solid var(--color-outline-dim);
  position: relative;
  transition: background 0.18s, border-color 0.18s;
}
.fe-switch.on { background: var(--color-primary); border-color: var(--color-primary); }
.fe-knob {
  position: absolute; top: 50%; left: 3px;
  width: 16px; height: 16px; border-radius: 50%;
  background: var(--color-text-dim);
  transform: translateY(-50%);
  transition: left 0.18s, background 0.18s;
}
.fe-switch.on .fe-knob { left: 19px; background: var(--color-on-primary); }

.fe-search { position: relative; display: flex; align-items: center; }
.fe-search .material-symbols-outlined {
  position: absolute; left: 12px; color: var(--color-text-dim); font-size: 20px; pointer-events: none;
}
.fe-search-input {
  width: 100%;
  padding: 9px 14px 9px 40px;
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-full);
  background: var(--glass-bg);
  box-shadow: var(--glass-edge);
  color: var(--color-text);
  font-size: 14px; outline: none;
}
.fe-search-input:focus { border-color: var(--color-primary); }

.fe-chats {
  display: flex; flex-direction: column; gap: 2px;
  max-height: 240px; overflow-y: auto;
  margin: 0 -4px;
  padding: 0 4px;
}

.fe-chat {
  display: flex; align-items: center; gap: 10px;
  padding: 8px 10px;
  border: none; border-radius: var(--radius-md);
  background: transparent; color: var(--color-text);
  cursor: pointer; text-align: left;
  transition: background 0.15s;
}
.fe-chat:hover { background: var(--glass-bg); }
.fe-chat.on { background: color-mix(in oklch, var(--color-tertiary-container) 45%, transparent); }

.fe-chat-ava {
  width: 36px; height: 36px; border-radius: 50%;
  flex-shrink: 0; overflow: hidden;
  display: grid; place-items: center;
  background: var(--color-surface-high); color: var(--color-text-dim);
}
.fe-chat-ava.group { background: var(--color-primary-container); color: var(--color-on-primary-container); }
.fe-chat-ava.dev { background: var(--color-tertiary-container); color: var(--color-on-tertiary-container); }
.fe-chat-ava img { width: 100%; height: 100%; object-fit: cover; }
.fe-chat-ava .material-symbols-outlined { font-size: 20px; font-variation-settings: 'FILL' 1; }

.fe-chat-name { flex: 1; min-width: 0; font-size: 14px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }

.fe-chat-check { font-size: 22px; color: var(--color-text-dim); flex-shrink: 0; }
.fe-chat.on .fe-chat-check { color: var(--color-tertiary); font-variation-settings: 'FILL' 1; }
</style>
