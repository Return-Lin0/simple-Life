// 待办状态：今日视图与列表视图
import { defineStore } from 'pinia'
import { todoApi } from '@/api/modules'
import type { TodoFilter, TodoReq, TodoView } from '@/types'

export const useTodoStore = defineStore('todo', {
  state: () => ({
    today: [] as TodoView[],
    list: [] as TodoView[],
    total: 0,
    loading: false,
    currentPage: 1,
    pageSize: 20,
  }),
  getters: {
    overdueCount: (s) => s.today.filter((t) => t.overdue).length,
  },
  actions: {
    async fetchToday() {
      this.loading = true
      try {
        this.today = await todoApi.today()
      } finally {
        this.loading = false
      }
    },
    async fetchList(filter: TodoFilter = {}) {
      this.loading = true
      try {
        const params: TodoFilter = {
          page: this.currentPage,
          page_size: this.pageSize,
          ...filter,
        }
        const data = await todoApi.list(params)
        this.list = data.list
        this.total = data.total
      } finally {
        this.loading = false
      }
    },
    async create(data: TodoReq) {
      const resp = await todoApi.create(data)
      return resp.id
    },
    async update(id: number, data: TodoReq) {
      await todoApi.update(id, data)
    },
    async remove(id: number) {
      await todoApi.remove(id)
    },
    async toggleStatus(todo: TodoView) {
      const next = todo.status === 0 ? 1 : 0
      await todoApi.setStatus(todo.id, next)
      todo.status = next
    },
    async convertToNote(id: number) {
      return todoApi.convertToNote(id)
    },
    async batchDelete(ids: number[]) {
      return todoApi.batchDelete(ids)
    },
    async batchUpdateStatus(ids: number[], status: number) {
      return todoApi.batchStatus(ids, status)
    },
  },
})
