<template>
  <div class="ann-card vibe-card" :class="{ today: item.is_today }" @click="$emit('edit')">
    <div class="ann-date">
      <span class="date-month">{{ month }}</span>
      <span class="date-day">{{ day }}</span>
    </div>
    <div class="ann-info">
      <div class="ann-name">{{ item.name }}</div>
      <div class="ann-sub">
        {{ item.event_date }}
        <span v-if="item.repeat_yearly">· 每年重复</span>
        <span v-if="item.remind_enabled">· 提前 {{ item.remind_days_before }} 天提醒</span>
      </div>
    </div>
    <div class="countdown" :class="{ today: item.is_today, passed: item.days_left < 0 }">
      {{ countdownText(item.days_left) }}
    </div>
    <button class="ann-more" @click.stop="$emit('remove')" title="删除">
      <el-icon><Delete /></el-icon>
    </button>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { countdownText } from '@/utils/format'
import type { Anniversary } from '@/types'

const props = defineProps<{ item: Anniversary }>()
defineEmits<{ (e: 'edit'): void; (e: 'remove'): void }>()

const month = computed(() => props.item.event_date.slice(5, 7))
const day = computed(() => props.item.event_date.slice(8, 10))
</script>

<style scoped>
.ann-card {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 18px;
  cursor: pointer;
}
.ann-card.today {
  border-color: rgba(255, 180, 84, 0.55);
  background: linear-gradient(0deg, #fffaf2, #ffffff);
}
.ann-date {
  width: 54px;
  height: 58px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  border-radius: 16px;
  background: var(--vibe-primary-soft);
  color: var(--vibe-primary);
  flex-shrink: 0;
}
.date-month {
  font-size: 11px;
  font-weight: 700;
}
.date-day {
  font-size: 22px;
  font-weight: 800;
  line-height: 1.1;
}
.ann-info {
  flex: 1;
  min-width: 0;
}
.ann-name {
  font-size: 15px;
  font-weight: 600;
  margin-bottom: 4px;
}
.ann-sub {
  font-size: 12.5px;
  color: var(--vibe-text-secondary);
}
.countdown {
  padding: 6px 14px;
  border-radius: 999px;
  font-size: 13px;
  font-weight: 700;
  color: var(--vibe-primary);
  background: var(--vibe-primary-soft);
  white-space: nowrap;
}
.countdown.today {
  color: #d48806;
  background: rgba(255, 180, 84, 0.16);
}
.countdown.passed {
  color: var(--vibe-text-tertiary);
  background: var(--vibe-surface-2);
}
.ann-more {
  border: none;
  background: transparent;
  color: var(--vibe-text-tertiary);
  cursor: pointer;
  padding: 6px;
  border-radius: 999px;
  transition: all 0.2s var(--vibe-ease);
}
.ann-more:hover {
  color: var(--vibe-danger);
  background: rgba(255, 107, 122, 0.1);
}
</style>
