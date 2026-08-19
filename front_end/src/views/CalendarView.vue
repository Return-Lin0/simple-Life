<template>
  <div class="calendar">
    <div class="calendar-header">
      <div class="month-nav">
        <button class="nav-btn" @click="shiftMonth(-1)"><el-icon><ArrowLeft /></el-icon></button>
        <span class="month-title">{{ currentMonth.format('YYYY 年 M 月') }}</span>
        <button class="nav-btn" @click="shiftMonth(1)"><el-icon><ArrowRight /></el-icon></button>
        <el-button size="small" round class="today-btn" @click="goToday">今天</el-button>
      </div>
      <div class="legend">
        <span><i class="dot todo-dot"></i>待办</span>
        <span><i class="dot ann-dot"></i>纪念日</span>
      </div>
    </div>

    <div class="calendar-card vibe-card">
      <div class="week-row">
        <span v-for="w in weekNames" :key="w" class="week-cell">{{ w }}</span>
      </div>
      <div class="day-grid">
        <div
          v-for="cell in cells"
          :key="cell.key"
          class="day-cell"
          :class="{ muted: !cell.inMonth, today: cell.isToday }"
          @click="openCreateOn(cell.date)"
        >
          <span class="day-num">{{ cell.dayNum }}</span>
          <div class="cell-events">
            <button
              v-for="ev in cell.todos"
              :key="'t' + ev.id"
              class="cell-chip todo-chip"
              :class="{ done: ev.status === 1 }"
              @click.stop="openEdit(ev)"
            >
              {{ ev.title }}
            </button>
            <button
              v-for="a in cell.anns"
              :key="'a' + a.id"
              class="cell-chip ann-chip"
              @click.stop="$router.push({ name: 'anniversaries' })"
            >
              {{ a.name }}
            </button>
          </div>
        </div>
      </div>
    </div>

    <TodoFormDialog
      v-model:visible="dialogVisible"
      :todo="editing"
      :tags="tags"
      :preset-date="presetDate"
      @saved="load"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import dayjs, { type Dayjs } from 'dayjs'
import TodoFormDialog from '@/components/TodoFormDialog.vue'
import { todoApi, tagApi } from '@/api/modules'
import { useAnniversaryStore } from '@/stores/anniversary'
import type { Anniversary, Tag, Todo, TodoView } from '@/types'

const currentMonth = ref(dayjs().startOf('month'))
const todoMap = ref<Record<string, TodoView[]>>({})
const annMap = ref<Record<string, Anniversary[]>>({})
const dialogVisible = ref(false)
const editing = ref<Todo | null>(null)
const tags = ref<Tag[]>([])
const presetDate = ref('')
const anniversaryStore = useAnniversaryStore()

const weekNames = ['一', '二', '三', '四', '五', '六', '日']

const cells = computed(() => {
  const start = currentMonth.value.startOf('week')
  const days: {
    key: string
    date: string
    dayNum: number
    inMonth: boolean
    isToday: boolean
    todos: TodoView[]
    anns: Anniversary[]
  }[] = []
  const todayStr = dayjs().format('YYYY-MM-DD')
  for (let i = 0; i < 42; i++) {
    const d = start.add(i, 'day')
    const dateStr = d.format('YYYY-MM-DD')
    days.push({
      key: dateStr,
      date: dateStr,
      dayNum: d.date(),
      inMonth: d.month() === currentMonth.value.month(),
      isToday: dateStr === todayStr,
      todos: todoMap.value[dateStr] || [],
      anns: annMap.value[dateStr] || [],
    })
  }
  return days
})

onMounted(async () => {
  tags.value = await tagApi.list()
  await Promise.all([load(), anniversaryStore.fetchList()])
  // 观察纪念日变化，保持日历同步
  computeAnnMap()
})

async function load() {
  const start = currentMonth.value.startOf('week').format('YYYY-MM-DD')
  const end = currentMonth.value.endOf('week').add(6, 'day').format('YYYY-MM-DD')
  const list = await todoApi.calendar(start, end)
  todoMap.value = {}
  for (const t of list) {
    if (!todoMap.value[t.event_date]) todoMap.value[t.event_date] = []
    todoMap.value[t.event_date].push(t)
  }
}

function computeAnnMap() {
  annMap.value = {}
  for (const a of anniversaryStore.list) {
    const date = a.next_occurrence || a.event_date
    if (!annMap.value[date]) annMap.value[date] = []
    annMap.value[date].push(a)
  }
}

function shiftMonth(n: number) {
  currentMonth.value = currentMonth.value.add(n, 'month')
  void load()
}

function goToday() {
  currentMonth.value = dayjs().startOf('month')
  void load()
}

function openCreateOn(date: string) {
  editing.value = null
  presetDate.value = date
  dialogVisible.value = true
}

function openEdit(t: Todo) {
  editing.value = t
  presetDate.value = ''
  dialogVisible.value = true
}
</script>

<style scoped>
.calendar-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
}
.month-nav {
  display: flex;
  align-items: center;
  gap: 10px;
}
.nav-btn {
  width: 34px;
  height: 34px;
  display: grid;
  place-items: center;
  border: 1px solid var(--vibe-border);
  border-radius: 999px;
  background: var(--vibe-surface);
  color: var(--vibe-text-secondary);
  cursor: pointer;
  transition: all 0.22s var(--vibe-ease);
}
.nav-btn:hover {
  color: var(--vibe-primary);
  border-color: var(--vibe-primary);
}
.month-title {
  font-size: 18px;
  font-weight: 700;
  min-width: 130px;
  text-align: center;
}
.legend {
  display: flex;
  gap: 16px;
  font-size: 12.5px;
  color: var(--vibe-text-secondary);
}
.legend span {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}
.dot {
  width: 8px;
  height: 8px;
  border-radius: 999px;
  display: inline-block;
}
.todo-dot {
  background: var(--vibe-primary);
}
.ann-dot {
  background: var(--vibe-warning);
}

.calendar-card {
  padding: 18px;
  overflow: hidden;
}
.week-row {
  display: grid;
  grid-template-columns: repeat(7, 1fr);
  margin-bottom: 8px;
}
.week-cell {
  text-align: center;
  font-size: 12.5px;
  font-weight: 600;
  color: var(--vibe-text-tertiary);
  padding: 6px 0;
}
.day-grid {
  display: grid;
  grid-template-columns: repeat(7, 1fr);
  gap: 6px;
}
.day-cell {
  min-height: 96px;
  padding: 8px;
  border-radius: var(--vibe-radius-md);
  background: var(--vibe-surface-2);
  cursor: pointer;
  transition: all 0.22s var(--vibe-ease);
}
.day-cell:hover {
  background: var(--vibe-primary-soft);
  transform: translateY(-1px);
}
.day-cell.muted {
  opacity: 0.45;
}
.day-cell.today {
  outline: 2px solid var(--vibe-primary);
  outline-offset: -2px;
}
.day-num {
  display: inline-block;
  font-size: 13px;
  font-weight: 700;
  color: var(--vibe-text-secondary);
}
.today .day-num {
  color: var(--vibe-primary);
}
.cell-events {
  display: flex;
  flex-direction: column;
  gap: 3px;
  margin-top: 6px;
}
.cell-chip {
  border: none;
  text-align: left;
  padding: 3px 8px;
  border-radius: 999px;
  font-size: 11px;
  font-weight: 600;
  cursor: pointer;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  transition: transform 0.18s var(--vibe-ease);
}
.cell-chip:hover {
  transform: scale(1.04);
}
.todo-chip {
  color: var(--vibe-primary);
  background: var(--vibe-primary-soft);
}
.todo-chip.done {
  color: var(--vibe-text-tertiary);
  background: var(--vibe-surface);
  text-decoration: line-through;
}
.ann-chip {
  color: #d48806;
  background: rgba(255, 180, 84, 0.16);
}
</style>
