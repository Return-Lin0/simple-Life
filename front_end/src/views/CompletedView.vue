<template>
  <div class="completed">
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
import { onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
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

onMounted(async () => {
  tags.value = await tagApi.list()
  await load()
})

async function load() {
  await todoStore.fetchList({ status: 1 })
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
