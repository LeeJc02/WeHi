package chat

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/LeeJc02/WeHi/backend/internal/platform/apperr"
	"github.com/LeeJc02/WeHi/backend/pkg/contracts"

	"gorm.io/gorm"
)

type ConversationService struct {
	deps *dependencies
}

func (s *ConversationService) EnsureDirectConversation(userID, targetID uint64) (*contracts.ConversationDTO, error) {
	if userID == targetID {
		return nil, apperr.BadRequest("INVALID_DIRECT_CONVERSATION_TARGET", "cannot create direct conversation with yourself")
	}
	if s.deps.ai != nil {
		if err := s.deps.ai.EnsureBotForUser(userID); err != nil {
			return nil, err
		}
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
	if hasBot, err := s.containsBotMember(memberIDs); err != nil {
		return nil, err
	} else if hasBot {
		return nil, apperr.BadRequest("BOT_GROUP_UNSUPPORTED", "AI Bot cannot be added to a group conversation")
	}
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
	if s.deps.ai != nil {
		if err := s.deps.ai.EnsureBotForUser(userID); err != nil {
			return nil, err
		}
	}
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
			Announcement:       row.Announcement,
			CreatorID:          row.CreatorID,
			MemberCount:        row.MemberCount,
			Pinned:             row.Pinned,
			PinnedAt:           formatTimePtr(row.PinnedAt),
			IsMuted:            row.IsMuted,
			Draft:              row.Draft,
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
			Announcement:       row.Announcement,
			CreatorID:          row.CreatorID,
			MemberCount:        row.MemberCount,
			Pinned:             row.Pinned,
			PinnedAt:           formatTimePtr(row.PinnedAt),
			IsMuted:            row.IsMuted,
			Draft:              row.Draft,
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
	rows, err := s.deps.repo.ListConversationMemberProfiles(conversationID, userID)
	if err != nil {
		return nil, err
	}
	result := make([]contracts.ConversationMemberDTO, 0, len(rows))
	for _, row := range rows {
		result = append(result, contracts.ConversationMemberDTO{
			UserID:      row.UserID,
			Username:    row.Username,
			DisplayName: row.DisplayName,
			AvatarURL:   row.AvatarURL,
			RemarkName:  row.RemarkName,
			Role:        row.Role,
			LastReadSeq: row.LastReadSeq,
			JoinedAt:    row.JoinedAt.Format(time.RFC3339),
			Online:      onlineUsers[row.UserID],
		})
	}
	return result, nil
}

func (s *ConversationService) AddConversationMembers(actorID, conversationID uint64, memberIDs []uint64) error {
	if err := s.requireGroupConversation(conversationID, "adding members"); err != nil {
		return err
	}
	member, err := s.deps.repo.FindConversationMember(conversationID, actorID)
	if err != nil {
		return apperr.Forbidden("CONVERSATION_MEMBERSHIP_REQUIRED", "not a conversation member")
	}
	if member.Role != "owner" && member.Role != "admin" {
		return apperr.Forbidden("FORBIDDEN_CONVERSATION_ACTION", "only owner/admin can invite members")
	}
	memberIDs = uniqueIDs(memberIDs)
	if hasBot, err := s.containsBotMember(memberIDs); err != nil {
		return err
	} else if hasBot {
		return apperr.BadRequest("BOT_GROUP_UNSUPPORTED", "AI Bot cannot be added to a group conversation")
	}
	for _, memberID := range memberIDs {
		if _, err := s.deps.repo.FindUserByID(memberID); err != nil {
			return apperr.NotFound("CONVERSATION_MEMBER_NOT_FOUND", fmt.Sprintf("user %d not found", memberID))
		}
	}
	if err := s.deps.repo.AddConversationMembers(conversationID, memberIDs); err != nil {
		return err
	}
	recipients, _ := s.deps.repo.AccessibleConversationIDsForConversation(conversationID)
	for _, memberID := range uniqueIDs(memberIDs) {
		s.deps.emitSyncEvent(recipients, "member.joined", conversationAggregateID(conversationID), contracts.ConversationMemberEvent{
			ConversationID: conversationID,
			UserID:         memberID,
		})
	}
	s.emitConversationUpsert(conversationID, recipients)
	return nil
}

func (s *ConversationService) RemoveConversationMember(actorID, conversationID, targetID uint64) error {
	if err := s.requireGroupConversation(conversationID, "removing members"); err != nil {
		return err
	}
	member, err := s.deps.repo.FindConversationMember(conversationID, actorID)
	if err != nil {
		return apperr.Forbidden("CONVERSATION_MEMBERSHIP_REQUIRED", "not a conversation member")
	}
	if actorID != targetID && member.Role != "owner" && member.Role != "admin" {
		return apperr.Forbidden("FORBIDDEN_CONVERSATION_ACTION", "only owner/admin can remove members")
	}
	targetMember, err := s.deps.repo.FindConversationMember(conversationID, targetID)
	if err != nil {
		return apperr.NotFound("CONVERSATION_MEMBER_NOT_FOUND", "conversation member not found")
	}
	if targetMember.Role == "owner" {
		members, err := s.deps.repo.ListConversationMembers(conversationID)
		if err != nil {
			return err
		}
		if len(members) == 1 {
			if err := s.deps.repo.DeleteGroupConversation(conversationID, conversationAggregateID(conversationID)); err != nil {
				return err
			}
			s.deps.emitSyncEvent([]uint64{targetID}, "conversation.removed", conversationAggregateID(conversationID), contracts.ConversationRemovedEvent{ConversationID: conversationID})
			return nil
		}
		if len(members) > 1 {
			return apperr.BadRequest("OWNER_TRANSFER_REQUIRED", "owner must transfer ownership before leaving or being removed")
		}
	}
	if err := s.deps.repo.RemoveConversationMember(conversationID, targetID); err != nil {
		return err
	}
	recipients, _ := s.deps.repo.AccessibleConversationIDsForConversation(conversationID)
	s.deps.emitSyncEvent(append(recipients, targetID), "member.left", conversationAggregateID(conversationID), contracts.ConversationMemberEvent{
		ConversationID: conversationID,
		UserID:         targetID,
	})
	s.deps.emitSyncEvent([]uint64{targetID}, "conversation.removed", conversationAggregateID(conversationID), contracts.ConversationRemovedEvent{ConversationID: conversationID})
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
	member, err := s.deps.repo.FindConversationMember(conversationID, userID)
	if err != nil {
		return apperr.Forbidden("CONVERSATION_MEMBERSHIP_REQUIRED", "not a conversation member")
	}
	if member.Role == "owner" {
		members, err := s.deps.repo.ListConversationMembers(conversationID)
		if err != nil {
			return err
		}
		if len(members) == 1 {
			if err := s.deps.repo.DeleteGroupConversation(conversationID, conversationAggregateID(conversationID)); err != nil {
				return err
			}
			s.deps.emitSyncEvent([]uint64{userID}, "conversation.removed", conversationAggregateID(conversationID), contracts.ConversationRemovedEvent{ConversationID: conversationID})
			return nil
		}
		if len(members) > 1 {
			return apperr.BadRequest("OWNER_TRANSFER_REQUIRED", "owner must transfer ownership before leaving or being removed")
		}
	}
	if err := s.deps.repo.RemoveConversationMember(conversationID, userID); err != nil {
		return err
	}
	recipients, _ := s.deps.repo.AccessibleConversationIDsForConversation(conversationID)
	s.deps.emitSyncEvent(append(recipients, userID), "member.left", conversationAggregateID(conversationID), contracts.ConversationMemberEvent{
		ConversationID: conversationID,
		UserID:         userID,
	})
	s.deps.emitSyncEvent([]uint64{userID}, "conversation.removed", conversationAggregateID(conversationID), contracts.ConversationRemovedEvent{ConversationID: conversationID})
	s.emitConversationUpsert(conversationID, recipients)
	return nil
}

func (s *ConversationService) TransferOwnership(actorID, conversationID, targetID uint64) error {
	if err := s.requireGroupConversation(conversationID, "transferring ownership"); err != nil {
		return err
	}
	member, err := s.deps.repo.FindConversationMember(conversationID, actorID)
	if err != nil {
		return apperr.Forbidden("CONVERSATION_MEMBERSHIP_REQUIRED", "not a conversation member")
	}
	if member.Role != "owner" {
		return apperr.Forbidden("FORBIDDEN_CONVERSATION_ACTION", "only owner can transfer ownership")
	}
	if actorID == targetID {
		return apperr.BadRequest("OWNER_TRANSFER_TARGET_INVALID", "ownership must be transferred to another group member")
	}
	targetMember, err := s.deps.repo.FindConversationMember(conversationID, targetID)
	if err != nil {
		return apperr.NotFound("CONVERSATION_MEMBER_NOT_FOUND", "conversation member not found")
	}
	if targetMember.Role == "owner" {
		return apperr.BadRequest("OWNER_TRANSFER_TARGET_INVALID", "ownership must be transferred to another group member")
	}
	if err := s.deps.repo.SetConversationMemberRole(conversationID, actorID, "admin"); err != nil {
		return err
	}
	return s.deps.repo.SetConversationMemberRole(conversationID, targetID, "owner")
}

func (s *ConversationService) UpdateSettings(userID, conversationID uint64, req contracts.UpdateConversationSettingsRequest) (*contracts.ConversationDTO, error) {
	member, err := s.deps.repo.FindConversationMember(conversationID, userID)
	if err != nil {
		return nil, apperr.Forbidden("CONVERSATION_MEMBERSHIP_REQUIRED", "not a conversation member")
	}
	conversation, err := s.deps.repo.FindConversationByID(conversationID)
	if err != nil {
		return nil, err
	}

	if req.Draft != nil {
		draft := strings.TrimSpace(*req.Draft)
		req.Draft = &draft
	}
	if req.Announcement != nil {
		if conversation.Type != "group" {
			return nil, apperr.BadRequest("CONVERSATION_ANNOUNCEMENT_UNSUPPORTED", "announcement is only supported for group conversations")
		}
		if member.Role != "owner" && member.Role != "admin" {
			return nil, apperr.Forbidden("FORBIDDEN_CONVERSATION_ACTION", "only owner/admin can update the announcement")
		}
		announcement := strings.TrimSpace(*req.Announcement)
		req.Announcement = &announcement
	}
	if req.Pinned != nil && !*req.Pinned && s.deps.ai != nil {
		isBotConversation, err := s.deps.ai.IsBotConversation(userID, conversationID)
		if err != nil {
			return nil, err
		}
		if isBotConversation {
			return nil, apperr.BadRequest("BOT_CONVERSATION_PIN_REQUIRED", "AI Bot conversation must stay pinned")
		}
	}

	if err := s.deps.repo.UpdateConversationSettings(userID, conversationID, req.Pinned, req.IsMuted, req.Draft); err != nil {
		return nil, err
	}
	if req.Announcement != nil {
		if err := s.deps.repo.UpdateConversationAnnouncement(conversationID, *req.Announcement); err != nil {
			return nil, err
		}
	}

	recipients := []uint64{userID}
	if req.Announcement != nil {
		recipients, _ = s.deps.repo.AccessibleConversationIDsForConversation(conversationID)
	}
	s.emitConversationUpsert(conversationID, recipients)
	return s.GetConversationSummary(userID, conversationID)
}

func (s *ConversationService) resolveDirectConversationName(userID, conversationID uint64) (string, error) {
	members, err := s.deps.repo.ListConversationMemberProfiles(conversationID, userID)
	if err != nil {
		return "", err
	}
	for _, member := range members {
		if member.UserID != userID {
			if member.RemarkName != "" {
				return member.RemarkName, nil
			}
			if member.DisplayName != "" {
				return member.DisplayName, nil
			}
			return member.Username, nil
		}
	}
	return "Direct chat", nil
}

func (s *ConversationService) publishConversationIndex(dto *contracts.ConversationDTO) {
	event := conversationIndexEvent(dto)
	if err := s.deps.publishJSON("search.conversation.index", event); err != nil {
		_ = s.deps.indexConversationCompensation(context.Background(), event)
	}
}

func (s *ConversationService) containsBotMember(memberIDs []uint64) (bool, error) {
	if s.deps.ai == nil {
		return false, nil
	}
	botUserID, err := s.deps.ai.BotUserID()
	if err != nil || botUserID == 0 {
		return false, err
	}
	for _, memberID := range memberIDs {
		if memberID == botUserID {
			return true, nil
		}
	}
	return false, nil
}

func (s *ConversationService) requireGroupConversation(conversationID uint64, action string) error {
	conversation, err := s.deps.repo.FindConversationByID(conversationID)
	if err != nil {
		return err
	}
	if conversation.Type != "group" {
		return apperr.BadRequest("GROUP_ACTION_UNSUPPORTED", action+" is only supported for group conversations")
	}
	return nil
}
