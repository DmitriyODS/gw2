<template>
  <div class="permission-matrix">
    <table class="matrix-table">
      <thead>
        <tr>
          <th class="section-col">Раздел</th>
          <th
            v-for="bit in maxBits"
            :key="bit"
            class="bit-col"
          ></th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="(section, sIdx) in SECTIONS" :key="sIdx">
          <td class="section-name">{{ section.name }}</td>
          <td
            v-for="bit in maxBits"
            :key="bit"
            class="bit-cell"
          >
            <template v-if="bit < section.bits.length">
              <div class="bit-item">
                <Checkbox
                  :model-value="getBit(sIdx, section.bits[bit].bit)"
                  @update:model-value="setBit(sIdx, section.bits[bit].bit, $event)"
                  :binary="true"
                />
                <label class="bit-label" @click="setBit(sIdx, section.bits[bit].bit, !getBit(sIdx, section.bits[bit].bit))">
                  {{ section.bits[bit].label }}
                </label>
              </div>
            </template>
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import Checkbox from 'primevue/checkbox'

const props = defineProps({
  modelValue: {
    type: [Number, String, BigInt],
    default: 0
  }
})

const emit = defineEmits(['update:modelValue'])

const SECTIONS = [
  {
    name: 'Задачи',
    bits: [
      { bit: 0, label: 'Просмотр' },
      { bit: 1, label: 'Создание своих' },
      { bit: 2, label: 'Редакт. своих' },
      { bit: 3, label: 'Удал. своих' },
      { bit: 4, label: 'Создание чужих' },
      { bit: 5, label: 'Редакт. чужих' },
      { bit: 6, label: 'Удал. чужих' }
    ]
  },
  {
    name: 'Юниты',
    bits: [
      { bit: 0, label: 'Просмотр' },
      { bit: 1, label: 'Создание своих' },
      { bit: 2, label: 'Редакт. своих' },
      { bit: 3, label: 'Удал. своих' },
      { bit: 4, label: 'Создание чужих' },
      { bit: 5, label: 'Редакт. чужих' },
      { bit: 6, label: 'Удал. чужих' }
    ]
  },
  {
    name: 'Пользователи',
    bits: [
      { bit: 0, label: 'Просмотр' },
      { bit: 1, label: 'Создание' },
      { bit: 2, label: 'Изменение' },
      { bit: 3, label: 'Удаление' }
    ]
  },
  {
    name: 'Роли',
    bits: [
      { bit: 0, label: 'Просмотр' },
      { bit: 1, label: 'Создание' },
      { bit: 2, label: 'Изменение' },
      { bit: 3, label: 'Удаление' },
      { bit: 4, label: 'Назначение' }
    ]
  },
  {
    name: 'Статистика',
    bits: [
      { bit: 0, label: 'Просмотр' },
      { bit: 1, label: 'Стат. пользоват.' },
      { bit: 2, label: 'Выгрузка общей' },
      { bit: 3, label: 'Выгрузка пользов.' }
    ]
  },
  {
    name: 'Копирование',
    bits: [
      { bit: 0, label: 'Просмотр' },
      { bit: 1, label: 'Выгрузка' },
      { bit: 2, label: 'Загрузка' }
    ]
  },
  {
    name: 'Отделы',
    bits: [
      { bit: 0, label: 'Просмотр' },
      { bit: 1, label: 'Создание' },
      { bit: 2, label: 'Изменение' },
      { bit: 3, label: 'Удаление' }
    ]
  },
  {
    name: 'Типы юнитов',
    bits: [
      { bit: 0, label: 'Просмотр' },
      { bit: 1, label: 'Создание' },
      { bit: 2, label: 'Изменение' },
      { bit: 3, label: 'Удаление' }
    ]
  }
]

const maxBits = computed(() => Math.max(...SECTIONS.map(s => s.bits.length)))

function getBit(section, bit) {
  const val = props.modelValue
  if (val === null || val === undefined) return false
  const sectionByte = (BigInt(val) >> BigInt(section * 8)) & 0xFFn
  return Boolean(sectionByte & BigInt(1 << bit))
}

function setBit(section, bit, value) {
  let access = BigInt(props.modelValue || 0)
  const bitPos = BigInt(section * 8 + bit)
  if (value) {
    access |= (1n << bitPos)
  } else {
    access &= ~(1n << bitPos)
  }
  emit('update:modelValue', access.toString())
}
</script>

<style scoped>
.permission-matrix {
  overflow-x: auto;
}

.matrix-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 13px;
}

.matrix-table th,
.matrix-table td {
  padding: 6px 8px;
  border-bottom: 1px solid var(--gw-border);
  vertical-align: middle;
}

.section-col {
  min-width: 120px;
  text-align: left;
  font-weight: 700;
  color: var(--gw-text-secondary);
  font-size: 11px;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.bit-col {
  min-width: 120px;
}

.section-name {
  font-weight: 600;
  color: var(--gw-text);
  white-space: nowrap;
}

.bit-cell {
  padding: 4px 8px;
}

.bit-item {
  display: flex;
  align-items: center;
  gap: 6px;
}

.bit-label {
  font-size: 12px;
  color: var(--gw-text);
  cursor: pointer;
  user-select: none;
  white-space: nowrap;
}

.matrix-table tr:hover td {
  background: var(--gw-bg);
}
</style>
