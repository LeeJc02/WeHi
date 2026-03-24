package controllers

import (
	"context"
	"net/http"
	"time"

	"github.com/LeeJc02/WeHi/backend/internal/app/auth"
	"github.com/LeeJc02/WeHi/backend/internal/app/presence"
	httpx "github.com/LeeJc02/WeHi/backend/internal/platform/httpx"
	"github.com/LeeJc02/WeHi/backend/internal/platform/observability"
	"github.com/LeeJc02/WeHi/backend/internal/realtime"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.opentelemetry.io/otel/attribute"
)

type RealtimeController struct {
	authService *auth.Service
	presence    *presence.Service
	hub         *realtime.Hub
}

const (
	// Ping cadence stays comfortably inside the pong deadline so the server
	// detects dead connections without churning healthy mobile clients.
	wsWriteWait = 10 * time.Second
	wsPongWait  = 60 * time.Second
	wsPingEvery = (wsPongWait * 9) / 10
)

func NewRealtimeController(authService *auth.Service, presenceService *presence.Service, hub *realtime.Hub) *RealtimeController {
	return &RealtimeController{authService: authService, presence: presenceService, hub: hub}
}

// ServeWS authenticates once, registers the socket in the per-user hub, and
// then relies on ping/pong plus read deadlines to clean up broken sessions.
func (ctl *RealtimeController) ServeWS(c *gin.Context) {
	ctx, span := observability.Tracer("realtime.ws").Start(c.Request.Context(), "websocket.connect")
	defer span.End()
	token := c.Query("token")
	claims, err := ctl.authService.ParseAccessToken(token)
	if err != nil {
		httpx.Fail(c, http.StatusUnauthorized, err.Error())
		return
	}
	span.SetAttributes(
		attribute.Int64("chat.user_id", int64(claims.UserID)),
		attribute.String("chat.session_id", claims.SessionID),
	)
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	_ = conn.SetReadDeadline(time.Now().Add(wsPongWait))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(wsPongWait))
	})

	ctl.hub.Add(claims.UserID, conn)
	_ = ctl.presence.MarkOnline(ctx, claims.UserID)

	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(wsPingEvery)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				deadline := time.Now().Add(wsWriteWait)
				if err := conn.WriteControl(websocket.PingMessage, []byte("ping"), deadline); err != nil {
					_ = conn.Close()
					return
				}
			case <-done:
				return
			}
		}
	}()

	defer func() {
		close(done)
		ctl.hub.Remove(claims.UserID, conn)
		_ = ctl.presence.MarkOffline(context.Background(), claims.UserID)
		_ = conn.Close()
	}()

	_ = conn.SetWriteDeadline(time.Now().Add(wsWriteWait))
	_ = conn.WriteJSON(gin.H{
		"type": "auth.ok",
		"payload": gin.H{
			"user_id":    claims.UserID,
			"session_id": claims.SessionID,
		},
	})
	for {
		_ = conn.SetReadDeadline(time.Now().Add(wsPongWait))
		if _, _, err := conn.ReadMessage(); err != nil {
			return
		}
	}
}
