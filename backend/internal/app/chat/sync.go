package chat

import (
	"strconv"

	"github.com/LeeJc02/WeHi/backend/pkg/contracts"
)

// emitSyncEvent persists a user-scoped replay event and nudges online clients to
// pull the delta stream so websocket delivery and offline compensation converge.
func (d *dependencies) emitSyncEvent(userIDs []uint64, eventType, aggregateID string, payload any) {
	userIDs = uniqueIDs(userIDs)
	if len(userIDs) == 0 {
		return
	}
	if d.repo != nil {
		_ = d.repo.AppendSyncEvents(userIDs, eventType, aggregateID, payload)
	}
	if d.rabbit != nil {
		_ = d.publishJSON("sync.notify", contracts.SyncNotifyEvent{Recipients: userIDs})
	}
}

// emitConversationUpsert rebuilds each recipient's conversation summary so the
// sync stream always carries data in the receiver's own permission context.
func (s *ConversationService) emitConversationUpsert(conversationID uint64, userIDs []uint64) {
	for _, userID := range uniqueIDs(userIDs) {
		dto, err := s.GetConversationSummary(userID, conversationID)
		if err != nil {
			continue
		}
		s.deps.emitSyncEvent([]uint64{userID}, "conversation.updated", conversationAggregateID(conversationID), contracts.ConversationSyncEvent{Conversation: *dto})
	}
}

func conversationAggregateID(conversationID uint64) string {
	return "conversation:" + formatUint(conversationID)
}

func messageAggregateID(messageID uint64) string {
	return "message:" + formatUint(messageID)
}

func friendRequestAggregateID(requestID uint64) string {
	return "friend_request:" + formatUint(requestID)
}

func userAggregateID(userID uint64) string {
	return "user:" + formatUint(userID)
}

func formatUint(value uint64) string {
	return strconv.FormatUint(value, 10)
}
