package infra

import (
	"os"
	"testing"
)

func TestCatalogLoadFromYAML(t *testing.T) {
	// Create a temporary hosts.yml file for testing
	testYAML := `
- name: test-host
  private_ip: 192.168.1.10
  ip: 203.0.113.5
  hostname: test-server
  tags:
    - test
    - development
`
	// Save the original hosts.yml if it exists
	originalContent, err := os.ReadFile("../../hosts.yml")
	hasOriginal := err == nil

	// Write test data to hosts.yml
	err = os.WriteFile("../../hosts.yml", []byte(testYAML), 0644)
	if err != nil {
		t.Fatalf("Failed to write test hosts.yml: %v", err)
	}

	// Ensure we restore the original file when done
	defer func() {
		if hasOriginal {
			os.WriteFile("../../hosts.yml", originalContent, 0644)
		} else {
			os.Remove("../../hosts.yml")
		}
	}()

	// Test loading the catalog
	catalog, err := NewCatalog()
	if err != nil {
		t.Fatalf("Failed to load catalog: %v", err)
	}

	// Verify the catalog contains the expected host
	if len(catalog.Hosts) != 1 {
		t.Fatalf("Expected 1 host, got %d", len(catalog.Hosts))
	}

	host := catalog.Hosts[0]
	if host.Name != "test-host" {
		t.Errorf("Expected host name 'test-host', got '%s'", host.Name)
	}
	if host.IP != "203.0.113.5" {
		t.Errorf("Expected host IP '203.0.113.5', got '%s'", host.IP)
	}
	if host.PrivateIP != "192.168.1.10" {
		t.Errorf("Expected private IP '192.168.1.10', got '%s'", host.PrivateIP)
	}
	if host.Hostname != "test-server" {
		t.Errorf("Expected hostname 'test-server', got '%s'", host.Hostname)
	}
	if len(host.Tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(host.Tags))
	}
	if host.Tags[0] != "test" || host.Tags[1] != "development" {
		t.Errorf("Expected tags ['test', 'development'], got %v", host.Tags)
	}
}

func TestCatalogGetTargets(t *testing.T) {
	// Create a catalog with test hosts
	catalog := &Catalog{
		Hosts: []Host{
			{
				Name:      "web-server",
				IP:        "203.0.113.10",
				PrivateIP: "192.168.1.10",
				Hostname:  "web1",
				Tags:      []string{"web", "production"},
			},
			{
				Name:      "db-server",
				IP:        "203.0.113.11",
				PrivateIP: "192.168.1.11",
				Hostname:  "db1",
				Tags:      []string{"database", "production"},
			},
			{
				Name:      "web-staging",
				IP:        "203.0.113.12",
				PrivateIP: "192.168.1.12",
				Hostname:  "web-stg",
				Tags:      []string{"web", "staging"},
			},
		},
	}

	// Test filtering by name
	targets, err := catalog.GetTargets("web", "", "")
	if err != nil {
		t.Fatalf("Failed to get targets by name: %v", err)
	}
	if len(targets) != 2 {
		t.Errorf("Expected 2 targets with name prefix 'web', got %d", len(targets))
	}

	// Test filtering by IP
	targets, err = catalog.GetTargets("", "203.0.113.11", "")
	if err != nil {
		t.Fatalf("Failed to get targets by IP: %v", err)
	}
	if len(targets) != 1 {
		t.Errorf("Expected 1 target with IP '203.0.113.11', got %d", len(targets))
	}
	if len(targets) > 0 && targets[0].Name != "db-server" {
		t.Errorf("Expected target with name 'db-server', got '%s'", targets[0].Name)
	}

	// Test getting all targets
	targets, err = catalog.GetTargets("", "", "")
	if err != nil {
		t.Fatalf("Failed to get all targets: %v", err)
	}
	if len(targets) != 3 {
		t.Errorf("Expected 3 targets, got %d", len(targets))
	}
}
