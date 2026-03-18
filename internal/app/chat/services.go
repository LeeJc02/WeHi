package chat

import (
	"awesomeproject/internal/app/repository"
	"awesomeproject/internal/platform/rabbit"
	"awesomeproject/internal/platform/search"
)

type dependencies struct {
	repo               *repository.Repository
	rabbit             *rabbit.Client
	search             *search.Client
	messagesIndex      string
	conversationsIndex string
}

type Services struct {
	User         *UserService
	Friend       *FriendService
	Conversation *ConversationService
	Message      *MessageService
	Search       *SearchService
}

func NewServices(repo *repository.Repository, rabbitClient *rabbit.Client, searchClient *search.Client, messagesIndex, conversationsIndex string) *Services {
	deps := &dependencies{
		repo:               repo,
		rabbit:             rabbitClient,
		search:             searchClient,
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
