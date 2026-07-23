// Картинка заметки с ресайзом и выравниванием. Расширяет штатный
// @tiptap/extension-image двумя атрибутами (width — % ширины блока, align —
// left|center|right) и подменяет рендер на Vue-NodeView с ручкой ресайза,
// слайдером ширины (удобно на тач) и кнопками выравнивания. Атрибуты
// сериализуются в HTML (style width + data-align) — картинка сохраняет
// размер и положение и в режиме чтения (публичная ссылка), и при экспорте.
import Image from '@tiptap/extension-image'
import { VueNodeViewRenderer } from '@tiptap/vue-3'
import ImageNodeView from './ImageNodeView.vue'

export const ResizableImage = Image.extend({
  addAttributes() {
    return {
      ...this.parent?.(),
      width: {
        default: null,
        parseHTML: (el) => el.style.width || el.getAttribute('width') || null,
        renderHTML: (attrs) => (attrs.width ? { style: `width: ${attrs.width}` } : {}),
      },
      align: {
        default: 'left',
        parseHTML: (el) => el.getAttribute('data-align') || 'left',
        renderHTML: (attrs) => ({ 'data-align': attrs.align || 'left' }),
      },
    }
  },
  addNodeView() {
    return VueNodeViewRenderer(ImageNodeView)
  },
})
