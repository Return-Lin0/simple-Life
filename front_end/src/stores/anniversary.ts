// 纪念日状态
import { defineStore } from 'pinia'
import { anniversaryApi } from '@/api/modules'
import type { Anniversary } from '@/types'

export const useAnniversaryStore = defineStore('anniversary', {
  state: () => ({
    list: [] as Anniversary[],
    loading: false,
  }),
  actions: {
    async fetchList() {
      this.loading = true
      try {
        this.list = await anniversaryApi.list()
      } finally {
        this.loading = false
      }
    },
    async create(data: Partial<Anniversary>) {
      await anniversaryApi.create(data)
      await this.fetchList()
    },
    async update(id: number, data: Partial<Anniversary>) {
      await anniversaryApi.update(id, data)
      await this.fetchList()
    },
    async remove(id: number) {
      await anniversaryApi.remove(id)
      await this.fetchList()
    },
  },
})
