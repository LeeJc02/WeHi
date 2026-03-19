package chat

import (
	"context"
	"errors"
	"testing"
	"time"

	"awesomeproject/internal/app/repository"
	"awesomeproject/internal/platform/apperr"
	"awesomeproject/internal/platform/search"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestConversationServiceAddConversationMembersRejectsDirectConversation(t *testing.T) {
	repo := newConversationServiceTestRepo(t, "conversation_service_reject_direct")
	now := time.Now().UTC()
	users := []repository.User{
		{ID: 1, Username: "owner", DisplayName: "Owner", PasswordHash: "hash", CreatedAt: now, UpdatedAt: now},
		{ID: 2, Username: "peer", DisplayName: "Peer", PasswordHash: "hash", CreatedAt: now, UpdatedAt: now},
		{ID: 3, Username: "extra", DisplayName: "Extra", PasswordHash: "hash", CreatedAt: now, UpdatedAt: now},
	}
	if err := repo.DB().Create(&users).Error; err != nil {
		t.Fatalf("seed users: %v", err)
	}

	conversation, _, err := repo.EnsureDirectConversation(1, 2)
	if err != nil {
		t.Fatalf("ensure direct conversation: %v", err)
	}

	service := newConversationServiceForTest(repo)
	err = service.AddConversationMembers(1, conversation.ID, []uint64{3})
	appErr := apperr.From(err)
	if appErr == nil || appErr.Code != "GROUP_ACTION_UNSUPPORTED" {
		t.Fatalf("expected GROUP_ACTION_UNSUPPORTED, got %#v", appErr)
	}
}

func TestConversationServiceLeaveConversationDissolvesLastGroupMember(t *testing.T) {
	repo := newConversationServiceTestRepo(t, "conversation_service_dissolve_last_member")
	now := time.Now().UTC()
	user := repository.User{
		ID:           1,
		Username:     "solo-owner",
		DisplayName:  "Solo Owner",
		PasswordHash: "hash",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := repo.DB().Create(&user).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}

	conversation := &repository.Conversation{
		Type:      "group",
		Name:      "Solo Group",
		CreatorID: user.ID,
		CreatedAt: now,
		UpdatedAt: now,
	}
	members := []repository.ConversationMember{
		{UserID: user.ID, Role: "owner", JoinedAt: now, UpdatedAt: now},
	}
	if err := repo.CreateConversation(conversation, members); err != nil {
		t.Fatalf("create conversation: %v", err)
	}
	if err := repo.AppendSyncEvents([]uint64{user.ID}, "conversation.updated", conversationAggregateID(conversation.ID), map[string]any{
		"conversation_id": conversation.ID,
	}); err != nil {
		t.Fatalf("seed sync event: %v", err)
	}

	service := newConversationServiceForTest(repo)
	if err := service.LeaveConversation(user.ID, conversation.ID); err != nil {
		t.Fatalf("leave conversation: %v", err)
	}

	_, err := repo.FindConversationByID(conversation.ID)
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected conversation to be deleted, got %v", err)
	}

	memberRows, err := repo.ListConversationMembers(conversation.ID)
	if err != nil {
		t.Fatalf("list conversation members: %v", err)
	}
	if len(memberRows) != 0 {
		t.Fatalf("expected no active members, got %d", len(memberRows))
	}

	syncRows, err := repo.ListSyncEventsByAggregate(conversationAggregateID(conversation.ID), 10)
	if err != nil {
		t.Fatalf("list sync events: %v", err)
	}
	if len(syncRows) != 1 {
		t.Fatalf("expected exactly one removal sync event, got %d", len(syncRows))
	}
	if syncRows[0].EventType != "conversation.removed" {
		t.Fatalf("expected final sync event to be conversation.removed, got %q", syncRows[0].EventType)
	}
}

func newConversationServiceForTest(repo *repository.Repository) *ConversationService {
	services := NewServices(repo, nil, search.New("mock://mysql"), nil, nil, "messages", "conversations")
	return services.Conversation
}

func newConversationServiceTestRepo(t *testing.T, dbName string) *repository.Repository {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+dbName+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&repository.User{},
		&repository.Conversation{},
		&repository.ConversationMember{},
		&repository.ConversationPin{},
		&repository.ConversationSetting{},
		&repository.Message{},
		&repository.SyncEvent{},
		&repository.AIAuditLog{},
		&repository.AIRetryJob{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return repository.New(db.WithContext(context.Background()))
}
