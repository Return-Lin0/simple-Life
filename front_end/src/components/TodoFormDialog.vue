<template>
  <el-dialog
    :model-value="visible"
    :title="isEdit ? '编辑待办' : '新建待办'"
    width="560px"
    align-center
    destroy-on-close
    @update:model-value="$emit('update:visible', $event)"
    @open="onOpen"
  >
    <el-form ref="formRef" :model="form" :rules="rules" label-position="top" class="todo-form">
      <el-form-item label="名称" prop="title">
        <el-input v-model="form.title" placeholder="要做什么？例如：14:00 与团队开会" maxlength="128" show-word-limit />
      </el-form-item>

      <el-form-item label="备注">
        <el-input v-model="form.description" type="textarea" :rows="2" placeholder="补充细节、地址、材料…" />
      </el-form-item>

      <div class="form-grid">
        <el-form-item label="日期" prop="event_date">
          <el-date-picker
            v-model="form.event_date"
            type="date"
            value-format="YYYY-MM-DD"
            placeholder="选择日期"
            :disabled-date="disabledDate"
            style="width: 100%"
          />
        </el-form-item>
        <el-form-item label="全天事项">
          <el-switch v-model="form.is_all_day" @change="onAllDayChange" />
        </el-form-item>
      </div>

      <div v-if="!form.is_all_day" class="form-grid">
        <el-form-item label="开始时间">
          <el-time-picker v-model="form.start_time" value-format="HH:mm:ss" format="HH:mm" placeholder="开始" style="width: 100%" />
        </el-form-item>
        <el-form-item label="结束时间">
          <el-time-picker v-model="form.end_time" value-format="HH:mm:ss" format="HH:mm" placeholder="结束（可选）" style="width: 100%" />
        </el-form-item>
      </div>

      <div class="form-grid">
        <el-form-item label="优先级">
          <el-radio-group v-model="form.priority">
            <el-radio-button :value="0">高</el-radio-button>
            <el-radio-button :value="1">中</el-radio-button>
            <el-radio-button :value="2">低</el-radio-button>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="分类">
          <el-select v-model="form.category" style="width: 100%">
            <el-option v-for="(label, key) in categoryOptions" :key="key" :label="label" :value="key" />
          </el-select>
        </el-form-item>
      </div>

      <el-form-item label="标签">
        <el-select v-model="form.tags" multiple collapse-tags collapse-tags-tooltip placeholder="选择标签" style="width: 100%">
          <el-option v-for="tag in tags" :key="tag.id" :label="tag.name" :value="tag.id" />
        </el-select>
      </el-form-item>

      <div class="form-grid">
        <el-form-item label="重复">
          <el-select v-model="form.recurrence_type" style="width: 100%" @change="onRecurrenceChange">
            <el-option :value="0" label="不重复" />
            <el-option :value="1" label="每天" />
            <el-option :value="2" label="每周" />
            <el-option :value="3" label="每月" />
          </el-select>
        </el-form-item>
        <el-form-item v-if="form.recurrence_type === 2" label="每周哪几天">
          <el-select v-model="weekdays" multiple collapse-tags placeholder="选择星期" style="width: 100%">
            <el-option v-for="(name, i) in weekdayOptions" :key="i" :label="name" :value="i + 1" />
          </el-select>
        </el-form-item>
      </div>

      <div class="remind-row">
        <el-form-item label="提醒" class="remind-switch">
          <el-switch v-model="form.reminder_enabled" />
        </el-form-item>
        <el-form-item v-if="form.reminder_enabled" label="提前" class="remind-offset">
          <el-input-number v-model="form.remind_offset_minutes" :min="0" :max="1440" />
          <span class="offset-unit">分钟</span>
        </el-form-item>
      </div>
    </el-form>

    <template #footer>
      <el-button round @click="$emit('update:visible', false)">取消</el-button>
      <el-button type="primary" round :loading="saving" @click="submit">保存</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { ElMessage, type FormInstance, type FormRules } from 'element-plus'
import dayjs from 'dayjs'
import { useTodoStore } from '@/stores/todo'
import { buildRecurrenceRule, WEEKDAY_NAMES } from '@/utils/format'
import type { Tag, Todo } from '@/types'

const props = defineProps<{
  visible: boolean
  todo: Todo | null
  tags: Tag[]
  presetDate?: string
}>()
const emit = defineEmits<{
  (e: 'update:visible', v: boolean): void
  (e: 'saved'): void
}>()

const todoStore = useTodoStore()
const formRef = ref<FormInstance>()
const saving = ref(false)
const weekdays = ref<number[]>([1])

const categoryOptions: Record<string, string> = { life: '生活', work: '工作', study: '学习', health: '健康', other: '其他' }
const weekdayOptions = WEEKDAY_NAMES
const isEdit = computed(() => props.todo !== null)

const form = reactive({
  title: '',
  description: '',
  event_date: '',
  start_time: '',
  end_time: '',
  is_all_day: false,
  priority: 1,
  category: 'life',
  tags: [] as number[],
  recurrence_type: 0,
  reminder_enabled: false,
  remind_offset_minutes: 10,
})

const rules: FormRules = {
  title: [
    { required: true, message: '请输入事项名称', trigger: 'blur' },
    { min: 1, max: 128, message: '名称长度 1~128 个字符', trigger: 'blur' },
  ],
  event_date: [{ required: true, message: '请选择日期', trigger: 'change' }],
}

// 禁止选择早于今天的日期；编辑已有逾期事项时允许保留原日期
function disabledDate(date: Date) {
  const d = dayjs(date).format('YYYY-MM-DD')
  const today = dayjs().format('YYYY-MM-DD')
  if (d < today) {
    return !(isEdit.value && props.todo && d === props.todo.event_date)
  }
  return false
}

function onOpen() {
  weekdays.value = [1]
  if (props.todo) {
    Object.assign(form, {
      title: props.todo.title,
      description: props.todo.description || '',
      event_date: props.todo.event_date,
      start_time: props.todo.start_time || '',
      end_time: props.todo.end_time || '',
      is_all_day: props.todo.is_all_day,
      priority: props.todo.priority,
      category: props.todo.category || 'life',
      tags: props.todo.tags.map((t) => t.id),
      recurrence_type: props.todo.recurrence_type,
      reminder_enabled: props.todo.reminder_enabled,
      remind_offset_minutes: props.todo.remind_offset_minutes ?? 10,
    })
    if (props.todo.recurrence_rule) {
      try {
        const rule = JSON.parse(props.todo.recurrence_rule)
        if (rule && Array.isArray(rule.weekdays)) weekdays.value = rule.weekdays
      } catch {
        weekdays.value = [1]
      }
    }
  } else {
    Object.assign(form, {
      title: '',
      description: '',
      event_date: '',
      start_time: '',
      end_time: '',
      is_all_day: false,
      priority: 1,
      category: 'life',
      tags: [],
      recurrence_type: 0,
      reminder_enabled: false,
      remind_offset_minutes: 10,
    })
    if (props.presetDate) {
      form.event_date = props.presetDate
    }
  }
}

function onAllDayChange(v: boolean) {
  if (v) {
    form.start_time = ''
    form.end_time = ''
  }
}

function onRecurrenceChange(v: number) {
  if (v !== 2) weekdays.value = []
}

async function submit() {
  if (!formRef.value) return
  const valid = await formRef.value.validate().catch(() => false)
  if (!valid) return

  // 前端时间段校验（与后端一致）
  if (!form.is_all_day && form.start_time && form.end_time && form.end_time <= form.start_time) {
    ElMessage.warning('结束时间必须晚于开始时间')
    return
  }
  // 日期校验：不能早于今天（编辑时允许保留原逾期日期）
  const today = dayjs().format('YYYY-MM-DD')
  if (form.event_date < today) {
    if (!(isEdit.value && props.todo && form.event_date === props.todo.event_date)) {
      ElMessage.warning('待办日期不能早于今天，请重新选择日期')
      return
    }
  }
  // 今天的事项：开始时间不能早于当前时间
  if (!form.is_all_day && form.start_time && form.event_date === today) {
    const nowClock = dayjs().format('HH:mm:ss')
    if (form.start_time < nowClock) {
      ElMessage.warning('今天的事项开始时间不能早于当前时间')
      return
    }
  }

  const payload = {
    title: form.title.trim(),
    description: form.description,
    event_date: form.event_date,
    start_time: form.is_all_day ? '' : form.start_time || '',
    end_time: form.is_all_day ? '' : form.end_time || '',
    is_all_day: form.is_all_day,
    priority: form.priority,
    category: form.category,
    tags: form.tags,
    recurrence_type: form.recurrence_type,
    recurrence_rule: form.recurrence_type === 2 ? buildRecurrenceRule([...weekdays.value].sort((a, b) => a - b)) : '',
    reminder_enabled: form.reminder_enabled,
    remind_offset_minutes: form.reminder_enabled ? form.remind_offset_minutes : 0,
  }

  saving.value = true
  try {
    if (isEdit.value && props.todo) {
      await todoStore.update(props.todo.id, payload)
      ElMessage.success('待办已更新')
    } else {
      await todoStore.create(payload)
      ElMessage.success('待办已创建')
    }
    emit('update:visible', false)
    emit('saved')
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
.todo-form {
  padding-top: 6px;
}
.form-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0 18px;
}
.remind-row {
  display: flex;
  align-items: flex-start;
  gap: 24px;
}
.remind-switch {
  margin-bottom: 0;
}
.offset-unit {
  margin-left: 8px;
  color: var(--vibe-text-secondary);
  font-size: 13px;
}
</style>
