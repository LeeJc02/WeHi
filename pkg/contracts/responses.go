package contracts

import "encoding/json"

type UserProfile struct {
	ID          uint64 `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	AvatarURL   string `json:"avatar_url"`
}

type SessionInfo struct {
	ID         string `json:"id"`
	DeviceID   string `json:"device_id"`
	UserAgent  string `json:"user_agent"`
	LastSeenAt string `json:"last_seen_at"`
	ExpiresAt  string `json:"expires_at"`
	Current    bool   `json:"current"`
}

type AuthPayload struct {
	AccessToken  string      `json:"access_token"`
	RefreshToken string      `json:"refresh_token,omitempty"`
	User         UserProfile `json:"user"`
}

type FriendDTO struct {
	ID          uint64 `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	AvatarURL   string `json:"avatar_url"`
	RemarkName  string `json:"remark_name"`
	IsAIBot     bool   `json:"is_ai_bot"`
}

type FriendRequestDTO struct {
	ID        uint64      `json:"id"`
	Status    string      `json:"status"`
	Direction string      `json:"direction"`
	Message   string      `json:"message"`
	Requester UserProfile `json:"requester"`
	Addressee UserProfile `json:"addressee"`
	CreatedAt string      `json:"created_at"`
	UpdatedAt string      `json:"updated_at"`
}

type ConversationDTO struct {
	ID                 uint64 `json:"id"`
	Type               string `json:"type"`
	Name               string `json:"name"`
	Announcement       string `json:"announcement"`
	CreatorID          uint64 `json:"creator_id"`
	MemberCount        int64  `json:"member_count"`
	Pinned             bool   `json:"pinned"`
	PinnedAt           string `json:"pinned_at"`
	IsMuted            bool   `json:"is_muted"`
	Draft              string `json:"draft"`
	LastReadSeq        uint64 `json:"last_read_seq"`
	UnreadCount        int64  `json:"unread_count"`
	LastMessageSeq     uint64 `json:"last_message_seq"`
	LastMessageSender  uint64 `json:"last_message_sender"`
	LastMessagePreview string `json:"last_message_preview"`
	LastMessageType    string `json:"last_message_type"`
	LastMessageAt      string `json:"last_message_at"`
}

type ConversationMemberDTO struct {
	UserID      uint64 `json:"user_id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	AvatarURL   string `json:"avatar_url"`
	RemarkName  string `json:"remark_name"`
	Role        string `json:"role"`
	LastReadSeq uint64 `json:"last_read_seq"`
	JoinedAt    string `json:"joined_at"`
	Online      bool   `json:"online"`
}

type MessageDTO struct {
	ID               uint64               `json:"id"`
	ConversationID   uint64               `json:"conversation_id"`
	Seq              uint64               `json:"seq"`
	SenderID         uint64               `json:"sender_id"`
	MessageType      string               `json:"message_type"`
	Content          string               `json:"content"`
	ReplyToMessageID *uint64              `json:"reply_to_message_id,omitempty"`
	ReplyTo          *MessageReferenceDTO `json:"reply_to,omitempty"`
	Attachment       *AttachmentDTO       `json:"attachment,omitempty"`
	ClientMsgID      string               `json:"client_msg_id"`
	Status           string               `json:"status"`
	DeliveryStatus   string               `json:"delivery_status"`
	CreatedAt        string               `json:"created_at"`
	RecalledAt       string               `json:"recalled_at"`
}

type SearchConversationHit struct {
	ConversationID uint64 `json:"conversation_id"`
	Name           string `json:"name"`
	Type           string `json:"type"`
	UpdatedAt      string `json:"updated_at"`
}

type SearchMessageHit struct {
	MessageID      uint64 `json:"message_id"`
	ConversationID uint64 `json:"conversation_id"`
	Conversation   string `json:"conversation_name"`
	SenderID       uint64 `json:"sender_id"`
	MessageType    string `json:"message_type"`
	Content        string `json:"content"`
	CreatedAt      string `json:"created_at"`
}

type SearchResponse struct {
	Conversations []SearchConversationHit `json:"conversations"`
	Messages      []SearchMessageHit      `json:"messages"`
	NextCursor    string                  `json:"next_cursor"`
}

type SyncEventDTO struct {
	EventID     uint64          `json:"event_id"`
	EventType   string          `json:"event_type"`
	AggregateID string          `json:"aggregate_id"`
	Cursor      uint64          `json:"cursor"`
	Type        string          `json:"type"`
	Payload     json.RawMessage `json:"payload"`
	CreatedAt   string          `json:"created_at"`
}

type SyncEventsResponse struct {
	Events        []SyncEventDTO `json:"events"`
	NextCursor    uint64         `json:"next_cursor"`
	CurrentCursor uint64         `json:"current_cursor"`
	HasMore       bool           `json:"has_more"`
}

type SyncCursorResponse struct {
	Cursor uint64 `json:"cursor"`
}
