package chat

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"awesomeproject/internal/platform/apperr"
	"awesomeproject/pkg/contracts"
)

type MessageService struct {
	deps          *dependencies
	conversations *ConversationService
}

func (s *MessageService) ListMessages(userID, conversationID uint64, cursor string, limit int) ([]contracts.MessageDTO, error) {
	if _, err := s.deps.repo.FindConversationMember(conversationID, userID); err != nil {
		return nil, apperr.Forbidden("CONVERSATION_MEMBERSHIP_REQUIRED", "not a conversation member")
	}
	var beforeSeq uint64
	if cursor != "" {
		value, err := strconv.ParseUint(cursor, 10, 64)
		if err != nil {
			return nil, apperr.BadRequest("INVALID_CURSOR", "invalid cursor")
		}
		beforeSeq = value
	}
	messages, err := s.deps.repo.ListMessages(conversationID, beforeSeq, limit)
	if err != nil {
		return nil, err
	}
	result := make([]contracts.MessageDTO, 0, len(messages))
	for index := len(messages) - 1; index >= 0; index-- {
		msg := messages[index]
		result = append(result, messageDTO(&msg))
	}
	return result, nil
}

func (s *MessageService) SendMessage(userID, conversationID uint64, messageType, content, clientMsgID string) (*contracts.MessageDTO, error) {
	if _, err := s.deps.repo.FindConversationMember(conversationID, userID); err != nil {
		return nil, apperr.Forbidden("CONVERSATION_MEMBERSHIP_REQUIRED", "not a conversation member")
	}
	messageType = strings.TrimSpace(messageType)
	if messageType == "" {
		messageType = "text"
	}
	if messageType != "text" && messageType != "system" && messageType != "image" && messageType != "file" {
		return nil, apperr.BadRequest("UNSUPPORTED_MESSAGE_TYPE", "unsupported message_type")
	}
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, apperr.BadRequest("MESSAGE_CONTENT_REQUIRED", "content is required")
	}
	if clientMsgID == "" {
		clientMsgID = fmt.Sprintf("%d-%d", conversationID, time.Now().UnixNano())
	}
	msg, conversation, err := s.deps.repo.SendMessage(conversationID, userID, messageType, content, clientMsgID)
	if err != nil {
		return nil, err
	}
	dto := messageDTO(msg)
	conversationDTO, err := s.conversations.GetConversationSummary(userID, conversationID)
	if err != nil {
		return nil, err
	}
	memberIDs, _ := s.deps.repo.AccessibleConversationIDsForConversation(conversationID)
	if s.deps.rabbit != nil {
		_ = s.deps.rabbit.PublishJSON("message.new", contracts.MessageFanoutEvent{
			Recipients:     memberIDs,
			ConversationID: conversationID,
			Message:        dto,
			Conversation:   *conversationDTO,
		})
		_ = s.deps.rabbit.PublishJSON("search.message.index", contracts.SearchMessageIndexEvent{
			MessageID:        msg.ID,
			ConversationID:   msg.ConversationID,
			ConversationName: conversationDTO.Name,
			SenderID:         msg.SenderID,
			MessageType:      msg.MessageType,
			Content:          msg.Content,
			CreatedAt:        dto.CreatedAt,
		})
		_ = s.deps.rabbit.PublishJSON("search.conversation.index", contracts.SearchConversationIndexEvent{
			ConversationID: conversation.ID,
			Name:           conversationDTO.Name,
			Type:           conversation.Type,
			UpdatedAt:      conversation.UpdatedAt.Format(time.RFC3339),
		})
	}
	for _, memberID := range uniqueIDs(memberIDs) {
		summary, err := s.conversations.GetConversationSummary(memberID, conversationID)
		if err != nil {
			continue
		}
		s.deps.emitSyncEvent([]uint64{memberID}, "message.new", contracts.MessageFanoutEvent{
			Recipients:     []uint64{memberID},
			ConversationID: conversationID,
			Message:        dto,
			Conversation:   *summary,
		})
	}
	return &dto, nil
}

func (s *MessageService) MarkRead(userID, conversationID uint64, seq uint64) error {
	if _, err := s.deps.repo.FindConversationMember(conversationID, userID); err != nil {
		return apperr.Forbidden("CONVERSATION_MEMBERSHIP_REQUIRED", "not a conversation member")
	}
	targetSeq := seq
	if targetSeq == 0 {
		conversation, err := s.deps.repo.FindConversationByID(conversationID)
		if err != nil {
			return err
		}
		targetSeq = conversation.LastMessageSeq
	}
	if err := s.deps.repo.MarkRead(conversationID, userID, targetSeq); err != nil {
		return err
	}
	memberIDs, _ := s.deps.repo.AccessibleConversationIDsForConversation(conversationID)
	if s.deps.rabbit != nil {
		_ = s.deps.rabbit.PublishJSON("conversation.read", contracts.ReadReceiptEvent{
			Recipients:     memberIDs,
			ConversationID: conversationID,
			ReaderID:       userID,
			LastReadSeq:    targetSeq,
		})
	}
	s.deps.emitSyncEvent(memberIDs, "conversation.read", contracts.ReadReceiptEvent{
		Recipients:     memberIDs,
		ConversationID: conversationID,
		ReaderID:       userID,
		LastReadSeq:    targetSeq,
	})
	return nil
}
