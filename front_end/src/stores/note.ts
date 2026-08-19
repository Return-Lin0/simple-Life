// 记事状态
import { defineStore } from 'pinia'
import { noteApi } from '@/api/modules'
import type { Note } from '@/types'

export const useNoteStore = defineStore('note', {
  state: () => ({
    list: [] as Note[],
    total: 0,
    loading: false,
    page: 1,
    pageSize: 20,
  }),
  actions: {
    async fetchList() {
      this.loading = true
      try {
        const data = await noteApi.list({ page: this.page, page_size: this.pageSize })
        this.list = data.list
        this.total = data.total
      } finally {
        this.loading = false
      }
    },
    async create(title: string, content: string) {
      await noteApi.create({ title, content })
      await this.fetchList()
    },
    async update(id: number, title: string, content: string) {
      await noteApi.update(id, { title, content })
      await this.fetchList()
    },
    async remove(id: number) {
      await noteApi.remove(id)
      await this.fetchList()
    },
  },
})
