// SSE 客户端：后端 EventSource 无法携带 Authorization 头，
// 故使用 fetch + ReadableStream 解析事件流（技术设计文档 8.7 节）。
import { getAccessToken } from './token'

export interface SseHandlers {
  onReminder: (data: Record<string, unknown>) => void
  onError?: (err: unknown) => void
}

let stopFlag = false
let retryCount = 0

export function stopSse() {
  stopFlag = true
}

export function startSse(handlers: SseHandlers) {
  stopFlag = false
  retryCount = 0
  void connect(handlers)
}

async function connect(handlers: SseHandlers) {
  // 指数退避：1s → 2s → 4s … 最大 30s
  const delay = Math.min(1000 * 2 ** retryCount, 30_000)
  if (retryCount > 0) {
    await new Promise((r) => setTimeout(r, delay))
  }
  if (stopFlag) return

  try {
    const token = getAccessToken()
    const res = await fetch('/api/v1/events', {
      headers: token ? { Authorization: `Bearer ${token}` } : {},
    })
    if (!res.ok || !res.body) {
      throw new Error(`SSE 连接失败: ${res.status}`)
    }
    retryCount = 0 // 连接成功重置退避
    const reader = res.body.getReader()
    const decoder = new TextDecoder()
    let buffer = ''

    while (!stopFlag) {
      const { done, value } = await reader.read()
      if (done) break
      buffer += decoder.decode(value, { stream: true })
      // 事件以空行分隔
      const blocks = buffer.split('\n\n')
      buffer = blocks.pop() ?? ''
      for (const block of blocks) {
        parseBlock(block, handlers)
      }
    }
  } catch (err) {
    handlers.onError?.(err)
  }

  if (!stopFlag) {
    retryCount += 1
    void connect(handlers)
  }
}

function parseBlock(block: string, handlers: SseHandlers) {
  const lines = block.split('\n')
  let eventName = 'message'
  let data = ''
  for (const line of lines) {
    if (line.startsWith('event:')) eventName = line.slice(6).trim()
    else if (line.startsWith('data:')) data += line.slice(5).trim()
  }
  if (eventName === 'reminder' && data) {
    try {
      handlers.onReminder(JSON.parse(data))
    } catch {
      // 忽略无法解析的事件
    }
  }
}
