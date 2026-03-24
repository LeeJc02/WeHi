package chat

import (
	"strings"

	"github.com/LeeJc02/WeHi/backend/internal/platform/apperr"
	"github.com/LeeJc02/WeHi/backend/pkg/contracts"
)

type UserService struct {
	deps *dependencies
}

func (s *UserService) ListUsers(currentUserID uint64) ([]contracts.UserProfile, error) {
	users, err := s.deps.repo.ListUsers(currentUserID)
	if err != nil {
		return nil, err
	}
	botUserID := uint64(0)
	if s.deps.ai != nil {
		botUserID, err = s.deps.ai.BotUserID()
		if err != nil {
			return nil, err
		}
	}
	result := make([]contracts.UserProfile, 0, len(users))
	for _, user := range users {
		if botUserID > 0 && user.ID == botUserID {
			continue
		}
		result = append(result, contracts.UserProfile{
			ID:          user.ID,
			Username:    user.Username,
			DisplayName: user.DisplayName,
			AvatarURL:   user.AvatarURL,
		})
	}
	return result, nil
}

func (s *UserService) UpdateProfile(userID uint64, displayName, avatarURL string) (*contracts.UserProfile, error) {
	displayName = strings.TrimSpace(displayName)
	if len(displayName) < 2 {
		return nil, apperr.BadRequest("INVALID_DISPLAY_NAME", "display_name must be at least 2 characters")
	}
	avatarURL = strings.TrimSpace(avatarURL)
	if err := s.deps.repo.UpdateUserProfile(userID, displayName, avatarURL); err != nil {
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
		AvatarURL:   user.AvatarURL,
	}
	s.deps.emitSyncEvent([]uint64{userID}, "profile.updated", userAggregateID(userID), profile)
	return profile, nil
}
