package chat

import (
	"testing"

	"github.com/LeeJc02/WeHi/backend/pkg/contracts"
)

func TestUniqueIDsFiltersZeroAndDuplicates(t *testing.T) {
	got := uniqueIDs([]uint64{3, 0, 2, 3, 1, 2})

	want := []uint64{1, 2, 3}
	if len(got) != len(want) {
		t.Fatalf("expected %d ids, got %d", len(want), len(got))
	}
	for idx := range want {
		if got[idx] != want[idx] {
			t.Fatalf("expected %v, got %v", want, got)
		}
	}
}

func TestConversationIndexEventFallsBackToCurrentTime(t *testing.T) {
	event := conversationIndexEvent(&contracts.ConversationDTO{
		ID:   7,
		Name: "Team Chat",
		Type: "group",
	})

	if event.ConversationID != 7 {
		t.Fatalf("expected conversation id 7, got %d", event.ConversationID)
	}
	if event.Name != "Team Chat" {
		t.Fatalf("expected name Team Chat, got %s", event.Name)
	}
	if event.UpdatedAt == "" {
		t.Fatal("expected updated_at to be populated")
	}
}
