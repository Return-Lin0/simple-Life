// 与后端 DTO 对应的类型定义（字段与《技术设计文档》第 9 章一致）

export interface ApiBody<T = unknown> {
  code: number
  message: string
  data?: T
}

export interface UserInfo {
  id: number
  username: string
  nickname: string
  email?: string
  avatar_url?: string
}

export interface LoginResp {
  access_token: string
  expires_in: number
  user: UserInfo
}

export interface Tag {
  id: number
  user_id: number
  name: string
  color?: string
}

export interface Todo {
  id: number
  user_id: number
  title: string
  description?: string
  event_date: string
  start_time?: string
  end_time?: string
  is_all_day: boolean
  priority: number // 0 高 / 1 中 / 2 低
  category: string
  status: number // 0 未完成 / 1 已完成
  recurrence_type: number // 0 无 / 1 每天 / 2 每周 / 3 每月
  recurrence_rule?: string
  parent_id?: number
  reminder_enabled: boolean
  remind_offset_minutes?: number
  completed_at?: string
  created_at: string
  updated_at: string
  tags: Tag[]
}

export interface TodoView extends Todo {
  overdue: boolean
}

export interface TodoReq {
  title: string
  description?: string
  event_date: string
  start_time?: string
  end_time?: string
  is_all_day?: boolean
  priority?: number
  category?: string
  tags?: number[]
  recurrence_type?: number
  recurrence_rule?: string
  reminder_enabled?: boolean
  remind_offset_minutes?: number
}

export interface Note {
  id: number
  user_id: number
  title: string
  content: string
  source_todo_id?: number
  created_at: string
  updated_at: string
}

export interface Habit {
  id: number
  user_id: number
  name: string
  icon?: string
  target_weekly_days: number
  created_at: string
}

export interface Anniversary {
  id: number
  user_id: number
  name: string
  event_date: string
  is_lunar: boolean
  repeat_yearly: boolean
  remind_enabled: boolean
  remind_days_before: number
  next_occurrence: string
  days_left: number
  is_today: boolean
}

export interface Notification {
  id: number
  user_id: number
  title: string
  content?: string
  target_type: number
  target_id: number
  is_read: boolean
  created_at: string
}

export interface SearchItem {
  type: 'todo' | 'note' | 'anniversary'
  id: number
  title: string
  subtitle: string
}

export interface PageData<T> {
  list: T[]
  total: number
  page: number
  page_size: number
}

export interface TodoFilter {
  view?: string
  status?: number
  category?: string
  start_date?: string
  end_date?: string
  keyword?: string
  tag_ids?: string
  sort_by?: string
  order?: string
  page?: number
  page_size?: number
}

export interface SseReminderEvent {
  user_id: number
  event: 'reminder'
  task_id: number
  type: string
  title: string
  content?: string
  remind_at: string
  notification_id: number
}
