package realtime

import (
	"sync"
	"time"

	"github.com/LeeJc02/WeHi/backend/pkg/contracts"

	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var wsConnections = promauto.NewGauge(prometheus.GaugeOpts{
	Name: "chat_ws_connections",
	Help: "Current number of live websocket connections.",
})

type Hub struct {
	mu    sync.RWMutex
	conns map[uint64]map[*websocket.Conn]struct{}
}

func NewHub() *Hub {
	return &Hub{conns: map[uint64]map[*websocket.Conn]struct{}{}}
}

func (h *Hub) Add(userID uint64, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.conns[userID] == nil {
		h.conns[userID] = map[*websocket.Conn]struct{}{}
	}
	h.conns[userID][conn] = struct{}{}
	wsConnections.Inc()
}

func (h *Hub) Remove(userID uint64, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.conns[userID] == nil {
		return
	}
	delete(h.conns[userID], conn)
	wsConnections.Dec()
	if len(h.conns[userID]) == 0 {
		delete(h.conns, userID)
	}
}

func (h *Hub) Broadcast(userIDs []uint64, event contracts.EventEnvelope) {
	h.mu.RLock()
	stale := make([]struct {
		userID uint64
		conn   *websocket.Conn
	}, 0)
	for _, userID := range userIDs {
		for conn := range h.conns[userID] {
			// Broadcast iterates the in-memory fan-out table only; it never blocks
			// on repository lookups, which keeps realtime delivery isolated from DB
			// latency and lets dead sockets be collected opportunistically.
			_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteJSON(event); err != nil {
				stale = append(stale, struct {
					userID uint64
					conn   *websocket.Conn
				}{userID: userID, conn: conn})
			}
		}
	}
	h.mu.RUnlock()
	for _, item := range stale {
		h.Remove(item.userID, item.conn)
		_ = item.conn.Close()
	}
}
