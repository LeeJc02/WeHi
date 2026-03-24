package chat

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/LeeJc02/WeHi/backend/internal/platform/apperr"
	"github.com/LeeJc02/WeHi/backend/pkg/contracts"
)

type SearchService struct {
	deps *dependencies
}

func (s *SearchService) Search(ctx context.Context, userID uint64, query, scope, cursor string, limit int, conversationID uint64) (*contracts.SearchResponse, error) {
	if s.deps.search == nil {
		return nil, apperr.BadGateway("SEARCH_UNAVAILABLE", "search is not configured")
	}
	ids, err := s.deps.repo.AccessibleConversationIDs(userID)
	if err != nil {
		return nil, err
	}
	if conversationID > 0 {
		allowed := false
		for _, id := range ids {
			if id == conversationID {
				allowed = true
				break
			}
		}
		if !allowed {
			return nil, apperr.Forbidden("CONVERSATION_MEMBERSHIP_REQUIRED", "not a conversation member")
		}
		ids = []uint64{conversationID}
	}
	if len(ids) == 0 {
		return &contracts.SearchResponse{
			Conversations: []contracts.SearchConversationHit{},
			Messages:      []contracts.SearchMessageHit{},
		}, nil
	}
	offset, _ := strconv.Atoi(cursor)
	if limit <= 0 || limit > 20 {
		limit = 10
	}
	scope = strings.TrimSpace(scope)
	if scope == "" {
		scope = "all"
	}
	response := &contracts.SearchResponse{
		Conversations: []contracts.SearchConversationHit{},
		Messages:      []contracts.SearchMessageHit{},
	}
	if scope == "all" || scope == "messages" {
		var hits []contracts.SearchMessageHit
		if s.deps.search.IsMock() {
			hits, err = s.deps.repo.SearchMessages(ids, query, offset, limit)
		} else {
			hits, err = s.deps.search.SearchMessages(ctx, s.deps.messagesIndex, query, ids, offset, limit)
		}
		if err != nil {
			return nil, err
		}
		response.Messages = hits
	}
	if scope == "all" || scope == "conversations" {
		var hits []contracts.SearchConversationHit
		if s.deps.search.IsMock() {
			hits, err = s.deps.repo.SearchConversations(ids, query, offset, limit)
		} else {
			hits, err = s.deps.search.SearchConversations(ctx, s.deps.conversationsIndex, query, ids, offset, limit)
		}
		if err != nil {
			return nil, err
		}
		response.Conversations = hits
	}
	response.NextCursor = strconv.Itoa(offset + limit)
	return response, nil
}

func (s *SearchService) Reindex(ctx context.Context) error {
	if s.deps.search == nil || s.deps.search.IsMock() {
		return apperr.BadGateway("SEARCH_UNAVAILABLE", "search is not configured")
	}
	conversations, err := s.deps.repo.ListConversationsForReindex()
	if err != nil {
		return err
	}
	for _, conversation := range conversations {
		if err := s.deps.search.IndexDocument(ctx, s.deps.conversationsIndex, conversation.ID, contracts.SearchConversationIndexEvent{
			ConversationID: conversation.ID,
			Name:           conversation.Name,
			Type:           conversation.Type,
			UpdatedAt:      conversation.UpdatedAt.Format(time.RFC3339),
		}); err != nil {
			return err
		}
	}
	conversationNames := make(map[uint64]string, len(conversations))
	for _, conversation := range conversations {
		conversationNames[conversation.ID] = conversation.Name
	}
	messages, err := s.deps.repo.ListMessagesForReindex()
	if err != nil {
		return err
	}
	for _, message := range messages {
		if err := s.deps.search.IndexDocument(ctx, s.deps.messagesIndex, message.ID, contracts.SearchMessageIndexEvent{
			MessageID:        message.ID,
			ConversationID:   message.ConversationID,
			ConversationName: conversationNames[message.ConversationID],
			SenderID:         message.SenderID,
			MessageType:      message.MessageType,
			Content:          message.Content,
			CreatedAt:        message.CreatedAt.Format(time.RFC3339),
		}); err != nil {
			return err
		}
	}
	return nil
}

func (s *SearchService) IndexMessageEvent(ctx context.Context, event contracts.SearchMessageIndexEvent) error {
	if s.deps.search == nil || s.deps.search.IsMock() {
		return nil
	}
	return s.deps.search.IndexDocument(ctx, s.deps.messagesIndex, event.MessageID, contracts.SearchMessageHit{
		MessageID:      event.MessageID,
		ConversationID: event.ConversationID,
		Conversation:   event.ConversationName,
		SenderID:       event.SenderID,
		MessageType:    event.MessageType,
		Content:        event.Content,
		CreatedAt:      event.CreatedAt,
	})
}

func (s *SearchService) IndexConversationEvent(ctx context.Context, event contracts.SearchConversationIndexEvent) error {
	if s.deps.search == nil || s.deps.search.IsMock() {
		return nil
	}
	return s.deps.search.IndexDocument(ctx, s.deps.conversationsIndex, event.ConversationID, contracts.SearchConversationHit{
		ConversationID: event.ConversationID,
		Name:           event.Name,
		Type:           event.Type,
		UpdatedAt:      event.UpdatedAt,
	})
}
