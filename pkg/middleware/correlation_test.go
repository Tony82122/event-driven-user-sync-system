package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestCorrelationIDMiddleware_GeneratesID(t *testing.T) {
	r := gin.New()
	r.Use(CorrelationID())
	r.GET("/test", func(c *gin.Context) {
		id := GetCorrelationID(c)
		c.String(http.StatusOK, id)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	// Should have set the header
	corrID := w.Header().Get(CorrelationIDHeader)
	if corrID == "" {
		t.Fatal("expected X-Correlation-ID header to be set")
	}

	// Body should match header
	if w.Body.String() != corrID {
		t.Errorf("body %q does not match header %q", w.Body.String(), corrID)
	}
}

func TestCorrelationIDMiddleware_UsesExistingID(t *testing.T) {
	r := gin.New()
	r.Use(CorrelationID())
	r.GET("/test", func(c *gin.Context) {
		id := GetCorrelationID(c)
		c.String(http.StatusOK, id)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set(CorrelationIDHeader, "my-custom-id")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	corrID := w.Header().Get(CorrelationIDHeader)
	if corrID != "my-custom-id" {
		t.Errorf("expected %q, got %q", "my-custom-id", corrID)
	}

	if w.Body.String() != "my-custom-id" {
		t.Errorf("body: expected %q, got %q", "my-custom-id", w.Body.String())
	}
}

func TestGetCorrelationID_NoContext(t *testing.T) {
	// When called with a context that has no correlation ID set,
	// it should generate a new UUID
	r := gin.New()
	r.GET("/test", func(c *gin.Context) {
		// No middleware, so no correlation ID in context
		id := GetCorrelationID(c)
		if id == "" {
			t.Error("expected a generated UUID, got empty string")
		}
		c.String(http.StatusOK, id)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
}
