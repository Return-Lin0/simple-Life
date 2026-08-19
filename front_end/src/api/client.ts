// Axios 封装：Bearer Token 注入 + 401 并发刷新队列
import axios, { AxiosError, AxiosRequestConfig } from 'axios'
import { getAccessToken, setAccessToken } from '@/utils/token'

export const http = axios.create({
  baseURL: '/api/v1',
  timeout: 10_000,
  withCredentials: true, // Refresh Token 依赖 HttpOnly Cookie
})

// 统一解包 {code, message, data}
http.interceptors.response.use(
  (res) => {
    const body = res.data
    if (body && typeof body.code === 'number' && body.code !== 0) {
      return Promise.reject(new ApiError(body.code, body.message))
    }
    return res
  },
  (error: AxiosError) => Promise.reject(normalizeError(error)),
)

export class ApiError extends Error {
  code: number
  constructor(code: number, message: string) {
    super(message)
    this.code = code
  }
}

function normalizeError(error: AxiosError): Error {
  if (error instanceof ApiError) return error
  const status = error.response?.status
  if (status === 401) return new ApiError(1002, '登录已过期，请重新登录')
  if (status === 429) return new ApiError(1006, '操作过于频繁，请稍后再试')
  if (status && status >= 500) return new ApiError(9000, '服务器开小差了，请稍后再试')
  if (error.code === 'ECONNABORTED') return new ApiError(9000, '请求超时，请检查网络')
  return new ApiError(9000, '网络异常，请稍后再试')
}

// ---------- 401 静默刷新：并发请求只刷新一次 ----------
let refreshing: Promise<boolean> | null = null

export async function refreshSession(): Promise<boolean> {
  if (!refreshing) {
    refreshing = doRefresh().finally(() => {
      refreshing = null
    })
  }
  return refreshing
}

async function doRefresh(): Promise<boolean> {
  try {
    const res = await http.post<{ code: number; data: { access_token: string } }>(
      '/auth/refresh',
      null,
      // 刷新请求不注入旧的失效 Token
      { headers: { Authorization: '' } },
    )
    const token = res.data.data?.access_token
    if (token) {
      setAccessToken(token)
      return true
    }
    return false
  } catch {
    return false
  }
}

// 带自动刷新重试的请求
export async function request<T>(config: AxiosRequestConfig): Promise<T> {
  const token = getAccessToken()
  const headers = { ...(config.headers || {}), ...(token ? { Authorization: `Bearer ${token}` } : {}) }
  try {
    const res = await http.request<{ code: number; message: string; data?: T }>({ ...config, headers })
    return res.data.data as T
  } catch (err) {
    const axErr = err as AxiosError
    if (axErr.response?.status === 401) {
      const ok = await refreshSession()
      if (ok) {
        const newToken = getAccessToken()
        const retry = await http.request<{ code: number; message: string; data?: T }>({
          ...config,
          headers: { ...headers, ...(newToken ? { Authorization: `Bearer ${newToken}` } : {}) },
        })
        return retry.data.data as T
      }
      // 刷新失败：清理会话并跳转登录页
      setAccessToken(null)
      if (window.location.pathname !== '/login') {
        window.location.assign('/login')
      }
    }
    throw err
  }
}
