package contracts

type EventEnvelope struct {
	Type    string `json:"type"`
	Payload any    `json:"payload"`
}

type SyncNotifyEvent struct {
	Recipients []uint64 `json:"recipients"`
}

type MessageFanoutEvent struct {
	Recipients     []uint64        `json:"recipients"`
	ConversationID uint64          `json:"conversation_id"`
	Message        MessageDTO      `json:"message"`
	Conversation   ConversationDTO `json:"conversation"`
}

type MessageAcceptedEvent struct {
	Recipients     []uint64 `json:"recipients"`
	ConversationID uint64   `json:"conversation_id"`
	ClientMsgID    string   `json:"client_msg_id"`
	AcceptedAt     string   `json:"accepted_at"`
}

type MessageDeliveryEvent struct {
	Recipients     []uint64 `json:"recipients"`
	ConversationID uint64   `json:"conversation_id"`
	MessageID      uint64   `json:"message_id"`
	ClientMsgID    string   `json:"client_msg_id"`
	DeliveryStatus string   `json:"delivery_status"`
	UpdatedAt      string   `json:"updated_at"`
}

type ReadReceiptEvent struct {
	Recipients     []uint64 `json:"recipients"`
	ConversationID uint64   `json:"conversation_id"`
	ReaderID       uint64   `json:"reader_id"`
	LastReadSeq    uint64   `json:"last_read_seq"`
}

type MessageRecalledEvent struct {
	Recipients     []uint64 `json:"recipients"`
	ConversationID uint64   `json:"conversation_id"`
	MessageID      uint64   `json:"message_id"`
	RecalledAt     string   `json:"recalled_at"`
}

type TypingUpdatedEvent struct {
	Recipients     []uint64 `json:"recipients"`
	ConversationID uint64   `json:"conversation_id"`
	UserID         uint64   `json:"user_id"`
	IsTyping       bool     `json:"is_typing"`
}

type FriendRequestEvent struct {
	Recipients []uint64         `json:"recipients"`
	Request    FriendRequestDTO `json:"request"`
}

type ConversationSyncEvent struct {
	Conversation ConversationDTO `json:"conversation"`
}

type ConversationRemovedEvent struct {
	ConversationID uint64 `json:"conversation_id"`
}

type ConversationMemberEvent struct {
	ConversationID uint64 `json:"conversation_id"`
	UserID         uint64 `json:"user_id"`
}

type SearchMessageIndexEvent struct {
	MessageID        uint64 `json:"message_id"`
	ConversationID   uint64 `json:"conversation_id"`
	ConversationName string `json:"conversation_name"`
	SenderID         uint64 `json:"sender_id"`
	MessageType      string `json:"message_type"`
	Content          string `json:"content"`
	CreatedAt        string `json:"created_at"`
}

type SearchConversationIndexEvent struct {
	ConversationID uint64 `json:"conversation_id"`
	Name           string `json:"name"`
	Type           string `json:"type"`
	UpdatedAt      string `json:"updated_at"`
}
