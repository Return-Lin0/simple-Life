// Package notify 实现 SSE 连接中枢：按用户维度维护事件通道，广播提醒事件。
// 注意：SSE 连接由 API 进程持有；worker 进程通过 Redis Pub/Sub 把事件
// 转发给 API 进程后再由 Hub 广播（跨进程解耦）。
package notify

import (
	"sync"
)

// Hub 管理所有在线用户的 SSE 事件通道。
type Hub struct {
	mu    sync.RWMutex
	conns map[uint64]map[chan []byte]struct{}
}

// NewHub 创建 Hub。
func NewHub() *Hub {
	return &Hub{conns: make(map[uint64]map[chan []byte]struct{})}
}

// Register 为用户注册新的事件通道，返回只读事件流。
func (h *Hub) Register(userID uint64) chan []byte {
	ch := make(chan []byte, 32) // 有缓冲：短暂积压不阻塞 worker
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.conns[userID] == nil {
		h.conns[userID] = make(map[chan []byte]struct{})
	}
	h.conns[userID][ch] = struct{}{}
	return ch
}

// Unregister 移除用户的事件通道（连接断开时调用）。
func (h *Hub) Unregister(userID uint64, ch chan []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if conns, ok := h.conns[userID]; ok {
		delete(conns, ch)
		close(ch)
		if len(conns) == 0 {
			delete(h.conns, userID)
		}
	}
}

// Broadcast 向用户全部在线连接推送事件（非阻塞，通道满时丢弃防止卡死）。
func (h *Hub) Broadcast(userID uint64, data []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for ch := range h.conns[userID] {
		select {
		case ch <- data:
		default:
			// 通道已满：跳过该连接，提醒中心仍可查询到记录
		}
	}
}

// OnlineCount 返回当前在线连接数（监控用）。
func (h *Hub) OnlineCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	total := 0
	for _, conns := range h.conns {
		total += len(conns)
	}
	return total
}
