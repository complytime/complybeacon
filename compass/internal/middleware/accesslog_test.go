package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
)

type captureHandler struct {
	mu      sync.Mutex
	records []slog.Record
}

func (h *captureHandler) Enabled(_ context.Context, _ slog.Level) bool { return true }

func (h *captureHandler) Handle(_ context.Context, r slog.Record) error {
	// Copy record to make attrs accessible later
	copied := slog.Record{Time: r.Time, Message: r.Message, Level: r.Level, PC: r.PC}
	r.Attrs(func(a slog.Attr) bool {
		copied.AddAttrs(a)
		return true
	})
	h.mu.Lock()
	h.records = append(h.records, copied)
	h.mu.Unlock()
	return nil
}

func (h *captureHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	// For tests, keep it simple: attach attrs by appending during Handle.
	return h
}

func (h *captureHandler) WithGroup(name string) slog.Handler { return h }

func TestAccessLogger_EmitsRecord(t *testing.T) {
	gin.SetMode(gin.TestMode)

	ch := &captureHandler{}
	prev := slog.Default()
	slog.SetDefault(slog.New(ch))
	t.Cleanup(func() { slog.SetDefault(prev) })

	r := gin.New()
	r.Use(RequestID(), AccessLogger())
	r.GET("/hello", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	req := httptest.NewRequest(http.MethodGet, "/hello", nil)
	req.Header.Set("X-Request-Id", "accesslog-test")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	ch.mu.Lock()
	defer ch.mu.Unlock()
	if len(ch.records) == 0 {
		t.Fatal("expected at least one log record")
	}

	// Find the http_request record
	var found *slog.Record
	for i := range ch.records {
		if ch.records[i].Message == "http_request" {
			found = &ch.records[i]
			break
		}
	}
	if found == nil {
		t.Fatalf("expected http_request log record, got %d records", len(ch.records))
	}

	// Extract attrs to a map for assertions
	got := map[string]any{}
	found.Attrs(func(a slog.Attr) bool { got[a.Key] = a.Value.Any(); return true })

	if got["request_id"] != "accesslog-test" {
		t.Errorf("request_id mismatch: %v", got["request_id"])
	}
	if got["method"] != http.MethodGet {
		t.Errorf("method mismatch: %v", got["method"])
	}
	if got["path"] != "/hello" {
		t.Errorf("path mismatch: %v", got["path"])
	}
	if _, ok := got["status"]; !ok {
		t.Errorf("missing status attr")
	}
}
