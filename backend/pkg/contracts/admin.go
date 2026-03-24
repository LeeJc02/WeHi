package contracts

type AdminProfile struct {
	ID                 uint64 `json:"id"`
	Username           string `json:"username"`
	MustChangePassword bool   `json:"must_change_password"`
}

type AdminAuthPayload struct {
	AccessToken string       `json:"access_token"`
	Admin       AdminProfile `json:"admin"`
}

type AdminLoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type AdminChangePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}
