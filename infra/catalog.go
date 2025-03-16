package infra

import (
	"encoding/json"
	"fmt"
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

func GetHostsFromFile() ([]Host, error) {
	out, err := os.ReadFile("hosts.yml")
	if err != nil {
		return nil, err
	}

	var hosts []Host
	err = yaml.Unmarshal(out, &hosts)
	return hosts, err
}

func NewCatalog() (*Catalog, error) {
	hosts, err := GetHostsFromFile()
	if err != nil {
		hosts, err = GetHostFromDOTerraform()
		if err != nil {
			return nil, err
		}
	}
	return &Catalog{Hosts: hosts}, nil
}

// infrastructureWrapper is used for marshaling/unmarshaling Infrastructure
type infrastructureWrapper struct {
	Provider string `yaml:"provider" json:"provider"`
	Metadata any    `yaml:"metadata" json:"metadata"`
}

// MarshalYAML implements the yaml.Marshaler interface
func (h Host) MarshalYAML() (interface{}, error) {
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
	type alias Host
	wrapped := struct {
		alias
		Infrastructure *infrastructureWrapper `yaml:"infrastructure,omitempty"`
	}{}

	if err := value.Decode(&wrapped); err != nil {
		return err
	}

	*h = Host(wrapped.alias)

	if wrapped.Infrastructure != nil {
		switch wrapped.Infrastructure.Provider {
		case "aws":
			var vm amazon.VM
			metadata, err := yaml.Marshal(wrapped.Infrastructure.Metadata)
			if err != nil {
				return err
			}
			if err := yaml.Unmarshal(metadata, &vm); err != nil {
				return err
			}
			h.Infrastructure = &vm
		default:
			return fmt.Errorf("unknown infrastructure provider: %s", wrapped.Infrastructure.Provider)
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
	type alias Host
	wrapped := struct {
		alias
		Infrastructure *infrastructureWrapper `json:"infrastructure,omitempty"`
	}{}

	if err := json.Unmarshal(data, &wrapped); err != nil {
		return err
	}

	*h = Host(wrapped.alias)

	if wrapped.Infrastructure != nil {
		switch wrapped.Infrastructure.Provider {
		case "aws":
			var vm amazon.VM
			metadata, err := json.Marshal(wrapped.Infrastructure.Metadata)
			if err != nil {
				return err
			}
			if err := json.Unmarshal(metadata, &vm); err != nil {
				return err
			}
			h.Infrastructure = &vm
		default:
			return fmt.Errorf("unknown infrastructure provider: %s", wrapped.Infrastructure.Provider)
		}
	}

	return nil
}
