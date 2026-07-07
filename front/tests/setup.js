// Глобальная оснастка юнит-тестов: заглушки тяжёлых внешних компонентов
// (PrimeVue, кастомные диалоги, router-link), чтобы mount не тянул за собой
// весь UI-стек. material-иконки — это просто <span>-текст, их не глушим.
import { config } from '@vue/test-utils'

// Заглушка диалога-обёртки: рендерит и тело (default), и подвал (footer) —
// именно в них живёт тестируемая разметка форм. Как и реальный AppDialog
// (PrimeVue Dialog внутри), тело не рендерится, пока modelValue=false —
// иначе закрытые вложенные диалоги (например «Перенести в другой ежедневник»)
// монтируют настоящие тяжёлые PrimeVue-компоненты (Select и т.п.) без
// плагина PrimeVue в тестовом окружении.
const AppDialogStub = {
  name: 'AppDialog',
  props: ['modelValue', 'title', 'subtitle', 'icon', 'tone', 'size', 'busy', 'closable'],
  emits: ['update:modelValue'],
  template: `<div v-if="modelValue" class="app-dialog-stub"><slot /><slot name="footer" /></div>`,
}

const InputStub = (cls) => ({
  props: ['modelValue', 'placeholder', 'clearable'],
  emits: ['update:modelValue'],
  template: `<input class="${cls}" :value="modelValue" @input="$emit('update:modelValue', $event.target.value)" />`,
})

const RouterLinkStub = {
  props: ['to'],
  template: `<a :href="typeof to === 'string' ? to : '#'"><slot /></a>`,
}

config.global.stubs = {
  AppDialog: AppDialogStub,
  DatePicker: InputStub('datepicker-stub'),
  TimePicker: InputStub('timepicker-stub'),
  ConfirmDialog: {
    props: ['visible', 'header', 'message', 'confirmLabel', 'dangerConfirm'],
    emits: ['confirm', 'cancel'],
    template: `<div v-if="visible" class="confirm-stub"><slot /></div>`,
  },
  RouterLink: RouterLinkStub,
  'router-link': RouterLinkStub,
}
