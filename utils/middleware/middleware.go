// Package middleware provides HTTP middleware functions for request/response handling
package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/yourusername/videostreamingplatform/utils/errors"
	"github.com/yourusername/videostreamingplatform/utils/observability"
)

// LoggingMiddleware logs all HTTP requests and responses
func LoggingMiddleware(logger *observability.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap response writer to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		logger.Printf("→ %s %s %s", r.Method, r.URL.Path, r.RemoteAddr)
		next.ServeHTTP(wrapped, r)

		duration := time.Since(start)
		logger.Printf("← %s %s completed in %dms [%d]",
			r.Method, r.URL.Path, duration.Milliseconds(), wrapped.statusCode)
	})
}

// errorResponseMiddleware handles errors from handlers
func ErrorHandlingMiddleware(logger *observability.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				logger.Printf("panic recovered: %v", err)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintf(w, `{"type":"INTERNAL_ERROR","message":"Internal server error","status_code":500}`)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

func (rw *responseWriter) WriteHeader(code int) {
	if !rw.written {
		rw.statusCode = code
		rw.written = true
		rw.ResponseWriter.WriteHeader(code)
	}
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.written {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.ResponseWriter.Write(b)
}

// ChainMiddleware chains multiple middleware handlers
func ChainMiddleware(handler http.Handler, middlewares ...func(http.Handler) http.Handler) http.Handler {
	// reverse apply middleware
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}
	return handler
}

// ErrorResponseWriter writes structured error responses
func WriteError(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "application/json")

	if appErr, ok := err.(*errors.AppError); ok {
		w.WriteHeader(appErr.HTTPStatus())
		fmt.Fprintf(w,
			`{"type":"%s","message":"%s","status_code":%d}`,
			appErr.Type, appErr.Message, appErr.StatusCode)
		return
	}

	w.WriteHeader(http.StatusInternalServerError)
	fmt.Fprintf(w,
		`{"type":"INTERNAL_ERROR","message":"Internal server error","status_code":500}`)
}
