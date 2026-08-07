<template>
  <!-- Перенос компании: выгрузка одним файлом и подъём из него новой компании
       (на другом сервере, в другом аккаунте или копией рядом). -->
  <AppCard
    title="Перенос компании"
    hint="Выгрузка одним файлом: работа, справочники, содержимое разделов и вложенные файлы."
  >
    <AppInfoBar
      icon="info"
      message="Сотрудники и питомцы в выгрузку не входят — аккаунты остаются на месте. При подъёме архива авторы записей ищутся по логину; кого не нашли, записи достаются тому, кто импортирует."
    />

    <AppRow
      title="Скачать архив"
      hint="Задачи, юниты, отделы и этапы, метки, реестры, календари и портал вместе с файлами."
    >
      <AppButton icon="download" label="Выгрузить" :loading="busy" @click="download" />
    </AppRow>

    <AppRow
      title="Поднять из архива"
      hint="Создаст НОВУЮ компанию с этим содержимым. Существующие данные не трогаются."
    >
      <AppButton icon="upload" label="Загрузить файл" @click="pick" />
    </AppRow>

    <input
      ref="fileInput"
      type="file"
      accept=".gwcompany,.zip"
      class="tr-file"
      @change="onFile"
    >

    <AppDialog
      v-if="confirmOpen"
      v-model="confirmOpen"
      size="sm"
      title="Поднять компанию из архива?"
      :subtitle="file?.name"
      :busy="busy"
      :actions="[
        { kind: 'cancel', label: 'Отмена', disabled: busy },
        { kind: 'confirm', label: 'Загрузить', icon: 'upload', disabled: busy },
      ]"
      @confirm="upload"
      @cancel="confirmOpen = false"
    >
      <AppStack :gap="10">
        <p class="tr-note">
          Появится новая компания — вы станете её создателем. Имя можно задать своё;
          пустое поле возьмёт имя из архива.
        </p>
        <InputText v-model="name" class="tr-input" placeholder="Название новой компании" />
      </AppStack>
    </AppDialog>
  </AppCard>
</template>

<script setup>
import { ref } from 'vue'
import InputText from 'primevue/inputtext'
import AppButton from '@/components/ui/AppButton.vue'
import AppCard from '@/components/ui/AppCard.vue'
import AppDialog from '@/components/ui/AppDialog.vue'
import AppInfoBar from '@/components/ui/AppInfoBar.vue'
import AppRow from '@/components/ui/AppRow.vue'
import AppStack from '@/components/ui/AppStack.vue'
import { exportCompany, importCompany } from '@/api/companies.js'
import { useNotificationsStore } from '@/stores/notifications.js'

const props = defineProps({
  companyId: { type: Number, required: true },
  companyName: { type: String, default: '' },
})

const emit = defineEmits(['imported'])

const notif = useNotificationsStore()
const busy = ref(false)
const fileInput = ref(null)
const file = ref(null)
const name = ref('')
const confirmOpen = ref(false)

async function download() {
  busy.value = true
  try {
    const res = await exportCompany(props.companyId)
    const blob = res instanceof Blob ? res : await res.blob()
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    // Имя из Content-Disposition знает сервер, но читать его из ответа не
    // обязательно: то же имя собирается здесь из названия компании и даты.
    a.download = `${slug(props.companyName)}-${new Date().toISOString().slice(0, 10)}.gwcompany`
    document.body.appendChild(a)
    a.click()
    a.remove()
    URL.revokeObjectURL(url)
    notif.success('Компания выгружена')
  } catch (e) {
    notif.error(e.message || 'Не удалось выгрузить компанию')
  } finally {
    busy.value = false
  }
}

function pick() {
  fileInput.value?.click()
}

function onFile(event) {
  const picked = event.target.files?.[0]
  event.target.value = ''
  if (!picked) return
  file.value = picked
  name.value = ''
  confirmOpen.value = true
}

async function upload() {
  if (!file.value) return
  busy.value = true
  try {
    const res = await importCompany(file.value, name.value.trim())
    confirmOpen.value = false
    notif.success(`Компания «${res.company_name}» поднята из архива`)
    emit('imported', res)
  } catch (e) {
    notif.error(e.message || 'Не удалось поднять компанию из архива')
  } finally {
    busy.value = false
  }
}

function slug(value) {
  const out = (value || '').toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-+|-+$/g, '')
  return out || 'company'
}
</script>

<style scoped>
.tr-file { display: none; }

.tr-note {
  margin: 0;
  color: var(--color-text-dim);
  font-size: 0.9rem;
}

.tr-input { width: 100%; }
</style>
