<template>
  <div class="completed">
    <!-- 筛选栏：与全部待办一致（状态固定为已完成） -->
    <div class="filter-bar vibe-card">
      <el-input
        v-model="filters.keyword"
        placeholder="搜索标题或备注"
        clearable
        class="filter-keyword"
        @input="onFilterChange"
      />
      <el-select v-model="filters.category" placeholder="分类" clearable class="filter-item" @change="onFilterChange">
        <el-option v-for="(label, key) in categoryOptions" :key="key" :label="label" :value="key" />
      </el-select>
      <el-select
        v-model="filters.tag_ids"
        placeholder="标签"
        clearable
        multiple
        collapse-tags
        class="filter-item"
        @change="onFilterChange"
      >
        <el-option v-for="tag in tags" :key="tag.id" :label="tag.name" :value="tag.id" />
      </el-select>
      <el-select v-model="filters.sort_by" placeholder="排序" clearable class="filter-item" @change="onFilterChange">
        <el-option label="优先级" value="priority" />
        <el-option label="日期" value="event_date" />
      </el-select>
      <el-button round :icon="Refresh" circle title="重置筛选" @click="resetFilters" />
    </div>

    <div v-if="todoStore.loading" class="loading-grid">
      <div v-for="i in 4" :key="i" class="skeleton-card skeleton-shimmer"></div>
    </div>
    <transition-group v-else-if="todoStore.list.length" name="list" tag="div">
      <TodoCard
        v-for="t in todoStore.list"
        :key="t.id"
        :todo="t"
        @toggle="onRestore(t)"
        @edit="openEdit(t)"
        @convert="onConvert(t)"
        @remove="onRemove(t)"
      />
    </transition-group>
    <EmptyState v-else title="还没有完成的事项" description="完成的事情都会躺在这里" icon="CircleCheck" />

    <div v-if="todoStore.total > todoStore.pageSize" class="pager">
      <el-pagination
        background
        layout="total, prev, pager, next"
        :total="todoStore.total"
        :page-size="todoStore.pageSize"
        :current-page="todoStore.currentPage"
        @current-change="onPageChange"
      />
    </div>

    <TodoFormDialog v-model:visible="dialogVisible" :todo="editing" :tags="tags" @saved="load" />
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Refresh } from '@element-plus/icons-vue'
import TodoCard from '@/components/TodoCard.vue'
import TodoFormDialog from '@/components/TodoFormDialog.vue'
import EmptyState from '@/components/EmptyState.vue'
import { useTodoStore } from '@/stores/todo'
import { tagApi } from '@/api/modules'
import type { Tag, Todo, TodoView } from '@/types'

const todoStore = useTodoStore()
const dialogVisible = ref(false)
const editing = ref<Todo | null>(null)
const tags = ref<Tag[]>([])

const categoryOptions: Record<string, string> = { life: '生活', work: '工作', study: '学习', health: '健康', other: '其他' }

const filters = reactive<Record<string, unknown>>({
  keyword: '',
  category: undefined,
  tag_ids: undefined,
  sort_by: undefined,
})

onMounted(async () => {
  tags.value = await tagApi.list()
  await load()
})

async function load() {
  const tagIds = (filters.tag_ids as number[] | undefined)?.join(',')
  await todoStore.fetchList({
    status: 1,
    keyword: (filters.keyword as string) || undefined,
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
  filters.category = undefined
  filters.tag_ids = undefined
  filters.sort_by = undefined
  onFilterChange()
}

function onPageChange(page: number) {
  todoStore.currentPage = page
  void load()
}

async function onRestore(t: TodoView) {
  try {
    await todoStore.toggleStatus(t)
    ElMessage.success('已恢复为未完成')
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

function openEdit(t: Todo) {
  editing.value = t
  dialogVisible.value = true
}

async function onConvert(t: TodoView) {
  try {
    await todoStore.convertToNote(t.id)
    ElMessage.success('已转为记事')
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

async function onRemove(t: TodoView) {
  try {
    await ElMessageBox.confirm(`确定删除「${t.title}」吗？删除后不可恢复。`, '删除待办', {
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
</script>

<style scoped>
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
</style>
