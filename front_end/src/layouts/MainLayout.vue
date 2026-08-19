<template>
  <div class="layout">
    <!-- 侧边导航 -->
    <aside class="sidebar">
      <div class="brand">
        <div class="brand-logo">V</div>
        <span class="brand-name">VIBE</span>
      </div>

      <nav class="nav">
        <router-link
          v-for="item in navItems"
          :key="item.name"
          :to="{ name: item.name }"
          class="nav-item"
          :class="{ active: isActive(item.name) }"
        >
          <el-icon :size="18"><component :is="item.icon" /></el-icon>
          <span>{{ item.label }}</span>
          <span v-if="item.name === 'notifications' && notificationStore.unreadCount > 0" class="nav-badge">
            {{ notificationStore.unreadCount }}
          </span>
        </router-link>
      </nav>

      <div class="sidebar-footer">
        <div class="user-chip">
          <el-avatar :size="34" class="user-avatar">{{ avatarText }}</el-avatar>
          <div class="user-meta">
            <div class="user-name">{{ authStore.user?.nickname }}</div>
            <div class="user-sub">@{{ authStore.user?.username }}</div>
          </div>
        </div>
      </div>
    </aside>

    <!-- 主区域 -->
    <div class="main">
      <header class="topbar">
        <div class="topbar-left">
          <h1 class="page-title">{{ pageTitle }}</h1>
          <p class="page-subtitle">{{ subtitle }}</p>
        </div>
        <div class="topbar-right">
          <!-- 全局搜索 -->
          <el-popover placement="bottom-end" :width="320" trigger="manual" v-model:visible="searchVisible">
            <template #reference>
              <div class="search-box">
                <el-icon class="search-icon"><Search /></el-icon>
                <el-input
                  v-model="keyword"
                  placeholder="搜索待办、记事、纪念日…"
                  clearable
                  @input="onSearchInput"
                  @focus="searchVisible = true"
                  @blur="hideSearchSoon"
                />
              </div>
            </template>
            <div class="search-results">
              <div v-if="searchLoading" class="search-tip">搜索中…</div>
              <div v-else-if="searchItems.length === 0" class="search-tip">没有找到相关内容</div>
              <div
                v-for="item in searchItems"
                :key="item.type + item.id"
                class="search-item"
                @mousedown.prevent="goSearchItem(item)"
              >
                <el-tag size="small" :type="tagType(item.type)" round>{{ typeLabel(item.type) }}</el-tag>
                <div class="search-item-body">
                  <div class="search-item-title">{{ item.title }}</div>
                  <div class="search-item-sub">{{ item.subtitle }}</div>
                </div>
              </div>
            </div>
          </el-popover>

          <!-- 提醒铃铛 -->
          <el-badge :value="notificationStore.unreadCount" :hidden="notificationStore.unreadCount === 0" class="bell">
            <button class="icon-btn" @click="router.push({ name: 'notifications' })">
              <el-icon :size="20"><Bell /></el-icon>
            </button>
          </el-badge>

          <!-- 用户菜单 -->
          <el-dropdown trigger="click" @command="onUserCommand">
            <button class="icon-btn avatar-btn">
              <el-avatar :size="32" class="user-avatar">{{ avatarText }}</el-avatar>
            </button>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item disabled>{{ authStore.user?.nickname }}</el-dropdown-item>
                <el-dropdown-item divided command="logout">
                  <el-icon><SwitchButton /></el-icon>退出登录
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </header>

      <main class="content">
        <router-view v-slot="{ Component }">
          <transition name="page" mode="out-in">
            <component :is="Component" />
          </transition>
        </router-view>
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { useAuthStore } from '@/stores/auth'
import { useNotificationStore } from '@/stores/notification'
import { searchApi } from '@/api/modules'
import { startSse, stopSse } from '@/utils/sse'
import { greeting } from '@/utils/format'
import type { SearchItem, SseReminderEvent } from '@/types'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
const notificationStore = useNotificationStore()

// 导航配置
const navItems = [
  { name: 'today', label: '今日', icon: 'Sunny' },
  { name: 'todos', label: '全部待办', icon: 'List' },
  { name: 'calendar', label: '日历', icon: 'Calendar' },
  { name: 'completed', label: '已完成', icon: 'CircleCheck' },
  { name: 'notes', label: '记事本', icon: 'Notebook' },
  { name: 'habits', label: '习惯打卡', icon: 'Medal' },
  { name: 'anniversaries', label: '纪念日', icon: 'Present' },
  { name: 'notifications', label: '提醒中心', icon: 'Bell' },
]

const pageTitle = computed(() => (route.meta.title as string) || 'VIBE')
const subtitle = computed(() => (route.name === 'today' ? `${greeting()}，今天也要元气满满` : '把生活安排得井井有条'))
const avatarText = computed(() => (authStore.user?.nickname || 'U').slice(0, 1).toUpperCase())
const isActive = (name: string) => route.name === name

// ---------- 搜索 ----------
const keyword = ref('')
const searchItems = ref<SearchItem[]>([])
const searchLoading = ref(false)
const searchVisible = ref(false)
let searchTimer: number | undefined

function onSearchInput() {
  window.clearTimeout(searchTimer)
  if (!keyword.value.trim()) {
    searchItems.value = []
    return
  }
  searchTimer = window.setTimeout(async () => {
    searchLoading.value = true
    try {
      searchItems.value = await searchApi.search(keyword.value.trim())
    } finally {
      searchLoading.value = false
    }
  }, 300)
}

function hideSearchSoon() {
  // 延迟隐藏，保证点击结果能触发
  window.setTimeout(() => {
    searchVisible.value = false
  }, 150)
}

function goSearchItem(item: SearchItem) {
  searchVisible.value = false
  const target = item.type === 'todo' ? 'todos' : item.type === 'note' ? 'notes' : 'anniversaries'
  router.push({ name: target })
}

function typeLabel(t: string) {
  return { todo: '待办', note: '记事', anniversary: '纪念日' }[t] || t
}

function tagType(t: string): 'primary' | 'success' | 'warning' {
  return t === 'todo' ? 'primary' : t === 'note' ? 'success' : 'warning'
}

// ---------- 用户与 SSE ----------
async function onUserCommand(cmd: string) {
  if (cmd === 'logout') {
    await authStore.logout()
    ElMessage.success('已安全退出')
    router.replace({ name: 'login' })
  }
}

onMounted(async () => {
  notificationStore.fetchList()
  // 登录态下启动 SSE 实时提醒
  startSse({
    onReminder: (raw) => {
      const ev = raw as unknown as SseReminderEvent
      if ('Notification' in window && Notification.permission === 'granted') {
        new Notification('VIBE 提醒', { body: ev.title })
      }
      ElMessage({ type: 'warning', message: ev.title, duration: 5000, grouping: true })
      notificationStore.pushLocal({
        id: ev.notification_id,
        user_id: ev.user_id,
        title: ev.title,
        content: ev.content || '',
        target_type: ev.type === 'todo' ? 1 : 2,
        target_id: ev.task_id,
        is_read: false,
        created_at: ev.remind_at,
      })
    },
  })
})

onUnmounted(() => {
  stopSse()
  window.clearTimeout(searchTimer)
})
</script>

<style scoped>
.layout {
  display: flex;
  height: 100%;
}

/* ---------- 侧边栏 ---------- */
.sidebar {
  width: 224px;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  padding: 24px 14px 18px;
  background: var(--vibe-surface);
  border-right: 1px solid var(--vibe-border);
}

.brand {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 0 10px 26px;
}
.brand-logo {
  width: 40px;
  height: 40px;
  display: grid;
  place-items: center;
  border-radius: 14px;
  background: linear-gradient(135deg, #7c8cff, #5a69e6);
  color: #fff;
  font-weight: 800;
  font-size: 20px;
  box-shadow: 0 8px 20px rgba(108, 123, 255, 0.35);
}
.brand-name {
  font-size: 20px;
  font-weight: 800;
  letter-spacing: 1.5px;
}

.nav {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 4px;
  overflow-y: auto;
}
.nav-item {
  position: relative;
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
  border-radius: var(--vibe-radius-md);
  color: var(--vibe-text-secondary);
  text-decoration: none;
  font-size: 14.5px;
  font-weight: 500;
  transition: all 0.25s var(--vibe-ease);
}
.nav-item:hover {
  color: var(--vibe-primary);
  background: var(--vibe-primary-soft);
}
.nav-item.active {
  color: var(--vibe-primary);
  background: var(--vibe-primary-soft);
  font-weight: 600;
}
.nav-item.active::before {
  content: '';
  position: absolute;
  left: 0;
  top: 50%;
  transform: translateY(-50%);
  width: 4px;
  height: 20px;
  border-radius: 999px;
  background: var(--vibe-primary);
}
.nav-badge {
  margin-left: auto;
  min-width: 20px;
  height: 20px;
  padding: 0 6px;
  display: grid;
  place-items: center;
  border-radius: 999px;
  background: var(--vibe-danger);
  color: #fff;
  font-size: 12px;
  font-weight: 600;
}

.sidebar-footer {
  padding-top: 14px;
}
.user-chip {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px;
  border-radius: var(--vibe-radius-md);
  background: var(--vibe-surface-2);
}
.user-avatar {
  background: linear-gradient(135deg, #a5b0ff, #6c7bff);
  color: #fff;
  font-weight: 700;
}
.user-meta {
  min-width: 0;
}
.user-name {
  font-size: 14px;
  font-weight: 600;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.user-sub {
  font-size: 12px;
  color: var(--vibe-text-tertiary);
}

/* ---------- 主区域 ---------- */
.main {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.topbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 22px 34px 14px;
}
.page-title {
  margin: 0;
  font-size: 24px;
  font-weight: 800;
}
.page-subtitle {
  margin: 4px 0 0;
  font-size: 13px;
  color: var(--vibe-text-secondary);
}

.topbar-right {
  display: flex;
  align-items: center;
  gap: 14px;
}

.search-box {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 300px;
  padding: 0 12px;
  background: var(--vibe-surface);
  border: 1px solid var(--vibe-border);
  border-radius: 999px;
  transition: box-shadow 0.25s var(--vibe-ease);
}
.search-box:focus-within {
  box-shadow: 0 0 0 4px var(--vibe-primary-soft);
}
.search-icon {
  color: var(--vibe-text-tertiary);
}
.search-box :deep(.el-input__wrapper) {
  background: transparent;
  box-shadow: none;
  padding: 6px 0;
}

.icon-btn {
  width: 40px;
  height: 40px;
  display: grid;
  place-items: center;
  border: 1px solid var(--vibe-border);
  border-radius: 999px;
  background: var(--vibe-surface);
  color: var(--vibe-text-secondary);
  cursor: pointer;
  transition: all 0.25s var(--vibe-ease);
}
.icon-btn:hover {
  color: var(--vibe-primary);
  border-color: var(--vibe-primary);
  background: var(--vibe-primary-soft);
  transform: translateY(-1px);
}

.search-results {
  max-height: 320px;
  overflow-y: auto;
}
.search-tip {
  padding: 18px;
  text-align: center;
  color: var(--vibe-text-tertiary);
  font-size: 13px;
}
.search-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 8px;
  border-radius: var(--vibe-radius-sm);
  cursor: pointer;
  transition: background 0.2s var(--vibe-ease);
}
.search-item:hover {
  background: var(--vibe-primary-soft);
}
.search-item-body {
  min-width: 0;
}
.search-item-title {
  font-size: 14px;
  font-weight: 600;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.search-item-sub {
  font-size: 12px;
  color: var(--vibe-text-tertiary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.content {
  flex: 1;
  overflow-y: auto;
  padding: 8px 34px 40px;
}
</style>
