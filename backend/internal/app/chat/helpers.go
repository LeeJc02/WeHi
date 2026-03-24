package chat

import (
	"encoding/json"
	"sort"
	"time"

	"github.com/LeeJc02/WeHi/backend/internal/app/repository"
	"github.com/LeeJc02/WeHi/backend/pkg/contracts"
)

func messageDTO(msg *repository.Message) contracts.MessageDTO {
	var attachment *contracts.AttachmentDTO
	if msg.AttachmentJSON != "" {
		var parsed contracts.AttachmentDTO
		if err := json.Unmarshal([]byte(msg.AttachmentJSON), &parsed); err == nil {
			attachment = &parsed
		}
	}
	return contracts.MessageDTO{
		ID:               msg.ID,
		ConversationID:   msg.ConversationID,
		Seq:              msg.Seq,
		SenderID:         msg.SenderID,
		MessageType:      msg.MessageType,
		Content:          msg.Content,
		ReplyToMessageID: msg.ReplyToMessageID,
		Attachment:       attachment,
		ClientMsgID:      msg.ClientMsgID,
		Status:           msg.Status,
		DeliveryStatus:   msg.DeliveryStatus,
		CreatedAt:        msg.CreatedAt.Format(time.RFC3339),
		RecalledAt:       formatTimePtr(msg.RecalledAt),
	}
}

func conversationIndexEvent(dto *contracts.ConversationDTO) contracts.SearchConversationIndexEvent {
	updatedAt := dto.LastMessageAt
	if updatedAt == "" {
		updatedAt = time.Now().Format(time.RFC3339)
	}
	return contracts.SearchConversationIndexEvent{
		ConversationID: dto.ID,
		Name:           dto.Name,
		Type:           dto.Type,
		UpdatedAt:      updatedAt,
	}
}

func formatTimePtr(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.Format(time.RFC3339)
}

func uniqueIDs(values []uint64) []uint64 {
	seen := map[uint64]struct{}{}
	result := make([]uint64, 0, len(values))
	for _, value := range values {
		if value == 0 {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}
