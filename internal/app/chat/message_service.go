package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"awesomeproject/internal/app/repository"
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
	replyIDs := make([]uint64, 0, len(messages))
	for _, msg := range messages {
		if msg.ReplyToMessageID != nil {
			replyIDs = append(replyIDs, *msg.ReplyToMessageID)
		}
	}
	replyRefs, err := s.messageReferences(replyIDs)
	if err != nil {
		return nil, err
	}
	for index := len(messages) - 1; index >= 0; index-- {
		msg := messages[index]
		dto := messageDTO(&msg)
		if msg.ReplyToMessageID != nil {
			dto.ReplyTo = replyRefs[*msg.ReplyToMessageID]
		}
		result = append(result, dto)
	}
	return result, nil
}

func (s *MessageService) SendMessage(userID, conversationID uint64, messageType, content, clientMsgID string, replyToMessageID *uint64, attachment *contracts.AttachmentDTO) (*contracts.MessageDTO, error) {
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
	if content == "" && attachment == nil {
		return nil, apperr.BadRequest("MESSAGE_CONTENT_REQUIRED", "content is required")
	}
	if replyToMessageID != nil {
		replyMessage, err := s.deps.repo.FindMessageByID(*replyToMessageID)
		if err != nil {
			return nil, apperr.NotFound("REPLY_MESSAGE_NOT_FOUND", "reply message not found")
		}
		if replyMessage.ConversationID != conversationID {
			return nil, apperr.BadRequest("REPLY_MESSAGE_MISMATCH", "reply message must belong to the same conversation")
		}
	}
	if clientMsgID == "" {
		clientMsgID = fmt.Sprintf("%d-%d", conversationID, time.Now().UnixNano())
	}
	dto, created, err := s.persistAndFanoutMessage(userID, conversationID, messageType, content, clientMsgID, replyToMessageID, attachment)
	if err != nil {
		return nil, err
	}
	if created {
		s.startBotReply(userID, conversationID)
	}
	return &dto, nil
}

func (s *MessageService) UpdateTyping(userID, conversationID uint64, isTyping bool) error {
	if _, err := s.deps.repo.FindConversationMember(conversationID, userID); err != nil {
		return apperr.Forbidden("CONVERSATION_MEMBERSHIP_REQUIRED", "not a conversation member")
	}
	memberIDs, err := s.deps.repo.AccessibleConversationIDsForConversation(conversationID)
	if err != nil {
		return err
	}
	recipients := make([]uint64, 0, len(memberIDs))
	for _, memberID := range memberIDs {
		if memberID == userID {
			continue
		}
		recipients = append(recipients, memberID)
	}
	if len(recipients) == 0 || s.deps.rabbit == nil {
		if len(recipients) == 0 {
			return nil
		}
		s.deps.emitSyncEvent(recipients, "typing.updated", conversationAggregateID(conversationID), contracts.TypingUpdatedEvent{
			Recipients:     recipients,
			ConversationID: conversationID,
			UserID:         userID,
			IsTyping:       isTyping,
		})
		return nil
	}
	event := contracts.TypingUpdatedEvent{
		Recipients:     recipients,
		ConversationID: conversationID,
		UserID:         userID,
		IsTyping:       isTyping,
	}
	s.deps.emitSyncEvent(recipients, "typing.updated", conversationAggregateID(conversationID), event)
	return s.deps.publishJSON("typing.updated", event)
}

func (s *MessageService) RecallMessage(userID, messageID uint64) error {
	msg, err := s.deps.repo.FindMessageByID(messageID)
	if err != nil {
		return err
	}
	if msg.SenderID != userID {
		return apperr.Forbidden("FORBIDDEN_MESSAGE_ACTION", "only sender can recall the message")
	}
	if msg.RecalledAt != nil {
		return nil
	}
	updated, err := s.deps.repo.RecallMessage(messageID)
	if err != nil {
		return err
	}
	memberIDs, _ := s.deps.repo.AccessibleConversationIDsForConversation(updated.ConversationID)
	event := contracts.MessageRecalledEvent{
		Recipients:     memberIDs,
		ConversationID: updated.ConversationID,
		MessageID:      updated.ID,
		RecalledAt:     formatTimePtr(updated.RecalledAt),
	}
	if s.deps.rabbit != nil {
		_ = s.deps.publishJSON("message.recalled", event)
	}
	s.deps.emitSyncEvent(memberIDs, "message.recalled", messageAggregateID(updated.ID), event)
	return nil
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
		_ = s.deps.publishJSON("message.read", contracts.ReadReceiptEvent{
			Recipients:     memberIDs,
			ConversationID: conversationID,
			ReaderID:       userID,
			LastReadSeq:    targetSeq,
		})
	}
	s.deps.emitSyncEvent(memberIDs, "message.read", conversationAggregateID(conversationID), contracts.ReadReceiptEvent{
		Recipients:     memberIDs,
		ConversationID: conversationID,
		ReaderID:       userID,
		LastReadSeq:    targetSeq,
	})
	return nil
}

func (s *MessageService) messageReferences(ids []uint64) (map[uint64]*contracts.MessageReferenceDTO, error) {
	rows, err := s.deps.repo.FindMessagesByIDs(uniqueIDs(ids))
	if err != nil {
		return nil, err
	}
	result := make(map[uint64]*contracts.MessageReferenceDTO, len(rows))
	for _, row := range rows {
		dto := messageDTO(&row)
		result[row.ID] = &contracts.MessageReferenceDTO{
			ID:          row.ID,
			SenderID:    row.SenderID,
			MessageType: row.MessageType,
			Content:     row.Content,
			Attachment:  dto.Attachment,
			RecalledAt:  dto.RecalledAt,
		}
	}
	return result, nil
}

func (s *MessageService) applyDeliveryStatus(msg *repository.Message, senderID uint64) *contracts.MessageDeliveryEvent {
	if s.deps.presence == nil {
		return nil
	}
	memberIDs, err := s.deps.repo.AccessibleConversationIDsForConversation(msg.ConversationID)
	if err != nil {
		return nil
	}
	recipients := make([]uint64, 0, len(memberIDs))
	for _, memberID := range memberIDs {
		if memberID == senderID {
			continue
		}
		recipients = append(recipients, memberID)
	}
	if len(recipients) == 0 {
		return nil
	}
	onlineMap, err := s.deps.presence.OnlineMap(context.Background(), recipients)
	if err != nil {
		return nil
	}
	delivered := false
	for _, recipientID := range recipients {
		if onlineMap[recipientID] {
			delivered = true
			break
		}
	}
	if !delivered {
		return nil
	}
	updated, err := s.deps.repo.UpdateMessageDeliveryStatus(msg.ID, "delivered")
	if err != nil {
		return nil
	}
	msg.DeliveryStatus = updated.DeliveryStatus
	msg.Status = updated.Status
	msg.UpdatedAt = updated.UpdatedAt
	return &contracts.MessageDeliveryEvent{
		Recipients:     []uint64{senderID},
		ConversationID: msg.ConversationID,
		MessageID:      msg.ID,
		ClientMsgID:    msg.ClientMsgID,
		DeliveryStatus: updated.DeliveryStatus,
		UpdatedAt:      updated.UpdatedAt.Format(time.RFC3339),
	}
}

func (s *MessageService) persistAndFanoutMessage(senderID, conversationID uint64, messageType, content, clientMsgID string, replyToMessageID *uint64, attachment *contracts.AttachmentDTO) (contracts.MessageDTO, bool, error) {
	attachmentJSON := ""
	if attachment != nil {
		payload, err := json.Marshal(attachment)
		if err != nil {
			return contracts.MessageDTO{}, false, err
		}
		attachmentJSON = string(payload)
	}
	msg, conversation, created, err := s.deps.repo.SendMessage(conversationID, senderID, messageType, content, clientMsgID, replyToMessageID, attachmentJSON)
	if err != nil {
		return contracts.MessageDTO{}, false, err
	}
	deliveryEvent := s.applyDeliveryStatus(msg, senderID)
	if deliveryEvent != nil {
		msg.DeliveryStatus = deliveryEvent.DeliveryStatus
		msg.Status = deliveryEvent.DeliveryStatus
	}
	dto := messageDTO(msg)
	if msg.ReplyToMessageID != nil {
		replyRefs, err := s.messageReferences([]uint64{*msg.ReplyToMessageID})
		if err == nil {
			dto.ReplyTo = replyRefs[*msg.ReplyToMessageID]
		}
	}
	if !created {
		return dto, false, nil
	}
	conversationDTO, err := s.conversations.GetConversationSummary(senderID, conversationID)
	if err != nil {
		return contracts.MessageDTO{}, false, err
	}
	memberIDs, _ := s.deps.repo.AccessibleConversationIDsForConversation(conversationID)
	acceptedEvent := contracts.MessageAcceptedEvent{
		Recipients:     []uint64{senderID},
		ConversationID: conversationID,
		ClientMsgID:    dto.ClientMsgID,
		AcceptedAt:     dto.CreatedAt,
	}
	s.deps.emitSyncEvent([]uint64{senderID}, "message.accepted", messageAggregateID(msg.ID), acceptedEvent)
	if s.deps.rabbit != nil {
		_ = s.deps.publishJSON("message.accepted", acceptedEvent)
		fanoutEvent := contracts.MessageFanoutEvent{
			Recipients:     memberIDs,
			ConversationID: conversationID,
			Message:        dto,
			Conversation:   *conversationDTO,
		}
		_ = s.deps.publishJSON("message.persisted", fanoutEvent)
		messageIndexEvent := contracts.SearchMessageIndexEvent{
			MessageID:        msg.ID,
			ConversationID:   msg.ConversationID,
			ConversationName: conversationDTO.Name,
			SenderID:         msg.SenderID,
			MessageType:      msg.MessageType,
			Content:          msg.Content,
			CreatedAt:        dto.CreatedAt,
		}
		if err := s.deps.publishJSON("search.message.index", messageIndexEvent); err != nil {
			_ = s.deps.indexMessageCompensation(context.Background(), messageIndexEvent)
		}
		conversationIndexEvent := contracts.SearchConversationIndexEvent{
			ConversationID: conversation.ID,
			Name:           conversationDTO.Name,
			Type:           conversation.Type,
			UpdatedAt:      conversation.UpdatedAt.Format(time.RFC3339),
		}
		if err := s.deps.publishJSON("search.conversation.index", conversationIndexEvent); err != nil {
			_ = s.deps.indexConversationCompensation(context.Background(), conversationIndexEvent)
		}
	}
	for _, memberID := range uniqueIDs(memberIDs) {
		summary, err := s.conversations.GetConversationSummary(memberID, conversationID)
		if err != nil {
			continue
		}
		s.deps.emitSyncEvent([]uint64{memberID}, "message.persisted", messageAggregateID(msg.ID), contracts.MessageFanoutEvent{
			Recipients:     []uint64{memberID},
			ConversationID: conversationID,
			Message:        dto,
			Conversation:   *summary,
		})
	}
	if deliveryEvent != nil {
		s.deps.emitSyncEvent([]uint64{senderID}, "message.delivered", messageAggregateID(msg.ID), *deliveryEvent)
		if s.deps.rabbit != nil {
			_ = s.deps.publishJSON("message.delivered", *deliveryEvent)
		}
	}
	return dto, true, nil
}

func (s *MessageService) startBotReply(userID, conversationID uint64) {
	if s.deps.ai == nil {
		return
	}
	isBotConversation, err := s.deps.ai.IsBotConversation(userID, conversationID)
	if err != nil || !isBotConversation {
		return
	}
	go func() {
		timeout, err := s.deps.ai.AsyncTimeout()
		if err != nil {
			timeout = 30 * time.Second
		}
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		var replyErr error
		for attempt := 0; attempt < 3; attempt++ {
			reply, err := s.deps.ai.GenerateReply(ctx, userID, conversationID)
			if err == nil {
				_, _, _ = s.persistAndFanoutMessage(reply.BotUserID, conversationID, "text", reply.Content, fmt.Sprintf("ai-%d-%d", conversationID, time.Now().UnixNano()), nil, nil)
				return
			}
			replyErr = err
			if ctx.Err() != nil {
				break
			}
			time.Sleep(time.Duration(attempt+1) * 300 * time.Millisecond)
		}
		if replyErr != nil {
			_ = s.deps.ai.EnqueueRetryJob(userID, conversationID, replyErr)
		}
	}()
}

func (s *MessageService) EmitInternalMessage(senderID, conversationID uint64, messageType, content string) error {
	_, _, err := s.persistAndFanoutMessage(senderID, conversationID, messageType, content, fmt.Sprintf("internal-%d-%d", conversationID, time.Now().UnixNano()), nil, nil)
	return err
}
