package web

import (
	"encoding/json"
	"net/http"

	"github.com/hillsidelabs/rd/infra"
)

func (s *Server) CatalogHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		catalog, err := infra.NewCatalog()
		if err != nil {
			http.Error(w, "Failed to load infrastructure catalog: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(catalog.Hosts); err != nil {
			http.Error(w, "Failed to encode catalog: "+err.Error(), http.StatusInternalServerError)
			return
		}
	})
}
