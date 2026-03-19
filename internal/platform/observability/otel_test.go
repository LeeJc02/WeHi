package observability

import (
	"context"
	"testing"

	"awesomeproject/internal/config"
)

func TestInit_DoesNotConflictWithDefaultResourceSchema(t *testing.T) {
	t.Setenv("OTEL_RESOURCE_ATTRIBUTES", "deployment.environment=test")

	shutdown, err := Init(context.Background(), config.Config{
		ServiceName: "observability-test",
		OTELExporter: "none",
	})
	if err != nil {
		t.Fatalf("init observability: %v", err)
	}
	t.Cleanup(func() {
		if shutdown != nil {
			_ = shutdown(context.Background())
		}
	})
}
