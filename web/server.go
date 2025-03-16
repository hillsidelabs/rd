package web

import (
	"fmt"
	"log"
	"net/http"
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
	// Define a basic handler for the root path
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Welcome to the server!")
	})

	// Log that the server is starting
	log.Printf("Server starting on port %s", s.port)

	// Start the server
	return http.ListenAndServe(":"+s.port, nil)
}
