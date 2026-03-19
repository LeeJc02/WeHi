package controllers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
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
func (s friendServiceStub) UpdateRemark(userID, friendID uint64, remarkName string) error {
	return nil
}

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
	listFn   func(userID, conversationID uint64, cursor string, limit int) ([]contracts.MessageDTO, error)
	sendFn   func(userID, conversationID uint64, messageType, content, clientMsgID string, replyToMessageID *uint64, attachment *contracts.AttachmentDTO) (*contracts.MessageDTO, error)
	typingFn func(userID, conversationID uint64, isTyping bool) error
	recallFn func(userID, messageID uint64) error
}

func (s messageServiceStub) ListMessages(userID, conversationID uint64, cursor string, limit int) ([]contracts.MessageDTO, error) {
	return s.listFn(userID, conversationID, cursor, limit)
}

func (s messageServiceStub) SendMessage(userID, conversationID uint64, messageType, content, clientMsgID string, replyToMessageID *uint64, attachment *contracts.AttachmentDTO) (*contracts.MessageDTO, error) {
	if s.sendFn != nil {
		return s.sendFn(userID, conversationID, messageType, content, clientMsgID, replyToMessageID, attachment)
	}
	return &contracts.MessageDTO{}, nil
}

func (s messageServiceStub) MarkRead(userID, conversationID uint64, seq uint64) error { return nil }
func (s messageServiceStub) UpdateTyping(userID, conversationID uint64, isTyping bool) error {
	if s.typingFn != nil {
		return s.typingFn(userID, conversationID, isTyping)
	}
	return nil
}
func (s messageServiceStub) RecallMessage(userID, messageID uint64) error {
	if s.recallFn != nil {
		return s.recallFn(userID, messageID)
	}
	return nil
}

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

func TestMessageControllerSendMessageBindsReplyAndAttachment(t *testing.T) {
	controller := NewMessageController(messageServiceStub{
		sendFn: func(userID, conversationID uint64, messageType, content, clientMsgID string, replyToMessageID *uint64, attachment *contracts.AttachmentDTO) (*contracts.MessageDTO, error) {
			if userID != 42 || conversationID != 8 {
				t.Fatalf("unexpected user/conversation ids: %d %d", userID, conversationID)
			}
			if messageType != "file" || content != "report.pdf" || clientMsgID != "client-1" {
				t.Fatalf("unexpected message payload: %q %q %q", messageType, content, clientMsgID)
			}
			if replyToMessageID == nil || *replyToMessageID != 5 {
				t.Fatalf("unexpected reply target: %#v", replyToMessageID)
			}
			if attachment == nil || attachment.ObjectKey != "object-1" || attachment.Filename != "report.pdf" {
				t.Fatalf("unexpected attachment: %#v", attachment)
			}
			return &contracts.MessageDTO{ID: 10}, nil
		},
	})
	body, _ := json.Marshal(contracts.SendMessageRequest{
		MessageType:      "file",
		Content:          "report.pdf",
		ClientMsgID:      "client-1",
		ReplyToMessageID: uint64Ptr(5),
		Attachment: &contracts.AttachmentDTO{
			ObjectKey:   "object-1",
			URL:         "/uploads/object-1",
			Filename:    "report.pdf",
			ContentType: "application/pdf",
			SizeBytes:   123,
		},
	})
	ctx, recorder := makeAuthorizedContext(http.MethodPost, "/conversations/8/messages", body)
	ctx.Params = gin.Params{{Key: "id", Value: "8"}}

	controller.SendMessage(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
}

func TestMessageControllerRecallUsesMessageID(t *testing.T) {
	controller := NewMessageController(messageServiceStub{
		recallFn: func(userID, messageID uint64) error {
			if userID != 42 || messageID != 12 {
				t.Fatalf("unexpected recall payload: %d %d", userID, messageID)
			}
			return nil
		},
	})
	ctx, recorder := makeAuthorizedContext(http.MethodPost, "/messages/12/recall", nil)
	ctx.Params = gin.Params{{Key: "id", Value: "12"}}

	controller.Recall(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
}

func TestMessageControllerUpdateTypingBindsRequest(t *testing.T) {
	controller := NewMessageController(messageServiceStub{
		typingFn: func(userID, conversationID uint64, isTyping bool) error {
			if userID != 42 || conversationID != 3 || !isTyping {
				t.Fatalf("unexpected typing payload: %d %d %v", userID, conversationID, isTyping)
			}
			return nil
		},
	})
	body, _ := json.Marshal(contracts.TypingStatusRequest{IsTyping: true})
	ctx, recorder := makeAuthorizedContext(http.MethodPost, "/conversations/3/typing", body)
	ctx.Params = gin.Params{{Key: "id", Value: "3"}}

	controller.UpdateTyping(ctx)

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

type adminDiagnosticsServiceStub struct {
	resolveFn func(clientMsgID string, senderID, conversationID uint64) (*contracts.MessageLookupResult, error)
}

func (s adminDiagnosticsServiceStub) MessageJourney(messageID uint64) (*contracts.MessageJourney, error) {
	return &contracts.MessageJourney{}, nil
}

func (s adminDiagnosticsServiceStub) ResolveMessageByClientMsgID(clientMsgID string, senderID, conversationID uint64) (*contracts.MessageLookupResult, error) {
	return s.resolveFn(clientMsgID, senderID, conversationID)
}

func (s adminDiagnosticsServiceStub) ConversationConsistency(conversationID uint64) (*contracts.ConversationConsistency, error) {
	return &contracts.ConversationConsistency{}, nil
}

func (s adminDiagnosticsServiceStub) ConversationEvents(conversationID uint64, limit int) ([]contracts.SyncEventDTO, error) {
	return []contracts.SyncEventDTO{}, nil
}

func TestAdminDiagnosticsControllerResolveMessageBindsQuery(t *testing.T) {
	controller := NewAdminDiagnosticsController(adminDiagnosticsServiceStub{
		resolveFn: func(clientMsgID string, senderID, conversationID uint64) (*contracts.MessageLookupResult, error) {
			if clientMsgID != "client-1" || senderID != 7 || conversationID != 9 {
				t.Fatalf("unexpected query values: %q %d %d", clientMsgID, senderID, conversationID)
			}
			return &contracts.MessageLookupResult{MessageID: 11, ConversationID: 9, SenderID: 7, ClientMsgID: clientMsgID}, nil
		},
	})
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/admin/messages/resolve?client_msg_id=client-1&sender_id=7&conversation_id=9", nil)
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = request

	controller.ResolveMessage(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
}

type adminAIServiceStub struct {
	listRetryFn func(query contracts.ListAIRetryJobsQuery) ([]contracts.AIRetryJobDTO, error)
	getRetryFn  func(id uint64) (*contracts.AIRetryJobDetailDTO, error)
	retryNowFn  func(id uint64) error
	retryJobsFn func(ids []uint64) error
	cleanupFn   func(statuses []string) error
}

func (s adminAIServiceStub) Load() (*contracts.AIConfig, error) {
	return &contracts.AIConfig{}, nil
}

func (s adminAIServiceStub) Save(cfg contracts.AIConfig) error {
	return nil
}

func (s adminAIServiceStub) ListRetryJobs(query contracts.ListAIRetryJobsQuery) ([]contracts.AIRetryJobDTO, error) {
	if s.listRetryFn != nil {
		return s.listRetryFn(query)
	}
	return []contracts.AIRetryJobDTO{}, nil
}

func (s adminAIServiceStub) GetRetryJob(id uint64) (*contracts.AIRetryJobDetailDTO, error) {
	if s.getRetryFn != nil {
		return s.getRetryFn(id)
	}
	return &contracts.AIRetryJobDetailDTO{}, nil
}

func (s adminAIServiceStub) RetryJobNow(id uint64) error {
	if s.retryNowFn != nil {
		return s.retryNowFn(id)
	}
	return nil
}

func (s adminAIServiceStub) RetryJobs(ids []uint64) error {
	if s.retryJobsFn != nil {
		return s.retryJobsFn(ids)
	}
	return nil
}

func (s adminAIServiceStub) CleanupRetryJobs(statuses []string) error {
	if s.cleanupFn != nil {
		return s.cleanupFn(statuses)
	}
	return nil
}

func TestAdminAIControllerListRetryJobsBindsQuery(t *testing.T) {
	stub := adminAIServiceStub{
		listRetryFn: func(query contracts.ListAIRetryJobsQuery) ([]contracts.AIRetryJobDTO, error) {
			if query.Limit != 12 || query.Status != "exhausted" {
				t.Fatalf("unexpected query: %#v", query)
			}
			return []contracts.AIRetryJobDTO{{ID: 3}}, nil
		},
	}
	controller := NewAdminAIController(stub, stub)
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/admin/ai/retry-jobs?limit=12&status=exhausted", nil)
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = request

	controller.ListRetryJobs(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
}

func TestAdminAIControllerRetryJobNowUsesPathID(t *testing.T) {
	stub := adminAIServiceStub{
		retryNowFn: func(id uint64) error {
			if id != 19 {
				t.Fatalf("unexpected job id: %d", id)
			}
			return nil
		},
	}
	controller := NewAdminAIController(stub, stub)
	ctx, recorder := makeAuthorizedContext(http.MethodPost, "/api/v1/admin/ai/retry-jobs/19/retry-now", nil)
	ctx.Params = gin.Params{{Key: "id", Value: "19"}}

	controller.RetryJobNow(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
}

func TestAdminAIControllerRetryJobDetailUsesPathID(t *testing.T) {
	stub := adminAIServiceStub{
		getRetryFn: func(id uint64) (*contracts.AIRetryJobDetailDTO, error) {
			if id != 23 {
				t.Fatalf("unexpected job id: %d", id)
			}
			return &contracts.AIRetryJobDetailDTO{
				AIRetryJobDTO: contracts.AIRetryJobDTO{ID: id},
			}, nil
		},
	}
	controller := NewAdminAIController(stub, stub)
	ctx, recorder := makeAuthorizedContext(http.MethodGet, "/api/v1/admin/ai/retry-jobs/23", nil)
	ctx.Params = gin.Params{{Key: "id", Value: "23"}}

	controller.RetryJobDetail(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
}

func TestAdminAIControllerRetryJobsBindsBody(t *testing.T) {
	stub := adminAIServiceStub{
		retryJobsFn: func(ids []uint64) error {
			if len(ids) != 2 || ids[0] != 5 || ids[1] != 8 {
				t.Fatalf("unexpected ids: %#v", ids)
			}
			return nil
		},
	}
	controller := NewAdminAIController(stub, stub)
	body, _ := json.Marshal(contracts.RetryAIRetryJobsRequest{IDs: []uint64{5, 8}})
	ctx, recorder := makeAuthorizedContext(http.MethodPost, "/api/v1/admin/ai/retry-jobs/retry-batch", body)

	controller.RetryJobs(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
}

func TestAdminAIControllerCleanupRetryJobsBindsBody(t *testing.T) {
	stub := adminAIServiceStub{
		cleanupFn: func(statuses []string) error {
			if len(statuses) != 2 || statuses[0] != "completed" || statuses[1] != "exhausted" {
				t.Fatalf("unexpected statuses: %#v", statuses)
			}
			return nil
		},
	}
	controller := NewAdminAIController(stub, stub)
	body, _ := json.Marshal(contracts.CleanupAIRetryJobsRequest{Statuses: []string{"completed", "exhausted"}})
	ctx, recorder := makeAuthorizedContext(http.MethodPost, "/api/v1/admin/ai/retry-jobs/cleanup", body)

	controller.CleanupRetryJobs(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
}

type conversationServiceStub struct {
	listMembersFn func(userID, conversationID uint64, onlineUsers map[uint64]bool) ([]contracts.ConversationMemberDTO, error)
	updateFn      func(userID, conversationID uint64, req contracts.UpdateConversationSettingsRequest) (*contracts.ConversationDTO, error)
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
func (s conversationServiceStub) UpdateSettings(userID, conversationID uint64, req contracts.UpdateConversationSettingsRequest) (*contracts.ConversationDTO, error) {
	if s.updateFn != nil {
		return s.updateFn(userID, conversationID, req)
	}
	return &contracts.ConversationDTO{}, nil
}

func TestConversationControllerUpdateConversationSettingsBindsRequest(t *testing.T) {
	controller := NewConversationController(conversationServiceStub{
		updateFn: func(userID, conversationID uint64, req contracts.UpdateConversationSettingsRequest) (*contracts.ConversationDTO, error) {
			if userID != 42 || conversationID != 7 {
				t.Fatalf("unexpected user/conversation ids: %d %d", userID, conversationID)
			}
			if req.Pinned == nil || !*req.Pinned {
				t.Fatal("expected pinned=true")
			}
			if req.IsMuted == nil || !*req.IsMuted {
				t.Fatal("expected is_muted=true")
			}
			if req.Draft == nil || *req.Draft != "draft" {
				t.Fatalf("unexpected draft: %#v", req.Draft)
			}
			return &contracts.ConversationDTO{ID: conversationID}, nil
		},
	}, (*presence.Service)(nil))

	body, _ := json.Marshal(contracts.UpdateConversationSettingsRequest{
		Pinned:  boolPtr(true),
		IsMuted: boolPtr(true),
		Draft:   stringPtr("draft"),
	})
	ctx, recorder := makeAuthorizedContext(http.MethodPatch, "/conversations/7/settings", body)
	ctx.Params = gin.Params{{Key: "id", Value: "7"}}

	controller.UpdateConversationSettings(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
}

type uploadServiceStub struct {
	presignFn  func(req contracts.UploadPresignRequest) (*contracts.UploadPresignResponse, error)
	putFn      func(objectKey string, reader io.Reader) error
	completeFn func(req contracts.UploadCompleteRequest) (*contracts.UploadCompleteResponse, error)
	openFn     func(objectKey string) (*os.File, string, error)
}

func (s uploadServiceStub) Presign(req contracts.UploadPresignRequest) (*contracts.UploadPresignResponse, error) {
	return s.presignFn(req)
}

func (s uploadServiceStub) PutObject(objectKey string, reader io.Reader) error {
	if s.putFn != nil {
		return s.putFn(objectKey, reader)
	}
	return nil
}

func (s uploadServiceStub) Complete(req contracts.UploadCompleteRequest) (*contracts.UploadCompleteResponse, error) {
	if s.completeFn != nil {
		return s.completeFn(req)
	}
	return &contracts.UploadCompleteResponse{}, nil
}

func (s uploadServiceStub) Open(objectKey string) (*os.File, string, error) {
	if s.openFn != nil {
		return s.openFn(objectKey)
	}
	return nil, "", nil
}

func TestUploadControllerPresignBindsRequest(t *testing.T) {
	controller := NewUploadController(uploadServiceStub{
		presignFn: func(req contracts.UploadPresignRequest) (*contracts.UploadPresignResponse, error) {
			if req.Filename != "avatar.png" || req.ContentType != "image/png" || req.SizeBytes != 256 {
				t.Fatalf("unexpected presign request: %#v", req)
			}
			return &contracts.UploadPresignResponse{ObjectKey: "object-1"}, nil
		},
	})
	body, _ := json.Marshal(contracts.UploadPresignRequest{
		Filename:    "avatar.png",
		ContentType: "image/png",
		SizeBytes:   256,
	})
	ctx, recorder := makeAuthorizedContext(http.MethodPost, "/uploads/presign", body)

	controller.Presign(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
}

func boolPtr(value bool) *bool {
	return &value
}

func stringPtr(value string) *string {
	return &value
}

func uint64Ptr(value uint64) *uint64 {
	return &value
}
