<template>
  <!-- Форма по внешней ссылке: страница для человека без аккаунта, но само
       заполнение — тот же компонент, что и в разделе (components/forms/
       FormFill.vue). Отличий два: данные берутся по коду ссылки, а ответ
       уходит публичной ручкой. -->
  <AppPage
    :title="view?.form?.title || 'Форма'"
    show-title
    :loading="booting"
    @narrow-change="narrow = $event"
  >
    <template #footer>
      <BrandWordmark :size="15" />
    </template>

    <!-- Ссылка «только для своих»: формы гость не увидит, пока не войдёт.
         Это не поломка, поэтому и вид другой — не отказ, а приглашение. -->
    <EmptyState
      v-if="needAuth"
      icon="lock"
      tone="soft"
      title="Нужен вход в аккаунт"
      subtitle="Автор открыл эту форму только для тех, кто вошёл в Groove Work. Войдите или заведите аккаунт — и ссылка откроется."
    >
      <AppButton variant="filled" icon="login" label="Войти" @click="goLogin" />
    </EmptyState>

    <EmptyState
      v-else-if="error"
      icon="link_off"
      tone="soft"
      title="Ссылка не открывается"
      :subtitle="error"
    />

    <div v-else-if="view" class="sf">
      <FormFill
        :form="view.form"
        :can-respond="view.can_respond"
        :reason="view.reason || ''"
        :mine="view.mine || null"
        :answer-keys="view.answer_keys || null"
        :booking="view.booking || {}"
        :ask-name="!authStore.user"
        :submit="submit"
        :upload="upload"
        @error="notif.error($event)"
        @submitted="onSubmitted"
      />
    </div>
  </AppPage>
</template>

<script setup>
/* Публичная страница заполнения формы (/form/:code).

   Открывается и гостем, и вошедшим: у вошедшего ответ подписывается его
   аккаунтом, гость представляется сам (если автор спрашивает имя). Черновик по
   ссылке не открывается вовсе — пока форма не запущена, её содержимое дело
   автора, и сервер отвечает отказом. */
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import AppButton from '@/components/ui/AppButton.vue'
import AppPage from '@/components/ui/AppPage.vue'
import BrandWordmark from '@/components/common/BrandWordmark.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import FormFill from '@/components/forms/FormFill.vue'
import { getSharedForm, submitSharedResponse, uploadSharedFile } from '@/api/forms.js'
import { useAuthStore } from '@/stores/auth.js'
import { useNotificationsStore } from '@/stores/notifications.js'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
const notif = useNotificationsStore()

const code = computed(() => String(route.params.code || ''))
const view = ref(null)
const booting = ref(true)
const error = ref('')
const needAuth = ref(false)
const narrow = ref(false)

onMounted(load)

async function load() {
  booting.value = true
  error.value = ''
  needAuth.value = false
  try {
    view.value = await getSharedForm(code.value)
  } catch (e) {
    if (e?.error === 'SHARE_AUTH_REQUIRED') needAuth.value = true
    else error.value = e?.message || 'Ссылка не найдена или отозвана'
  } finally {
    booting.value = false
  }
}

const submit = (payload) => submitSharedResponse(code.value, payload)

// По ссылке файл едет одним запросом: чанковые ручки требуют входа, а
// вложения анкет — обычные документы и снимки.
const upload = (file) => uploadSharedFile(code.value, file)

async function onSubmitted() {
  notif.success('Ответ отправлен')
  // Перечитываем: форма могла набрать потолок ответов, и повторная отправка
  // должна честно закрыться.
  view.value = await getSharedForm(code.value)
}

function goLogin() {
  router.push({ path: '/login', query: { redirect: route.fullPath } })
}
</script>

<style scoped>
.sf { max-width: 760px; margin: 0 auto; width: 100%; }
</style>
