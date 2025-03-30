package infra

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/hillsidelabs/rd/infra/amazon"
	"gopkg.in/yaml.v3"
)

// Infrastructure defines the interface for infrastructure providers
type Infrastructure interface {
	Provider() string
	Metadata() any
}

type Host struct {
	Name           string         `yaml:"name" json:"name"`
	IP             string         `yaml:"ip" json:"ip"`
	PrivateIP      string         `yaml:"private_ip" json:"private_ip"`
	Hostname       string         `yaml:"hostname" json:"hostname"`
	Tags           []string       `yaml:"tags" json:"tags"`
	Infrastructure Infrastructure `yaml:"infrastructure,omitempty" json:"infrastructure,omitempty"`
}

func (h Host) String() string {
	return fmt.Sprintf("%s:\t%s\t[%s]", h.Name, h.IP, h.PrivateIP)
}

type Catalog struct {
	Hosts []Host
}

func (c *Catalog) GetTargets(name, ip, private string) ([]Host, error) {
	targets := []Host{}

	for _, h := range c.Hosts {
		if name != "" && strings.HasPrefix(h.Name, name) {
			targets = append(targets, h)
		}
		if ip != "" && h.IP == ip {
			targets = append(targets, h)
		}
		if private != "" && h.Name == private {
			targets = append(targets, h)
		}

		if name == "" && ip == "" && private == "" {
			targets = append(targets, h)
		}
	}

	return targets, nil
}

func GetHostsFromFile(fn string) ([]Host, error) {
	out, err := os.ReadFile(fn)
	if err != nil {
		log.Printf("Error reading %s: %v", fn, err)
		return nil, err
	}

	var hosts []Host
	if err = yaml.Unmarshal(out, &hosts); err != nil {
		log.Printf("Error unmarshaling YAML: %v", err)
		return nil, err
	}

	for i, h := range hosts {
		log.Printf("Host %d: %s (%s)", i, h.Name, h.IP)
	}
	return hosts, nil
}

func NewCatalogFromFile(fn string) (*Catalog, error) {
	if fn == "" {
		return nil, fmt.Errorf("no filename given")
	}

	hosts, err := GetHostsFromFile(fn)
	if err != nil {
		// hosts, err = GetHostFromDOTerraform()
		// if err != nil {
		return nil, err
		// }
	}

	log.Println("Hosts: ", hosts)
	return &Catalog{Hosts: hosts}, nil
}

func NewCatalog() (*Catalog, error) {
	return NewCatalogFromFile("hosts.yml")
}

// infrastructureWrapper is used for marshaling/unmarshaling Infrastructure
type infrastructureWrapper struct {
	Provider string `yaml:"provider" json:"provider"`
	Metadata any    `yaml:"metadata" json:"metadata"`
}

// MarshalYAML implements the yaml.Marshaler interface
func (h Host) MarshalYAML() (any, error) {
	type alias Host
	wrapped := struct {
		alias
		Infrastructure *infrastructureWrapper `yaml:"infrastructure,omitempty"`
	}{
		alias: alias(h),
	}

	if h.Infrastructure != nil {
		wrapped.Infrastructure = &infrastructureWrapper{
			Provider: h.Infrastructure.Provider(),
			Metadata: h.Infrastructure.Metadata(),
		}
	}

	return wrapped, nil
}

// UnmarshalYAML implements the yaml.Unmarshaler interface
func (h *Host) UnmarshalYAML(value *yaml.Node) error {
	// Create a temporary struct without the custom unmarshaler
	type HostAlias struct {
		Name           string                 `yaml:"name"`
		IP             string                 `yaml:"ip"`
		PrivateIP      string                 `yaml:"private_ip"`
		Hostname       string                 `yaml:"hostname"`
		Tags           []string               `yaml:"tags"`
		Infrastructure *infrastructureWrapper `yaml:"infrastructure,omitempty"`
	}
	
	var alias HostAlias
	if err := value.Decode(&alias); err != nil {
		return err
	}
	
	// Copy the basic fields
	h.Name = alias.Name
	h.IP = alias.IP
	h.PrivateIP = alias.PrivateIP
	h.Hostname = alias.Hostname
	h.Tags = alias.Tags
	
	// Handle the infrastructure field
	if alias.Infrastructure != nil {
		switch alias.Infrastructure.Provider {
		case "aws":
			var vm amazon.VM
			metadata, err := yaml.Marshal(alias.Infrastructure.Metadata)
			if err != nil {
				return err
			}
			if err := yaml.Unmarshal(metadata, &vm); err != nil {
				return err
			}
			h.Infrastructure = &vm
		default:
			return fmt.Errorf("unknown infrastructure provider: %s", alias.Infrastructure.Provider)
		}
	}
	
	return nil
}

// MarshalJSON implements the json.Marshaler interface
func (h Host) MarshalJSON() ([]byte, error) {
	type alias Host
	wrapped := struct {
		alias
		Infrastructure *infrastructureWrapper `json:"infrastructure,omitempty"`
	}{
		alias: alias(h),
	}

	if h.Infrastructure != nil {
		wrapped.Infrastructure = &infrastructureWrapper{
			Provider: h.Infrastructure.Provider(),
			Metadata: h.Infrastructure.Metadata(),
		}
	}

	return json.Marshal(wrapped)
}

// UnmarshalJSON implements the json.Unmarshaler interface
func (h *Host) UnmarshalJSON(data []byte) error {
	// Create a temporary struct without the custom unmarshaler
	type HostAlias struct {
		Name           string                 `json:"name"`
		IP             string                 `json:"ip"`
		PrivateIP      string                 `json:"private_ip"`
		Hostname       string                 `json:"hostname"`
		Tags           []string               `json:"tags"`
		Infrastructure *infrastructureWrapper `json:"infrastructure,omitempty"`
	}
	
	var alias HostAlias
	if err := json.Unmarshal(data, &alias); err != nil {
		return err
	}
	
	// Copy the basic fields
	h.Name = alias.Name
	h.IP = alias.IP
	h.PrivateIP = alias.PrivateIP
	h.Hostname = alias.Hostname
	h.Tags = alias.Tags
	
	// Handle the infrastructure field
	if alias.Infrastructure != nil {
		switch alias.Infrastructure.Provider {
		case "aws":
			var vm amazon.VM
			metadata, err := json.Marshal(alias.Infrastructure.Metadata)
			if err != nil {
				return err
			}
			if err := json.Unmarshal(metadata, &vm); err != nil {
				return err
			}
			h.Infrastructure = &vm
		default:
			return fmt.Errorf("unknown infrastructure provider: %s", alias.Infrastructure.Provider)
		}
	}
	
	return nil
}
