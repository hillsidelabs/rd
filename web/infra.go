package web

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"sync"
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
		// hostname := r.FormValue("hostname")
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

		// Load configuration
		config, err := infra.LoadConfig("")
		if err != nil {
			log.Printf("Failed to load config: %v, using defaults", err)
			config = infra.DefaultConfig()
		}

		// Get latest AMI ID based on configuration
		var imageID string
		if config.Provider == infra.ProviderAWS {
			latestAMI, err := config.GetLatestAMI()
			if err != nil {
				log.Printf("Failed to get latest AMI: %v, using fallback", err)
				imageID = "ami-0c55b159cbfafe1f0" // Fallback AMI
			} else {
				imageID = latestAMI
			}
		} else {
			// For DigitalOcean, we'll use the image slug from config
			imageID = config.DO.ImageSlug
		}

		// Create VM using the appropriate provider
		vm := amazon.NewVM(name, imageID, instanceType, tags)

		if vpcName != "" {
			vm.VPC = vpcName
		} else if config.AWS.VPC != "default" {
			vm.VPC = config.AWS.VPC
		}

		// Create command logs for the UI
		logs := []modules.CommandLogLine{
			{Content: fmt.Sprintf("Creating VM '%s' with instance type '%s'", name, instanceType), Type: modules.LogTypeInfo, Time: time.Now()},
			{Content: fmt.Sprintf("Using image ID: %s", imageID), Type: modules.LogTypeInfo, Time: time.Now().Add(time.Millisecond * 100)},
		}

		// Generate a unique job ID for this VM creation
		jobID := fmt.Sprintf("vm-create-%s-%d", name, time.Now().Unix())

		// Create a channel to stream logs
		logChan := make(chan modules.CommandLogLine, 100)

		// Store the log channel in a global map for the SSE endpoint to access
		s.registerJobChannel(jobID, logChan)

		// Start VM creation in a goroutine to avoid blocking the response
		go func() {
			defer s.closeJobChannel(jobID)

			// Send initial logs
			for _, log := range logs {
				logChan <- log
			}
			// Command to create VPC
			logChan <- modules.CommandLogLine{
				Content: fmt.Sprintf("Creating VPC for VM '%s'...", name),
				Type:    modules.LogTypeCommand,
				Time:    time.Now(),
			}

			// Simulate VPC creation (replace with actual AWS API call)
			time.Sleep(3 * time.Second)

			// Check if we're using AWS VM type that has CreateVPC
			// Create VPC if needed
			// if err := vm.CreateVPC(); err != nil {
			// 	logChan <- modules.CommandLogLine{
			// 		Content: fmt.Sprintf("Failed to create VPC: %v", err),
			// 		Type:    modules.LogTypeError,
			// 		Time:    time.Now(),
			// 	}
			// 	return
			// }

			logChan <- modules.CommandLogLine{
				Content: "VPC created successfully",
				Type:    modules.LogTypeSuccess,
				Time:    time.Now(),
			}

			time.Sleep(3 * time.Second)

			// Simulate launching instance
			logChan <- modules.CommandLogLine{
				Content: fmt.Sprintf("Launching instance '%s' with type '%s'...", name, instanceType),
				Type:    modules.LogTypeCommand,
				Time:    time.Now().Add(time.Second * 2),
			}
			time.Sleep(3 * time.Second)

			// Simulate waiting for instance
			logChan <- modules.CommandLogLine{
				Content: "Waiting for instance to be available...",
				Type:    modules.LogTypeInfo,
				Time:    time.Now().Add(time.Second * 4),
			}
			time.Sleep(3 * time.Second)

			// Success message
			logChan <- modules.CommandLogLine{
				Content: fmt.Sprintf("VM '%s' created successfully", name),
				Type:    modules.LogTypeSuccess,
				Time:    time.Now().Add(time.Second * 5),
			}
			time.Sleep(3 * time.Second)

			// Add IP information
			logChan <- modules.CommandLogLine{
				Content: "Instance details:",
				Type:    modules.LogTypeInfo,
				Time:    time.Now().Add(time.Second * 6),
			}
			time.Sleep(3 * time.Second)

			logChan <- modules.CommandLogLine{
				Content: "  Public IP: 203.0.113." + fmt.Sprintf("%d", rand.Intn(255)),
				Type:    modules.LogTypeInfo,
				Time:    time.Now().Add(time.Second * 7),
			}
			time.Sleep(3 * time.Second)

			logChan <- modules.CommandLogLine{
				Content: "  Private IP: 10.0.1." + fmt.Sprintf("%d", rand.Intn(255)),
				Type:    modules.LogTypeInfo,
				Time:    time.Now().Add(time.Second * 8),
			}
			time.Sleep(3 * time.Second)
		}()

		// Render the creation status page
		component := pages.InfraNewVMCreate(pages.InfraNewVMCreateProps{
			VMName:       name,
			InstanceType: instanceType,
			Logs:         logs,
			IsRunning:    true,
			SSEUrl:       "/infra/vm/events?job=" + jobID,
		})
		component.Render(r.Context(), w)
	})
}

// Add SSE endpoint for streaming VM creation logs
func (s *Server) VMCreationSSE() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jobID := r.URL.Query().Get("job")
		if jobID == "" {
			http.Error(w, "Missing job ID", http.StatusBadRequest)
			return
		}

		// Get the log channel for this job
		logChan, ok := s.getJobChannel(jobID)
		if !ok {
			http.Error(w, "Job not found", http.StatusNotFound)
			return
		}

		// Set headers for SSE
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		// Create a notification channel for client disconnection
		clientGone := r.Context().Done()

		// Stream logs until channel is closed or client disconnects
		for {
			select {
			case log, ok := <-logChan:
				if !ok {
					// Channel closed, send completion event and exit
					fmt.Fprintf(w, "event: complete\ndata: {\"status\":\"complete\"}\n\n")
					w.(http.Flusher).Flush()
					return
				}

				// Send the log as an SSE event
				body := new(bytes.Buffer)
				modules.CommandStatusLogLine(log).Render(context.Background(), body)
				fmt.Fprintf(w, "event: log\ndata: %s\n\n", body.String())
				w.(http.Flusher).Flush()

			case <-clientGone:
				// Client disconnected
				fmt.Fprintf(w, "event: close\ndata: ")
				w.(http.Flusher).Flush()
				return
			}
		}
	})
}

// Job channel management
var (
	jobChannels     = make(map[string]chan modules.CommandLogLine)
	jobChannelMutex sync.Mutex
)

func (s *Server) registerJobChannel(jobID string, ch chan modules.CommandLogLine) {
	jobChannelMutex.Lock()
	defer jobChannelMutex.Unlock()
	jobChannels[jobID] = ch
}

func (s *Server) getJobChannel(jobID string) (chan modules.CommandLogLine, bool) {
	jobChannelMutex.Lock()
	defer jobChannelMutex.Unlock()
	ch, ok := jobChannels[jobID]
	return ch, ok
}

func (s *Server) closeJobChannel(jobID string) {
	jobChannelMutex.Lock()
	defer jobChannelMutex.Unlock()
	if ch, ok := jobChannels[jobID]; ok {
		close(ch)
		delete(jobChannels, jobID)
	}
}
