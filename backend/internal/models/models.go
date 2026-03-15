package models

import "time"

type User struct {
	ID           uint64    `gorm:"primaryKey" json:"id"`
	Username     string    `gorm:"uniqueIndex;size:64;not null" json:"username"`
	DisplayName  string    `gorm:"size:128;not null" json:"display_name"`
	PasswordHash string    `gorm:"size:255;not null" json:"-"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Session struct {
	ID        uint64    `gorm:"primaryKey"`
	Token     string    `gorm:"uniqueIndex;size:128;not null"`
	UserID    uint64    `gorm:"index;not null"`
	CreatedAt time.Time `json:"created_at"`
}

type Friendship struct {
	ID        uint64    `gorm:"primaryKey"`
	UserID    uint64    `gorm:"uniqueIndex:idx_friend_pair;not null"`
	FriendID  uint64    `gorm:"uniqueIndex:idx_friend_pair;not null"`
	CreatedAt time.Time `json:"created_at"`
}

type Conversation struct {
	ID        uint64    `gorm:"primaryKey" json:"id"`
	Type      string    `gorm:"size:32;index;not null" json:"type"`
	Name      string    `gorm:"size:128" json:"name"`
	CreatorID uint64    `gorm:"index;not null" json:"creator_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ConversationMember struct {
	ID                uint64    `gorm:"primaryKey"`
	ConversationID    uint64    `gorm:"uniqueIndex:idx_conversation_member;not null"`
	UserID            uint64    `gorm:"uniqueIndex:idx_conversation_member;not null"`
	Role              string    `gorm:"size:32;not null" json:"role"`
	LastReadMessageID uint64    `gorm:"default:0" json:"last_read_message_id"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type Message struct {
	ID             uint64    `gorm:"primaryKey" json:"id"`
	ConversationID uint64    `gorm:"index;not null" json:"conversation_id"`
	SenderID       uint64    `gorm:"index;not null" json:"sender_id"`
	Content        string    `gorm:"type:text;not null" json:"content"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type FriendDTO struct {
	ID          uint64 `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
}

type ConversationDTO struct {
	ID                 uint64 `json:"id"`
	Type               string `json:"type"`
	Name               string `json:"name"`
	CreatorID          uint64 `json:"creator_id"`
	MemberCount        int    `json:"member_count"`
	UnreadCount        int64  `json:"unread_count"`
	LastMessageID      uint64 `json:"last_message_id,omitempty"`
	LastMessageSender  uint64 `json:"last_message_sender,omitempty"`
	LastMessagePreview string `json:"last_message_preview,omitempty"`
	LastMessageAt      string `json:"last_message_at,omitempty"`
}
