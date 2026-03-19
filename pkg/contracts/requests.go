package contracts

type RegisterRequest struct {
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Password    string `json:"password"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type UpdateProfileRequest struct {
	DisplayName string `json:"display_name"`
	AvatarURL   string `json:"avatar_url"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type LogoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type CreateFriendRequestRequest struct {
	AddresseeID uint64 `json:"addressee_id"`
	Message     string `json:"message"`
}

type UpdateFriendRemarkRequest struct {
	RemarkName string `json:"remark_name"`
}

type CreateDirectConversationRequest struct {
	TargetUserID uint64 `json:"target_user_id"`
}

type CreateGroupConversationRequest struct {
	Name      string   `json:"name"`
	MemberIDs []uint64 `json:"member_ids"`
}

type RenameConversationRequest struct {
	Name string `json:"name"`
}

type AddConversationMembersRequest struct {
	MemberIDs []uint64 `json:"member_ids"`
}

type TransferOwnershipRequest struct {
	UserID uint64 `json:"user_id"`
}

type UpdateConversationSettingsRequest struct {
	Pinned       *bool   `json:"pinned"`
	IsMuted      *bool   `json:"is_muted"`
	Draft        *string `json:"draft"`
	Announcement *string `json:"announcement"`
}

type SendMessageRequest struct {
	MessageType      string         `json:"message_type"`
	Content          string         `json:"content"`
	ClientMsgID      string         `json:"client_msg_id"`
	ReplyToMessageID *uint64        `json:"reply_to_message_id"`
	Attachment       *AttachmentDTO `json:"attachment"`
}

type MarkReadRequest struct {
	Seq uint64 `json:"seq"`
}

type TypingStatusRequest struct {
	IsTyping bool `json:"is_typing"`
}
