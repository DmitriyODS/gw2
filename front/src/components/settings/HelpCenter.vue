<template>
  <div class="hc">
    <SearchField
      v-model="query"
      placeholder="О чём рассказать? — «окно», «юнит», «напоминание»"
      :collapsible="false"
    />

    <Transition name="hc-swap" mode="out-in">
      <!-- Открытая статья -->
      <AppStack v-if="article" key="article" :gap="12">
        <AppButton variant="text" size="sm" icon="arrow_back" label="К списку" @click="article = null" />

        <AppCard>
          <h3 class="hc-title">{{ article.title }}</h3>
          <p class="hc-sub">{{ article.subtitle }}</p>
          <p v-for="(p, i) in article.text" :key="`p${i}`" class="hc-p">{{ p }}</p>

          <template v-if="article.tips?.length">
            <h4 class="hc-h4">Полезно знать</h4>
            <ul class="hc-list">
              <li v-for="(t, i) in article.tips" :key="`t${i}`">{{ t }}</li>
            </ul>
          </template>

          <AppButton
            v-if="article.route"
            class="hc-cta"
            variant="filled"
            icon="arrow_forward"
            :label="article.cta || 'Открыть раздел'"
            @click="goTo(article.route)"
          />
        </AppCard>
      </AppStack>

      <!-- Каталог -->
      <AppStack v-else key="list" :gap="16">
        <AppCard v-if="!query">
          <AppRow
            title="Не нашли ответ — напишите нам"
            hint="Личный чат с командой разработки: вопросы, идеи, сообщения об ошибках"
            icon="support_agent"
            clickable
            @click="openSupport"
          />
        </AppCard>

        <div v-for="group in groups" :key="group.key" class="hc-group">
          <div class="hc-group-label">{{ group.label }}</div>
          <AppCard :gap="0" class="hc-group-card">
            <AppRow
              v-for="a in group.articles"
              :key="a.id"
              :title="a.title"
              :hint="a.subtitle"
              :icon="a.icon"
              plain
              clickable
              @click="article = a"
            />
          </AppCard>
        </div>

        <EmptyState
          v-if="!groups.length"
          size="sm"
          icon="search_off"
          title="Ничего не нашли"
          subtitle="Попробуйте другие слова — например, «окно», «время» или «звонок»."
        />
      </AppStack>
    </Transition>
  </div>
</template>

<script setup>
/* Справка по платформе: каталог коротких статей о том, как устроены разделы и
   каркас-«рабочий стол». Содержимое зависит от прав: без активной компании
   компанийных статей нет, системные — только супер-админу.

   Интерактивного тура тут нет и не будет: он вёл по прежней навигации, а
   объяснять окна и плитки текстом честнее, чем стрелкой по экрану. */
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { usePermission, ROLES } from '@/composables/usePermission.js'
import { useCompanySettings } from '@/composables/useCompanySettings.js'
import { useMessengerStore } from '@/stores/messenger.js'
import { helpGroups } from '@/utils/helpArticles.js'
import AppButton from '@/components/ui/AppButton.vue'
import AppCard from '@/components/ui/AppCard.vue'
import AppRow from '@/components/ui/AppRow.vue'
import AppStack from '@/components/ui/AppStack.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import SearchField from '@/components/common/SearchField.vue'

const route = useRoute()
const router = useRouter()
const messenger = useMessengerStore()
const { isAtLeast, isSuperAdmin, canManageCompanies } = usePermission()
const { usesGroove } = useCompanySettings()

// Роль ≥ Сотрудник = есть активная компания (у супер-админа её нет).
const hasCompany = () => isAtLeast(ROLES.EMPLOYEE)

const query = ref('')
const article = ref(null)

const GROUPS = computed(() => helpGroups({
  hasCompany: hasCompany(),
  isSuperAdmin: isSuperAdmin(),
  canManageCompanies: canManageCompanies(),
  isManager: isAtLeast(ROLES.MANAGER),
  usesGroove: usesGroove.value,
}))

// Ссылка из поиска Hola — ?article=<id> открывает статью сразу, минуя список
// (та же логика, что у deep-link вкладки в CompanyManagePanel).
onMounted(() => {
  const id = route.query.article
  if (!id) return
  for (const g of GROUPS.value) {
    const found = g.articles.find((a) => a.id === id)
    if (found) { article.value = found; return }
  }
})

const groups = computed(() => {
  const q = query.value.trim().toLowerCase()
  if (!q) return GROUPS.value
  return GROUPS.value
    .map((g) => ({
      ...g,
      articles: g.articles.filter((a) => (
        [a.title, a.subtitle, ...(a.text || []), ...(a.tips || [])].join(' ').toLowerCase().includes(q)
      )),
    }))
    .filter((g) => g.articles.length)
})

function goTo(path) {
  router.push(path)
}

/* Поддержка — тот же dev-чат мессенджера: отдельной формы обращений нет,
   переписка с командой ведётся там же, где остальная. */
async function openSupport() {
  try {
    // Адрес ведёт СРАЗУ в чат: раздел мессенджера считает открытую переписку по
    // нему, и просто «/messenger» тут же снял бы активный чат.
    const id = await messenger.openDevChat()
    router.push(`/messenger/${id}`)
  } catch {
    router.push('/messenger')
  }
}
</script>

<style scoped>
.hc { display: flex; flex-direction: column; gap: 16px; }

.hc-group { display: flex; flex-direction: column; gap: 8px; }

.hc-group-label {
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  color: var(--color-text-dim);
  padding-left: 4px;
}

.hc-group-card { padding: 6px; }

.hc-title { margin: 0; font-size: 18px; font-weight: 700; }
.hc-sub { margin: 0; color: var(--color-text-dim); font-size: 13.5px; }
.hc-p { margin: 0; font-size: 14px; line-height: 1.6; }
.hc-h4 { margin: 4px 0 0; font-size: 13px; font-weight: 700; color: var(--color-text-dim); }

.hc-list {
  margin: 0;
  padding-left: 18px;
  display: flex;
  flex-direction: column;
  gap: 6px;
  font-size: 13.5px;
  line-height: 1.55;
  color: var(--color-text-dim);
}

.hc-cta { align-self: flex-start; margin-top: 4px; }

.hc-swap-enter-active, .hc-swap-leave-active { transition: opacity 0.15s, transform 0.15s; }
.hc-swap-enter-from, .hc-swap-leave-to { opacity: 0; transform: translateY(4px); }
</style>
