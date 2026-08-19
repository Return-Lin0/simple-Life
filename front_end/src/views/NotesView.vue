<template>
  <div class="notes">
    <div v-if="noteStore.loading" class="notes-grid">
      <div v-for="i in 6" :key="i" class="skeleton-note skeleton-shimmer"></div>
    </div>
    <transition-group v-else-if="noteStore.list.length" name="list" tag="div" class="notes-grid">
      <div v-for="note in noteStore.list" :key="note.id" class="note-card vibe-card" @click="openEdit(note)">
        <div class="note-title-row">
          <span class="note-title">{{ note.title }}</span>
          <el-tag v-if="note.source_todo_id" size="small" type="info" round>来自待办</el-tag>
        </div>
        <p class="note-content">{{ note.content || '（无内容）' }}</p>
        <div class="note-footer">
          <span class="note-time">{{ formatDateTime(note.created_at) }}</span>
          <button class="note-delete" @click.stop="onRemove(note)">
            <el-icon :size="14"><Delete /></el-icon>
          </button>
        </div>
      </div>
    </transition-group>
    <EmptyState v-else title="记事本还空着" description="随手记下灵感、购物清单或日记片段" icon="Notebook">
      <template #action>
        <el-button type="primary" round @click="openCreate">写一条</el-button>
      </template>
    </EmptyState>

    <button class="fab" title="新建记事" @click="openCreate">
      <el-icon :size="22"><Plus /></el-icon>
    </button>

    <el-dialog v-model="dialogVisible" :title="editing ? '编辑记事' : '新建记事'" width="560px" align-center destroy-on-close>
      <el-form label-position="top">
        <el-form-item label="标题">
          <el-input v-model="form.title" placeholder="记点什么？" maxlength="128" show-word-limit />
        </el-form-item>
        <el-form-item label="内容">
          <el-input v-model="form.content" type="textarea" :rows="8" placeholder="写下想法…" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button round @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" round :loading="saving" @click="save">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import EmptyState from '@/components/EmptyState.vue'
import { useNoteStore } from '@/stores/note'
import { formatDateTime } from '@/utils/format'
import type { Note } from '@/types'

const noteStore = useNoteStore()
const dialogVisible = ref(false)
const editing = ref<Note | null>(null)
const saving = ref(false)
const form = reactive({ title: '', content: '' })

onMounted(() => noteStore.fetchList())

function openCreate() {
  editing.value = null
  form.title = ''
  form.content = ''
  dialogVisible.value = true
}

function openEdit(note: Note) {
  editing.value = note
  form.title = note.title
  form.content = note.content
  dialogVisible.value = true
}

async function save() {
  if (!form.title.trim()) {
    ElMessage.warning('请输入标题')
    return
  }
  saving.value = true
  try {
    if (editing.value) {
      await noteStore.update(editing.value.id, form.title, form.content)
      ElMessage.success('记事已更新')
    } else {
      await noteStore.create(form.title, form.content)
      ElMessage.success('记事已保存')
    }
    dialogVisible.value = false
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    saving.value = false
  }
}

async function onRemove(note: Note) {
  try {
    await ElMessageBox.confirm(`确定删除「${note.title}」吗？`, '删除记事', {
      confirmButtonText: '删除',
      cancelButtonText: '取消',
      type: 'warning',
      roundButton: true,
    })
    await noteStore.remove(note.id)
    ElMessage.success('已删除')
  } catch (e) {
    if (e === 'cancel' || e === 'close') return
    ElMessage.error((e as Error).message)
  }
}
</script>

<style scoped>
.notes {
  padding-bottom: 80px;
}
.notes-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 16px;
}
.note-card {
  padding: 18px;
  display: flex;
  flex-direction: column;
  gap: 10px;
  cursor: pointer;
}
.note-title-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}
.note-title {
  font-size: 15px;
  font-weight: 700;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.note-content {
  margin: 0;
  font-size: 13px;
  color: var(--vibe-text-secondary);
  line-height: 1.7;
  display: -webkit-box;
  -webkit-line-clamp: 3;
  -webkit-box-orient: vertical;
  overflow: hidden;
  flex: 1;
}
.note-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  border-top: 1px solid var(--vibe-border);
  padding-top: 10px;
}
.note-time {
  font-size: 12px;
  color: var(--vibe-text-tertiary);
}
.note-delete {
  border: none;
  background: transparent;
  color: var(--vibe-text-tertiary);
  cursor: pointer;
  padding: 4px;
  border-radius: 999px;
  opacity: 0;
  transition: all 0.2s var(--vibe-ease);
}
.note-card:hover .note-delete {
  opacity: 1;
}
.note-delete:hover {
  color: var(--vibe-danger);
  background: rgba(255, 107, 122, 0.1);
}
.skeleton-note {
  height: 150px;
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
}
</style>
