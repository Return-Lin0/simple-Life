<template>
  <div class="todo-card vibe-card" :class="{ done: todo.status === 1, overdue: todo.overdue }">
    <!-- 完成切换：圆形勾选，动画丝滑 -->
    <button class="check" :class="{ checked: todo.status === 1 }" @click="$emit('toggle')">
      <transition name="pop">
        <el-icon v-if="todo.status === 1" :size="14"><Check /></el-icon>
      </transition>
    </button>

    <div class="todo-body" @click="$emit('edit')">
      <div class="todo-title-row">
        <span class="todo-title">{{ todo.title }}</span>
        <span v-if="todo.overdue" class="badge overdue-badge">已逾期</span>
        <span v-if="todo.recurrence_type > 0" class="badge recur-badge">
          <el-icon :size="12"><RefreshRight /></el-icon>{{ recurrenceLabel }}
        </span>
      </div>
      <div class="todo-meta">
        <span class="meta-item"><el-icon :size="13"><Calendar /></el-icon>{{ dateText }}</span>
        <span v-if="!todo.is_all_day && todo.start_time" class="meta-item">
          <el-icon :size="13"><Clock /></el-icon>{{ formatTime(todo.start_time) }}{{ todo.end_time ? ' – ' + formatTime(todo.end_time) : '' }}
        </span>
        <span v-if="todo.reminder_enabled" class="meta-item remind"><el-icon :size="13"><Bell /></el-icon>已设提醒</span>
        <span class="category-chip" :class="'cat-' + todo.category">{{ categoryLabel }}</span>
        <el-tag v-for="tag in todo.tags" :key="tag.id" size="small" round effect="plain" class="tag-chip">
          {{ tag.name }}
        </el-tag>
      </div>
    </div>

    <div class="priority-dot" :style="{ background: priorityColor }" :title="'优先级：' + priorityLabel"></div>

    <el-dropdown trigger="click" @command="onCommand" class="todo-actions">
      <button class="more-btn" @click.stop><el-icon><MoreFilled /></el-icon></button>
      <template #dropdown>
        <el-dropdown-menu>
          <el-dropdown-item command="toggle">{{ todo.status === 1 ? '恢复未完成' : '标记完成' }}</el-dropdown-item>
          <el-dropdown-item command="convert" :disabled="todo.status === 0">转为记事</el-dropdown-item>
          <el-dropdown-item divided command="remove">
            <span class="danger-text">删除</span>
          </el-dropdown-item>
        </el-dropdown-menu>
      </template>
    </el-dropdown>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import {
  CATEGORY_LABELS,
  PRIORITY_COLORS,
  PRIORITY_LABELS,
  RECURRENCE_LABELS,
  formatTime,
  weekdayCN,
} from '@/utils/format'
import type { TodoView } from '@/types'

const props = defineProps<{ todo: TodoView }>()
const emit = defineEmits<{
  (e: 'toggle'): void
  (e: 'edit'): void
  (e: 'convert'): void
  (e: 'remove'): void
}>()

const categoryLabel = computed(() => CATEGORY_LABELS[props.todo.category] || '其他')
const priorityLabel = computed(() => PRIORITY_LABELS[props.todo.priority] || '中')
const priorityColor = computed(() => PRIORITY_COLORS[props.todo.priority] || '#8a90a3')
const recurrenceLabel = computed(() => RECURRENCE_LABELS[props.todo.recurrence_type] || '')
const dateText = computed(() => `${props.todo.event_date} ${weekdayCN(props.todo.event_date)}`)

function onCommand(cmd: string) {
  if (cmd === 'toggle') emit('toggle')
  else if (cmd === 'convert') emit('convert')
  else if (cmd === 'remove') emit('remove')
}
</script>

<style scoped>
.todo-card {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 16px 18px;
  margin-bottom: 12px;
  cursor: pointer;
}
.todo-card:hover {
  border-color: var(--vibe-primary-light);
}
.todo-card.overdue {
  border-color: rgba(255, 107, 122, 0.4);
  background: linear-gradient(0deg, #fff7f8, #ffffff);
}

.check {
  width: 26px;
  height: 26px;
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
.check:hover {
  border-color: var(--vibe-primary);
  transform: scale(1.08);
}
.check.checked {
  background: var(--vibe-success);
  border-color: var(--vibe-success);
}

.todo-body {
  flex: 1;
  min-width: 0;
}
.todo-title-row {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 6px;
}
.todo-title {
  font-size: 15px;
  font-weight: 600;
  transition: color 0.25s var(--vibe-ease);
}
.done .todo-title {
  color: var(--vibe-text-tertiary);
  text-decoration: line-through;
}

.badge {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  padding: 2px 8px;
  border-radius: 999px;
  font-size: 11px;
  font-weight: 600;
}
.overdue-badge {
  color: var(--vibe-danger);
  background: rgba(255, 107, 122, 0.12);
}
.recur-badge {
  color: var(--vibe-primary);
  background: var(--vibe-primary-soft);
}

.todo-meta {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  font-size: 12.5px;
  color: var(--vibe-text-secondary);
}
.meta-item {
  display: inline-flex;
  align-items: center;
  gap: 4px;
}
.meta-item.remind {
  color: var(--vibe-warning);
}
.category-chip {
  padding: 2px 10px;
  border-radius: 999px;
  font-size: 11.5px;
  font-weight: 600;
}
.cat-life {
  color: #4fc3f7;
  background: rgba(79, 195, 247, 0.12);
}
.cat-work {
  color: var(--vibe-primary);
  background: var(--vibe-primary-soft);
}
.cat-study {
  color: #9b7bff;
  background: rgba(155, 123, 255, 0.12);
}
.cat-health {
  color: var(--vibe-success);
  background: rgba(52, 195, 143, 0.12);
}
.cat-other {
  color: var(--vibe-text-secondary);
  background: rgba(138, 144, 163, 0.12);
}
.tag-chip {
  --el-tag-bg-color: var(--vibe-surface-2);
  --el-tag-border-color: var(--vibe-border);
  --el-tag-text-color: var(--vibe-text-secondary);
}

.priority-dot {
  width: 8px;
  height: 8px;
  flex-shrink: 0;
  border-radius: 999px;
}

.more-btn {
  border: none;
  background: transparent;
  color: var(--vibe-text-tertiary);
  cursor: pointer;
  padding: 6px;
  border-radius: 999px;
  transition: all 0.2s var(--vibe-ease);
}
.more-btn:hover {
  color: var(--vibe-text);
  background: var(--vibe-surface-2);
}
.danger-text {
  color: var(--vibe-danger);
}
</style>
