<template>
  <div class="todo-list">
    <!-- 筛选栏 -->
    <div class="filter-bar vibe-card">
      <el-input v-model="filters.keyword" placeholder="搜索标题或备注" clearable class="filter-keyword" @input="onFilterChange" />
      <el-select v-model="filters.status" placeholder="状态" clearable class="filter-item" @change="onFilterChange">
        <el-option label="未完成" :value="0" />
        <el-option label="已完成" :value="1" />
      </el-select>
      <el-select v-model="filters.category" placeholder="分类" clearable class="filter-item" @change="onFilterChange">
        <el-option v-for="(label, key) in categoryOptions" :key="key" :label="label" :value="key" />
      </el-select>
      <el-select v-model="filters.tag_ids" placeholder="标签" clearable multiple collapse-tags class="filter-item" @change="onFilterChange">
        <el-option v-for="tag in tags" :key="tag.id" :label="tag.name" :value="tag.id" />
      </el-select>
      <el-select v-model="filters.sort_by" placeholder="排序" clearable class="filter-item" @change="onFilterChange">
        <el-option label="优先级" value="priority" />
        <el-option label="日期" value="event_date" />
      </el-select>
      <el-button round :icon="Refresh" circle title="重置筛选" @click="resetFilters" />
      <el-button
        round
        :type="selectionMode ? 'primary' : 'default'"
        :icon="selectionMode ? 'Finished' : 'Select'"
        @click="toggleSelectionMode"
      >
        {{ selectionMode ? '退出多选' : '多选' }}
      </el-button>
    </div>

    <!-- 多选工具条 -->
    <div v-if="selectionMode" class="select-bar vibe-card">
      <el-checkbox
        :model-value="allSelected"
        :indeterminate="partialSelected"
        @change="toggleSelectAll"
      >
        全选
      </el-checkbox>
      <span class="select-count">已选 {{ selectedIds.length }} 项</span>
      <div class="select-actions">
        <el-button size="small" round type="primary" :disabled="selectedIds.length === 0" @click="batchComplete">
          批量完成
        </el-button>
        <el-button size="small" round :disabled="selectedIds.length === 0" @click="batchRestore">
          批量恢复
        </el-button>
        <el-button size="small" round type="danger" :disabled="selectedIds.length === 0" @click="batchRemove">
          批量删除
        </el-button>
        <el-button size="small" round text @click="clearSelection">清空选择</el-button>
      </div>
    </div>

    <!-- 列表 -->
    <div v-if="todoStore.loading" class="loading-grid">
      <div v-for="i in 5" :key="i" class="skeleton-card skeleton-shimmer"></div>
    </div>
    <transition-group v-else-if="todoStore.list.length" name="list" tag="div">
      <TodoCard
        v-for="t in todoStore.list"
        :key="t.id"
        :todo="t"
        :selectable="selectionMode"
        :selected="selectedIds.includes(t.id)"
        @toggle="onToggle(t)"
        @edit="openEdit(t)"
        @convert="onConvert(t)"
        @remove="onRemove(t)"
        @select="toggleSelect(t.id)"
      />
    </transition-group>
    <EmptyState v-else title="没有符合条件的待办" description="换个筛选条件，或新建一条待办" icon="List">
      <template #action>
        <el-button type="primary" round @click="openCreate">新建待办</el-button>
      </template>
    </EmptyState>

    <div v-if="todoStore.total > 0" class="pager">
      <el-pagination
        background
        layout="total, prev, pager, next"
        :total="todoStore.total"
        :page-size="todoStore.pageSize"
        :current-page="todoStore.currentPage"
        @current-change="onPageChange"
      />
    </div>

    <button class="fab" title="新建待办" @click="openCreate">
      <el-icon :size="22"><Plus /></el-icon>
    </button>

    <TodoFormDialog v-model:visible="dialogVisible" :todo="editing" :tags="tags" @saved="onSaved" />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Refresh } from '@element-plus/icons-vue'
import TodoCard from '@/components/TodoCard.vue'
import TodoFormDialog from '@/components/TodoFormDialog.vue'
import EmptyState from '@/components/EmptyState.vue'
import { useTodoStore } from '@/stores/todo'
import { tagApi } from '@/api/modules'
import type { Tag, Todo, TodoView } from '@/types'

const todoStore = useTodoStore()
const tags = ref<Tag[]>([])
const dialogVisible = ref(false)
const editing = ref<Todo | null>(null)

const categoryOptions: Record<string, string> = { life: '生活', work: '工作', study: '学习', health: '健康', other: '其他' }

const filters = reactive<Record<string, unknown>>({
  keyword: '',
  status: undefined,
  category: undefined,
  tag_ids: undefined,
  sort_by: undefined,
})

// ---------- 多选状态 ----------
const selectionMode = ref(false)
const selectedIds = ref<number[]>([])

const allSelected = computed(() =>
  todoStore.list.length > 0 && todoStore.list.every((t) => selectedIds.value.includes(t.id)),
)
const partialSelected = computed(() => {
  const onPage = todoStore.list.filter((t) => selectedIds.value.includes(t.id)).length
  return onPage > 0 && onPage < todoStore.list.length
})

function toggleSelectionMode() {
  selectionMode.value = !selectionMode.value
  if (!selectionMode.value) selectedIds.value = []
}

function toggleSelect(id: number) {
  const idx = selectedIds.value.indexOf(id)
  if (idx >= 0) selectedIds.value.splice(idx, 1)
  else selectedIds.value.push(id)
}

function toggleSelectAll() {
  if (allSelected.value) {
    const onPage = new Set(todoStore.list.map((t) => t.id))
    selectedIds.value = selectedIds.value.filter((id) => !onPage.has(id))
  } else {
    const onPage = todoStore.list.map((t) => t.id)
    selectedIds.value = Array.from(new Set([...selectedIds.value, ...onPage]))
  }
}

function clearSelection() {
  selectedIds.value = []
}

async function batchComplete() {
  try {
    await todoStore.batchUpdateStatus([...selectedIds.value], 1)
    ElMessage.success('已批量标记完成')
    afterBatch()
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

async function batchRestore() {
  try {
    await todoStore.batchUpdateStatus([...selectedIds.value], 0)
    ElMessage.success('已批量恢复未完成')
    afterBatch()
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

async function batchRemove() {
  try {
    await ElMessageBox.confirm(`确定删除选中的 ${selectedIds.value.length} 条待办吗？删除后不可恢复。`, '批量删除', {
      confirmButtonText: '删除',
      cancelButtonText: '取消',
      type: 'warning',
      roundButton: true,
    })
    await todoStore.batchDelete([...selectedIds.value])
    ElMessage.success('已批量删除')
    afterBatch()
  } catch (e) {
    if (e === 'cancel' || e === 'close') return
    ElMessage.error((e as Error).message)
  }
}

async function afterBatch() {
  selectedIds.value = []
  await load()
}

onMounted(async () => {
  tags.value = await tagApi.list()
  await load()
})

async function load() {
  const tagIds = (filters.tag_ids as number[] | undefined)?.join(',')
  await todoStore.fetchList({
    keyword: (filters.keyword as string) || undefined,
    status: filters.status as number | undefined,
    category: filters.category as string | undefined,
    tag_ids: tagIds,
    sort_by: filters.sort_by as string | undefined,
  })
}

function onFilterChange() {
  todoStore.currentPage = 1
  void load()
}

function resetFilters() {
  filters.keyword = ''
  filters.status = undefined
  filters.category = undefined
  filters.tag_ids = undefined
  filters.sort_by = undefined
  onFilterChange()
}

function onPageChange(page: number) {
  todoStore.currentPage = page
  void load()
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
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

async function onConvert(t: TodoView) {
  try {
    await ElMessageBox.confirm(`将「${t.title}」转为记事？`, '转为记事', {
      confirmButtonText: '转换',
      cancelButtonText: '取消',
      type: 'info',
      roundButton: true,
    })
    await todoStore.convertToNote(t.id)
    ElMessage.success('已转为记事')
    await load()
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
    await load()
  } catch (e) {
    if (e === 'cancel' || e === 'close') return
    ElMessage.error((e as Error).message)
  }
}

function onSaved() {
  void load()
}
</script>

<style scoped>
.todo-list {
  padding-bottom: 80px;
}
.filter-bar {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 14px 16px;
  margin-bottom: 18px;
}
.filter-keyword {
  flex: 1;
  max-width: 260px;
}
.filter-item {
  width: 128px;
}
.select-bar {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 12px 18px;
  margin-bottom: 14px;
  border-color: rgba(108, 123, 255, 0.35);
  background: linear-gradient(0deg, #f6f7ff, #ffffff);
}
.select-count {
  font-size: 13px;
  color: var(--vibe-primary);
  font-weight: 600;
}
.select-actions {
  margin-left: auto;
  display: flex;
  align-items: center;
  gap: 8px;
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
.pager {
  display: flex;
  justify-content: center;
  margin-top: 20px;
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
}
</style>
