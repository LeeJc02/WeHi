package httpx

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/LeeJc02/WeHi/backend/internal/platform/apperr"
	"github.com/LeeJc02/WeHi/backend/pkg/contracts"

	"github.com/gin-gonic/gin"
)

func TestFailErrorWritesEnvelope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)

	FailError(ctx, apperr.NotFound("NOT_FOUND", "missing"))

	var envelope contracts.Envelope
	if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if recorder.Code != 404 {
		t.Fatalf("expected status 404, got %d", recorder.Code)
	}
	if envelope.ErrorCode != "NOT_FOUND" {
		t.Fatalf("expected error code NOT_FOUND, got %s", envelope.ErrorCode)
	}
	if envelope.Message != "missing" {
		t.Fatalf("expected message missing, got %s", envelope.Message)
	}
}

func TestRequestIDSetsResponseHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest("GET", "/health", nil)

	RequestID()(ctx)

	if recorder.Header().Get("X-Request-Id") == "" {
		t.Fatal("expected X-Request-Id header to be set")
	}
}
