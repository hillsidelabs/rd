package middleware

import (
	"net/http"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Initialize the logger with default settings
func init() {
	// Set up pretty console logging for development
	if os.Getenv("GO_ENV") != "production" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})
	} else {
		// In production, use structured JSON logging
		zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
		log.Logger = zerolog.New(os.Stderr).With().Timestamp().Logger()
	}
}

// RequestLogger is a middleware that logs HTTP requests using zerolog
func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create a response wrapper to capture the status code
		ww := NewResponseWriter(w)

		// Process the request
		next.ServeHTTP(ww, r)

		// Calculate duration
		duration := time.Since(start)

		// Log the request details
		log.Info().
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Str("remote_addr", r.RemoteAddr).
			Int("status", ww.Status()).
			Dur("duration", duration).
			Str("user_agent", r.UserAgent()).
			Msg("Request processed")
	})
}

// ResponseWriter is a wrapper for http.ResponseWriter that captures the status code
type ResponseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

// NewResponseWriter creates a new ResponseWriter
func NewResponseWriter(w http.ResponseWriter) *ResponseWriter {
	return &ResponseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK, // Default to 200 OK
	}
}

// WriteHeader captures the status code and passes it to the underlying ResponseWriter
func (rw *ResponseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.written = true
	rw.ResponseWriter.WriteHeader(code)
}

// Write captures that a response has been written
func (rw *ResponseWriter) Write(b []byte) (int, error) {
	rw.written = true
	return rw.ResponseWriter.Write(b)
}

// Status returns the HTTP status code
func (rw *ResponseWriter) Status() int {
	return rw.statusCode
}
