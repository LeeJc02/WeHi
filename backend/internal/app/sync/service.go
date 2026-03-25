package sync

import (
	"github.com/LeeJc02/WeHi/backend/internal/app/repository"
	"github.com/LeeJc02/WeHi/backend/pkg/contracts"
)

type Service struct {
	repo *repository.Repository
}

func NewService(repo *repository.Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) CurrentCursor(userID uint64) (*contracts.SyncCursorResponse, error) {
	cursor, err := s.repo.CurrentSyncCursor(userID)
	if err != nil {
		return nil, err
	}
	return &contracts.SyncCursorResponse{Cursor: cursor}, nil
}

func (s *Service) ListEvents(userID, cursor uint64, limit int) (*contracts.SyncEventsResponse, error) {
	// The sync API returns both the page boundary and the user's current head so
	// clients can tell whether they have fully caught up after replaying events.
	rows, currentCursor, err := s.repo.ListSyncEvents(userID, cursor, limit)
	if err != nil {
		return nil, err
	}
	events := make([]contracts.SyncEventDTO, 0, len(rows))
	nextCursor := cursor
	for _, row := range rows {
		nextCursor = row.ID
		events = append(events, contracts.SyncEventDTO{
			EventID:     row.ID,
			EventType:   row.EventType,
			AggregateID: row.AggregateID,
			Cursor:      row.ID,
			Type:        row.EventType,
			Payload:     []byte(row.Payload),
			CreatedAt:   row.CreatedAt.Format(timeLayout),
		})
	}
	return &contracts.SyncEventsResponse{
		Events:        events,
		NextCursor:    nextCursor,
		CurrentCursor: currentCursor,
		HasMore:       nextCursor < currentCursor,
	}, nil
}

const timeLayout = "2006-01-02T15:04:05Z07:00"
