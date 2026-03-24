package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/LeeJc02/WeHi/backend/internal/app/repository"
	"github.com/LeeJc02/WeHi/backend/pkg/contracts"
)

type PresenceChecker interface {
	OnlineMap(ctx context.Context, userIDs []uint64) (map[uint64]bool, error)
}

type DiagnosticsService struct {
	repo     *repository.Repository
	presence PresenceChecker
}

func NewDiagnosticsService(repo *repository.Repository, presence PresenceChecker) *DiagnosticsService {
	return &DiagnosticsService{repo: repo, presence: presence}
}

func (s *DiagnosticsService) MessageJourney(messageID uint64) (*contracts.MessageJourney, error) {
	message, err := s.repo.FindMessageByID(messageID)
	if err != nil {
		return nil, err
	}
	messageEvents, err := s.repo.ListSyncEventsByAggregate(fmt.Sprintf("message:%d", messageID), 500)
	if err != nil {
		return nil, err
	}
	conversationEvents, err := s.repo.ListSyncEventsByAggregate(fmt.Sprintf("conversation:%d", message.ConversationID), 500)
	if err != nil {
		return nil, err
	}
	stages := []contracts.MessageJourneyStage{
		{Name: "accepted", OccurredAt: message.CreatedAt.Format(timeLayout), Note: "API 接收消息"},
		{Name: "persisted", OccurredAt: message.CreatedAt.Format(timeLayout), Note: "消息已落库"},
	}
	for _, row := range messageEvents {
		switch row.EventType {
		case "message.delivered":
			var payload contracts.MessageDeliveryEvent
			if json.Unmarshal([]byte(row.Payload), &payload) == nil && payload.MessageID == messageID {
				stages = append(stages, contracts.MessageJourneyStage{
					Name:        "delivered",
					OccurredAt:  row.CreatedAt.Format(timeLayout),
					RecipientID: payload.Recipients[0],
					Note:        payload.DeliveryStatus,
				})
			}
		case "message.recalled":
			stages = append(stages, contracts.MessageJourneyStage{
				Name:       "recalled",
				OccurredAt: row.CreatedAt.Format(timeLayout),
				Note:       "消息已撤回",
			})
		}
	}
	for _, row := range conversationEvents {
		if row.EventType != "message.read" {
			continue
		}
		var payload contracts.ReadReceiptEvent
		if json.Unmarshal([]byte(row.Payload), &payload) != nil {
			continue
		}
		if payload.LastReadSeq < message.Seq {
			continue
		}
		stages = append(stages, contracts.MessageJourneyStage{
			Name:        "read",
			OccurredAt:  row.CreatedAt.Format(timeLayout),
			RecipientID: payload.ReaderID,
			Note:        fmt.Sprintf("读至 seq %d", payload.LastReadSeq),
		})
	}
	sort.SliceStable(stages, func(i, j int) bool {
		return stages[i].OccurredAt < stages[j].OccurredAt
	})
	return &contracts.MessageJourney{
		MessageID:      message.ID,
		ConversationID: message.ConversationID,
		ClientMsgID:    message.ClientMsgID,
		SenderID:       message.SenderID,
		MessageType:    message.MessageType,
		DeliveryStatus: message.DeliveryStatus,
		CreatedAt:      message.CreatedAt.Format(timeLayout),
		RecalledAt:     formatTimePtr(message.RecalledAt),
		Stages:         stages,
	}, nil
}

func (s *DiagnosticsService) ResolveMessageByClientMsgID(clientMsgID string, senderID, conversationID uint64) (*contracts.MessageLookupResult, error) {
	message, err := s.repo.FindMessageByClientMsgIDForAdmin(clientMsgID, senderID, conversationID)
	if err != nil {
		return nil, err
	}
	return &contracts.MessageLookupResult{
		MessageID:      message.ID,
		ConversationID: message.ConversationID,
		SenderID:       message.SenderID,
		ClientMsgID:    message.ClientMsgID,
	}, nil
}

func (s *DiagnosticsService) ConversationConsistency(conversationID uint64) (*contracts.ConversationConsistency, error) {
	conversation, err := s.repo.FindConversationByID(conversationID)
	if err != nil {
		return nil, err
	}
	rows, err := s.repo.ListConversationReadStates(conversationID)
	if err != nil {
		return nil, err
	}
	userIDs := make([]uint64, 0, len(rows))
	for _, row := range rows {
		userIDs = append(userIDs, row.UserID)
	}
	onlineMap := map[uint64]bool{}
	if s.presence != nil && len(userIDs) > 0 {
		onlineMap, _ = s.presence.OnlineMap(context.Background(), userIDs)
	}
	currentCursor := uint64(0)
	members := make([]contracts.ConversationConsistencyMember, 0, len(rows))
	onlineCount := 0
	for _, row := range rows {
		if row.CurrentCursor > currentCursor {
			currentCursor = row.CurrentCursor
		}
		if onlineMap[row.UserID] {
			onlineCount++
		}
		members = append(members, contracts.ConversationConsistencyMember{
			UserID:        row.UserID,
			Username:      row.Username,
			DisplayName:   row.DisplayName,
			AvatarURL:     row.AvatarURL,
			Role:          row.Role,
			LastReadSeq:   row.LastReadSeq,
			UnreadCount:   row.UnreadCount,
			CurrentCursor: row.CurrentCursor,
			Online:        onlineMap[row.UserID],
		})
	}
	return &contracts.ConversationConsistency{
		ConversationID:  conversationID,
		LastMessageSeq:  conversation.LastMessageSeq,
		LastMessageAt:   formatTimePtr(conversation.LastMessageAt),
		OnlineCount:     onlineCount,
		CurrentEventLag: currentCursor,
		Members:         members,
	}, nil
}

func (s *DiagnosticsService) ConversationEvents(conversationID uint64, limit int) ([]contracts.SyncEventDTO, error) {
	rows, err := s.repo.ListSyncEventsByAggregate(fmt.Sprintf("conversation:%d", conversationID), limit)
	if err != nil {
		return nil, err
	}
	result := make([]contracts.SyncEventDTO, 0, len(rows))
	for _, row := range rows {
		result = append(result, contracts.SyncEventDTO{
			EventID:     row.ID,
			EventType:   row.EventType,
			AggregateID: row.AggregateID,
			Cursor:      row.ID,
			Type:        row.EventType,
			Payload:     []byte(row.Payload),
			CreatedAt:   row.CreatedAt.Format(timeLayout),
		})
	}
	return result, nil
}

const timeLayout = "2006-01-02T15:04:05Z07:00"

func formatTimePtr(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.Format(timeLayout)
}
