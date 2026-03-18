package apperr

import (
	"errors"
	"testing"
)

func TestFromReturnsAppError(t *testing.T) {
	input := Conflict("CONFLICT", "conflict happened")

	got := From(input)

	if got != input {
		t.Fatalf("expected same app error instance")
	}
}

func TestFromWrapsGenericError(t *testing.T) {
	got := From(errors.New("boom"))

	if got.Code != "INTERNAL_ERROR" {
		t.Fatalf("expected INTERNAL_ERROR, got %s", got.Code)
	}
	if got.Status != 500 {
		t.Fatalf("expected status 500, got %d", got.Status)
	}
	if got.Message != "boom" {
		t.Fatalf("expected original message, got %s", got.Message)
	}
}
