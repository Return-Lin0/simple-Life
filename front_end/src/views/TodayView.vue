<template>
  <div class="today">
    <!-- 顶部概览 -->
    <div class="summary-row">
      <div class="summary-card vibe-card">
        <div class="summary-num">{{ todoStore.today.length }}</div>
        <div class="summary-label">今日事项</div>
      </div>
      <div class="summary-card vibe-card warn">
        <div class="summary-num">{{ todoStore.overdueCount }}</div>
        <div class="summary-label">已逾期</div>
      </div>
      <div class="summary-card vibe-card done-card">
        <div class="summary-num">{{ doneToday }}</div>
        <div class="summary-label">今日已完成</div>
      </div>
    </div>

    <!-- 逾期区块 -->
    <section v-if="overdueList.length" class="section">
      <h2 class="section-title danger"><el-icon><WarningFilled /></el-icon>逾期未完成</h2>
      <transition-group name="list" tag="div">
        <TodoCard
          v-for="t in overdueList"
          :key="t.id"
          :todo="t"
          @toggle="onToggle(t)"
          @edit="openEdit(t)"
          @convert="onConvert(t)"
          @remove="onRemove(t)"
        />
      </transition-group>
    </section>

    <!-- 今日事项 -->
    <section class="section">
      <h2 class="section-title"><el-icon><Sunny /></el-icon>今天要做</h2>
      <div v-if="todoStore.loading" class="loading-grid">
        <div v-for="i in 3" :key="i" class="skeleton-card skeleton-shimmer"></div>
      </div>
      <transition-group v-else-if="pendingToday.length" name="list" tag="div">
        <TodoCard
          v-for="t in pendingToday"
          :key="t.id"
          :todo="t"
          @toggle="onToggle(t)"
          @edit="openEdit(t)"
          @convert="onConvert(t)"
          @remove="onRemove(t)"
        />
      </transition-group>
      <EmptyState v-else title="今天很轻松" description="没有待办事项，好好享受今天吧" icon="CoffeeCup">
        <template #action>
          <el-button type="primary" round @click="openCreate">记一件事</el-button>
        </template>
      </EmptyState>
    </section>

    <!-- 今日已完成 -->
    <section v-if="doneList.length" class="section">
      <h2 class="section-title success">
        <el-icon><CircleCheckFilled /></el-icon>
        今日已完成
        <span class="section-count">{{ doneList.length }}</span>
      </h2>
      <transition-group name="list" tag="div">
        <TodoCard
          v-for="t in doneList"
          :key="t.id"
          :todo="t"
          @toggle="onToggle(t)"
          @edit="openEdit(t)"
          @convert="onConvert(t)"
          @remove="onRemove(t)"
        />
      </transition-group>
    </section>

    <!-- 新建按钮 -->
    <button class="fab" title="新建待办" @click="openCreate">
      <el-icon :size="22"><Plus /></el-icon>
    </button>

    <TodoFormDialog
      v-model:visible="dialogVisible"
      :todo="editing"
      :tags="tags"
      @saved="onSaved"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import TodoCard from '@/components/TodoCard.vue'
import TodoFormDialog from '@/components/TodoFormDialog.vue'
import EmptyState from '@/components/EmptyState.vue'
import { useTodoStore } from '@/stores/todo'
import { tagApi, todoApi } from '@/api/modules'
import type { Tag, Todo, TodoView } from '@/types'

const todoStore = useTodoStore()
const dialogVisible = ref(false)
const editing = ref<Todo | null>(null)
const tags = ref<Tag[]>([])

const overdueList = computed(() => todoStore.today.filter((t) => t.overdue && t.status === 0))
const pendingToday = computed(() => todoStore.today.filter((t) => !t.overdue && t.status === 0))
const doneToday = ref(0)
const doneList = ref<TodoView[]>([])

onMounted(async () => {
  await Promise.all([
    todoStore.fetchToday(),
    tagApi.list().then((t) => (tags.value = t)),
    fetchDoneToday(),
  ])
})

// 今日已完成数量：后端 today 视图只含未完成与逾期，需单独查询
async function fetchDoneToday() {
  try {
    const today = new Date()
    const pad = (n: number) => String(n).padStart(2, '0')
    const dateStr = `${today.getFullYear()}-${pad(today.getMonth() + 1)}-${pad(today.getDate())}`
    const data = await todoApi.list({ start_date: dateStr, end_date: dateStr, status: 1, page_size: 100 })
    doneToday.value = data.total
    doneList.value = data.list
  } catch {
    doneToday.value = 0
    doneList.value = []
  }
}

function openCreate() {
  editing.value = null
  dialogVisible.value = true
}

function openEdit(t: Todo) {
  editing.value = t
  dialogVisible.value = true
}

async function onToggle(t: TodoView) {
  try {
    await todoStore.toggleStatus(t)
    await refreshAll()
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

async function onConvert(t: TodoView) {
  try {
    await ElMessageBox.confirm(`将「${t.title}」转为记事并标记完成？`, '转为记事', {
      confirmButtonText: '转换',
      cancelButtonText: '取消',
      type: 'info',
      roundButton: true,
    })
    await todoStore.convertToNote(t.id)
    ElMessage.success('已转为记事')
    await refreshAll()
  } catch (e) {
    if (e === 'cancel' || e === 'close') return
    ElMessage.error((e as Error).message)
  }
}

async function onRemove(t: TodoView) {
  try {
    await ElMessageBox.confirm(`确定删除「${t.title}」吗？`, '删除待办', {
      confirmButtonText: '删除',
      cancelButtonText: '取消',
      type: 'warning',
      roundButton: true,
    })
    await todoStore.remove(t.id)
    ElMessage.success('已删除')
    await refreshAll()
  } catch (e) {
    if (e === 'cancel' || e === 'close') return
    ElMessage.error((e as Error).message)
  }
}

async function onSaved() {
  await refreshAll()
}

// 主看板联动刷新：今日事项列表 + 今日已完成统计
async function refreshAll() {
  await Promise.all([todoStore.fetchToday(), fetchDoneToday()])
}
</script>

<style scoped>
.today {
  padding-bottom: 80px;
}
.summary-row {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 16px;
  margin-bottom: 26px;
}
.summary-card {
  padding: 20px;
  text-align: center;
}
.summary-num {
  font-size: 30px;
  font-weight: 800;
  color: var(--vibe-primary);
}
.summary-card.warn .summary-num {
  color: var(--vibe-danger);
}
.summary-card.done-card .summary-num {
  color: var(--vibe-success);
}
.summary-label {
  margin-top: 4px;
  font-size: 13px;
  color: var(--vibe-text-secondary);
}
.section {
  margin-bottom: 24px;
}
.section-title {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 0 0 14px;
  font-size: 16px;
  font-weight: 700;
  color: var(--vibe-primary);
}
.section-title.danger {
  color: var(--vibe-danger);
}
.section-title.success {
  color: var(--vibe-success);
}
.section-count {
  margin-left: auto;
  padding: 2px 10px;
  border-radius: 999px;
  font-size: 12px;
  font-weight: 700;
  color: var(--vibe-success);
  background: rgba(52, 195, 143, 0.14);
}
.loading-grid {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.skeleton-card {
  height: 78px;
  border-radius: var(--vibe-radius-lg);
}
.fab {
  position: fixed;
  right: 44px;
  bottom: 40px;
  width: 54px;
  height: 54px;
  display: grid;
  place-items: center;
  border: none;
  border-radius: 999px;
  background: linear-gradient(135deg, #7c8cff, #5a69e6);
  color: #fff;
  cursor: pointer;
  box-shadow: 0 12px 28px rgba(108, 123, 255, 0.45);
  transition: all 0.28s var(--vibe-ease);
  z-index: 10;
}
.fab:hover {
  transform: translateY(-3px) scale(1.05);
  box-shadow: 0 16px 36px rgba(108, 123, 255, 0.55);
}
.fab:active {
  transform: scale(0.94);
}
</style>
