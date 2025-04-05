package web

import (
	"net/http"

	"github.com/hillsidelabs/rd/infra"
	"github.com/hillsidelabs/rd/web/ui/pages"
)

func (s *Server) Infra() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		catalog, err := infra.NewCatalog()
		if err != nil {
			http.Error(w, "Failed to load infrastructure catalog: "+err.Error(), http.StatusInternalServerError)
			return
		}

		component := pages.Infra(catalog)
		component.Render(r.Context(), w)
	})
}

func (s *Server) InfraNewVM() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		component := pages.InfraNewVM()
		component.Render(r.Context(), w)
	})
}
