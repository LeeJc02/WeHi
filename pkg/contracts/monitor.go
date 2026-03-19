package contracts

type MonitorServiceStatus struct {
	Name      string `json:"name"`
	Healthy   bool   `json:"healthy"`
	Status    string `json:"status"`
	Error     string `json:"error,omitempty"`
	CheckedAt string `json:"checked_at"`
}

type MonitorOverview struct {
	Services             []MonitorServiceStatus `json:"services"`
	TotalRequests        float64                `json:"total_requests"`
	ClientErrors         float64                `json:"client_errors"`
	ServerErrors         float64                `json:"server_errors"`
	AverageLatencyMs     float64                `json:"average_latency_ms"`
	WebSocketConnections float64                `json:"websocket_connections"`
	AIRetryPending       float64                `json:"ai_retry_pending"`
	AIRetryCompleted     float64                `json:"ai_retry_completed"`
	AIRetryExhausted     float64                `json:"ai_retry_exhausted"`
	SnapshotAt           string                 `json:"snapshot_at"`
}

type MonitorPoint struct {
	Timestamp            string  `json:"timestamp"`
	TotalRequests        float64 `json:"total_requests"`
	ClientErrors         float64 `json:"client_errors"`
	ServerErrors         float64 `json:"server_errors"`
	AverageLatencyMs     float64 `json:"average_latency_ms"`
	WebSocketConnections float64 `json:"websocket_connections"`
	AIRetryPending       float64 `json:"ai_retry_pending"`
	AIRetryCompleted     float64 `json:"ai_retry_completed"`
	AIRetryExhausted     float64 `json:"ai_retry_exhausted"`
}

type MonitorTimeseries struct {
	Points []MonitorPoint `json:"points"`
}

type MessageJourneyStage struct {
	Name        string `json:"name"`
	OccurredAt  string `json:"occurred_at"`
	RecipientID uint64 `json:"recipient_id"`
	Note        string `json:"note"`
}

type MessageJourney struct {
	MessageID      uint64                `json:"message_id"`
	ConversationID uint64                `json:"conversation_id"`
	ClientMsgID    string                `json:"client_msg_id"`
	SenderID       uint64                `json:"sender_id"`
	MessageType    string                `json:"message_type"`
	DeliveryStatus string                `json:"delivery_status"`
	CreatedAt      string                `json:"created_at"`
	RecalledAt     string                `json:"recalled_at"`
	Stages         []MessageJourneyStage `json:"stages"`
}

type MessageLookupResult struct {
	MessageID      uint64 `json:"message_id"`
	ConversationID uint64 `json:"conversation_id"`
	SenderID       uint64 `json:"sender_id"`
	ClientMsgID    string `json:"client_msg_id"`
}

type ConversationConsistencyMember struct {
	UserID        uint64 `json:"user_id"`
	Username      string `json:"username"`
	DisplayName   string `json:"display_name"`
	AvatarURL     string `json:"avatar_url"`
	Role          string `json:"role"`
	LastReadSeq   uint64 `json:"last_read_seq"`
	UnreadCount   int64  `json:"unread_count"`
	CurrentCursor uint64 `json:"current_cursor"`
	Online        bool   `json:"online"`
}

type ConversationConsistency struct {
	ConversationID  uint64                          `json:"conversation_id"`
	LastMessageSeq  uint64                          `json:"last_message_seq"`
	LastMessageAt   string                          `json:"last_message_at"`
	OnlineCount     int                             `json:"online_count"`
	CurrentEventLag uint64                          `json:"current_event_lag"`
	Members         []ConversationConsistencyMember `json:"members"`
}
