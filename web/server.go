package web

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/hillsidelabs/rd/web/assets"
	"github.com/hillsidelabs/rd/web/middleware"
	"github.com/rs/zerolog/log"
)

// Server represents the HTTP server configuration
type Server struct {
	port string
}

// NewServer creates a new server instance
func NewServer(port string) *Server {
	return &Server{
		port: port,
	}
}

// Start initializes and starts the HTTP server
func (s *Server) Start() error {
	mux := http.NewServeMux()

	SetupAssetsRoutes(mux)

	// Define a basic handler for the root path
	mux.Handle("GET /", s.CatalogHandler())
	mux.Handle("GET /infra", s.Infra())
	mux.Handle("GET /infra/newvm", s.InfraNewVM())
	mux.Handle("POST /infra/newvm", s.InfraNewVMCreate())
	mux.Handle("GET /infra/vm/events", s.VMCreationSSE())

	// Fill in the following handler with a SSE handler that accepts a `num_events` query param and sends that of SSE events where the `event` is `log` and the data is some html that has `<div>This is the { i } event</div>`. AI!
	mux.Handle("GET /test/sse", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set headers for SSE
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		// Parse the num_events query parameter
		numEventsStr := r.URL.Query().Get("num_events")
		numEvents := 5 // Default value
		if numEventsStr != "" {
			if n, err := strconv.Atoi(numEventsStr); err == nil && n > 0 {
				numEvents = n
			}
		}

		// Create a flusher to ensure data is sent immediately
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming not supported", http.StatusInternalServerError)
			return
		}

		// Send the specified number of SSE events
		for i := 1; i <= numEvents; i++ {
			// Format the event data
			data := fmt.Sprintf("<div>This is the %d event</div>", i)
			
			// Write the event
			fmt.Fprintf(w, "event: log\n")
			fmt.Fprintf(w, "data: %s\n\n", data)
			
			// Flush to send the data immediately
			flusher.Flush()
			
			// Add a small delay between events
			time.Sleep(500 * time.Millisecond)
		}
	}))

	// Log that the server is starting
	log.Info().Str("port", s.port).Msg("Server starting")

	// Apply middleware
	handler := middleware.RequestLogger(mux)

	// Start the server with the wrapped handler
	return http.ListenAndServe(":"+s.port, handler)
}

func SetupAssetsRoutes(mux *http.ServeMux) {
	var isDevelopment = os.Getenv("GO_ENV") != "production"
	log.Printf("isDevelopment: %v", isDevelopment)

	assetHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isDevelopment {
			w.Header().Set("Cache-Control", "no-store")
		}

		var fs http.Handler
		if isDevelopment {
			fs = http.FileServer(http.Dir("./web/assets"))
		} else {
			fs = http.FileServer(http.FS(assets.Assets))
		}

		fs.ServeHTTP(w, r)
	})

	mux.Handle("GET /assets/", http.StripPrefix("/assets/", assetHandler))
}
