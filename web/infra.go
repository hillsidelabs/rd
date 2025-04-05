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

func (s *Server) InfraNewVMCreate() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// We want to use the aws package to create a VM based on the forms defined in `web/ui/pages/infra/new_vm.go`. AI!

		component := pages.InfraNewVMCreate()
		component.Render(r.Context(), w)
	})
}
