// 展示格式化工具
import dayjs from 'dayjs'

export const CATEGORY_LABELS: Record<string, string> = {
  life: '生活',
  work: '工作',
  study: '学习',
  health: '健康',
  other: '其他',
}

export const PRIORITY_LABELS = ['高', '中', '低'] as const
export const PRIORITY_COLORS = ['#ff6b7a', '#ffb454', '#8a90a3'] as const

export const RECURRENCE_LABELS = ['不重复', '每天', '每周', '每月'] as const

// 后端使用 ISO 星期：1=周一 … 7=周日；dayjs day() 为 0=周日 … 6=周六
export const WEEKDAY_NAMES = ['周一', '周二', '周三', '周四', '周五', '周六', '周日']

export function toIsoWeekday(day: number): number {
  return day === 0 ? 7 : day
}

export function fromIsoWeekday(iso: number): number {
  return iso === 7 ? 0 : iso
}

export function formatTime(t?: string): string {
  if (!t) return '全天'
  return t.slice(0, 5) // HH:mm:ss -> HH:mm
}

export function formatDateTime(s?: string): string {
  return s ? dayjs(s).format('MM月DD日 HH:mm') : ''
}

export function formatDateCN(s: string): string {
  return dayjs(s).format('M月D日')
}

export function weekdayCN(date: string): string {
  return WEEKDAY_NAMES[toIsoWeekday(dayjs(date).day()) - 1]
}

export function greeting(): string {
  const h = dayjs().hour()
  if (h < 6) return '夜深了'
  if (h < 12) return '早上好'
  if (h < 14) return '中午好'
  if (h < 18) return '下午好'
  return '晚上好'
}

export function countdownText(days: number): string {
  if (days === 0) return '就是今天'
  if (days > 0) return `还有 ${days} 天`
  return '已过'
}

export function buildRecurrenceRule(weekdays: number[]): string {
  return JSON.stringify({ weekdays })
}
