<template>
  <div class="anns">
    <div class="anns-head">
      <p class="hint">重要的日子，一个都不错过</p>
      <el-button type="primary" round :icon="Plus" @click="openCreate">添加纪念日</el-button>
    </div>

    <div v-if="store.loading" class="anns-grid">
      <div v-for="i in 3" :key="i" class="skeleton-ann skeleton-shimmer"></div>
    </div>
    <transition-group v-else-if="store.list.length" name="list" tag="div" class="anns-grid">
      <AnniversaryCard
        v-for="a in store.list"
        :key="a.id"
        :item="a"
        @edit="openEdit(a)"
        @remove="onRemove(a)"
      />
    </transition-group>
    <EmptyState v-else title="还没有纪念日" description="添加生日、结婚纪念日等，自动计算倒计时" icon="Present">
      <template #action>
        <el-button type="primary" round @click="openCreate">添加纪念日</el-button>
      </template>
    </EmptyState>

    <el-dialog v-model="dialogVisible" :title="editing ? '编辑纪念日' : '添加纪念日'" width="460px" align-center destroy-on-close>
      <el-form label-position="top">
        <el-form-item label="名称">
          <el-input v-model="form.name" placeholder="例如：妈妈的生日" maxlength="64" show-word-limit />
        </el-form-item>
        <el-form-item label="日期">
          <el-date-picker v-model="form.event_date" type="date" value-format="YYYY-MM-DD" placeholder="选择日期" style="width: 100%" />
        </el-form-item>
        <el-form-item label="每年重复">
          <el-switch v-model="form.repeat_yearly" />
          <span class="form-tip">开启后每年自动计算倒计时</span>
        </el-form-item>
        <el-form-item label="提醒">
          <el-switch v-model="form.remind_enabled" />
        </el-form-item>
        <el-form-item v-if="form.remind_enabled" label="提前几天提醒">
          <el-input-number v-model="form.remind_days_before" :min="0" :max="30" />
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
import { Plus } from '@element-plus/icons-vue'
import AnniversaryCard from '@/components/AnniversaryCard.vue'
import EmptyState from '@/components/EmptyState.vue'
import { useAnniversaryStore } from '@/stores/anniversary'
import type { Anniversary } from '@/types'

const store = useAnniversaryStore()
const dialogVisible = ref(false)
const editing = ref<Anniversary | null>(null)
const saving = ref(false)
const form = reactive({
  name: '',
  event_date: '',
  repeat_yearly: true,
  remind_enabled: false,
  remind_days_before: 1,
})

onMounted(() => store.fetchList())

function openCreate() {
  editing.value = null
  Object.assign(form, { name: '', event_date: '', repeat_yearly: true, remind_enabled: false, remind_days_before: 1 })
  dialogVisible.value = true
}

function openEdit(a: Anniversary) {
  editing.value = a
  Object.assign(form, {
    name: a.name,
    event_date: a.event_date,
    repeat_yearly: a.repeat_yearly,
    remind_enabled: a.remind_enabled,
    remind_days_before: a.remind_days_before,
  })
  dialogVisible.value = true
}

async function save() {
  if (!form.name.trim()) return ElMessage.warning('请输入名称')
  if (!form.event_date) return ElMessage.warning('请选择日期')
  saving.value = true
  try {
    const payload = { ...form, is_lunar: false }
    if (editing.value) {
      await store.update(editing.value.id, payload)
      ElMessage.success('纪念日已更新')
    } else {
      await store.create(payload)
      ElMessage.success('纪念日已添加')
    }
    dialogVisible.value = false
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    saving.value = false
  }
}

async function onRemove(a: Anniversary) {
  try {
    await ElMessageBox.confirm(`确定删除「${a.name}」吗？`, '删除纪念日', {
      confirmButtonText: '删除',
      cancelButtonText: '取消',
      type: 'warning',
      roundButton: true,
    })
    await store.remove(a.id)
    ElMessage.success('已删除')
  } catch (e) {
    if (e === 'cancel' || e === 'close') return
    ElMessage.error((e as Error).message)
  }
}
</script>

<style scoped>
.anns-head {
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
.anns-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 16px;
}
.skeleton-ann {
  height: 104px;
  border-radius: var(--vibe-radius-lg);
}
.form-tip {
  margin-left: 10px;
  font-size: 12px;
  color: var(--vibe-text-tertiary);
}
</style>
