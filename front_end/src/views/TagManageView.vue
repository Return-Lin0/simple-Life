<template>
  <div class="tags">
    <!-- 新建标签 -->
    <section class="card vibe-card">
      <h2 class="card-title"><el-icon><Plus /></el-icon>新建标签</h2>
      <div class="create-row">
        <el-input v-model="newName" placeholder="标签名称（如：重要、购物）" maxlength="32" class="create-input" />
        <el-color-picker v-model="newColor" :predefine="presetColors" />
        <el-button type="primary" round :loading="creating" @click="createTag">创建</el-button>
      </div>
    </section>

    <!-- 标签列表 -->
    <section class="card vibe-card">
      <h2 class="card-title"><el-icon><CollectionTag /></el-icon>我的标签（{{ tags.length }}）</h2>
      <p class="card-desc">删除标签会同时移除待办上的关联，请谨慎操作</p>

      <div v-if="loading" class="loading-grid">
        <div v-for="i in 3" :key="i" class="skeleton-tag skeleton-shimmer"></div>
      </div>
      <transition-group v-else-if="tags.length" name="list" tag="div" class="tag-list">
        <div v-for="tag in tags" :key="tag.id" class="tag-row">
          <span class="tag-dot" :style="{ background: tag.color || '#8a90a3' }"></span>
          <span class="tag-name">{{ tag.name }}</span>
          <div class="tag-actions">
            <el-button size="small" round :icon="Edit" @click="openEdit(tag)">编辑</el-button>
            <el-button size="small" round type="danger" plain :icon="Delete" @click="removeTag(tag)">删除</el-button>
          </div>
        </div>
      </transition-group>
      <div v-else class="empty-hint">还没有标签，先在上方创建一个吧</div>
    </section>

    <!-- 编辑弹窗 -->
    <el-dialog v-model="editVisible" title="编辑标签" width="420px" align-center destroy-on-close>
      <el-form label-position="top">
        <el-form-item label="标签名称">
          <el-input v-model="editForm.name" maxlength="32" show-word-limit />
        </el-form-item>
        <el-form-item label="颜色">
          <el-color-picker v-model="editForm.color" :predefine="presetColors" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button round @click="editVisible = false">取消</el-button>
        <el-button type="primary" round :loading="saving" @click="saveEdit">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Delete, Edit } from '@element-plus/icons-vue'
import { tagApi } from '@/api/modules'
import type { Tag } from '@/types'

const tags = ref<Tag[]>([])
const loading = ref(false)

// 新建
const newName = ref('')
const newColor = ref('#6c7bff')
const creating = ref(false)
const presetColors = ['#6c7bff', '#34c38f', '#ffb454', '#ff6b7a', '#4fc3f7', '#9b7bff', '#8a90a3']

// 编辑
const editVisible = ref(false)
const saving = ref(false)
const editForm = reactive({ id: 0, name: '', color: '#6c7bff' })

onMounted(() => fetchTags())

async function fetchTags() {
  loading.value = true
  try {
    tags.value = await tagApi.list()
  } finally {
    loading.value = false
  }
}

async function createTag() {
  const name = newName.value.trim()
  if (!name) {
    ElMessage.warning('请输入标签名称')
    return
  }
  creating.value = true
  try {
    await tagApi.create({ name, color: newColor.value })
    ElMessage.success('标签已创建')
    newName.value = ''
    await fetchTags()
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    creating.value = false
  }
}

function openEdit(tag: Tag) {
  editForm.id = tag.id
  editForm.name = tag.name
  editForm.color = tag.color || '#6c7bff'
  editVisible.value = true
}

async function saveEdit() {
  if (!editForm.name.trim()) {
    ElMessage.warning('标签名称不能为空')
    return
  }
  saving.value = true
  try {
    await tagApi.update(editForm.id, { name: editForm.name.trim(), color: editForm.color })
    ElMessage.success('标签已更新')
    editVisible.value = false
    await fetchTags()
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    saving.value = false
  }
}

async function removeTag(tag: Tag) {
  try {
    await ElMessageBox.confirm(
      `确定删除标签「${tag.name}」吗？所有待办上的该标签关联都会被移除。`,
      '删除标签',
      { confirmButtonText: '删除', cancelButtonText: '取消', type: 'warning', roundButton: true },
    )
    await tagApi.remove(tag.id)
    ElMessage.success('标签已删除')
    await fetchTags()
  } catch (e) {
    if (e === 'cancel' || e === 'close') return
    ElMessage.error((e as Error).message)
  }
}
</script>

<style scoped>
.tags {
  max-width: 720px;
  display: flex;
  flex-direction: column;
  gap: 20px;
}
.card {
  padding: 24px 28px;
}
.card-title {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 0 0 16px;
  font-size: 17px;
  font-weight: 700;
  color: var(--vibe-primary);
}
.card-desc {
  margin: 0 0 14px;
  font-size: 13px;
  color: var(--vibe-text-secondary);
}
.create-row {
  display: flex;
  align-items: center;
  gap: 12px;
}
.create-input {
  flex: 1;
  max-width: 320px;
}
.tag-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.tag-row {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
  border: 1px solid var(--vibe-border);
  border-radius: var(--vibe-radius-md);
  transition: all 0.22s var(--vibe-ease);
}
.tag-row:hover {
  border-color: var(--vibe-primary-light);
  background: var(--vibe-primary-soft);
}
.tag-dot {
  width: 12px;
  height: 12px;
  flex-shrink: 0;
  border-radius: 999px;
}
.tag-name {
  flex: 1;
  font-size: 14.5px;
  font-weight: 600;
}
.tag-actions {
  display: flex;
  gap: 8px;
}
.empty-hint {
  padding: 24px;
  text-align: center;
  color: var(--vibe-text-tertiary);
  font-size: 13px;
}
.loading-grid {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.skeleton-tag {
  height: 48px;
  border-radius: var(--vibe-radius-md);
}
</style>
