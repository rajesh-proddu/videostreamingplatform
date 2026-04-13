package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/yourusername/videostreamingplatform/utils/observability"
)

func testLogger() *observability.Logger {
	return observability.NewLogger("test")
}

func TestLoggingMiddleware_LogsRequestAndResponse(t *testing.T) {
	t.Parallel()

	called := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	logger := testLogger()
	wrapped := LoggingMiddleware(logger, handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if !called {
		t.Error("handler was not called")
	}
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if rec.Body.String() != "ok" {
		t.Errorf("body = %q, want %q", rec.Body.String(), "ok")
	}
}

func TestErrorHandlingMiddleware_RecoversPanic(t *testing.T) {
	t.Parallel()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("something went wrong")
	})

	logger := testLogger()
	wrapped := ErrorHandlingMiddleware(logger, handler)

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
	if rec.Body.Len() == 0 {
		t.Error("response body should not be empty")
	}
}

func TestErrorHandlingMiddleware_NoPanic(t *testing.T) {
	t.Parallel()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("all good"))
	})

	logger := testLogger()
	wrapped := ErrorHandlingMiddleware(logger, handler)

	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if rec.Body.String() != "all good" {
		t.Errorf("body = %q, want %q", rec.Body.String(), "all good")
	}
}

func TestChainMiddleware_Order(t *testing.T) {
	t.Parallel()

	var order []string

	mw1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "mw1-before")
			next.ServeHTTP(w, r)
			order = append(order, "mw1-after")
		})
	}

	mw2 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "mw2-before")
			next.ServeHTTP(w, r)
			order = append(order, "mw2-after")
		})
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		order = append(order, "handler")
	})

	chained := ChainMiddleware(handler, mw1, mw2)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	chained.ServeHTTP(rec, req)

	expected := []string{"mw1-before", "mw2-before", "handler", "mw2-after", "mw1-after"}
	if len(order) != len(expected) {
		t.Fatalf("execution order length = %d, want %d: %v", len(order), len(expected), order)
	}
	for i, v := range expected {
		if order[i] != v {
			t.Errorf("order[%d] = %q, want %q", i, order[i], v)
		}
	}
}

func TestResponseWriter_CapturesStatusCode(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	rw := &responseWriter{ResponseWriter: rec, statusCode: http.StatusOK}

	rw.WriteHeader(http.StatusNotFound)

	if rw.statusCode != http.StatusNotFound {
		t.Errorf("statusCode = %d, want %d", rw.statusCode, http.StatusNotFound)
	}
	if !rw.written {
		t.Error("written should be true after WriteHeader")
	}
}

func TestResponseWriter_DefaultsTo200(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	rw := &responseWriter{ResponseWriter: rec, statusCode: http.StatusOK}

	_, _ = rw.Write([]byte("hello"))

	if rw.statusCode != http.StatusOK {
		t.Errorf("statusCode = %d, want %d", rw.statusCode, http.StatusOK)
	}
	if !rw.written {
		t.Error("written should be true after Write")
	}
}
