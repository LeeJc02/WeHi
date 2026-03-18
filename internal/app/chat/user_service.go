package chat

import (
	"strings"

	"awesomeproject/internal/platform/apperr"
	"awesomeproject/pkg/contracts"
)

type UserService struct {
	deps *dependencies
}

func (s *UserService) ListUsers(currentUserID uint64) ([]contracts.UserProfile, error) {
	users, err := s.deps.repo.ListUsers(currentUserID)
	if err != nil {
		return nil, err
	}
	result := make([]contracts.UserProfile, 0, len(users))
	for _, user := range users {
		result = append(result, contracts.UserProfile{
			ID:          user.ID,
			Username:    user.Username,
			DisplayName: user.DisplayName,
		})
	}
	return result, nil
}

func (s *UserService) UpdateProfile(userID uint64, displayName string) (*contracts.UserProfile, error) {
	displayName = strings.TrimSpace(displayName)
	if len(displayName) < 2 {
		return nil, apperr.BadRequest("INVALID_DISPLAY_NAME", "display_name must be at least 2 characters")
	}
	if err := s.deps.repo.UpdateUserDisplayName(userID, displayName); err != nil {
		return nil, err
	}
	user, err := s.deps.repo.FindUserByID(userID)
	if err != nil {
		return nil, err
	}
	profile := &contracts.UserProfile{
		ID:          user.ID,
		Username:    user.Username,
		DisplayName: user.DisplayName,
	}
	s.deps.emitSyncEvent([]uint64{userID}, "profile.updated", profile)
	return profile, nil
}
