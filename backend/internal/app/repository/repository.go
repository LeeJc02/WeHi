package repository

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/LeeJc02/WeHi/backend/pkg/contracts"

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

func (r *Repository) CreateAdminUser(user *AdminUser) error {
	return r.db.Create(user).Error
}

func (r *Repository) FindAdminByUsername(username string) (*AdminUser, error) {
	var user AdminUser
	if err := r.db.Where("username = ? AND deleted_at IS NULL", username).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *Repository) FindAdminByID(id uint64) (*AdminUser, error) {
	var user AdminUser
	if err := r.db.Where("id = ? AND deleted_at IS NULL", id).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *Repository) UpdateAdminPassword(id uint64, passwordHash string, mustChangePassword bool) error {
	return r.db.Model(&AdminUser{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Updates(map[string]any{
			"password_hash":        passwordHash,
			"must_change_password": mustChangePassword,
			"updated_at":           time.Now(),
		}).Error
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

func (r *Repository) UpdateUserProfile(id uint64, displayName, avatarURL string) error {
	return r.db.Model(&User{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Updates(map[string]any{
			"display_name": displayName,
			"avatar_url":   avatarURL,
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
		Select("u.id, u.username, u.display_name, u.avatar_url, f.remark_name").
		Joins("JOIN users u ON u.id = f.friend_id AND u.deleted_at IS NULL").
		Where("f.user_id = ? AND f.deleted_at IS NULL", userID).
		Order("CASE WHEN f.remark_name = '' THEN u.display_name ELSE f.remark_name END asc, u.username asc").
		Scan(&rows).Error
	return rows, err
}

func (r *Repository) UpdateFriendRemark(userID, friendID uint64, remarkName string) error {
	return r.db.Model(&Friendship{}).
		Where("user_id = ? AND friend_id = ? AND deleted_at IS NULL", userID, friendID).
		Update("remark_name", remarkName).Error
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

func (r *Repository) EnsureFriendship(userID, friendID uint64) error {
	if userID == 0 || friendID == 0 || userID == friendID {
		return nil
	}
	now := time.Now()
	return r.db.Transaction(func(tx *gorm.DB) error {
		row := Friendship{
			UserID:    userID,
			FriendID:  friendID,
			CreatedAt: now,
		}
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}, {Name: "friend_id"}},
			DoUpdates: clause.Assignments(map[string]any{"deleted_at": gorm.Expr("NULL")}),
		}).Create(&row).Error; err != nil {
			return err
		}
		return tx.Unscoped().
			Model(&Friendship{}).
			Where("user_id = ? AND friend_id = ?", userID, friendID).
			Update("deleted_at", gorm.Expr("NULL")).Error
	})
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
		createTx := tx
		if strings.TrimSpace(conversation.DirectKey) == "" {
			createTx = createTx.Omit("DirectKey")
		}
		if err := createTx.Create(conversation).Error; err != nil {
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
	AvatarURL   string
	RemarkName  string
	Role        string
	LastReadSeq uint64
	JoinedAt    time.Time
}

func (r *Repository) ListConversationMemberProfiles(conversationID, viewerID uint64) ([]ConversationMemberRow, error) {
	var rows []ConversationMemberRow
	err := r.db.Table("conversation_members cm").
		Select("cm.user_id, u.username, u.display_name, u.avatar_url, COALESCE(f.remark_name, '') AS remark_name, cm.role, cm.last_read_seq, cm.joined_at").
		Joins("JOIN users u ON u.id = cm.user_id AND u.deleted_at IS NULL").
		Joins("LEFT JOIN friendships f ON f.user_id = ? AND f.friend_id = cm.user_id AND f.deleted_at IS NULL", viewerID).
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
	Announcement       string
	CreatorID          uint64
	MemberCount        int64
	Pinned             bool
	PinnedAt           *time.Time
	IsMuted            bool
	Draft              string
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
			c.announcement,
			c.creator_id,
			member_stats.member_count,
			CASE WHEN cs.pinned_at IS NULL THEN FALSE ELSE TRUE END AS pinned,
			cs.pinned_at,
			COALESCE(cs.is_muted, FALSE) AS is_muted,
			COALESCE(cs.draft, '') AS draft,
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
		LEFT JOIN conversation_settings cs
			ON cs.conversation_id = c.id AND cs.user_id = ? AND cs.deleted_at IS NULL
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
			CASE WHEN cs.pinned_at IS NULL THEN 1 ELSE 0 END,
			cs.pinned_at DESC,
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
			var existing ConversationMember
			err := tx.Unscoped().
				Where("conversation_id = ? AND user_id = ?", conversationID, memberID).
				First(&existing).Error
			if errors.Is(err, gorm.ErrRecordNotFound) {
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
				continue
			}
			if err != nil {
				return err
			}
			if !existing.DeletedAt.Valid {
				continue
			}
			if err := tx.Unscoped().
				Model(&ConversationMember{}).
				Where("id = ?", existing.ID).
				Updates(map[string]any{
					"deleted_at":    gorm.Expr("NULL"),
					"updated_at":    now,
					"joined_at":     now,
					"role":          "member",
					"last_read_seq": 0,
				}).Error; err != nil {
				return err
			}
		}
		return tx.Model(&Conversation{}).Where("id = ?", conversationID).Update("updated_at", now).Error
	})
}

func (r *Repository) EnsureConversationPinned(userID, conversationID uint64) error {
	now := time.Now()
	return r.db.Transaction(func(tx *gorm.DB) error {
		var setting ConversationSetting
		err := tx.
			Where("conversation_id = ? AND user_id = ? AND deleted_at IS NULL", conversationID, userID).
			First(&setting).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return tx.Create(&ConversationSetting{
				ConversationID: conversationID,
				UserID:         userID,
				PinnedAt:       &now,
				CreatedAt:      now,
				UpdatedAt:      now,
			}).Error
		}
		if err != nil {
			return err
		}
		return tx.Model(&ConversationSetting{}).
			Where("id = ?", setting.ID).
			Updates(map[string]any{
				"pinned_at":  now,
				"updated_at": now,
				"deleted_at": gorm.Expr("NULL"),
			}).Error
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

func (r *Repository) DeleteGroupConversation(conversationID uint64, aggregateID string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("aggregate_id = ?", strings.TrimSpace(aggregateID)).Delete(&SyncEvent{}).Error; err != nil {
			return err
		}
		if err := tx.Where("conversation_id = ?", conversationID).Delete(&AIAuditLog{}).Error; err != nil {
			return err
		}
		if err := tx.Unscoped().Where("conversation_id = ?", conversationID).Delete(&AIRetryJob{}).Error; err != nil {
			return err
		}
		if err := tx.Unscoped().Where("conversation_id = ?", conversationID).Delete(&ConversationSetting{}).Error; err != nil {
			return err
		}
		if err := tx.Where("conversation_id = ?", conversationID).Delete(&ConversationPin{}).Error; err != nil {
			return err
		}
		if err := tx.Unscoped().Where("conversation_id = ?", conversationID).Delete(&Message{}).Error; err != nil {
			return err
		}
		if err := tx.Unscoped().Where("conversation_id = ?", conversationID).Delete(&ConversationMember{}).Error; err != nil {
			return err
		}
		return tx.Unscoped().Where("id = ?", conversationID).Delete(&Conversation{}).Error
	})
}

func (r *Repository) SetConversationMemberRole(conversationID, userID uint64, role string) error {
	return r.db.Model(&ConversationMember{}).
		Where("conversation_id = ? AND user_id = ? AND deleted_at IS NULL", conversationID, userID).
		Updates(map[string]any{"role": role, "updated_at": time.Now()}).Error
}

func (r *Repository) UpdateConversationSettings(userID, conversationID uint64, pinned, isMuted *bool, draft *string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		var setting ConversationSetting
		err := tx.
			Where("conversation_id = ? AND user_id = ? AND deleted_at IS NULL", conversationID, userID).
			First(&setting).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			setting = ConversationSetting{
				ConversationID: conversationID,
				UserID:         userID,
				CreatedAt:      now,
				UpdatedAt:      now,
			}
			if draft != nil {
				setting.Draft = *draft
			}
			if isMuted != nil {
				setting.IsMuted = *isMuted
			}
			if pinned != nil && *pinned {
				setting.PinnedAt = &now
			}
			return tx.Create(&setting).Error
		}
		if err != nil {
			return err
		}

		updates := map[string]any{
			"updated_at": now,
		}
		if draft != nil {
			updates["draft"] = *draft
		}
		if isMuted != nil {
			updates["is_muted"] = *isMuted
		}
		if pinned != nil {
			if *pinned {
				updates["pinned_at"] = now
			} else {
				updates["pinned_at"] = gorm.Expr("NULL")
			}
		}
		return tx.Model(&ConversationSetting{}).
			Where("id = ?", setting.ID).
			Updates(updates).Error
	})
}

func (r *Repository) UpdateConversationAnnouncement(conversationID uint64, announcement string) error {
	return r.db.Model(&Conversation{}).
		Where("id = ? AND deleted_at IS NULL", conversationID).
		Updates(map[string]any{
			"announcement": announcement,
			"updated_at":   time.Now(),
		}).Error
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

func (r *Repository) FindMessageByID(id uint64) (*Message, error) {
	var message Message
	if err := r.db.Where("id = ? AND deleted_at IS NULL", id).First(&message).Error; err != nil {
		return nil, err
	}
	return &message, nil
}

func (r *Repository) FindMessagesByIDs(ids []uint64) ([]Message, error) {
	if len(ids) == 0 {
		return []Message{}, nil
	}
	var rows []Message
	err := r.db.Where("id IN ? AND deleted_at IS NULL", ids).Find(&rows).Error
	return rows, err
}

func (r *Repository) FindMessageByClientMsgID(senderID uint64, clientMsgID string) (*Message, error) {
	var message Message
	if err := r.db.Where("sender_id = ? AND client_msg_id = ? AND deleted_at IS NULL", senderID, clientMsgID).First(&message).Error; err != nil {
		return nil, err
	}
	return &message, nil
}

func (r *Repository) FindMessageByClientMsgIDForAdmin(clientMsgID string, senderID, conversationID uint64) (*Message, error) {
	query := r.db.Where("client_msg_id = ? AND deleted_at IS NULL", clientMsgID)
	if senderID > 0 {
		query = query.Where("sender_id = ?", senderID)
	}
	if conversationID > 0 {
		query = query.Where("conversation_id = ?", conversationID)
	}
	var message Message
	if err := query.Order("id desc").First(&message).Error; err != nil {
		return nil, err
	}
	return &message, nil
}

func (r *Repository) SendMessage(conversationID, senderID uint64, messageType, content, clientMsgID string, replyToMessageID *uint64, attachmentJSON string) (*Message, *Conversation, bool, error) {
	var output Message
	var conversation Conversation
	created := false
	err := r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND deleted_at IS NULL", conversationID).
			First(&conversation).Error; err != nil {
			return err
		}
		if clientMsgID != "" {
			var existing Message
			if err := tx.Where("sender_id = ? AND client_msg_id = ? AND deleted_at IS NULL", senderID, clientMsgID).First(&existing).Error; err == nil {
				output = existing
				return nil
			} else if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
		}
		nextSeq := conversation.LastMessageSeq + 1
		now := time.Now()
		output = Message{
			ConversationID:   conversationID,
			Seq:              nextSeq,
			SenderID:         senderID,
			MessageType:      messageType,
			Content:          content,
			ReplyToMessageID: replyToMessageID,
			AttachmentJSON:   attachmentJSON,
			ClientMsgID:      clientMsgID,
			Status:           "sent",
			DeliveryStatus:   "sent",
			CreatedAt:        now,
			UpdatedAt:        now,
		}
		if err := tx.Create(&output).Error; err != nil {
			return err
		}
		created = true
		preview := buildMessagePreview(messageType, content, attachmentJSON, false)
		if err := tx.Model(&Conversation{}).
			Where("id = ?", conversationID).
			Updates(map[string]any{
				"last_message_seq":     nextSeq,
				"last_message_sender":  senderID,
				"last_message_type":    messageType,
				"last_message_preview": preview,
				"last_message_at":      now,
				"updated_at":           now,
			}).Error; err != nil {
			return err
		}
		conversation.LastMessageSeq = nextSeq
		conversation.LastMessageSender = senderID
		conversation.LastMessageType = messageType
		conversation.LastMessagePreview = preview
		conversation.LastMessageAt = &now
		conversation.UpdatedAt = now
		return nil
	})
	return &output, &conversation, created, err
}

func (r *Repository) RecallMessage(messageID uint64) (*Message, error) {
	var output Message
	err := r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND deleted_at IS NULL", messageID).
			First(&output).Error; err != nil {
			return err
		}
		now := time.Now()
		if err := tx.Model(&Message{}).
			Where("id = ?", messageID).
			Updates(map[string]any{
				"recalled_at": now,
				"updated_at":  now,
			}).Error; err != nil {
			return err
		}
		output.RecalledAt = &now
		return tx.Model(&Conversation{}).
			Where("id = ? AND last_message_seq = ?", output.ConversationID, output.Seq).
			Updates(map[string]any{
				"last_message_preview": "[消息已撤回]",
				"updated_at":           now,
			}).Error
	})
	return &output, err
}

func (r *Repository) UpdateMessageDeliveryStatus(messageID uint64, deliveryStatus string) (*Message, error) {
	var output Message
	err := r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND deleted_at IS NULL", messageID).
			First(&output).Error; err != nil {
			return err
		}
		now := time.Now()
		if err := tx.Model(&Message{}).
			Where("id = ?", messageID).
			Updates(map[string]any{
				"status":          deliveryStatus,
				"delivery_status": deliveryStatus,
				"updated_at":      now,
			}).Error; err != nil {
			return err
		}
		output.Status = deliveryStatus
		output.DeliveryStatus = deliveryStatus
		output.UpdatedAt = now
		return nil
	})
	return &output, err
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

func (r *Repository) AppendSyncEvents(userIDs []uint64, eventType, aggregateID string, payload any) error {
	userIDs = uniqueUint64s(userIDs)
	if len(userIDs) == 0 {
		return nil
	}
	// Sync events are written per recipient instead of per conversation so
	// reconnect replay can rebuild one user's view without re-filtering a shared
	// stream on the client.
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	now := time.Now()
	rows := make([]SyncEvent, 0, len(userIDs))
	for _, userID := range userIDs {
		rows = append(rows, SyncEvent{
			UserID:      userID,
			EventType:   eventType,
			AggregateID: aggregateID,
			Payload:     string(body),
			CreatedAt:   now,
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
	// Pagination always moves forward on the auto-increment id, which means the
	// cursor doubles as both ordering key and replay watermark.
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

func (r *Repository) ListSyncEventsByAggregate(aggregateID string, limit int) ([]SyncEvent, error) {
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	var rows []SyncEvent
	err := r.db.Where("aggregate_id = ?", aggregateID).
		Order("id asc").
		Limit(limit).
		Find(&rows).Error
	return rows, err
}

type ConversationReadStateRow struct {
	UserID        uint64
	LastReadSeq   uint64
	UnreadCount   int64
	Online        bool
	DisplayName   string
	Username      string
	AvatarURL     string
	RemarkName    string
	Role          string
	CurrentCursor uint64
}

func (r *Repository) ListConversationReadStates(conversationID uint64) ([]ConversationReadStateRow, error) {
	var rows []ConversationReadStateRow
	err := r.db.Raw(`
		SELECT
			cm.user_id,
			cm.last_read_seq,
			COALESCE(unread_stats.unread_count, 0) AS unread_count,
			u.display_name,
			u.username,
			u.avatar_url,
			cm.role,
			COALESCE(sync_stats.current_cursor, 0) AS current_cursor
		FROM conversation_members cm
		JOIN users u ON u.id = cm.user_id AND u.deleted_at IS NULL
		LEFT JOIN (
			SELECT cm2.user_id, cm2.conversation_id, COUNT(*) AS unread_count
			FROM messages m
			JOIN conversation_members cm2
				ON cm2.conversation_id = m.conversation_id AND cm2.deleted_at IS NULL
			WHERE m.deleted_at IS NULL
				AND m.seq > cm2.last_read_seq
				AND m.sender_id <> cm2.user_id
			GROUP BY cm2.user_id, cm2.conversation_id
		) unread_stats ON unread_stats.user_id = cm.user_id AND unread_stats.conversation_id = cm.conversation_id
		LEFT JOIN (
			SELECT user_id, MAX(id) AS current_cursor
			FROM sync_events
			GROUP BY user_id
		) sync_stats ON sync_stats.user_id = cm.user_id
		WHERE cm.conversation_id = ? AND cm.deleted_at IS NULL
		ORDER BY cm.id ASC
	`, conversationID).Scan(&rows).Error
	return rows, err
}

func (r *Repository) CreateAIAuditLog(log *AIAuditLog) error {
	return r.db.Create(log).Error
}

func (r *Repository) UpdateAIAuditLog(id uint64, updates map[string]any) error {
	if len(updates) == 0 {
		return nil
	}
	return r.db.Model(&AIAuditLog{}).
		Where("id = ?", id).
		Updates(updates).Error
}

func (r *Repository) FindAIAuditLogByID(id uint64) (*AIAuditLog, error) {
	var row AIAuditLog
	if err := r.db.Where("id = ?", id).First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *Repository) ListAIAuditLogs(limit int, status, provider, model string, userID, conversationID uint64) ([]AIAuditLog, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	query := r.db.Model(&AIAuditLog{})
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if provider != "" {
		query = query.Where("provider = ?", provider)
	}
	if model != "" {
		query = query.Where("model = ?", model)
	}
	if userID > 0 {
		query = query.Where("user_id = ?", userID)
	}
	if conversationID > 0 {
		query = query.Where("conversation_id = ?", conversationID)
	}
	var rows []AIAuditLog
	err := query.Order("id desc").Limit(limit).Find(&rows).Error
	return rows, err
}

func (r *Repository) DeleteAIAuditLogsBefore(before time.Time) error {
	return r.db.Where("created_at < ?", before).Delete(&AIAuditLog{}).Error
}

func (r *Repository) CreateAIRetryJob(job *AIRetryJob) error {
	return r.db.Create(job).Error
}

func (r *Repository) ListPendingAIRetryJobs(limit int, now time.Time) ([]AIRetryJob, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	var rows []AIRetryJob
	err := r.db.Where("status = ? AND next_attempt_at <= ? AND deleted_at IS NULL", "pending", now).
		Order("next_attempt_at asc, id asc").
		Limit(limit).
		Find(&rows).Error
	return rows, err
}

func (r *Repository) UpdateAIRetryJob(id uint64, updates map[string]any) error {
	if len(updates) == 0 {
		return nil
	}
	updates["updated_at"] = time.Now()
	return r.db.Model(&AIRetryJob{}).Where("id = ? AND deleted_at IS NULL", id).Updates(updates).Error
}

func (r *Repository) UpdateAIRetryJobs(ids []uint64, updates map[string]any) error {
	if len(ids) == 0 || len(updates) == 0 {
		return nil
	}
	updates["updated_at"] = time.Now()
	return r.db.Model(&AIRetryJob{}).
		Where("id IN ? AND deleted_at IS NULL", ids).
		Updates(updates).Error
}

func (r *Repository) FindAIRetryJobByID(id uint64) (*AIRetryJob, error) {
	var row AIRetryJob
	if err := r.db.Where("id = ? AND deleted_at IS NULL", id).First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *Repository) ListAIRetryJobs(limit int, status string) ([]AIRetryJob, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	query := r.db.Where("deleted_at IS NULL")
	if strings.TrimSpace(status) != "" {
		query = query.Where("status = ?", strings.TrimSpace(status))
	}
	var rows []AIRetryJob
	err := query.Order("id desc").Limit(limit).Find(&rows).Error
	return rows, err
}

func (r *Repository) DeleteAIRetryJobsByStatuses(statuses []string) error {
	cleaned := make([]string, 0, len(statuses))
	for _, status := range statuses {
		status = strings.TrimSpace(status)
		if status != "" {
			cleaned = append(cleaned, status)
		}
	}
	if len(cleaned) == 0 {
		return nil
	}
	return r.db.Where("status IN ? AND deleted_at IS NULL", cleaned).Delete(&AIRetryJob{}).Error
}

func (r *Repository) DeleteAIRetryJobsBefore(statuses []string, before time.Time) error {
	cleaned := make([]string, 0, len(statuses))
	for _, status := range statuses {
		status = strings.TrimSpace(status)
		if status != "" {
			cleaned = append(cleaned, status)
		}
	}
	if len(cleaned) == 0 {
		return nil
	}
	return r.db.Where("status IN ? AND updated_at < ? AND deleted_at IS NULL", cleaned, before).Delete(&AIRetryJob{}).Error
}

func (r *Repository) CountAIRetryJobsByStatus(statuses []string) (map[string]int64, error) {
	result := make(map[string]int64, len(statuses))
	cleaned := make([]string, 0, len(statuses))
	for _, status := range statuses {
		status = strings.TrimSpace(status)
		if status != "" {
			cleaned = append(cleaned, status)
			result[status] = 0
		}
	}
	if len(cleaned) == 0 {
		return result, nil
	}
	type row struct {
		Status string
		Count  int64
	}
	var rows []row
	if err := r.db.Model(&AIRetryJob{}).
		Select("status, COUNT(*) AS count").
		Where("status IN ? AND deleted_at IS NULL", cleaned).
		Group("status").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		result[row.Status] = row.Count
	}
	return result, nil
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

func buildMessagePreview(messageType, content, attachmentJSON string, recalled bool) string {
	if recalled {
		return "[消息已撤回]"
	}
	switch messageType {
	case "image":
		return "[图片]"
	case "file":
		if attachmentJSON != "" {
			var attachment contracts.AttachmentDTO
			if json.Unmarshal([]byte(attachmentJSON), &attachment) == nil && attachment.Filename != "" {
				return truncate("[文件] "+attachment.Filename, 255)
			}
		}
		return "[文件]"
	default:
		return truncate(content, 255)
	}
}
