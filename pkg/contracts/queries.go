package contracts

type ListMessagesQuery struct {
	Cursor string `form:"cursor"`
	Limit  int    `form:"limit"`
}

type SearchQuery struct {
	Q              string `form:"q"`
	Scope          string `form:"scope"`
	Cursor         string `form:"cursor"`
	Limit          int    `form:"limit"`
	ConversationID uint64 `form:"conversation_id"`
}

type SyncEventsQuery struct {
	Cursor uint64 `form:"cursor"`
	Limit  int    `form:"limit"`
}
