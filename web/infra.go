package web

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hillsidelabs/rd/infra"
	"github.com/hillsidelabs/rd/infra/amazon"
	"github.com/hillsidelabs/rd/web/ui/modules"
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
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Parse form data
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Failed to parse form: "+err.Error(), http.StatusBadRequest)
			return
		}

		// Extract form values
		name := r.FormValue("name")
		hostname := r.FormValue("hostname")
		instanceType := r.FormValue("instance_type")
		vpcName := r.FormValue("vpc")
		tagsStr := r.FormValue("tags")

		// Validate required fields
		if name == "" {
			http.Error(w, "VM name is required", http.StatusBadRequest)
			return
		}

		if instanceType == "" {
			instanceType = "t2.micro" // Default instance type
		}

		// Process tags
		var tags []string
		if tagsStr != "" {
			tags = strings.Split(tagsStr, ",")
			// Trim whitespace from tags
			for i := range tags {
				tags[i] = strings.TrimSpace(tags[i])
			}
		}

		// Create VM using AWS package
		imageID := "ami-0c55b159cbfafe1f0" // Default Amazon Linux 2 AMI - this should be configurable
		vm := infra.amazon.NewVM(name, imageID, instanceType, tags)
		
		if vpcName != "" {
			vm.VPC = vpcName
		}

		// Create command logs for the UI
		logs := []modules.CommandLogLine{
			{Content: fmt.Sprintf("Creating VM '%s' with instance type '%s'", name, instanceType), Type: modules.LogTypeInfo, Time: time.Now()},
			{Content: fmt.Sprintf("Using image ID: %s", imageID), Type: modules.LogTypeInfo, Time: time.Now().Add(time.Millisecond * 100)},
		}

		// Start VM creation in a goroutine to avoid blocking the response
		go func() {
			logs = append(logs, modules.CommandLogLine{
				Content: fmt.Sprintf("Creating VPC for VM '%s'...", name),
				Type:    modules.LogTypeCommand,
				Time:    time.Now().Add(time.Millisecond * 200),
			})

			// Create VPC if needed
			if err := vm.CreateVPC(); err != nil {
				logs = append(logs, modules.CommandLogLine{
					Content: fmt.Sprintf("Failed to create VPC: %v", err),
					Type:    modules.LogTypeError,
					Time:    time.Now().Add(time.Millisecond * 300),
				})
				// Store logs for status page
				// TODO: Implement a way to store and retrieve command logs
				return
			}

			logs = append(logs, modules.CommandLogLine{
				Content: "VPC created successfully",
				Type:    modules.LogTypeSuccess,
				Time:    time.Now().Add(time.Millisecond * 400),
			})

			// TODO: Implement the actual VM creation logic
			// This would involve calling the appropriate AWS API methods
			
			logs = append(logs, modules.CommandLogLine{
				Content: fmt.Sprintf("VM '%s' created successfully", name),
				Type:    modules.LogTypeSuccess,
				Time:    time.Now().Add(time.Second * 2),
			})
		}()

		// Render the creation status page
		component := pages.InfraNewVMCreate(pages.InfraNewVMCreateProps{
			VMName:       name,
			InstanceType: instanceType,
			Logs:         logs,
			IsRunning:    true,
		})
		component.Render(r.Context(), w)
	})
}
