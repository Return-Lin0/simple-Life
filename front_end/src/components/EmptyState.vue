<template>
  <div class="empty">
    <div class="empty-icon">
      <el-icon :size="30"><component :is="icon" /></el-icon>
    </div>
    <div class="empty-title">{{ title }}</div>
    <div v-if="description" class="empty-desc">{{ description }}</div>
    <div v-if="$slots.action" class="empty-action">
      <slot name="action" />
    </div>
  </div>
</template>

<script setup lang="ts">
// 通用空状态：柔和图标 + 文案 + 可选操作按钮
withDefaults(
  defineProps<{
    title: string
    description?: string
    icon?: string
  }>(),
  { icon: 'Box', description: '' },
)
</script>

<style scoped>
.empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
  padding: 60px 20px;
  text-align: center;
}
.empty-icon {
  width: 72px;
  height: 72px;
  display: grid;
  place-items: center;
  border-radius: 24px;
  background: var(--vibe-primary-soft);
  color: var(--vibe-primary);
  animation: floaty 3.2s ease-in-out infinite;
}
@keyframes floaty {
  0%,
  100% {
    transform: translateY(0);
  }
  50% {
    transform: translateY(-6px);
  }
}
.empty-title {
  font-size: 16px;
  font-weight: 700;
}
.empty-desc {
  font-size: 13px;
  color: var(--vibe-text-secondary);
}
.empty-action {
  margin-top: 6px;
}
</style>
