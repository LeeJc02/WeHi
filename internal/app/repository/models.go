package repository

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID           uint64         `gorm:"primaryKey"`
	Username     string         `gorm:"uniqueIndex;size:64;not null"`
	DisplayName  string         `gorm:"size:128;not null"`
	PasswordHash string         `gorm:"size:255;not null"`
	CreatedAt    time.Time      `gorm:"not null"`
	UpdatedAt    time.Time      `gorm:"not null"`
	DeletedAt    gorm.DeletedAt `gorm:"index"`
}

type FriendRequest struct {
	ID          uint64         `gorm:"primaryKey"`
	RequesterID uint64         `gorm:"index;not null"`
	AddresseeID uint64         `gorm:"index;not null"`
	Message     string         `gorm:"size:255"`
	Status      string         `gorm:"size:32;index;not null"`
	CreatedAt   time.Time      `gorm:"not null"`
	UpdatedAt   time.Time      `gorm:"not null"`
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

type Friendship struct {
	ID        uint64         `gorm:"primaryKey"`
	UserID    uint64         `gorm:"uniqueIndex:idx_friend_pair;not null"`
	FriendID  uint64         `gorm:"uniqueIndex:idx_friend_pair;not null"`
	CreatedAt time.Time      `gorm:"not null"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

type Conversation struct {
	ID                 uint64 `gorm:"primaryKey"`
	Type               string `gorm:"size:32;index;not null"`
	DirectKey          string `gorm:"size:128;uniqueIndex"`
	Name               string `gorm:"size:128;not null"`
	CreatorID          uint64 `gorm:"index;not null"`
	LastMessageSeq     uint64 `gorm:"not null;default:0"`
	LastMessageSender  uint64 `gorm:"not null;default:0"`
	LastMessageType    string `gorm:"size:32;not null;default:'system'"`
	LastMessagePreview string `gorm:"size:255;not null;default:''"`
	LastMessageAt      *time.Time
	CreatedAt          time.Time      `gorm:"not null"`
	UpdatedAt          time.Time      `gorm:"not null"`
	DeletedAt          gorm.DeletedAt `gorm:"index"`
}

type ConversationMember struct {
	ID             uint64         `gorm:"primaryKey"`
	ConversationID uint64         `gorm:"uniqueIndex:idx_conversation_member;not null"`
	UserID         uint64         `gorm:"uniqueIndex:idx_conversation_member;not null"`
	Role           string         `gorm:"size:32;not null"`
	LastReadSeq    uint64         `gorm:"not null;default:0"`
	JoinedAt       time.Time      `gorm:"not null"`
	UpdatedAt      time.Time      `gorm:"not null"`
	DeletedAt      gorm.DeletedAt `gorm:"index"`
}

type ConversationPin struct {
	ID             uint64    `gorm:"primaryKey"`
	ConversationID uint64    `gorm:"uniqueIndex:idx_conversation_pin;not null"`
	UserID         uint64    `gorm:"uniqueIndex:idx_conversation_pin;not null"`
	PinnedAt       time.Time `gorm:"not null"`
}

type Message struct {
	ID             uint64         `gorm:"primaryKey"`
	ConversationID uint64         `gorm:"index:idx_message_conversation_seq,priority:1;not null"`
	Seq            uint64         `gorm:"index:idx_message_conversation_seq,priority:2;not null"`
	SenderID       uint64         `gorm:"index;not null"`
	MessageType    string         `gorm:"size:32;not null"`
	Content        string         `gorm:"type:text;not null"`
	ClientMsgID    string         `gorm:"size:128;uniqueIndex;not null"`
	Status         string         `gorm:"size:32;not null"`
	CreatedAt      time.Time      `gorm:"not null"`
	UpdatedAt      time.Time      `gorm:"not null"`
	DeletedAt      gorm.DeletedAt `gorm:"index"`
}

type SyncEvent struct {
	ID        uint64    `gorm:"primaryKey"`
	UserID    uint64    `gorm:"index:idx_sync_events_user_cursor,priority:1;not null"`
	EventType string    `gorm:"size:64;not null"`
	Payload   string    `gorm:"type:json;not null"`
	CreatedAt time.Time `gorm:"not null"`
}
