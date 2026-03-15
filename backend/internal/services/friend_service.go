package services

import (
	"errors"

	"awesomeproject/backend/internal/models"
	"awesomeproject/backend/internal/repositories"
)

type FriendService struct {
	users   *repositories.UserRepository
	friends *repositories.FriendRepository
}

func NewFriendService(users *repositories.UserRepository, friends *repositories.FriendRepository) *FriendService {
	return &FriendService{users: users, friends: friends}
}

func (s *FriendService) AddFriend(userID, friendID uint64) error {
	if userID == 0 || friendID == 0 {
		return errors.New("user_id and friend_id are required")
	}
	if userID == friendID {
		return errors.New("cannot add yourself")
	}
	if _, err := s.users.FindByID(friendID); err != nil {
		return errors.New("friend user not found")
	}
	if exists, err := s.friends.Exists(userID, friendID); err != nil {
		return err
	} else if exists {
		return errors.New("friendship already exists")
	}
	return s.friends.CreatePair(userID, friendID)
}

func (s *FriendService) ListFriends(userID uint64) ([]models.FriendDTO, error) {
	rows, err := s.friends.ListByUser(userID)
	if err != nil {
		return nil, err
	}

	result := make([]models.FriendDTO, 0, len(rows))
	for _, row := range rows {
		user, err := s.users.FindByID(row.FriendID)
		if err != nil {
			return nil, err
		}
		result = append(result, models.FriendDTO{
			ID:          user.ID,
			Username:    user.Username,
			DisplayName: user.DisplayName,
		})
	}
	return result, nil
}
