package repository

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID           uint64         `gorm:"primaryKey"`
	Username     string         `gorm:"uniqueIndex;size:64;not null"`
	DisplayName  string         `gorm:"size:128;not null"`
	AvatarURL    string         `gorm:"size:512;not null;default:''"`
	PasswordHash string         `gorm:"size:255;not null"`
	CreatedAt    time.Time      `gorm:"not null"`
	UpdatedAt    time.Time      `gorm:"not null"`
	DeletedAt    gorm.DeletedAt `gorm:"index"`
}

type AdminUser struct {
	ID                 uint64         `gorm:"primaryKey"`
	Username           string         `gorm:"uniqueIndex;size:64;not null"`
	PasswordHash       string         `gorm:"size:255;not null"`
	MustChangePassword bool           `gorm:"not null;default:true"`
	CreatedAt          time.Time      `gorm:"not null"`
	UpdatedAt          time.Time      `gorm:"not null"`
	DeletedAt          gorm.DeletedAt `gorm:"index"`
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
	ID         uint64         `gorm:"primaryKey"`
	UserID     uint64         `gorm:"uniqueIndex:idx_friend_pair;not null"`
	FriendID   uint64         `gorm:"uniqueIndex:idx_friend_pair;not null"`
	RemarkName string         `gorm:"size:128;not null;default:''"`
	CreatedAt  time.Time      `gorm:"not null"`
	DeletedAt  gorm.DeletedAt `gorm:"index"`
}

type Conversation struct {
	ID                 uint64 `gorm:"primaryKey"`
	Type               string `gorm:"size:32;index;not null"`
	DirectKey          string `gorm:"size:128;uniqueIndex"`
	Name               string `gorm:"size:128;not null"`
	Announcement       string `gorm:"size:512;not null;default:''"`
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

type ConversationSetting struct {
	ID             uint64 `gorm:"primaryKey"`
	ConversationID uint64 `gorm:"uniqueIndex:idx_conversation_setting;not null"`
	UserID         uint64 `gorm:"uniqueIndex:idx_conversation_setting;not null"`
	IsMuted        bool   `gorm:"not null;default:false"`
	Draft          string `gorm:"type:text;not null;default:''"`
	PinnedAt       *time.Time
	CreatedAt      time.Time      `gorm:"not null"`
	UpdatedAt      time.Time      `gorm:"not null"`
	DeletedAt      gorm.DeletedAt `gorm:"index"`
}

type Message struct {
	ID               uint64 `gorm:"primaryKey"`
	ConversationID   uint64 `gorm:"index:idx_message_conversation_seq,priority:1;not null"`
	Seq              uint64 `gorm:"index:idx_message_conversation_seq,priority:2;not null"`
	SenderID         uint64 `gorm:"index;uniqueIndex:uk_messages_sender_client_msg_id;not null"`
	MessageType      string `gorm:"size:32;not null"`
	Content          string `gorm:"type:text;not null"`
	ReplyToMessageID *uint64
	AttachmentJSON   string    `gorm:"type:text"`
	ClientMsgID      string    `gorm:"size:128;uniqueIndex:uk_messages_sender_client_msg_id;not null"`
	Status           string    `gorm:"size:32;not null"`
	DeliveryStatus   string    `gorm:"size:32;not null;default:'sent'"`
	CreatedAt        time.Time `gorm:"not null"`
	UpdatedAt        time.Time `gorm:"not null"`
	RecalledAt       *time.Time
	DeletedAt        gorm.DeletedAt `gorm:"index"`
}

type SyncEvent struct {
	ID          uint64    `gorm:"primaryKey"`
	UserID      uint64    `gorm:"index:idx_sync_events_user_cursor,priority:1;not null"`
	EventType   string    `gorm:"size:64;not null"`
	AggregateID string    `gorm:"size:128;index:idx_sync_events_user_aggregate,priority:2;not null;default:''"`
	Payload     string    `gorm:"type:json;not null"`
	CreatedAt   time.Time `gorm:"not null"`
}

type AIAuditLog struct {
	ID                  uint64    `gorm:"primaryKey"`
	UserID              uint64    `gorm:"index;not null"`
	ConversationID      uint64    `gorm:"index;not null"`
	RequestID           string    `gorm:"size:128;index;not null;default:''"`
	Provider            string    `gorm:"size:64;index;not null"`
	Model               string    `gorm:"size:128;index;not null"`
	Status              string    `gorm:"size:32;index;not null"`
	DurationMS          int64     `gorm:"not null;default:0"`
	InputTokens         int       `gorm:"not null;default:0"`
	OutputTokens        int       `gorm:"not null;default:0"`
	TotalTokens         int       `gorm:"not null;default:0"`
	InputPreview        string    `gorm:"type:text;not null;default:''"`
	OutputPreview       string    `gorm:"type:text;not null;default:''"`
	RequestPayloadJSON  string    `gorm:"type:longtext;not null"`
	ResponsePayloadJSON string    `gorm:"type:longtext;not null"`
	ErrorCode           string    `gorm:"size:64;not null;default:''"`
	ErrorMessage        string    `gorm:"type:text;not null;default:''"`
	CreatedAt           time.Time `gorm:"index;not null"`
}

type AIRetryJob struct {
	ID             uint64         `gorm:"primaryKey"`
	UserID         uint64         `gorm:"index;not null"`
	ConversationID uint64         `gorm:"index;not null"`
	Status         string         `gorm:"size:32;index;not null"`
	AttemptCount   int            `gorm:"not null;default:0"`
	NextAttemptAt  time.Time      `gorm:"index;not null"`
	LastError      string         `gorm:"type:text;not null;default:''"`
	CreatedAt      time.Time      `gorm:"not null"`
	UpdatedAt      time.Time      `gorm:"not null"`
	DeletedAt      gorm.DeletedAt `gorm:"index"`
}
