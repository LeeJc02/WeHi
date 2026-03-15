package services

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"awesomeproject/backend/internal/models"
	"awesomeproject/backend/internal/repositories"

	"gorm.io/gorm"
)

type ConversationService struct {
	users         *repositories.UserRepository
	conversations *repositories.ConversationRepository
	messages      *repositories.MessageRepository
}

func NewConversationService(
	users *repositories.UserRepository,
	conversations *repositories.ConversationRepository,
	messages *repositories.MessageRepository,
) *ConversationService {
	return &ConversationService{
		users:         users,
		conversations: conversations,
		messages:      messages,
	}
}

func (s *ConversationService) EnsureDirectConversation(userID, targetUserID uint64) (*models.Conversation, error) {
	if userID == 0 || targetUserID == 0 {
		return nil, errors.New("user ids are required")
	}
	if userID == targetUserID {
		return nil, errors.New("cannot create direct conversation with yourself")
	}
	if _, err := s.users.FindByID(targetUserID); err != nil {
		return nil, errors.New("target user not found")
	}

	conversation, err := s.conversations.FindDirect(userID, targetUserID)
	if err == nil {
		return conversation, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	conversation = &models.Conversation{
		Type:      "direct",
		Name:      fmt.Sprintf("direct-%d-%d", min(userID, targetUserID), max(userID, targetUserID)),
		CreatorID: userID,
	}
	members := []models.ConversationMember{
		{UserID: userID, Role: "owner"},
		{UserID: targetUserID, Role: "member"},
	}
	if err := s.conversations.CreateConversation(conversation, members); err != nil {
		return nil, err
	}
	return conversation, nil
}

func (s *ConversationService) CreateGroupConversation(creatorID uint64, name string, memberIDs []uint64) (*models.Conversation, error) {
	name = strings.TrimSpace(name)
	if creatorID == 0 || name == "" {
		return nil, errors.New("creator_id and name are required")
	}

	memberIDs = append(memberIDs, creatorID)
	memberIDs = uniqueIDs(memberIDs)
	if len(memberIDs) < 3 {
		return nil, errors.New("group conversation requires at least three distinct members")
	}

	for _, memberID := range memberIDs {
		if _, err := s.users.FindByID(memberID); err != nil {
			return nil, fmt.Errorf("user %d not found", memberID)
		}
	}

	conversation := &models.Conversation{
		Type:      "group",
		Name:      name,
		CreatorID: creatorID,
	}
	members := make([]models.ConversationMember, 0, len(memberIDs))
	for _, memberID := range memberIDs {
		role := "member"
		if memberID == creatorID {
			role = "owner"
		}
		members = append(members, models.ConversationMember{UserID: memberID, Role: role})
	}
	if err := s.conversations.CreateConversation(conversation, members); err != nil {
		return nil, err
	}
	return conversation, nil
}

func (s *ConversationService) ListConversations(userID uint64) ([]models.ConversationDTO, error) {
	conversations, err := s.conversations.ListByUser(userID)
	if err != nil {
		return nil, err
	}

	result := make([]models.ConversationDTO, 0, len(conversations))
	for _, conversation := range conversations {
		members, err := s.conversations.ListMembers(conversation.ID)
		if err != nil {
			return nil, err
		}
		member, err := s.conversations.FindMember(conversation.ID, userID)
		if err != nil {
			return nil, err
		}
		unreadCount, err := s.messages.CountUnread(conversation.ID, userID, member.LastReadMessageID)
		if err != nil {
			return nil, err
		}

		dto := models.ConversationDTO{
			ID:          conversation.ID,
			Type:        conversation.Type,
			Name:        conversation.Name,
			CreatorID:   conversation.CreatorID,
			MemberCount: len(members),
			UnreadCount: unreadCount,
		}

		latest, err := s.messages.FindLatest(conversation.ID)
		if err == nil {
			dto.LastMessageID = latest.ID
			dto.LastMessageSender = latest.SenderID
			dto.LastMessagePreview = latest.Content
			dto.LastMessageAt = latest.CreatedAt.Format("2006-01-02 15:04:05")
		}
		result = append(result, dto)
	}
	return result, nil
}

func (s *ConversationService) SendMessage(userID, conversationID uint64, content string) (*models.Message, error) {
	content = strings.TrimSpace(content)
	if conversationID == 0 || content == "" {
		return nil, errors.New("conversation_id and content are required")
	}
	member, err := s.conversations.FindMember(conversationID, userID)
	if err != nil || member == nil {
		return nil, errors.New("current user is not a conversation member")
	}

	message := &models.Message{
		ConversationID: conversationID,
		SenderID:       userID,
		Content:        content,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	if err := s.messages.Create(message); err != nil {
		return nil, err
	}
	if err := s.conversations.TouchConversation(conversationID); err != nil {
		return nil, err
	}
	return message, nil
}

func (s *ConversationService) ListMessages(userID, conversationID uint64, limit int) ([]models.Message, error) {
	if _, err := s.conversations.FindMember(conversationID, userID); err != nil {
		return nil, errors.New("current user is not a conversation member")
	}
	return s.messages.ListByConversation(conversationID, limit)
}

func (s *ConversationService) MarkRead(userID, conversationID uint64) error {
	member, err := s.conversations.FindMember(conversationID, userID)
	if err != nil {
		return errors.New("current user is not a conversation member")
	}
	latest, err := s.messages.FindLatest(conversationID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if latest != nil {
		member.LastReadMessageID = latest.ID
	}
	return s.conversations.SaveMember(member)
}

func uniqueIDs(values []uint64) []uint64 {
	result := make([]uint64, 0, len(values))
	for _, value := range values {
		if value == 0 || slices.Contains(result, value) {
			continue
		}
		result = append(result, value)
	}
	return result
}

func min(a, b uint64) uint64 {
	if a < b {
		return a
	}
	return b
}

func max(a, b uint64) uint64 {
	if a > b {
		return a
	}
	return b
}
