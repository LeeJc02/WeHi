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

type SetConversationPinRequest struct {
	Pinned bool `json:"pinned"`
}

type SendMessageRequest struct {
	MessageType string `json:"message_type"`
	Content     string `json:"content"`
	ClientMsgID string `json:"client_msg_id"`
}

type MarkReadRequest struct {
	Seq uint64 `json:"seq"`
}
