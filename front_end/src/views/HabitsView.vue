<template>
  <div class="habits">
    <div class="habits-head">
      <p class="hint">每天一小步，坚持看得见</p>
      <el-button type="primary" round :icon="Plus" @click="openCreate">新建习惯</el-button>
    </div>

    <div v-if="habitStore.loading" class="habits-grid">
      <div v-for="i in 4" :key="i" class="skeleton-habit skeleton-shimmer"></div>
    </div>
    <transition-group v-else-if="habitStore.habits.length" name="list" tag="div" class="habits-grid">
      <HabitCard
        v-for="h in habitStore.habits"
        :key="h.id"
        :habit="h"
        :checked="habitStore.isCheckedToday(h.id)"
        :streak="habitStore.streaks[h.id] || 0"
        @toggle="onToggle(h)"
        @remove="onRemove(h)"
      />
    </transition-group>
    <EmptyState v-else title="还没有习惯" description="比如：喝水、运动、早睡，从今天开始坚持" icon="Medal">
      <template #action>
        <el-button type="primary" round @click="openCreate">新建习惯</el-button>
      </template>
    </EmptyState>

    <el-dialog v-model="dialogVisible" title="新建习惯" width="420px" align-center destroy-on-close>
      <el-form label-position="top">
        <el-form-item label="习惯名称">
          <el-input v-model="name" placeholder="例如：喝水 2000ml" maxlength="64" show-word-limit />
        </el-form-item>
        <el-form-item label="图标（可选，输入 emoji）">
          <el-input v-model="icon" placeholder="例如：💧 🏃 🌙" maxlength="4" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button round @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" round :loading="saving" @click="save">创建</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'
import HabitCard from '@/components/HabitCard.vue'
import EmptyState from '@/components/EmptyState.vue'
import { useHabitStore } from '@/stores/habit'
import type { Habit } from '@/types'

const habitStore = useHabitStore()
const dialogVisible = ref(false)
const saving = ref(false)
const name = ref('')
const icon = ref('')

onMounted(() => habitStore.fetchAll())

function openCreate() {
  name.value = ''
  icon.value = ''
  dialogVisible.value = true
}

async function save() {
  if (!name.value.trim()) {
    ElMessage.warning('请输入习惯名称')
    return
  }
  saving.value = true
  try {
    await habitStore.create(name.value.trim(), icon.value || undefined)
    ElMessage.success('习惯已创建，今天就开始吧')
    dialogVisible.value = false
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    saving.value = false
  }
}

async function onToggle(h: Habit) {
  try {
    if (habitStore.isCheckedToday(h.id)) {
      await habitStore.uncheckin(h.id)
      ElMessage.success('已取消今日打卡')
    } else {
      await habitStore.checkin(h.id)
      ElMessage.success(`「${h.name}」打卡成功`)
    }
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

async function onRemove(h: Habit) {
  try {
    await ElMessageBox.confirm(`确定删除习惯「${h.name}」吗？打卡记录将保留在后台。`, '删除习惯', {
      confirmButtonText: '删除',
      cancelButtonText: '取消',
      type: 'warning',
      roundButton: true,
    })
    await habitStore.remove(h.id)
    ElMessage.success('已删除')
  } catch (e) {
    if (e === 'cancel' || e === 'close') return
    ElMessage.error((e as Error).message)
  }
}
</script>

<style scoped>
.habits-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 18px;
}
.hint {
  margin: 0;
  font-size: 13px;
  color: var(--vibe-text-secondary);
}
.habits-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 16px;
}
.skeleton-habit {
  height: 96px;
  border-radius: var(--vibe-radius-lg);
}
</style>
