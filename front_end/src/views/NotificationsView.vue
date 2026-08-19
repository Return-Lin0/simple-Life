<template>
  <div class="notifications">
    <div v-if="store.loading" class="loading-grid">
      <div v-for="i in 4" :key="i" class="skeleton-card skeleton-shimmer"></div>
    </div>
    <transition-group v-else-if="store.list.length" name="list" tag="div">
      <div
        v-for="n in store.list"
        :key="n.id"
        class="notify-card vibe-card"
        :class="{ unread: !n.is_read }"
        @click="onClick(n)"
      >
        <span class="unread-dot" :class="{ read: n.is_read }"></span>
        <div class="notify-body">
          <div class="notify-title">{{ n.title }}</div>
          <div v-if="n.content" class="notify-content">{{ n.content }}</div>
          <div class="notify-time">{{ formatDateTime(n.created_at) }}</div>
        </div>
        <el-tag size="small" :type="n.target_type === 1 ? 'primary' : 'warning'" round effect="plain">
          {{ n.target_type === 1 ? '待办' : '纪念日' }}
        </el-tag>
      </div>
    </transition-group>
    <EmptyState v-else title="暂无提醒" description="提醒到达后会出现在这里" icon="Bell" />
  </div>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import EmptyState from '@/components/EmptyState.vue'
import { useNotificationStore } from '@/stores/notification'
import { formatDateTime } from '@/utils/format'
import type { Notification } from '@/types'

const store = useNotificationStore()

onMounted(() => store.fetchList())

async function onClick(n: Notification) {
  if (!n.is_read) {
    try {
      await store.markRead(n.id)
    } catch (e) {
      ElMessage.error((e as Error).message)
    }
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
  height: 84px;
  border-radius: var(--vibe-radius-lg);
}
.notify-card {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 16px 18px;
  margin-bottom: 12px;
  cursor: pointer;
}
.notify-card.unread {
  background: linear-gradient(0deg, #f6f7ff, #ffffff);
  border-color: rgba(108, 123, 255, 0.3);
}
.unread-dot {
  width: 10px;
  height: 10px;
  flex-shrink: 0;
  border-radius: 999px;
  background: var(--vibe-primary);
  box-shadow: 0 0 0 4px var(--vibe-primary-soft);
  transition: all 0.25s var(--vibe-ease);
}
.unread-dot.read {
  background: #d8dbe8;
  box-shadow: none;
}
.notify-body {
  flex: 1;
  min-width: 0;
}
.notify-title {
  font-size: 14.5px;
  font-weight: 600;
  margin-bottom: 4px;
}
.notify-content {
  font-size: 13px;
  color: var(--vibe-text-secondary);
  margin-bottom: 6px;
}
.notify-time {
  font-size: 12px;
  color: var(--vibe-text-tertiary);
}
</style>
