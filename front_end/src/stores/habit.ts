// 习惯打卡状态：打卡状态存 localStorage（UI 状态，非敏感数据），
// 服务端以唯一约束保证同一天不重复打卡。
import { defineStore } from 'pinia'
import { habitApi } from '@/api/modules'
import type { Habit } from '@/types'

const CHECKED_KEY = 'vibe.habit.checked'

function loadChecked(): Record<string, boolean> {
  try {
    return JSON.parse(localStorage.getItem(CHECKED_KEY) || '{}')
  } catch {
    return {}
  }
}

export const useHabitStore = defineStore('habit', {
  state: () => ({
    habits: [] as Habit[],
    streaks: {} as Record<number, number>,
    loading: false,
  }),
  getters: {
    checked: (s) => {
      const date = todayStr()
      const map = loadChecked()
      return s.habits.filter((h) => map[`${h.id}:${date}`])
    },
  },
  actions: {
    async fetchAll() {
      this.loading = true
      try {
        this.habits = await habitApi.list()
        await Promise.all(
          this.habits.map(async (h) => {
            const resp = await habitApi.streak(h.id)
            this.streaks[h.id] = resp.streak
          }),
        )
      } finally {
        this.loading = false
      }
    },
    isCheckedToday(id: number): boolean {
      const map = loadChecked()
      return !!map[`${id}:${todayStr()}`]
    },
    async checkin(id: number) {
      try {
        await habitApi.checkin(id, todayStr())
      } catch (err) {
        // 服务端 409（今天已打卡）视为成功状态
        const code = (err as { code?: number })?.code
        if (code !== 1005) throw err
      }
      const map = loadChecked()
      map[`${id}:${todayStr()}`] = true
      localStorage.setItem(CHECKED_KEY, JSON.stringify(map))
      await this.fetchAll()
    },
    async uncheckin(id: number) {
      await habitApi.uncheckin(id, todayStr())
      const map = loadChecked()
      delete map[`${id}:${todayStr()}`]
      localStorage.setItem(CHECKED_KEY, JSON.stringify(map))
      await this.fetchAll()
    },
    async create(name: string, icon?: string) {
      await habitApi.create({ name, icon })
      await this.fetchAll()
    },
    async remove(id: number) {
      await habitApi.remove(id)
      delete this.streaks[id]
      await this.fetchAll()
    },
  },
})

function todayStr(): string {
  const d = new Date()
  const m = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  return `${d.getFullYear()}-${m}-${day}`
}
