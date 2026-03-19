package repository

import (
	"database/sql"
	"strconv"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestCreateConversation_AllowsMultipleGroupConversationsWithoutDirectKey(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:test_create_group_without_direct_key?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&User{}, &Conversation{}, &ConversationMember{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	now := time.Now().UTC()
	users := []User{
		{ID: 1, Username: "creator", DisplayName: "Creator", PasswordHash: "hash", CreatedAt: now, UpdatedAt: now},
		{ID: 2, Username: "alice", DisplayName: "Alice", PasswordHash: "hash", CreatedAt: now, UpdatedAt: now},
		{ID: 3, Username: "bob", DisplayName: "Bob", PasswordHash: "hash", CreatedAt: now, UpdatedAt: now},
	}
	if err := db.Create(&users).Error; err != nil {
		t.Fatalf("seed users: %v", err)
	}

	repo := New(db)
	groupA := &Conversation{
		Type:      "group",
		Name:      "Group A",
		CreatorID: 1,
		CreatedAt: now,
		UpdatedAt: now,
	}
	groupB := &Conversation{
		Type:      "group",
		Name:      "Group B",
		CreatorID: 1,
		CreatedAt: now,
		UpdatedAt: now,
	}

	membersA := []ConversationMember{
		{UserID: 1, Role: "owner", JoinedAt: now, UpdatedAt: now},
		{UserID: 2, Role: "member", JoinedAt: now, UpdatedAt: now},
	}
	membersB := []ConversationMember{
		{UserID: 1, Role: "owner", JoinedAt: now, UpdatedAt: now},
		{UserID: 3, Role: "member", JoinedAt: now, UpdatedAt: now},
	}

	if err := repo.CreateConversation(groupA, membersA); err != nil {
		t.Fatalf("create first group: %v", err)
	}
	if err := repo.CreateConversation(groupB, membersB); err != nil {
		t.Fatalf("create second group: %v", err)
	}

	var directKeyA sql.NullString
	if err := db.Raw("SELECT direct_key FROM conversations WHERE id = ?", groupA.ID).Scan(&directKeyA).Error; err != nil {
		t.Fatalf("query first direct_key: %v", err)
	}
	if directKeyA.Valid {
		t.Fatalf("expected first group direct_key to be NULL, got %q", directKeyA.String)
	}

	var directKeyB sql.NullString
	if err := db.Raw("SELECT direct_key FROM conversations WHERE id = ?", groupB.ID).Scan(&directKeyB).Error; err != nil {
		t.Fatalf("query second direct_key: %v", err)
	}
	if directKeyB.Valid {
		t.Fatalf("expected second group direct_key to be NULL, got %q", directKeyB.String)
	}
}

func TestAddConversationMembers_RestoresSoftDeletedMember(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:test_restore_removed_group_member?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&User{}, &Conversation{}, &ConversationMember{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	now := time.Now().UTC()
	users := []User{
		{ID: 1, Username: "creator-2", DisplayName: "Creator", PasswordHash: "hash", CreatedAt: now, UpdatedAt: now},
		{ID: 2, Username: "alice-2", DisplayName: "Alice", PasswordHash: "hash", CreatedAt: now, UpdatedAt: now},
		{ID: 3, Username: "bob-2", DisplayName: "Bob", PasswordHash: "hash", CreatedAt: now, UpdatedAt: now},
	}
	if err := db.Create(&users).Error; err != nil {
		t.Fatalf("seed users: %v", err)
	}

	repo := New(db)
	group := &Conversation{
		Type:      "group",
		Name:      "Restore Group",
		CreatorID: 1,
		CreatedAt: now,
		UpdatedAt: now,
	}
	members := []ConversationMember{
		{UserID: 1, Role: "owner", JoinedAt: now, UpdatedAt: now},
		{UserID: 2, Role: "member", JoinedAt: now, UpdatedAt: now},
		{UserID: 3, Role: "member", JoinedAt: now, UpdatedAt: now},
	}
	if err := repo.CreateConversation(group, members); err != nil {
		t.Fatalf("create group: %v", err)
	}
	if err := repo.RemoveConversationMember(group.ID, 3); err != nil {
		t.Fatalf("remove member: %v", err)
	}
	if err := repo.AddConversationMembers(group.ID, []uint64{3}); err != nil {
		t.Fatalf("re-add member: %v", err)
	}

	rows, err := repo.ListConversationMembers(group.ID)
	if err != nil {
		t.Fatalf("list members: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("expected 3 active members after restore, got %d", len(rows))
	}

	var count int64
	if err := db.Model(&ConversationMember{}).Unscoped().
		Where("conversation_id = ? AND user_id = ?", group.ID, 3).
		Count(&count).Error; err != nil {
		t.Fatalf("count restored rows: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected restored member to reuse existing row, got %d rows", count)
	}
}

func TestDeleteGroupConversation_RemovesConversationData(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:test_delete_group_conversation?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&User{},
		&Conversation{},
		&ConversationMember{},
		&ConversationSetting{},
		&ConversationPin{},
		&Message{},
		&SyncEvent{},
		&AIAuditLog{},
		&AIRetryJob{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	now := time.Now().UTC()
	users := []User{
		{ID: 1, Username: "creator-3", DisplayName: "Creator", PasswordHash: "hash", CreatedAt: now, UpdatedAt: now},
		{ID: 2, Username: "alice-3", DisplayName: "Alice", PasswordHash: "hash", CreatedAt: now, UpdatedAt: now},
	}
	if err := db.Create(&users).Error; err != nil {
		t.Fatalf("seed users: %v", err)
	}

	repo := New(db)
	group := &Conversation{
		Type:      "group",
		Name:      "Delete Group",
		CreatorID: 1,
		CreatedAt: now,
		UpdatedAt: now,
	}
	members := []ConversationMember{
		{UserID: 1, Role: "owner", JoinedAt: now, UpdatedAt: now},
		{UserID: 2, Role: "member", JoinedAt: now, UpdatedAt: now},
	}
	if err := repo.CreateConversation(group, members); err != nil {
		t.Fatalf("create group: %v", err)
	}

	if err := db.Create(&ConversationSetting{
		ConversationID: group.ID,
		UserID:         1,
		Draft:          "draft",
		CreatedAt:      now,
		UpdatedAt:      now,
	}).Error; err != nil {
		t.Fatalf("seed setting: %v", err)
	}
	if err := db.Create(&ConversationPin{
		ConversationID: group.ID,
		UserID:         1,
		PinnedAt:       now,
	}).Error; err != nil {
		t.Fatalf("seed pin: %v", err)
	}
	if _, _, _, err := repo.SendMessage(group.ID, 1, "text", "hello", "delete-group-message", nil, ""); err != nil {
		t.Fatalf("seed message: %v", err)
	}
	aggregateID := "conversation:" + strconv.FormatUint(group.ID, 10)
	if err := repo.AppendSyncEvents([]uint64{1, 2}, "conversation.updated", aggregateID, map[string]any{"conversation_id": group.ID}); err != nil {
		t.Fatalf("seed sync events: %v", err)
	}
	if err := db.Create(&AIAuditLog{
		UserID:              1,
		ConversationID:      group.ID,
		Provider:            "test",
		Model:               "model",
		Status:              "failed",
		RequestPayloadJSON:  "{}",
		ResponsePayloadJSON: "{}",
		ErrorMessage:        "none",
		CreatedAt:           now,
	}).Error; err != nil {
		t.Fatalf("seed audit log: %v", err)
	}
	if err := db.Create(&AIRetryJob{
		UserID:         1,
		ConversationID: group.ID,
		Status:         "pending",
		NextAttemptAt:  now,
		LastError:      "",
		CreatedAt:      now,
		UpdatedAt:      now,
	}).Error; err != nil {
		t.Fatalf("seed retry job: %v", err)
	}

	if err := repo.DeleteGroupConversation(group.ID, aggregateID); err != nil {
		t.Fatalf("delete group conversation: %v", err)
	}

	assertConversationScopedCount(t, db, &Conversation{}, "id = ?", group.ID)
	assertConversationScopedCount(t, db, &ConversationMember{}, "conversation_id = ?", group.ID)
	assertConversationScopedCount(t, db, &ConversationSetting{}, "conversation_id = ?", group.ID)
	assertConversationScopedCount(t, db, &ConversationPin{}, "conversation_id = ?", group.ID)
	assertConversationScopedCount(t, db, &Message{}, "conversation_id = ?", group.ID)
	assertConversationScopedCount(t, db, &SyncEvent{}, "aggregate_id = ?", aggregateID)
	assertConversationScopedCount(t, db, &AIAuditLog{}, "conversation_id = ?", group.ID)
	assertConversationScopedCount(t, db, &AIRetryJob{}, "conversation_id = ?", group.ID)
}

func assertConversationScopedCount(t *testing.T, db *gorm.DB, model any, query string, args ...any) {
	t.Helper()
	var count int64
	tx := db.Model(model)
	switch model.(type) {
	case *Conversation, *ConversationMember, *ConversationSetting, *Message, *AIRetryJob:
		tx = tx.Unscoped()
	}
	if err := tx.Where(query, args...).Count(&count).Error; err != nil {
		t.Fatalf("count %T: %v", model, err)
	}
	if count != 0 {
		t.Fatalf("expected %T count to be 0, got %d", model, count)
	}
}
