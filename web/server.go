package web

import (
	"log"
	"net/http"
	"os"

	"github.com/a-h/templ"

	"github.com/hillsidelabs/rd/web/assets"
	"github.com/hillsidelabs/rd/web/ui/pages"
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
	mux.Handle("GET /", templ.Handler(pages.Landing()))

	// Log that the server is starting
	log.Printf("Server starting on port %s", s.port)

	// Start the server with the wrapped handler
	return http.ListenAndServe(":"+s.port, mux)
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
