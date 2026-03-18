package controllers

import (
	"context"
	"net/http"
	"time"

	"awesomeproject/internal/app/auth"
	"awesomeproject/internal/app/presence"
	httpx "awesomeproject/internal/platform/httpx"
	"awesomeproject/internal/realtime"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type RealtimeController struct {
	authService *auth.Service
	presence    *presence.Service
	hub         *realtime.Hub
}

const (
	wsWriteWait = 10 * time.Second
	wsPongWait  = 60 * time.Second
	wsPingEvery = (wsPongWait * 9) / 10
)

func NewRealtimeController(authService *auth.Service, presenceService *presence.Service, hub *realtime.Hub) *RealtimeController {
	return &RealtimeController{authService: authService, presence: presenceService, hub: hub}
}

func (ctl *RealtimeController) ServeWS(c *gin.Context) {
	token := c.Query("token")
	claims, err := ctl.authService.ParseAccessToken(token)
	if err != nil {
		httpx.Fail(c, http.StatusUnauthorized, err.Error())
		return
	}
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
	_ = ctl.presence.MarkOnline(c.Request.Context(), claims.UserID)

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
