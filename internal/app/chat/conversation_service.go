package chat

import (
	"fmt"
	"strings"
	"time"

	"awesomeproject/internal/platform/apperr"
	"awesomeproject/pkg/contracts"

	"gorm.io/gorm"
)

type ConversationService struct {
	deps *dependencies
}

func (s *ConversationService) EnsureDirectConversation(userID, targetID uint64) (*contracts.ConversationDTO, error) {
	if userID == targetID {
		return nil, apperr.BadRequest("INVALID_DIRECT_CONVERSATION_TARGET", "cannot create direct conversation with yourself")
	}
	conversation, _, err := s.deps.repo.EnsureDirectConversation(userID, targetID)
	if err != nil {
		return nil, err
	}
	dto, err := s.GetConversationSummary(userID, conversation.ID)
	if err != nil {
		return nil, err
	}
	s.publishConversationIndex(dto)
	s.emitConversationUpsert(dto.ID, []uint64{userID, targetID})
	return dto, nil
}

func (s *ConversationService) CreateGroupConversation(creatorID uint64, name string, memberIDs []uint64) (*contracts.ConversationDTO, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, apperr.BadRequest("GROUP_NAME_REQUIRED", "group name is required")
	}
	memberIDs = append(memberIDs, creatorID)
	memberIDs = uniqueIDs(memberIDs)
	if len(memberIDs) < 3 {
		return nil, apperr.BadRequest("GROUP_MEMBER_COUNT_INVALID", "group requires at least three distinct members")
	}
	for _, memberID := range memberIDs {
		if _, err := s.deps.repo.FindUserByID(memberID); err != nil {
			return nil, apperr.NotFound("GROUP_MEMBER_NOT_FOUND", fmt.Sprintf("user %d not found", memberID))
		}
	}
	conversation, err := s.deps.repo.CreateGroupConversation(creatorID, name, memberIDs)
	if err != nil {
		return nil, err
	}
	dto, err := s.GetConversationSummary(creatorID, conversation.ID)
	if err != nil {
		return nil, err
	}
	s.publishConversationIndex(dto)
	s.emitConversationUpsert(dto.ID, memberIDs)
	return dto, nil
}

func (s *ConversationService) RenameConversation(userID, conversationID uint64, name string) (*contracts.ConversationDTO, error) {
	member, err := s.deps.repo.FindConversationMember(conversationID, userID)
	if err != nil {
		return nil, apperr.Forbidden("CONVERSATION_MEMBERSHIP_REQUIRED", "not a conversation member")
	}
	if member.Role != "owner" && member.Role != "admin" {
		return nil, apperr.Forbidden("FORBIDDEN_CONVERSATION_ACTION", "only owner/admin can rename the conversation")
	}
	if err := s.deps.repo.RenameConversation(conversationID, strings.TrimSpace(name)); err != nil {
		return nil, err
	}
	dto, err := s.GetConversationSummary(userID, conversationID)
	if err != nil {
		return nil, err
	}
	s.publishConversationIndex(dto)
	recipients, _ := s.deps.repo.AccessibleConversationIDsForConversation(conversationID)
	s.emitConversationUpsert(conversationID, recipients)
	return dto, nil
}

func (s *ConversationService) ListConversations(userID uint64) ([]contracts.ConversationDTO, error) {
	rows, err := s.deps.repo.ListConversations(userID)
	if err != nil {
		return nil, err
	}
	result := make([]contracts.ConversationDTO, 0, len(rows))
	for _, row := range rows {
		name := row.Name
		if row.Type == "direct" {
			peerName, err := s.resolveDirectConversationName(userID, row.ID)
			if err == nil && peerName != "" {
				name = peerName
			}
		}
		result = append(result, contracts.ConversationDTO{
			ID:                 row.ID,
			Type:               row.Type,
			Name:               name,
			CreatorID:          row.CreatorID,
			MemberCount:        row.MemberCount,
			Pinned:             row.Pinned,
			LastReadSeq:        row.LastReadSeq,
			UnreadCount:        row.UnreadCount,
			LastMessageSeq:     row.LastMessageSeq,
			LastMessageSender:  row.LastMessageSender,
			LastMessagePreview: row.LastMessagePreview,
			LastMessageType:    row.LastMessageType,
			LastMessageAt:      formatTimePtr(row.LastMessageAt),
		})
	}
	return result, nil
}

func (s *ConversationService) GetConversationSummary(userID, conversationID uint64) (*contracts.ConversationDTO, error) {
	rows, err := s.deps.repo.ListConversations(userID)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		if row.ID != conversationID {
			continue
		}
		name := row.Name
		if row.Type == "direct" {
			name, _ = s.resolveDirectConversationName(userID, row.ID)
		}
		return &contracts.ConversationDTO{
			ID:                 row.ID,
			Type:               row.Type,
			Name:               name,
			CreatorID:          row.CreatorID,
			MemberCount:        row.MemberCount,
			Pinned:             row.Pinned,
			LastReadSeq:        row.LastReadSeq,
			UnreadCount:        row.UnreadCount,
			LastMessageSeq:     row.LastMessageSeq,
			LastMessageSender:  row.LastMessageSender,
			LastMessagePreview: row.LastMessagePreview,
			LastMessageType:    row.LastMessageType,
			LastMessageAt:      formatTimePtr(row.LastMessageAt),
		}, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (s *ConversationService) ListConversationMembers(userID, conversationID uint64, onlineUsers map[uint64]bool) ([]contracts.ConversationMemberDTO, error) {
	if _, err := s.deps.repo.FindConversationMember(conversationID, userID); err != nil {
		return nil, apperr.Forbidden("CONVERSATION_MEMBERSHIP_REQUIRED", "not a conversation member")
	}
	rows, err := s.deps.repo.ListConversationMemberProfiles(conversationID)
	if err != nil {
		return nil, err
	}
	result := make([]contracts.ConversationMemberDTO, 0, len(rows))
	for _, row := range rows {
		result = append(result, contracts.ConversationMemberDTO{
			UserID:      row.UserID,
			Username:    row.Username,
			DisplayName: row.DisplayName,
			Role:        row.Role,
			LastReadSeq: row.LastReadSeq,
			JoinedAt:    row.JoinedAt.Format(time.RFC3339),
			Online:      onlineUsers[row.UserID],
		})
	}
	return result, nil
}

func (s *ConversationService) AddConversationMembers(actorID, conversationID uint64, memberIDs []uint64) error {
	member, err := s.deps.repo.FindConversationMember(conversationID, actorID)
	if err != nil {
		return apperr.Forbidden("CONVERSATION_MEMBERSHIP_REQUIRED", "not a conversation member")
	}
	if member.Role != "owner" && member.Role != "admin" {
		return apperr.Forbidden("FORBIDDEN_CONVERSATION_ACTION", "only owner/admin can invite members")
	}
	memberIDs = uniqueIDs(memberIDs)
	for _, memberID := range memberIDs {
		if _, err := s.deps.repo.FindUserByID(memberID); err != nil {
			return apperr.NotFound("CONVERSATION_MEMBER_NOT_FOUND", fmt.Sprintf("user %d not found", memberID))
		}
	}
	if err := s.deps.repo.AddConversationMembers(conversationID, memberIDs); err != nil {
		return err
	}
	recipients, _ := s.deps.repo.AccessibleConversationIDsForConversation(conversationID)
	s.emitConversationUpsert(conversationID, recipients)
	return nil
}

func (s *ConversationService) RemoveConversationMember(actorID, conversationID, targetID uint64) error {
	member, err := s.deps.repo.FindConversationMember(conversationID, actorID)
	if err != nil {
		return apperr.Forbidden("CONVERSATION_MEMBERSHIP_REQUIRED", "not a conversation member")
	}
	if actorID != targetID && member.Role != "owner" && member.Role != "admin" {
		return apperr.Forbidden("FORBIDDEN_CONVERSATION_ACTION", "only owner/admin can remove members")
	}
	if err := s.deps.repo.RemoveConversationMember(conversationID, targetID); err != nil {
		return err
	}
	s.deps.emitSyncEvent([]uint64{targetID}, "conversation.removed", contracts.ConversationRemovedEvent{ConversationID: conversationID})
	recipients, _ := s.deps.repo.AccessibleConversationIDsForConversation(conversationID)
	s.emitConversationUpsert(conversationID, recipients)
	return nil
}

func (s *ConversationService) LeaveConversation(userID, conversationID uint64) error {
	conversation, err := s.deps.repo.FindConversationByID(conversationID)
	if err != nil {
		return err
	}
	if conversation.Type == "direct" {
		return apperr.BadRequest("DIRECT_CONVERSATION_CANNOT_LEAVE", "direct conversation cannot be left")
	}
	if err := s.deps.repo.RemoveConversationMember(conversationID, userID); err != nil {
		return err
	}
	s.deps.emitSyncEvent([]uint64{userID}, "conversation.removed", contracts.ConversationRemovedEvent{ConversationID: conversationID})
	recipients, _ := s.deps.repo.AccessibleConversationIDsForConversation(conversationID)
	s.emitConversationUpsert(conversationID, recipients)
	return nil
}

func (s *ConversationService) TransferOwnership(actorID, conversationID, targetID uint64) error {
	member, err := s.deps.repo.FindConversationMember(conversationID, actorID)
	if err != nil {
		return apperr.Forbidden("CONVERSATION_MEMBERSHIP_REQUIRED", "not a conversation member")
	}
	if member.Role != "owner" {
		return apperr.Forbidden("FORBIDDEN_CONVERSATION_ACTION", "only owner can transfer ownership")
	}
	if err := s.deps.repo.SetConversationMemberRole(conversationID, actorID, "admin"); err != nil {
		return err
	}
	return s.deps.repo.SetConversationMemberRole(conversationID, targetID, "owner")
}

func (s *ConversationService) SetPinned(userID, conversationID uint64, pinned bool) error {
	if _, err := s.deps.repo.FindConversationMember(conversationID, userID); err != nil {
		return apperr.Forbidden("CONVERSATION_MEMBERSHIP_REQUIRED", "not a conversation member")
	}
	return s.deps.repo.SetPin(userID, conversationID, pinned)
}

func (s *ConversationService) resolveDirectConversationName(userID, conversationID uint64) (string, error) {
	members, err := s.deps.repo.ListConversationMemberProfiles(conversationID)
	if err != nil {
		return "", err
	}
	for _, member := range members {
		if member.UserID != userID {
			if member.DisplayName != "" {
				return member.DisplayName, nil
			}
			return member.Username, nil
		}
	}
	return "Direct chat", nil
}

func (s *ConversationService) publishConversationIndex(dto *contracts.ConversationDTO) {
	if s.deps.rabbit == nil {
		return
	}
	_ = s.deps.rabbit.PublishJSON("search.conversation.index", conversationIndexEvent(dto))
}
