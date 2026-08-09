<template>
  <div class="hr" :class="{ compact }">
    <section v-for="section in sections" :key="section.key" class="hr-section">
      <h3 v-if="section.label" class="hr-title">{{ section.label }}</h3>
      <button
        v-for="item in section.items"
        :key="item.key"
        class="hr-item"
        :class="{ active: activeKey === item.key, command: item.command, web: item.web }"
        type="button"
        @click="$emit('pick', item)"
        @mouseenter="$emit('hover', item)"
      >
        <img v-if="item.avatar" class="hr-avatar" :src="item.avatar" :alt="item.title" />
        <span v-else class="hr-icon material-symbols-outlined">{{ item.icon }}</span>
        <span class="hr-text">
          <span class="hr-item-title">{{ item.title }}</span>
          <span v-if="item.subtitle" class="hr-item-sub">{{ item.subtitle }}</span>
        </span>
        <span class="hr-go material-symbols-outlined">{{ goIcon(item) }}</span>
      </button>
    </section>
  </div>
</template>

<script setup>
/* Общий список выдачи Hola: им рисуются и результаты поиска, и каталог
   быстрых команд — строки у них одинаковые, различается только источник. */
defineProps({
  /* [{ key, label?, items: [{ key, icon, avatar?, title, subtitle?, command?, web? }] }] */
  sections: { type: Array, required: true },
  activeKey: { type: String, default: null },
  /** Строка в одну линию: панель зажата клавиатурой — важнее число видимых. */
  compact: { type: Boolean, default: false },
})

defineEmits(['pick', 'hover'])

function goIcon(item) {
  if (item.web) return 'open_in_new'
  return item.command ? 'bolt' : 'arrow_outward'
}
</script>

<style scoped>
.hr { display: flex; flex-direction: column; gap: 14px; }

.hr-section { display: flex; flex-direction: column; gap: 6px; }

.hr-title {
  margin: 0;
  padding: 0 4px;
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0.4px;
  text-transform: uppercase;
  color: var(--color-text-dim);
}

.hr-item {
  display: flex;
  align-items: center;
  gap: 12px;
  width: 100%;
  padding: 12px 14px;
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-lg);
  background: var(--glass-bg);
  box-shadow: var(--glass-edge);
  color: var(--color-text);
  text-align: left;
  cursor: pointer;
  transition: border-color 0.14s, background 0.14s;
}

.hr-item:hover,
.hr-item.active {
  border-color: color-mix(in oklch, var(--color-primary) 34%, var(--acrylic-border));
  background: color-mix(in oklch, var(--color-primary) 8%, var(--glass-bg));
}

.hr-icon { font-size: 21px; color: var(--color-text-dim); flex-shrink: 0; }
.hr-item:hover .hr-icon,
.hr-item.active .hr-icon { color: var(--color-primary); }

/* Команда — не переход, а действие: её значок подсвечен и в покое. */
.hr-item.command .hr-icon,
.hr-item.command .hr-go { color: var(--color-primary); opacity: 1; }

.hr-avatar {
  width: 26px;
  height: 26px;
  flex-shrink: 0;
  border-radius: 50%;
  object-fit: cover;
}

.hr-text { flex: 1; min-width: 0; display: flex; flex-direction: column; }

.hr-item-title {
  font-size: 14.5px;
  font-weight: 500;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.hr-item-sub {
  font-size: 12.5px;
  color: var(--color-text-dim);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.hr-go { font-size: 18px; color: var(--color-text-dim); opacity: 0; }
.hr-item:hover .hr-go,
.hr-item.active .hr-go { opacity: 1; }

/* ── Компактная выдача ──
   Панель зажата клавиатурой: строка складывается в одну линию (подпись уходит
   за названием), поля урезаны, стрелка перехода убрана — на тач-экране она и
   так появлялась только по наведению. Строка с ~54px до ~34px: в тех же
   двухстах пикселях видно шесть результатов вместо трёх. */
.hr.compact { gap: 8px; }
.hr.compact .hr-section { gap: 3px; }
.hr.compact .hr-title { font-size: 10.5px; letter-spacing: 0.3px; }

.hr.compact .hr-item {
  gap: 8px;
  padding: 7px 10px;
  border-radius: var(--radius-md);
}

.hr.compact .hr-icon { font-size: 18px; }
.hr.compact .hr-avatar { width: 20px; height: 20px; }
.hr.compact .hr-go { display: none; }

.hr.compact .hr-text {
  flex-direction: row;
  align-items: baseline;
  gap: 8px;
}

.hr.compact .hr-item-title { flex: 0 1 auto; font-size: 13.5px; }
.hr.compact .hr-item-sub { flex: 1 1 auto; min-width: 0; font-size: 11.5px; }
</style>
