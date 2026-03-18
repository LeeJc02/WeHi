package repository

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"awesomeproject/pkg/contracts"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Repository struct {
	db *gorm.DB
}

func New(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) DB() *gorm.DB {
	return r.db
}

func (r *Repository) CreateUser(user *User) error {
	return r.db.Create(user).Error
}

func (r *Repository) FindUserByUsername(username string) (*User, error) {
	var user User
	if err := r.db.Where("username = ? AND deleted_at IS NULL", username).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *Repository) FindUserByID(id uint64) (*User, error) {
	var user User
	if err := r.db.Where("id = ? AND deleted_at IS NULL", id).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *Repository) UpdateUserDisplayName(id uint64, displayName string) error {
	return r.db.Model(&User{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Updates(map[string]any{
			"display_name": displayName,
			"updated_at":   time.Now(),
		}).Error
}

func (r *Repository) ListUsers(excludeID uint64) ([]User, error) {
	var users []User
	err := r.db.Where("id <> ? AND deleted_at IS NULL", excludeID).Order("id asc").Find(&users).Error
	return users, err
}

func (r *Repository) ListFriends(userID uint64) ([]contracts.FriendDTO, error) {
	var rows []contracts.FriendDTO
	err := r.db.Table("friendships f").
		Select("u.id, u.username, u.display_name").
		Joins("JOIN users u ON u.id = f.friend_id AND u.deleted_at IS NULL").
		Where("f.user_id = ? AND f.deleted_at IS NULL", userID).
		Order("u.display_name asc, u.username asc").
		Scan(&rows).Error
	return rows, err
}

func (r *Repository) CreateFriendRequest(request *FriendRequest) error {
	return r.db.Create(request).Error
}

func (r *Repository) FindPendingFriendRequest(requesterID, addresseeID uint64) (*FriendRequest, error) {
	var request FriendRequest
	err := r.db.
		Where("requester_id = ? AND addressee_id = ? AND status = 'pending' AND deleted_at IS NULL", requesterID, addresseeID).
		First(&request).Error
	if err != nil {
		return nil, err
	}
	return &request, nil
}

func (r *Repository) FindFriendRequestByID(id uint64) (*FriendRequest, error) {
	var request FriendRequest
	if err := r.db.Where("id = ? AND deleted_at IS NULL", id).First(&request).Error; err != nil {
		return nil, err
	}
	return &request, nil
}

func (r *Repository) ListFriendRequests(userID uint64) ([]FriendRequest, error) {
	var rows []FriendRequest
	err := r.db.Where("(requester_id = ? OR addressee_id = ?) AND deleted_at IS NULL", userID, userID).
		Order("updated_at desc, id desc").
		Find(&rows).Error
	return rows, err
}

func (r *Repository) FriendshipExists(userID, friendID uint64) (bool, error) {
	var count int64
	err := r.db.Model(&Friendship{}).
		Where("user_id = ? AND friend_id = ? AND deleted_at IS NULL", userID, friendID).
		Count(&count).Error
	return count > 0, err
}

func (r *Repository) AcceptFriendRequest(request *FriendRequest) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(request).Updates(map[string]any{
			"status":     "accepted",
			"updated_at": time.Now(),
		}).Error; err != nil {
			return err
		}

		rows := []Friendship{
			{UserID: request.RequesterID, FriendID: request.AddresseeID, CreatedAt: time.Now()},
			{UserID: request.AddresseeID, FriendID: request.RequesterID, CreatedAt: time.Now()},
		}
		return tx.Create(&rows).Error
	})
}

func (r *Repository) UpdateFriendRequestStatus(request *FriendRequest, status string) error {
	return r.db.Model(request).Updates(map[string]any{
		"status":     status,
		"updated_at": time.Now(),
	}).Error
}

func (r *Repository) EnsureDirectConversation(userID, targetID uint64) (*Conversation, []ConversationMember, error) {
	directKey := formatDirectKey(userID, targetID)
	var conversation Conversation
	err := r.db.Where("direct_key = ? AND deleted_at IS NULL", directKey).First(&conversation).Error
	if err == nil {
		if err := r.restoreDirectConversationMembers(&conversation, userID, targetID); err != nil {
			return nil, nil, err
		}
		members, err := r.ListConversationMembers(conversation.ID)
		return &conversation, members, err
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil, err
	}

	now := time.Now()
	conversation = Conversation{
		Type:      "direct",
		DirectKey: directKey,
		Name:      "",
		CreatorID: userID,
		CreatedAt: now,
		UpdatedAt: now,
	}
	members := []ConversationMember{
		{UserID: userID, Role: "owner", JoinedAt: now, UpdatedAt: now},
		{UserID: targetID, Role: "member", JoinedAt: now, UpdatedAt: now},
	}
	if err := r.CreateConversation(&conversation, members); err != nil {
		return nil, nil, err
	}
	return &conversation, members, nil
}

func (r *Repository) restoreDirectConversationMembers(conversation *Conversation, userID, targetID uint64) error {
	now := time.Now()
	desired := []ConversationMember{
		{ConversationID: conversation.ID, UserID: userID, Role: "owner", JoinedAt: now, UpdatedAt: now},
		{ConversationID: conversation.ID, UserID: targetID, Role: "member", JoinedAt: now, UpdatedAt: now},
	}
	return r.db.Transaction(func(tx *gorm.DB) error {
		restored := false
		for _, member := range desired {
			var existing ConversationMember
			err := tx.Unscoped().
				Where("conversation_id = ? AND user_id = ?", conversation.ID, member.UserID).
				First(&existing).Error
			if errors.Is(err, gorm.ErrRecordNotFound) {
				if err := tx.Create(&member).Error; err != nil {
					return err
				}
				restored = true
				continue
			}
			if err != nil {
				return err
			}
			if existing.DeletedAt.Valid {
				if err := tx.Unscoped().
					Model(&ConversationMember{}).
					Where("id = ?", existing.ID).
					Updates(map[string]any{
						"deleted_at":    gorm.Expr("NULL"),
						"updated_at":    now,
						"joined_at":     existing.JoinedAt,
						"role":          member.Role,
						"last_read_seq": existing.LastReadSeq,
					}).Error; err != nil {
					return err
				}
				restored = true
			}
		}
		if restored {
			if err := tx.Model(&Conversation{}).
				Where("id = ?", conversation.ID).
				Update("updated_at", now).Error; err != nil {
				return err
			}
			conversation.UpdatedAt = now
		}
		return nil
	})
}

func (r *Repository) CreateConversation(conversation *Conversation, members []ConversationMember) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(conversation).Error; err != nil {
			return err
		}
		for idx := range members {
			members[idx].ConversationID = conversation.ID
		}
		return tx.Create(&members).Error
	})
}

func (r *Repository) FindConversationByID(id uint64) (*Conversation, error) {
	var conversation Conversation
	if err := r.db.Where("id = ? AND deleted_at IS NULL", id).First(&conversation).Error; err != nil {
		return nil, err
	}
	return &conversation, nil
}

func (r *Repository) FindConversationMember(conversationID, userID uint64) (*ConversationMember, error) {
	var member ConversationMember
	if err := r.db.Where("conversation_id = ? AND user_id = ? AND deleted_at IS NULL", conversationID, userID).First(&member).Error; err != nil {
		return nil, err
	}
	return &member, nil
}

func (r *Repository) ListConversationMembers(conversationID uint64) ([]ConversationMember, error) {
	var members []ConversationMember
	err := r.db.Where("conversation_id = ? AND deleted_at IS NULL", conversationID).
		Order("id asc").Find(&members).Error
	return members, err
}

type ConversationMemberRow struct {
	UserID      uint64
	Username    string
	DisplayName string
	Role        string
	LastReadSeq uint64
	JoinedAt    time.Time
}

func (r *Repository) ListConversationMemberProfiles(conversationID uint64) ([]ConversationMemberRow, error) {
	var rows []ConversationMemberRow
	err := r.db.Table("conversation_members cm").
		Select("cm.user_id, u.username, u.display_name, cm.role, cm.last_read_seq, cm.joined_at").
		Joins("JOIN users u ON u.id = cm.user_id AND u.deleted_at IS NULL").
		Where("cm.conversation_id = ? AND cm.deleted_at IS NULL", conversationID).
		Order("cm.id asc").
		Scan(&rows).Error
	return rows, err
}

func (r *Repository) CreateGroupConversation(creatorID uint64, name string, memberIDs []uint64) (*Conversation, error) {
	now := time.Now()
	conversation := &Conversation{
		Type:      "group",
		Name:      strings.TrimSpace(name),
		CreatorID: creatorID,
		CreatedAt: now,
		UpdatedAt: now,
	}
	members := make([]ConversationMember, 0, len(memberIDs))
	for _, memberID := range memberIDs {
		role := "member"
		if memberID == creatorID {
			role = "owner"
		}
		members = append(members, ConversationMember{
			UserID:    memberID,
			Role:      role,
			JoinedAt:  now,
			UpdatedAt: now,
		})
	}
	if err := r.CreateConversation(conversation, members); err != nil {
		return nil, err
	}
	return conversation, nil
}

type conversationListRow struct {
	ID                 uint64
	Type               string
	Name               string
	CreatorID          uint64
	MemberCount        int64
	Pinned             bool
	LastReadSeq        uint64
	UnreadCount        int64
	LastMessageSeq     uint64
	LastMessageSender  uint64
	LastMessagePreview string
	LastMessageType    string
	LastMessageAt      *time.Time
}

func (r *Repository) ListConversations(userID uint64) ([]conversationListRow, error) {
	var rows []conversationListRow
	err := r.db.Raw(`
		SELECT
			c.id,
			c.type,
			c.name,
			c.creator_id,
			member_stats.member_count,
			CASE WHEN cp.id IS NULL THEN FALSE ELSE TRUE END AS pinned,
			cm.last_read_seq,
			COALESCE(unread_stats.unread_count, 0) AS unread_count,
			c.last_message_seq,
			c.last_message_sender,
			c.last_message_preview,
			c.last_message_type,
			c.last_message_at
		FROM conversation_members cm
		JOIN conversations c ON c.id = cm.conversation_id AND c.deleted_at IS NULL
		JOIN (
			SELECT conversation_id, COUNT(*) AS member_count
			FROM conversation_members
			WHERE deleted_at IS NULL
			GROUP BY conversation_id
		) member_stats ON member_stats.conversation_id = c.id
		LEFT JOIN conversation_pins cp
			ON cp.conversation_id = c.id AND cp.user_id = ? 
		LEFT JOIN (
			SELECT m.conversation_id, COUNT(*) AS unread_count
			FROM messages m
			JOIN conversation_members cm2
				ON cm2.conversation_id = m.conversation_id AND cm2.user_id = ? AND cm2.deleted_at IS NULL
			WHERE m.deleted_at IS NULL
				AND m.seq > cm2.last_read_seq
				AND m.sender_id <> ?
			GROUP BY m.conversation_id
		) unread_stats ON unread_stats.conversation_id = c.id
		WHERE cm.user_id = ? AND cm.deleted_at IS NULL
		ORDER BY
			CASE WHEN cp.id IS NULL THEN 1 ELSE 0 END,
			cp.pinned_at DESC,
			c.updated_at DESC,
			c.id DESC
	`, userID, userID, userID, userID).Scan(&rows).Error
	return rows, err
}

func (r *Repository) RenameConversation(conversationID uint64, name string) error {
	return r.db.Model(&Conversation{}).
		Where("id = ? AND deleted_at IS NULL", conversationID).
		Updates(map[string]any{"name": name, "updated_at": time.Now()}).Error
}

func (r *Repository) AddConversationMembers(conversationID uint64, memberIDs []uint64) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		for _, memberID := range memberIDs {
			member := ConversationMember{
				ConversationID: conversationID,
				UserID:         memberID,
				Role:           "member",
				JoinedAt:       now,
				UpdatedAt:      now,
			}
			if err := tx.Create(&member).Error; err != nil {
				return err
			}
		}
		return tx.Model(&Conversation{}).Where("id = ?", conversationID).Update("updated_at", now).Error
	})
}

func (r *Repository) RemoveConversationMember(conversationID, userID uint64) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&ConversationMember{}).
			Where("conversation_id = ? AND user_id = ? AND deleted_at IS NULL", conversationID, userID).
			Updates(map[string]any{"deleted_at": time.Now(), "updated_at": time.Now()}).Error; err != nil {
			return err
		}
		return tx.Model(&Conversation{}).Where("id = ?", conversationID).Update("updated_at", time.Now()).Error
	})
}

func (r *Repository) SetConversationMemberRole(conversationID, userID uint64, role string) error {
	return r.db.Model(&ConversationMember{}).
		Where("conversation_id = ? AND user_id = ? AND deleted_at IS NULL", conversationID, userID).
		Updates(map[string]any{"role": role, "updated_at": time.Now()}).Error
}

func (r *Repository) SetPin(userID, conversationID uint64, pinned bool) error {
	if pinned {
		return r.db.Where("user_id = ? AND conversation_id = ?", userID, conversationID).
			Assign(ConversationPin{PinnedAt: time.Now()}).
			FirstOrCreate(&ConversationPin{}).Error
	}
	return r.db.Where("user_id = ? AND conversation_id = ?", userID, conversationID).Delete(&ConversationPin{}).Error
}

func (r *Repository) ListMessages(conversationID uint64, beforeSeq uint64, limit int) ([]Message, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	query := r.db.Where("conversation_id = ? AND deleted_at IS NULL", conversationID)
	if beforeSeq > 0 {
		query = query.Where("seq < ?", beforeSeq)
	}
	var messages []Message
	err := query.Order("seq desc").Limit(limit).Find(&messages).Error
	return messages, err
}

func (r *Repository) SendMessage(conversationID, senderID uint64, messageType, content, clientMsgID string) (*Message, *Conversation, error) {
	var output Message
	var conversation Conversation
	err := r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND deleted_at IS NULL", conversationID).
			First(&conversation).Error; err != nil {
			return err
		}
		nextSeq := conversation.LastMessageSeq + 1
		now := time.Now()
		output = Message{
			ConversationID: conversationID,
			Seq:            nextSeq,
			SenderID:       senderID,
			MessageType:    messageType,
			Content:        content,
			ClientMsgID:    clientMsgID,
			Status:         "sent",
			CreatedAt:      now,
			UpdatedAt:      now,
		}
		if err := tx.Create(&output).Error; err != nil {
			return err
		}
		if err := tx.Model(&Conversation{}).
			Where("id = ?", conversationID).
			Updates(map[string]any{
				"last_message_seq":     nextSeq,
				"last_message_sender":  senderID,
				"last_message_type":    messageType,
				"last_message_preview": truncate(content, 255),
				"last_message_at":      now,
				"updated_at":           now,
			}).Error; err != nil {
			return err
		}
		conversation.LastMessageSeq = nextSeq
		conversation.LastMessageSender = senderID
		conversation.LastMessageType = messageType
		conversation.LastMessagePreview = truncate(content, 255)
		conversation.LastMessageAt = &now
		conversation.UpdatedAt = now
		return nil
	})
	return &output, &conversation, err
}

func (r *Repository) MarkRead(conversationID, userID, seq uint64) error {
	return r.db.Model(&ConversationMember{}).
		Where("conversation_id = ? AND user_id = ? AND deleted_at IS NULL", conversationID, userID).
		Update("last_read_seq", gorm.Expr("GREATEST(last_read_seq, ?)", seq)).Error
}

func (r *Repository) AccessibleConversationIDsForConversation(conversationID uint64) ([]uint64, error) {
	var ids []uint64
	err := r.db.Table("conversation_members").
		Select("user_id").
		Where("conversation_id = ? AND deleted_at IS NULL", conversationID).
		Scan(&ids).Error
	return ids, err
}

func (r *Repository) AccessibleConversationIDs(userID uint64) ([]uint64, error) {
	var ids []uint64
	err := r.db.Table("conversation_members").
		Select("conversation_id").
		Where("user_id = ? AND deleted_at IS NULL", userID).
		Scan(&ids).Error
	return ids, err
}

func (r *Repository) ListMessagesForReindex() ([]Message, error) {
	var rows []Message
	err := r.db.Where("deleted_at IS NULL").Order("id asc").Find(&rows).Error
	return rows, err
}

func (r *Repository) ListConversationsForReindex() ([]Conversation, error) {
	var rows []Conversation
	err := r.db.Where("deleted_at IS NULL").Order("id asc").Find(&rows).Error
	return rows, err
}

func (r *Repository) AppendSyncEvents(userIDs []uint64, eventType string, payload any) error {
	userIDs = uniqueUint64s(userIDs)
	if len(userIDs) == 0 {
		return nil
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	now := time.Now()
	rows := make([]SyncEvent, 0, len(userIDs))
	for _, userID := range userIDs {
		rows = append(rows, SyncEvent{
			UserID:    userID,
			EventType: eventType,
			Payload:   string(body),
			CreatedAt: now,
		})
	}
	return r.db.Create(&rows).Error
}

func (r *Repository) CurrentSyncCursor(userID uint64) (uint64, error) {
	var cursor uint64
	err := r.db.Model(&SyncEvent{}).
		Where("user_id = ?", userID).
		Select("COALESCE(MAX(id), 0)").
		Scan(&cursor).Error
	return cursor, err
}

func (r *Repository) ListSyncEvents(userID, cursor uint64, limit int) ([]SyncEvent, uint64, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	var rows []SyncEvent
	if err := r.db.Where("user_id = ? AND id > ?", userID, cursor).
		Order("id asc").
		Limit(limit).
		Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	currentCursor, err := r.CurrentSyncCursor(userID)
	if err != nil {
		return nil, 0, err
	}
	return rows, currentCursor, nil
}

func (r *Repository) SearchMessages(conversationIDs []uint64, query string, offset, limit int) ([]contracts.SearchMessageHit, error) {
	query = "%" + strings.TrimSpace(query) + "%"
	var rows []contracts.SearchMessageHit
	err := r.db.Table("messages m").
		Select("m.id AS message_id, m.conversation_id, c.name AS conversation_name, m.sender_id, m.message_type, m.content, DATE_FORMAT(m.created_at, '%Y-%m-%dT%H:%i:%sZ') AS created_at").
		Joins("JOIN conversations c ON c.id = m.conversation_id AND c.deleted_at IS NULL").
		Where("m.deleted_at IS NULL AND m.conversation_id IN ? AND (m.content LIKE ? OR c.name LIKE ?)", conversationIDs, query, query).
		Order("m.created_at DESC").
		Offset(offset).
		Limit(limit).
		Scan(&rows).Error
	return rows, err
}

func (r *Repository) SearchConversations(conversationIDs []uint64, query string, offset, limit int) ([]contracts.SearchConversationHit, error) {
	query = "%" + strings.TrimSpace(query) + "%"
	var rows []contracts.SearchConversationHit
	err := r.db.Table("conversations").
		Select("id AS conversation_id, name, type, DATE_FORMAT(updated_at, '%Y-%m-%dT%H:%i:%sZ') AS updated_at").
		Where("deleted_at IS NULL AND id IN ? AND name LIKE ?", conversationIDs, query).
		Order("updated_at DESC").
		Offset(offset).
		Limit(limit).
		Scan(&rows).Error
	return rows, err
}

func (r *Repository) FindUserProfiles(ids []uint64) (map[uint64]User, error) {
	var rows []User
	err := r.db.Where("id IN ? AND deleted_at IS NULL", ids).Find(&rows).Error
	if err != nil {
		return nil, err
	}
	result := make(map[uint64]User, len(rows))
	for _, row := range rows {
		result[row.ID] = row
	}
	return result, nil
}

func formatDirectKey(userA, userB uint64) string {
	if userA > userB {
		userA, userB = userB, userA
	}
	return fmt.Sprintf("%d:%d", userA, userB)
}

func uniqueUint64s(values []uint64) []uint64 {
	seen := make(map[uint64]struct{}, len(values))
	result := make([]uint64, 0, len(values))
	for _, value := range values {
		if value == 0 {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func truncate(value string, max int) string {
	if len(value) <= max {
		return value
	}
	return value[:max]
}
