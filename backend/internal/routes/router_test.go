package routes

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"awesomeproject/backend/internal/config"
	"awesomeproject/backend/internal/database"
)

func TestAuthAndConversationFlow(t *testing.T) {
	db, err := database.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	router := NewRouter(db, config.Config{FrontendOrigin: "http://127.0.0.1:5173"})

	register := func(username string) {
		body := bytes.NewBufferString(`{"username":"` + username + `","display_name":"` + username + `","password":"pass123"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", body)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("register %s failed: %s", username, rec.Body.String())
		}
	}

	login := func(username string) string {
		body := bytes.NewBufferString(`{"username":"` + username + `","password":"pass123"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", body)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("login %s failed: %s", username, rec.Body.String())
		}
		var payload struct {
			Data struct {
				Token string `json:"token"`
			} `json:"data"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decode login: %v", err)
		}
		return payload.Data.Token
	}

	register("alice")
	register("bob")
	aliceToken := login("alice")

	friendReq := httptest.NewRequest(http.MethodPost, "/api/v1/friends", bytes.NewBufferString(`{"friend_id":2}`))
	friendReq.Header.Set("Authorization", "Bearer "+aliceToken)
	friendReq.Header.Set("Content-Type", "application/json")
	friendRec := httptest.NewRecorder()
	router.ServeHTTP(friendRec, friendReq)
	if friendRec.Code != http.StatusOK {
		t.Fatalf("add friend failed: %s", friendRec.Body.String())
	}

	directReq := httptest.NewRequest(http.MethodPost, "/api/v1/conversations/direct", bytes.NewBufferString(`{"target_user_id":2}`))
	directReq.Header.Set("Authorization", "Bearer "+aliceToken)
	directReq.Header.Set("Content-Type", "application/json")
	directRec := httptest.NewRecorder()
	router.ServeHTTP(directRec, directReq)
	if directRec.Code != http.StatusOK {
		t.Fatalf("create direct failed: %s", directRec.Body.String())
	}
}
