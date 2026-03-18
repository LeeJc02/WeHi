package controllers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"awesomeproject/internal/app/auth"
	"awesomeproject/internal/app/presence"
	"awesomeproject/internal/app/repository"
	"awesomeproject/internal/platform/apperr"
	"awesomeproject/pkg/contracts"

	"github.com/gin-gonic/gin"
)

func makeAuthorizedContext(method, target string, body []byte) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(method, target, bytes.NewReader(body))
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = request
	ctx.Set(auth.CurrentUserKey, &repository.User{ID: 42, Username: "alice", DisplayName: "Alice"})
	ctx.Set("session_id", "session-1")
	return ctx, recorder
}

type friendServiceStub struct {
	createFn func(userID, addresseeID uint64, message string) (*contracts.FriendRequestDTO, error)
}

func (s friendServiceStub) ListFriends(userID uint64) ([]contracts.FriendDTO, error) {
	return nil, nil
}

func (s friendServiceStub) ListFriendRequests(userID uint64) ([]contracts.FriendRequestDTO, error) {
	return nil, nil
}

func (s friendServiceStub) CreateFriendRequest(userID, addresseeID uint64, message string) (*contracts.FriendRequestDTO, error) {
	return s.createFn(userID, addresseeID, message)
}

func (s friendServiceStub) ApproveFriendRequest(userID, requestID uint64) error { return nil }
func (s friendServiceStub) RejectFriendRequest(userID, requestID uint64) error  { return nil }

func TestFriendControllerCreateFriendRequestReturnsErrorCode(t *testing.T) {
	controller := NewFriendController(friendServiceStub{
		createFn: func(userID, addresseeID uint64, message string) (*contracts.FriendRequestDTO, error) {
			return nil, apperr.Conflict("FRIENDSHIP_ALREADY_EXISTS", "friendship already exists")
		},
	})
	body, _ := json.Marshal(contracts.CreateFriendRequestRequest{AddresseeID: 7, Message: "hi"})
	ctx, recorder := makeAuthorizedContext(http.MethodPost, "/friend-requests", body)

	controller.CreateFriendRequest(ctx)

	if recorder.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", recorder.Code)
	}
	var envelope contracts.Envelope
	if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if envelope.ErrorCode != "FRIENDSHIP_ALREADY_EXISTS" {
		t.Fatalf("expected conflict error code, got %s", envelope.ErrorCode)
	}
}

type messageServiceStub struct {
	listFn func(userID, conversationID uint64, cursor string, limit int) ([]contracts.MessageDTO, error)
}

func (s messageServiceStub) ListMessages(userID, conversationID uint64, cursor string, limit int) ([]contracts.MessageDTO, error) {
	return s.listFn(userID, conversationID, cursor, limit)
}

func (s messageServiceStub) SendMessage(userID, conversationID uint64, messageType, content, clientMsgID string) (*contracts.MessageDTO, error) {
	return nil, nil
}

func (s messageServiceStub) MarkRead(userID, conversationID uint64, seq uint64) error { return nil }

func TestMessageControllerListMessagesUsesDefaultLimit(t *testing.T) {
	controller := NewMessageController(messageServiceStub{
		listFn: func(userID, conversationID uint64, cursor string, limit int) ([]contracts.MessageDTO, error) {
			if limit != 20 {
				t.Fatalf("expected default limit 20, got %d", limit)
			}
			if conversationID != 5 {
				t.Fatalf("expected conversation id 5, got %d", conversationID)
			}
			return []contracts.MessageDTO{}, nil
		},
	})
	ctx, recorder := makeAuthorizedContext(http.MethodGet, "/conversations/5/messages", nil)
	ctx.Params = gin.Params{{Key: "id", Value: "5"}}

	controller.ListMessages(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
}

type searchServiceStub struct {
	searchFn func(ctx context.Context, userID uint64, query, scope, cursor string, limit int, conversationID uint64) (*contracts.SearchResponse, error)
}

func (s searchServiceStub) Search(ctx context.Context, userID uint64, query, scope, cursor string, limit int, conversationID uint64) (*contracts.SearchResponse, error) {
	return s.searchFn(ctx, userID, query, scope, cursor, limit, conversationID)
}

func TestSearchControllerUsesQueryDTO(t *testing.T) {
	controller := NewSearchController(searchServiceStub{
		searchFn: func(ctx context.Context, userID uint64, query, scope, cursor string, limit int, conversationID uint64) (*contracts.SearchResponse, error) {
			if query != "hello" || scope != "messages" || cursor != "10" || limit != 5 || conversationID != 9 {
				t.Fatalf("unexpected query binding: %q %q %q %d %d", query, scope, cursor, limit, conversationID)
			}
			return &contracts.SearchResponse{}, nil
		},
	})
	ctx, recorder := makeAuthorizedContext(http.MethodGet, "/search?q=hello&scope=messages&cursor=10&limit=5&conversation_id=9", nil)

	controller.Search(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
}

func TestConversationControllerHandlesNilPresence(t *testing.T) {
	controller := NewConversationController(conversationServiceStub{
		listMembersFn: func(userID, conversationID uint64, onlineUsers map[uint64]bool) ([]contracts.ConversationMemberDTO, error) {
			if len(onlineUsers) != 0 {
				t.Fatalf("expected empty online map, got %v", onlineUsers)
			}
			return []contracts.ConversationMemberDTO{}, nil
		},
	}, (*presence.Service)(nil))
	ctx, recorder := makeAuthorizedContext(http.MethodGet, "/conversations/9/members", nil)
	ctx.Params = gin.Params{{Key: "id", Value: "9"}}

	controller.ListConversationMembers(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
}

type conversationServiceStub struct {
	listMembersFn func(userID, conversationID uint64, onlineUsers map[uint64]bool) ([]contracts.ConversationMemberDTO, error)
}

func (s conversationServiceStub) ListConversations(userID uint64) ([]contracts.ConversationDTO, error) {
	return nil, nil
}
func (s conversationServiceStub) EnsureDirectConversation(userID, targetID uint64) (*contracts.ConversationDTO, error) {
	return nil, nil
}
func (s conversationServiceStub) CreateGroupConversation(creatorID uint64, name string, memberIDs []uint64) (*contracts.ConversationDTO, error) {
	return nil, nil
}
func (s conversationServiceStub) RenameConversation(userID, conversationID uint64, name string) (*contracts.ConversationDTO, error) {
	return nil, nil
}
func (s conversationServiceStub) ListConversationMembers(userID, conversationID uint64, onlineUsers map[uint64]bool) ([]contracts.ConversationMemberDTO, error) {
	return s.listMembersFn(userID, conversationID, onlineUsers)
}
func (s conversationServiceStub) AddConversationMembers(actorID, conversationID uint64, memberIDs []uint64) error {
	return nil
}
func (s conversationServiceStub) RemoveConversationMember(actorID, conversationID, targetID uint64) error {
	return nil
}
func (s conversationServiceStub) LeaveConversation(userID, conversationID uint64) error { return nil }
func (s conversationServiceStub) TransferOwnership(actorID, conversationID, targetID uint64) error {
	return nil
}
func (s conversationServiceStub) SetPinned(userID, conversationID uint64, pinned bool) error {
	return nil
}
