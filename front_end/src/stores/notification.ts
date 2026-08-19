// 提醒中心状态
import { defineStore } from 'pinia'
import { notificationApi } from '@/api/modules'
import type { Notification } from '@/types'

export const useNotificationStore = defineStore('notification', {
  state: () => ({
    list: [] as Notification[],
    total: 0,
    loading: false,
  }),
  getters: {
    unreadCount: (s) => s.list.filter((n) => !n.is_read).length,
  },
  actions: {
    async fetchList() {
      this.loading = true
      try {
        const data = await notificationApi.list({ page: 1, page_size: 50 })
        this.list = data.list
        this.total = data.total
      } finally {
        this.loading = false
      }
    },
    async markRead(id: number) {
      await notificationApi.markRead(id)
      const item = this.list.find((n) => n.id === id)
      if (item) item.is_read = true
    },
    // SSE 实时到达：插入到列表头部
    pushLocal(n: Notification) {
      if (!this.list.some((x) => x.id === n.id)) {
        this.list.unshift(n)
        this.total += 1
      }
    },
  },
})
