<template>
  <div class="habit-card vibe-card" :class="{ checked }">
    <div class="habit-icon">{{ icon }}</div>
    <div class="habit-info">
      <div class="habit-name">{{ habit.name }}</div>
      <div class="habit-streak">
        <el-icon :size="13"><Fire /></el-icon>
        连续 {{ streak }} 天
      </div>
    </div>
    <button class="check-btn" :class="{ checked }" :title="checked ? '取消今日打卡' : '今日打卡'" @click="onClick">
      <transition name="pop">
        <el-icon v-if="checked" :size="16" class="bounce"><Check /></el-icon>
        <span v-else class="check-text">打</span>
      </transition>
    </button>
    <button class="remove-btn" title="删除习惯" @click="$emit('remove')">
      <el-icon :size="14"><Close /></el-icon>
    </button>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { Habit } from '@/types'

const props = defineProps<{
  habit: Habit
  checked: boolean
  streak: number
}>()
const emit = defineEmits<{ (e: 'toggle'): void; (e: 'remove'): void }>()

const icon = computed(() => props.habit.icon || '🎯')

function onClick() {
  emit('toggle')
}
</script>

<style scoped>
.habit-card {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 18px;
}
.habit-card.checked {
  border-color: rgba(52, 195, 143, 0.45);
  background: linear-gradient(0deg, #f3fbf7, #ffffff);
}
.habit-icon {
  width: 48px;
  height: 48px;
  display: grid;
  place-items: center;
  font-size: 24px;
  border-radius: 16px;
  background: var(--vibe-primary-soft);
}
.habit-info {
  flex: 1;
  min-width: 0;
}
.habit-name {
  font-size: 15px;
  font-weight: 600;
  margin-bottom: 4px;
}
.habit-streak {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 12.5px;
  color: var(--vibe-warning);
}
.check-btn {
  width: 40px;
  height: 40px;
  flex-shrink: 0;
  display: grid;
  place-items: center;
  border: 2px solid #d8dbe8;
  border-radius: 999px;
  background: transparent;
  color: #fff;
  cursor: pointer;
  transition: all 0.28s var(--vibe-ease);
}
.check-btn:hover {
  border-color: var(--vibe-success);
  transform: scale(1.1);
}
.check-btn.checked {
  background: var(--vibe-success);
  border-color: var(--vibe-success);
}
.check-text {
  color: var(--vibe-text-tertiary);
  font-size: 13px;
  font-weight: 600;
}
.remove-btn {
  border: none;
  background: transparent;
  color: var(--vibe-text-tertiary);
  cursor: pointer;
  padding: 4px;
  border-radius: 999px;
  opacity: 0;
  transition: all 0.2s var(--vibe-ease);
}
.habit-card:hover .remove-btn {
  opacity: 1;
}
.remove-btn:hover {
  color: var(--vibe-danger);
  background: rgba(255, 107, 122, 0.1);
}
</style>
