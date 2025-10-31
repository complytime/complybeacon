package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func TestRequestID_EchoHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestID())
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-Id", "test-req-echo")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if got := w.Header().Get("X-Request-Id"); got != "test-req-echo" {
		t.Fatalf("expected X-Request-Id to echo, got %q", got)
	}
}

func TestRequestID_GenerateWhenMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestID())
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	rid := w.Header().Get("X-Request-Id")
	if rid == "" {
		t.Fatal("expected generated X-Request-Id header")
	}
	if _, err := uuid.Parse(rid); err != nil {
		t.Fatalf("expected UUID X-Request-Id, got %q: %v", rid, err)
	}
}
