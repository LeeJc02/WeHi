package chat

import (
	"sort"
	"strings"
	"time"

	"awesomeproject/internal/app/repository"
	"awesomeproject/internal/platform/apperr"
	"awesomeproject/pkg/contracts"
)

type FriendService struct {
	deps *dependencies
}

func (s *FriendService) CreateFriendRequest(userID, addresseeID uint64, message string) (*contracts.FriendRequestDTO, error) {
	if userID == addresseeID || userID == 0 || addresseeID == 0 {
		return nil, apperr.BadRequest("INVALID_FRIEND_REQUEST_TARGET", "invalid friend request target")
	}
	if _, err := s.deps.repo.FindUserByID(addresseeID); err != nil {
		return nil, apperr.NotFound("TARGET_USER_NOT_FOUND", "target user not found")
	}
	if exists, err := s.deps.repo.FriendshipExists(userID, addresseeID); err != nil {
		return nil, err
	} else if exists {
		return nil, apperr.Conflict("FRIENDSHIP_ALREADY_EXISTS", "friendship already exists")
	}
	if pending, err := s.deps.repo.FindPendingFriendRequest(userID, addresseeID); err == nil && pending != nil {
		return nil, apperr.Conflict("FRIEND_REQUEST_ALREADY_PENDING", "pending friend request already exists")
	}
	now := time.Now()
	request := &repository.FriendRequest{
		RequesterID: userID,
		AddresseeID: addresseeID,
		Message:     strings.TrimSpace(message),
		Status:      "pending",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.deps.repo.CreateFriendRequest(request); err != nil {
		return nil, err
	}
	dto, err := s.inflateFriendRequest(request, userID)
	if err != nil {
		return nil, err
	}
	if requesterView, err := s.inflateFriendRequest(request, userID); err == nil {
		s.deps.emitSyncEvent([]uint64{userID}, "friend.request", friendRequestAggregateID(request.ID), contracts.FriendRequestEvent{
			Recipients: []uint64{userID},
			Request:    *requesterView,
		})
	}
	addresseeView, addresseeViewErr := s.inflateFriendRequest(request, addresseeID)
	if addresseeViewErr == nil {
		s.deps.emitSyncEvent([]uint64{addresseeID}, "friend.request", friendRequestAggregateID(request.ID), contracts.FriendRequestEvent{
			Recipients: []uint64{addresseeID},
			Request:    *addresseeView,
		})
	}
	if s.deps.rabbit != nil {
		if addresseeViewErr == nil {
			_ = s.deps.publishJSON("friend.request", contracts.FriendRequestEvent{
				Recipients: []uint64{addresseeID},
				Request:    *addresseeView,
			})
		}
	}
	return dto, nil
}

func (s *FriendService) ListFriendRequests(userID uint64) ([]contracts.FriendRequestDTO, error) {
	requests, err := s.deps.repo.ListFriendRequests(userID)
	if err != nil {
		return nil, err
	}
	result := make([]contracts.FriendRequestDTO, 0, len(requests))
	for _, request := range requests {
		dto, err := s.inflateFriendRequest(&request, userID)
		if err != nil {
			return nil, err
		}
		result = append(result, *dto)
	}
	return result, nil
}

func (s *FriendService) ApproveFriendRequest(userID, requestID uint64) error {
	request, err := s.deps.repo.FindFriendRequestByID(requestID)
	if err != nil {
		return err
	}
	if request.AddresseeID != userID {
		return apperr.Forbidden("FORBIDDEN_FRIEND_REQUEST_ACTION", "cannot approve another user's friend request")
	}
	if request.Status != "pending" {
		return apperr.Conflict("FRIEND_REQUEST_NOT_PENDING", "friend request is no longer pending")
	}
	if err := s.deps.repo.AcceptFriendRequest(request); err != nil {
		return err
	}
	request.Status = "accepted"
	request.UpdatedAt = time.Now()
	if requesterView, err := s.inflateFriendRequest(request, request.RequesterID); err == nil {
		s.deps.emitSyncEvent([]uint64{request.RequesterID}, "friend.request", friendRequestAggregateID(request.ID), contracts.FriendRequestEvent{
			Recipients: []uint64{request.RequesterID},
			Request:    *requesterView,
		})
		if s.deps.rabbit != nil {
			_ = s.deps.publishJSON("friend.request", contracts.FriendRequestEvent{
				Recipients: []uint64{request.RequesterID},
				Request:    *requesterView,
			})
		}
	}
	if addresseeView, err := s.inflateFriendRequest(request, request.AddresseeID); err == nil {
		s.deps.emitSyncEvent([]uint64{request.AddresseeID}, "friend.request", friendRequestAggregateID(request.ID), contracts.FriendRequestEvent{
			Recipients: []uint64{request.AddresseeID},
			Request:    *addresseeView,
		})
		if s.deps.rabbit != nil {
			_ = s.deps.publishJSON("friend.request", contracts.FriendRequestEvent{
				Recipients: []uint64{request.AddresseeID},
				Request:    *addresseeView,
			})
		}
	}
	return nil
}

func (s *FriendService) RejectFriendRequest(userID, requestID uint64) error {
	request, err := s.deps.repo.FindFriendRequestByID(requestID)
	if err != nil {
		return err
	}
	if request.AddresseeID != userID {
		return apperr.Forbidden("FORBIDDEN_FRIEND_REQUEST_ACTION", "cannot reject another user's friend request")
	}
	if request.Status != "pending" {
		return apperr.Conflict("FRIEND_REQUEST_NOT_PENDING", "friend request is no longer pending")
	}
	if err := s.deps.repo.UpdateFriendRequestStatus(request, "rejected"); err != nil {
		return err
	}
	request.Status = "rejected"
	request.UpdatedAt = time.Now()
	if requesterView, err := s.inflateFriendRequest(request, request.RequesterID); err == nil {
		s.deps.emitSyncEvent([]uint64{request.RequesterID}, "friend.request", friendRequestAggregateID(request.ID), contracts.FriendRequestEvent{
			Recipients: []uint64{request.RequesterID},
			Request:    *requesterView,
		})
		if s.deps.rabbit != nil {
			_ = s.deps.publishJSON("friend.request", contracts.FriendRequestEvent{
				Recipients: []uint64{request.RequesterID},
				Request:    *requesterView,
			})
		}
	}
	if addresseeView, err := s.inflateFriendRequest(request, request.AddresseeID); err == nil {
		s.deps.emitSyncEvent([]uint64{request.AddresseeID}, "friend.request", friendRequestAggregateID(request.ID), contracts.FriendRequestEvent{
			Recipients: []uint64{request.AddresseeID},
			Request:    *addresseeView,
		})
		if s.deps.rabbit != nil {
			_ = s.deps.publishJSON("friend.request", contracts.FriendRequestEvent{
				Recipients: []uint64{request.AddresseeID},
				Request:    *addresseeView,
			})
		}
	}
	return nil
}

func (s *FriendService) ListFriends(userID uint64) ([]contracts.FriendDTO, error) {
	if s.deps.ai != nil {
		if err := s.deps.ai.EnsureBotForUser(userID); err != nil {
			return nil, err
		}
	}
	friends, err := s.deps.repo.ListFriends(userID)
	if err != nil {
		return nil, err
	}
	if s.deps.ai != nil {
		botUserID, err := s.deps.ai.BotUserID()
		if err != nil {
			return nil, err
		}
		for index := range friends {
			friends[index].IsAIBot = friends[index].ID == botUserID
		}
		sort.SliceStable(friends, func(i, j int) bool {
			if friends[i].IsAIBot != friends[j].IsAIBot {
				return friends[i].IsAIBot
			}
			return false
		})
	}
	if friends == nil {
		return []contracts.FriendDTO{}, nil
	}
	return friends, nil
}

func (s *FriendService) UpdateRemark(userID, friendID uint64, remarkName string) error {
	remarkName = strings.TrimSpace(remarkName)
	if _, err := s.deps.repo.FindUserByID(friendID); err != nil {
		return apperr.NotFound("TARGET_USER_NOT_FOUND", "target user not found")
	}
	exists, err := s.deps.repo.FriendshipExists(userID, friendID)
	if err != nil {
		return err
	}
	if !exists {
		return apperr.Forbidden("FRIENDSHIP_REQUIRED", "friendship required")
	}
	return s.deps.repo.UpdateFriendRemark(userID, friendID, remarkName)
}

func (s *FriendService) inflateFriendRequest(request *repository.FriendRequest, userID uint64) (*contracts.FriendRequestDTO, error) {
	users, err := s.deps.repo.FindUserProfiles([]uint64{request.RequesterID, request.AddresseeID})
	if err != nil {
		return nil, err
	}
	direction := "incoming"
	if request.RequesterID == userID {
		direction = "outgoing"
	}
	return &contracts.FriendRequestDTO{
		ID:        request.ID,
		Status:    request.Status,
		Direction: direction,
		Message:   request.Message,
		Requester: contracts.UserProfile{
			ID:          users[request.RequesterID].ID,
			Username:    users[request.RequesterID].Username,
			DisplayName: users[request.RequesterID].DisplayName,
			AvatarURL:   users[request.RequesterID].AvatarURL,
		},
		Addressee: contracts.UserProfile{
			ID:          users[request.AddresseeID].ID,
			Username:    users[request.AddresseeID].Username,
			DisplayName: users[request.AddresseeID].DisplayName,
			AvatarURL:   users[request.AddresseeID].AvatarURL,
		},
		CreatedAt: request.CreatedAt.Format(time.RFC3339),
		UpdatedAt: request.UpdatedAt.Format(time.RFC3339),
	}, nil
}
