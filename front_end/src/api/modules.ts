// 全部后端接口的 TypeScript 封装，路径与《技术设计文档》第 9 章一一对应
import { request } from './client'
import type {
  Anniversary,
  Habit,
  LoginResp,
  Note,
  Notification,
  PageData,
  SearchItem,
  Tag,
  Todo,
  TodoFilter,
  TodoReq,
  TodoView,
  UserInfo,
} from '@/types'

// ---------- 认证 ----------
export const authApi = {
  register: (data: { username: string; password: string; nickname: string; email?: string }) =>
    request<UserInfo>({ url: '/auth/register', method: 'POST', data }),
  login: (data: { username: string; password: string }) =>
    request<LoginResp>({ url: '/auth/login', method: 'POST', data }),
  refresh: () => request<{ access_token: string; expires_in: number }>({ url: '/auth/refresh', method: 'POST' }),
  logout: () => request<null>({ url: '/auth/logout', method: 'POST' }),
  me: () => request<UserInfo>({ url: '/auth/me', method: 'GET' }),
}

// ---------- 待办 ----------
export const todoApi = {
  list: (params: TodoFilter) =>
    request<PageData<TodoView>>({ url: '/todos', method: 'GET', params }),
  today: () => request<TodoView[]>({ url: '/todos', method: 'GET', params: { view: 'today' } }),
  calendar: (startDate: string, endDate: string) =>
    request<TodoView[]>({ url: '/todos/calendar', method: 'GET', params: { start_date: startDate, end_date: endDate } }),
  detail: (id: number) => request<TodoView>({ url: `/todos/${id}`, method: 'GET' }),
  create: (data: TodoReq) => request<{ id: number }>({ url: '/todos', method: 'POST', data }),
  update: (id: number, data: TodoReq) => request<Todo>({ url: `/todos/${id}`, method: 'PUT', data }),
  remove: (id: number) => request<null>({ url: `/todos/${id}`, method: 'DELETE' }),
  setStatus: (id: number, status: number) =>
    request<null>({ url: `/todos/${id}/status`, method: 'PATCH', data: { status } }),
  convertToNote: (id: number) => request<{ note_id: number }>({ url: `/todos/${id}/convert-to-note`, method: 'POST' }),
}

// ---------- 标签 ----------
export const tagApi = {
  list: () => request<Tag[]>({ url: '/tags', method: 'GET' }),
  create: (data: { name: string; color?: string }) => request<Tag>({ url: '/tags', method: 'POST', data }),
  update: (id: number, data: { name: string; color?: string }) =>
    request<Tag>({ url: `/tags/${id}`, method: 'PUT', data }),
  remove: (id: number) => request<null>({ url: `/tags/${id}`, method: 'DELETE' }),
}

// ---------- 记事 ----------
export const noteApi = {
  list: (params: { page?: number; page_size?: number }) =>
    request<PageData<Note>>({ url: '/notes', method: 'GET', params }),
  detail: (id: number) => request<Note>({ url: `/notes/${id}`, method: 'GET' }),
  create: (data: { title: string; content: string }) => request<Note>({ url: '/notes', method: 'POST', data }),
  update: (id: number, data: { title: string; content: string }) =>
    request<Note>({ url: `/notes/${id}`, method: 'PUT', data }),
  remove: (id: number) => request<null>({ url: `/notes/${id}`, method: 'DELETE' }),
}

// ---------- 习惯打卡 ----------
export const habitApi = {
  list: () => request<Habit[]>({ url: '/habits', method: 'GET' }),
  create: (data: { name: string; icon?: string; target_weekly_days?: number }) =>
    request<Habit>({ url: '/habits', method: 'POST', data }),
  update: (id: number, data: { name: string; icon?: string; target_weekly_days?: number }) =>
    request<Habit>({ url: `/habits/${id}`, method: 'PUT', data }),
  remove: (id: number) => request<null>({ url: `/habits/${id}`, method: 'DELETE' }),
  checkin: (id: number, date: string) =>
    request<null>({ url: `/habits/${id}/checkin`, method: 'POST', params: { date } }),
  uncheckin: (id: number, date: string) =>
    request<null>({ url: `/habits/${id}/checkin/${date}`, method: 'DELETE' }),
  streak: (id: number) => request<{ streak: number }>({ url: `/habits/${id}/streak`, method: 'GET' }),
}

// ---------- 纪念日 ----------
export const anniversaryApi = {
  list: () => request<Anniversary[]>({ url: '/anniversaries', method: 'GET' }),
  create: (data: Partial<Anniversary>) => request<Anniversary>({ url: '/anniversaries', method: 'POST', data }),
  update: (id: number, data: Partial<Anniversary>) =>
    request<Anniversary>({ url: `/anniversaries/${id}`, method: 'PUT', data }),
  remove: (id: number) => request<null>({ url: `/anniversaries/${id}`, method: 'DELETE' }),
}

// ---------- 提醒中心 ----------
export const notificationApi = {
  list: (params: { page?: number; page_size?: number }) =>
    request<PageData<Notification>>({ url: '/notifications', method: 'GET', params }),
  markRead: (id: number) => request<null>({ url: `/notifications/${id}/read`, method: 'PATCH' }),
}

// ---------- 搜索 ----------
export const searchApi = {
  search: (q: string, type?: string) =>
    request<SearchItem[]>({ url: '/search', method: 'GET', params: { q, type } }),
}
