package chat

import "awesomeproject/pkg/contracts"

func (d *dependencies) emitSyncEvent(userIDs []uint64, eventType string, payload any) {
	userIDs = uniqueIDs(userIDs)
	if len(userIDs) == 0 {
		return
	}
	if d.repo != nil {
		_ = d.repo.AppendSyncEvents(userIDs, eventType, payload)
	}
	if d.rabbit != nil {
		_ = d.rabbit.PublishJSON("sync.notify", contracts.SyncNotifyEvent{Recipients: userIDs})
	}
}

func (s *ConversationService) emitConversationUpsert(conversationID uint64, userIDs []uint64) {
	for _, userID := range uniqueIDs(userIDs) {
		dto, err := s.GetConversationSummary(userID, conversationID)
		if err != nil {
			continue
		}
		s.deps.emitSyncEvent([]uint64{userID}, "conversation.upsert", contracts.ConversationSyncEvent{Conversation: *dto})
	}
}
