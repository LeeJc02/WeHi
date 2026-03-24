package chat

import (
	"context"
	"time"

	"github.com/LeeJc02/WeHi/backend/internal/app/ai"
	"github.com/LeeJc02/WeHi/backend/internal/app/repository"
	"github.com/LeeJc02/WeHi/backend/internal/platform/rabbit"
	"github.com/LeeJc02/WeHi/backend/internal/platform/search"
	"github.com/LeeJc02/WeHi/backend/pkg/contracts"
)

type PresenceChecker interface {
	OnlineMap(ctx context.Context, userIDs []uint64) (map[uint64]bool, error)
}

type dependencies struct {
	repo               *repository.Repository
	rabbit             *rabbit.Client
	search             *search.Client
	presence           PresenceChecker
	ai                 *ai.Service
	messagesIndex      string
	conversationsIndex string
}

func (d *dependencies) publishJSON(routingKey string, payload any) error {
	if d.rabbit == nil {
		return nil
	}
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if err := d.rabbit.PublishJSON(routingKey, payload); err == nil {
			return nil
		} else {
			lastErr = err
		}
		time.Sleep(time.Duration(attempt+1) * 100 * time.Millisecond)
	}
	return lastErr
}

func (d *dependencies) indexMessageCompensation(ctx context.Context, event contracts.SearchMessageIndexEvent) error {
	if d.search == nil || d.search.IsMock() {
		return nil
	}
	return d.search.IndexDocument(ctx, d.messagesIndex, event.MessageID, contracts.SearchMessageHit{
		MessageID:      event.MessageID,
		ConversationID: event.ConversationID,
		Conversation:   event.ConversationName,
		SenderID:       event.SenderID,
		MessageType:    event.MessageType,
		Content:        event.Content,
		CreatedAt:      event.CreatedAt,
	})
}

func (d *dependencies) indexConversationCompensation(ctx context.Context, event contracts.SearchConversationIndexEvent) error {
	if d.search == nil || d.search.IsMock() {
		return nil
	}
	return d.search.IndexDocument(ctx, d.conversationsIndex, event.ConversationID, contracts.SearchConversationHit{
		ConversationID: event.ConversationID,
		Name:           event.Name,
		Type:           event.Type,
		UpdatedAt:      event.UpdatedAt,
	})
}

type Services struct {
	User         *UserService
	Friend       *FriendService
	Conversation *ConversationService
	Message      *MessageService
	Search       *SearchService
}

func NewServices(repo *repository.Repository, rabbitClient *rabbit.Client, searchClient *search.Client, presenceChecker PresenceChecker, aiService *ai.Service, messagesIndex, conversationsIndex string) *Services {
	deps := &dependencies{
		repo:               repo,
		rabbit:             rabbitClient,
		search:             searchClient,
		presence:           presenceChecker,
		ai:                 aiService,
		messagesIndex:      messagesIndex,
		conversationsIndex: conversationsIndex,
	}
	conversationService := &ConversationService{deps: deps}
	return &Services{
		User:         &UserService{deps: deps},
		Friend:       &FriendService{deps: deps},
		Conversation: conversationService,
		Message:      &MessageService{deps: deps, conversations: conversationService},
		Search:       &SearchService{deps: deps},
	}
}
