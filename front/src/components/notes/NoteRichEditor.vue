<template>
  <div class="ne" :class="{ readonly: !editable }">
    <!-- Sticky-панель форматирования (правило sticky-шапок: плотное стекло) -->
    <div v-if="editable" class="ne-toolbar">
      <!-- Действия с выделенным (ИИ/создать/в чат/копировать): постоянная
           кнопка, активна только при непустом выделении. На таче это
           ЕДИНСТВЕННЫЙ путь к меню — contextmenu отдан браузеру под
           нативное выделение/копирование. -->
      <div v-if="selectionMenu" class="ne-tgroup">
        <button
          class="ne-tbtn ne-selbtn"
          :disabled="!selectionHasText()"
          title="Действия с выделенным"
          aria-haspopup="menu"
          @mousedown.prevent
          @click="openSelectionActions"
        >
          <span class="material-symbols-outlined">auto_awesome</span>
          <span class="material-symbols-outlined ne-selbtn-caret">arrow_drop_down</span>
        </button>
      </div>
      <span v-if="selectionMenu" class="ne-tsep" />

      <div class="ne-tgroup">
        <button
          v-for="lvl in [1, 2, 3]"
          :key="lvl"
          class="ne-tbtn"
          :class="{ active: editor?.isActive('heading', { level: lvl }) }"
          :title="`Заголовок ${lvl}`"
          @click="chain().toggleHeading({ level: lvl }).run()"
        >H{{ lvl }}</button>
      </div>

      <span class="ne-tsep" />

      <div class="ne-tgroup">
        <button class="ne-tbtn" :class="{ active: editor?.isActive('bold') }" title="Жирный (⌘B)" @click="chain().toggleBold().run()">
          <span class="material-symbols-outlined">format_bold</span>
        </button>
        <button class="ne-tbtn" :class="{ active: editor?.isActive('italic') }" title="Курсив (⌘I)" @click="chain().toggleItalic().run()">
          <span class="material-symbols-outlined">format_italic</span>
        </button>
        <button class="ne-tbtn" :class="{ active: editor?.isActive('underline') }" title="Подчёркнутый (⌘U)" @click="chain().toggleUnderline().run()">
          <span class="material-symbols-outlined">format_underlined</span>
        </button>
        <button class="ne-tbtn" :class="{ active: editor?.isActive('strike') }" title="Зачёркнутый" @click="chain().toggleStrike().run()">
          <span class="material-symbols-outlined">strikethrough_s</span>
        </button>
      </div>

      <span class="ne-tsep" />

      <!-- Выделение цветом: палитра из токенов задач -->
      <div class="ne-tgroup ne-hl">
        <button class="ne-tbtn" :class="{ active: editor?.isActive('highlight') }" title="Выделить цветом" @click="hlOpen = !hlOpen">
          <span class="material-symbols-outlined">format_ink_highlighter</span>
        </button>
        <template v-if="hlOpen">
          <div class="ne-pop-backdrop" @click="hlOpen = false" />
          <div class="ne-pop">
            <button
              v-for="c in TASK_COLORS"
              :key="c.id"
              class="ne-swatch"
              :style="{ background: `var(--tag-${c.id}-surface)`, borderColor: `var(--tag-${c.id}-border)` }"
              :title="c.label"
              @click="setHighlight(c.id)"
            />
            <button class="ne-swatch ne-swatch-off" title="Снять выделение" @click="setHighlight(null)">
              <span class="material-symbols-outlined">format_color_reset</span>
            </button>
          </div>
        </template>
      </div>

      <span class="ne-tsep" />

      <div class="ne-tgroup">
        <button class="ne-tbtn" :class="{ active: editor?.isActive('link') }" title="Ссылка" @click="editLink">
          <span class="material-symbols-outlined">link</span>
        </button>
        <button class="ne-tbtn" :class="{ active: editor?.isActive('code') }" title="Код в строке" @click="chain().toggleCode().run()">
          <span class="material-symbols-outlined">code</span>
        </button>
        <button class="ne-tbtn" :class="{ active: editor?.isActive('codeBlock') }" title="Блок кода" @click="chain().toggleCodeBlock().run()">
          <span class="material-symbols-outlined">terminal</span>
        </button>
        <button v-if="canUpload" class="ne-tbtn" title="Вставить изображение" @click="imageInput?.click()">
          <span class="material-symbols-outlined">image</span>
        </button>
        <input ref="imageInput" type="file" accept="image/*" hidden @change="onImageFile" />
      </div>

      <span class="ne-tsep" />

      <div class="ne-tgroup">
        <button v-if="!editor?.isActive('table')" class="ne-tbtn" title="Вставить таблицу 3×3" @click="chain().insertTable({ rows: 3, cols: 3, withHeaderRow: true }).run()">
          <span class="material-symbols-outlined">table</span>
        </button>
        <template v-else>
          <button class="ne-tbtn" title="Столбец справа" @click="chain().addColumnAfter().run()">
            <span class="material-symbols-outlined">splitscreen_right</span>
          </button>
          <button class="ne-tbtn" title="Строка ниже" @click="chain().addRowAfter().run()">
            <span class="material-symbols-outlined">splitscreen_bottom</span>
          </button>
          <button class="ne-tbtn" title="Удалить столбец" @click="chain().deleteColumn().run()">
            <span class="material-symbols-outlined">variable_remove</span>
          </button>
          <button class="ne-tbtn" title="Удалить строку" @click="chain().deleteRow().run()">
            <span class="material-symbols-outlined">disabled_by_default</span>
          </button>
          <button class="ne-tbtn danger" title="Удалить таблицу" @click="chain().deleteTable().run()">
            <span class="material-symbols-outlined">delete_sweep</span>
          </button>
        </template>
      </div>

      <span class="ne-tsep" />

      <div class="ne-tgroup">
        <button class="ne-tbtn" :class="{ active: editor?.isActive('bulletList') }" title="Маркированный список" @click="chain().toggleBulletList().run()">
          <span class="material-symbols-outlined">format_list_bulleted</span>
        </button>
        <button class="ne-tbtn" :class="{ active: editor?.isActive('orderedList') }" title="Нумерованный список" @click="chain().toggleOrderedList().run()">
          <span class="material-symbols-outlined">format_list_numbered</span>
        </button>
        <button class="ne-tbtn" :class="{ active: editor?.isActive('taskList') }" title="Чекбоксы" @click="chain().toggleTaskList().run()">
          <span class="material-symbols-outlined">checklist</span>
        </button>
      </div>

      <span class="ne-tsep" />

      <div class="ne-tgroup">
        <button class="ne-tbtn" title="Отменить (⌘Z)" :disabled="!editor?.can().undo()" @click="chain().undo().run()">
          <span class="material-symbols-outlined">undo</span>
        </button>
        <button class="ne-tbtn" title="Повторить (⇧⌘Z)" :disabled="!editor?.can().redo()" @click="chain().redo().run()">
          <span class="material-symbols-outlined">redo</span>
        </button>
      </div>
    </div>

    <!-- zoom масштабирует только «лист» (текст, картинки, таблицы); панель
         форматирования остаётся обычного размера. -->
    <EditorContent class="ne-content" :style="zoom !== 1 ? { zoom } : {}" :editor="editor" @contextmenu="onContextMenu" />
  </div>
</template>

<script setup>
// Rich-редактор заметки на TipTap: live-форматирование выделенного текста,
// документ — TipTap JSON (не markdown: highlight-цвета и таблицы в md не
// выражаются). Переиспользуется страницей заметки и публичной ссылкой
// (view — editable:false без панели; edit по ссылке — без загрузки картинок).
import { onBeforeUnmount, ref, watch } from 'vue'
import { EditorContent, useEditor } from '@tiptap/vue-3'
import StarterKit from '@tiptap/starter-kit'
import Underline from '@tiptap/extension-underline'
import Link from '@tiptap/extension-link'
import Image from '@tiptap/extension-image'
import Table from '@tiptap/extension-table'
import TableRow from '@tiptap/extension-table-row'
import TableCell from '@tiptap/extension-table-cell'
import TableHeader from '@tiptap/extension-table-header'
import Highlight from '@tiptap/extension-highlight'
import TaskList from '@tiptap/extension-task-list'
import TaskItem from '@tiptap/extension-task-item'
import Placeholder from '@tiptap/extension-placeholder'
import { TASK_COLORS } from '@/utils/taskColors.js'

const props = defineProps({
  // Документ TipTap (JSON-объект). Компонент не пишет его обратно на каждый
  // ввод — наружу уходит событие change, родитель сам решает, когда сохранять.
  doc: { type: Object, default: null },
  editable: { type: Boolean, default: true },
  placeholder: { type: String, default: 'Начните писать…' },
  // async (file) => url — загрузка картинки; null скрывает кнопку (публичная
  // edit-ссылка: аплоад доступен только владельцу).
  uploadImage: { type: Function, default: null },
  // Масштаб листа (1 = 100%): управляется кнопками страницы заметки.
  zoom: { type: Number, default: 1 },
  // ПКМ на выделении открывает меню действий (событие selection-menu) вместо
  // браузерного. Включает только страница заметки владельца — на публичных
  // ссылках ИИ/создание задач недоступны.
  selectionMenu: { type: Boolean, default: false },
})
const emit = defineEmits(['change', 'blur', 'selection-menu'])

const canUpload = !!props.uploadImage
const hlOpen = ref(false)
const imageInput = ref(null)

const editor = useEditor({
  content: props.doc && Object.keys(props.doc).length ? props.doc : null,
  editable: props.editable,
  extensions: [
    StarterKit.configure({ heading: { levels: [1, 2, 3] } }),
    Underline,
    Link.configure({ openOnClick: !props.editable, autolink: true }),
    Image,
    Table.configure({ resizable: false }),
    TableRow,
    TableHeader,
    TableCell,
    Highlight.configure({ multicolor: true }),
    TaskList,
    TaskItem.configure({ nested: true }),
    Placeholder.configure({ placeholder: props.placeholder }),
  ],
  onUpdate: ({ editor: ed }) => emit('change', ed.getJSON()),
  onBlur: () => emit('blur'),
})

// Смена editable (публичная ссылка узнаёт режим после загрузки).
watch(() => props.editable, (v) => editor.value?.setEditable(v))

// Внешняя замена документа (загрузка с сервера) — не трогаем, если редактор в
// фокусе: иначе перезатрём набираемый текст.
watch(() => props.doc, (doc) => {
  const ed = editor.value
  if (!ed || ed.isFocused || !doc) return
  ed.commands.setContent(doc, false)
})

onBeforeUnmount(() => editor.value?.destroy())

function chain() { return editor.value?.chain().focus() }

function setHighlight(colorId) {
  hlOpen.value = false
  if (!colorId) chain().unsetHighlight().run()
  else chain().setHighlight({ color: `var(--tag-${colorId}-surface)` }).run()
}

function editLink() {
  const ed = editor.value
  if (!ed) return
  const prev = ed.getAttributes('link').href || ''
  // Промпт достаточен для ссылки: значение видно и правится в одном месте.
  const url = window.prompt('Адрес ссылки (пусто — убрать):', prev)
  if (url === null) return
  if (url === '') ed.chain().focus().extendMarkRange('link').unsetLink().run()
  else ed.chain().focus().extendMarkRange('link').setLink({ href: url }).run()
}

// На таче contextmenu приходит от long-press выделения — не перехватываем,
// иначе меню вылезает поверх нативных ручек выделения (правило мессенджера:
// выделение текста отдано браузеру). Путь к действиям там — кнопка тулбара.
const isTouchDevice = window.matchMedia?.('(hover: none) and (pointer: coarse)').matches ?? false

// Текст текущего выделения ('' — выделения нет). Зовётся из шаблона —
// пересчитывается на каждой транзакции редактора.
function selectionText() {
  const ed = editor.value
  if (!ed) return ''
  const { from, to } = ed.state.selection
  if (to <= from) return ''
  return ed.state.doc.textBetween(from, to, '\n', ' ')
}

function selectionHasText() {
  return !!selectionText().trim()
}

// ПКМ на непустом выделении → меню действий с фрагментом; пустое выделение
// оставляет системное меню (правописание, вставка).
function onContextMenu(e) {
  if (!props.selectionMenu || isTouchDevice || !editor.value) return
  const text = selectionText()
  if (!text.trim()) return
  e.preventDefault()
  const { from, to } = editor.value.state.selection
  emit('selection-menu', { x: e.clientX, y: e.clientY, text, from, to })
}

// Кнопка тулбара «Действия с выделенным» — то же меню, якорь под кнопкой.
function openSelectionActions(e) {
  const text = selectionText()
  if (!text.trim() || !editor.value) return
  const { from, to } = editor.value.state.selection
  const r = e.currentTarget.getBoundingClientRect()
  emit('selection-menu', { x: r.left, y: r.bottom + 6, text, from, to })
}

async function onImageFile(e) {
  const file = e.target.files?.[0]
  e.target.value = ''
  if (!file || !props.uploadImage) return
  const url = await props.uploadImage(file)
  if (url) chain().setImage({ src: url }).run()
}

defineExpose({ editor })
</script>

<style scoped>
.ne { display: flex; flex-direction: column; min-height: 0; }

/* Панель форматирования — sticky, плотное стекло, на мобиле скроллится горизонтально. */
.ne-toolbar {
  position: sticky;
  top: 0;
  z-index: 5;
  display: flex;
  align-items: center;
  gap: 2px;
  flex-wrap: wrap;
  padding: 6px 8px;
  border-radius: var(--radius-lg);
  background: var(--acrylic-bg-strong);
  -webkit-backdrop-filter: var(--acrylic-blur);
  backdrop-filter: var(--acrylic-blur);
  border: 1px solid var(--acrylic-border);
}
.ne-tgroup { display: inline-flex; align-items: center; gap: 2px; }
.ne-tsep {
  width: 1px;
  height: 20px;
  margin: 0 4px;
  background: var(--color-outline-variant);
  flex-shrink: 0;
}
.ne-tbtn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 30px;
  height: 30px;
  padding: 0 5px;
  border: none;
  border-radius: var(--radius-sm);
  background: none;
  color: var(--color-text-dim);
  font-size: 12.5px;
  font-weight: 700;
  cursor: pointer;
}
.ne-tbtn .material-symbols-outlined { font-size: 19px; }
.ne-tbtn:hover { background: color-mix(in oklch, var(--color-primary) 10%, transparent); color: var(--color-text); }
.ne-tbtn.active { background: color-mix(in oklch, var(--color-primary) 16%, transparent); color: var(--color-primary); }
.ne-tbtn:disabled { opacity: 0.4; cursor: default; }
.ne-tbtn.danger:hover { background: color-mix(in oklch, var(--color-error) 12%, transparent); color: var(--color-error); }

/* Кнопка действий с выделенным: primary-акцент, чтобы отличалась от
   форматирования; каретка намекает на выпадающее меню. */
.ne-selbtn:not(:disabled) { color: var(--color-primary); }
.ne-selbtn-caret { font-size: 16px !important; margin-left: -6px; }

/* Палитра выделения */
.ne-hl { position: relative; }
.ne-pop-backdrop { position: fixed; inset: 0; z-index: 9; }
.ne-pop {
  position: absolute;
  top: calc(100% + 6px);
  left: 0;
  z-index: 10;
  display: flex;
  gap: 6px;
  padding: 8px;
  border-radius: var(--radius-md);
  background: var(--acrylic-bg);
  -webkit-backdrop-filter: var(--acrylic-blur);
  backdrop-filter: var(--acrylic-blur);
  border: 1px solid var(--acrylic-border);
  box-shadow: var(--shadow-md);
}
.ne-swatch {
  width: 24px;
  height: 24px;
  border-radius: var(--radius-sm);
  border: 1px solid;
  cursor: pointer;
  padding: 0;
}
.ne-swatch-off {
  display: grid;
  place-items: center;
  background: var(--color-surface);
  border-color: var(--color-outline-variant);
  color: var(--color-text-dim);
}
.ne-swatch-off .material-symbols-outlined { font-size: 16px; }

/* Содержимое */
.ne-content { flex: 1; min-height: 0; }
.ne-content :deep(.tiptap) {
  outline: none;
  min-height: 240px;
  padding: 20px 18px 40px;
  color: var(--color-text);
  font-size: 15px;
  line-height: 1.65;
}
.ne-content :deep(.tiptap ul),
.ne-content :deep(.tiptap ol) {
  padding-left: 28px;
  margin: 8px 0;
}
.ne-content :deep(.tiptap li) { margin: 3px 0; }
.ne-content :deep(.tiptap li p) { margin: 0; }
.ne-content :deep(.tiptap h1) { font-size: 26px; margin: 20px 0 8px; }
.ne-content :deep(.tiptap h2) { font-size: 21px; margin: 16px 0 6px; }
.ne-content :deep(.tiptap h3) { font-size: 17px; margin: 12px 0 4px; }
.ne-content :deep(.tiptap p) { margin: 0 0 6px; }
.ne-content :deep(.tiptap a) { color: var(--color-primary); }
.ne-content :deep(.tiptap code) {
  padding: 1px 5px;
  border-radius: var(--radius-sm);
  background: var(--color-surface-high);
  font-size: 0.9em;
}
.ne-content :deep(.tiptap pre) {
  padding: 12px 14px;
  border-radius: var(--radius-md);
  background: var(--color-surface-high);
  overflow-x: auto;
}
.ne-content :deep(.tiptap pre code) { padding: 0; background: none; }
.ne-content :deep(.tiptap blockquote) {
  margin: 8px 0;
  padding: 4px 14px;
  border-left: 3px solid var(--color-primary);
  color: var(--color-text-dim);
}
.ne-content :deep(.tiptap img) {
  max-width: 100%;
  border-radius: var(--radius-md);
}
.ne-content :deep(.tiptap img.ProseMirror-selectednode) { outline: 2px solid var(--color-primary); }
.ne-content :deep(.tiptap mark) { border-radius: 3px; padding: 0 2px; color: inherit; }

.ne-content :deep(.tiptap table) {
  border-collapse: collapse;
  width: 100%;
  margin: 10px 0;
  table-layout: fixed;
}
.ne-content :deep(.tiptap th),
.ne-content :deep(.tiptap td) {
  border: 1px solid var(--color-outline-variant);
  padding: 6px 10px;
  vertical-align: top;
  word-break: break-word;
}
.ne-content :deep(.tiptap th) { background: var(--color-surface-high); font-weight: 700; text-align: left; }
.ne-content :deep(.tiptap .selectedCell) { background: color-mix(in oklch, var(--color-primary) 10%, transparent); }

.ne-content :deep(.tiptap ul[data-type='taskList']) { list-style: none; padding-left: 4px; }
.ne-content :deep(.tiptap ul[data-type='taskList'] li) { display: flex; gap: 8px; }
.ne-content :deep(.tiptap ul[data-type='taskList'] input) { accent-color: var(--color-primary); margin-top: 5px; }
.ne-content :deep(.tiptap ul[data-type='taskList'] li > div) { flex: 1; min-width: 0; }

/* Плейсхолдер пустого документа */
.ne-content :deep(.tiptap p.is-editor-empty:first-child::before) {
  content: attr(data-placeholder);
  float: left;
  height: 0;
  pointer-events: none;
  color: var(--color-text-dim);
  opacity: 0.7;
}

@media (max-width: 768px) {
  /* Панель — одна строка с горизонтальным скроллом. */
  .ne-toolbar { flex-wrap: nowrap; overflow-x: auto; scrollbar-width: none; }
  .ne-toolbar::-webkit-scrollbar { display: none; }
}
</style>
