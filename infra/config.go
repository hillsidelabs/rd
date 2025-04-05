package infra

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"gopkg.in/yaml.v3"
)

// ProviderType represents the supported cloud providers
type ProviderType string

const (
	ProviderAWS          ProviderType = "aws"
	ProviderDigitalOcean ProviderType = "digitalocean"
)

// Config represents the application configuration
type Config struct {
	Provider ProviderType `yaml:"provider"`
	AWS      AWSConfig    `yaml:"aws,omitempty"`
	DO       DOConfig     `yaml:"digitalocean,omitempty"`
}

// AWSConfig contains AWS-specific configuration
type AWSConfig struct {
	Region          string            `yaml:"region"`
	VPC             string            `yaml:"vpc"`
	Subnet          string            `yaml:"subnet"`
	SecurityGroup   string            `yaml:"security_group"`
	KeyPair         string            `yaml:"key_pair"`
	InstanceType    string            `yaml:"instance_type"`
	ImageOwner      string            `yaml:"image_owner"`
	ImageNameFilter string            `yaml:"image_name_filter"`
	Tags            map[string]string `yaml:"tags"`
}

// DOConfig contains DigitalOcean-specific configuration
type DOConfig struct {
	Region       string            `yaml:"region"`
	Size         string            `yaml:"size"`
	ImageSlug    string            `yaml:"image_slug"`
	Tags         []string          `yaml:"tags"`
	SSHKeys      []string          `yaml:"ssh_keys"`
	VPCName      string            `yaml:"vpc_name"`
	CustomLabels map[string]string `yaml:"custom_labels"`
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		Provider: ProviderAWS,
		AWS: AWSConfig{
			Region:          "us-west-2",
			VPC:             "default",
			Subnet:          "default",
			SecurityGroup:   "default",
			KeyPair:         "default",
			InstanceType:    "t2.micro",
			ImageOwner:      "099720109477", // Canonical (Ubuntu)
			ImageNameFilter: "ubuntu/images/hvm-ssd/ubuntu-*-22.04-amd64-server-*",
			Tags: map[string]string{
				"ManagedBy": "rd",
			},
		},
		DO: DOConfig{
			Region:    "nyc1",
			Size:      "s-1vcpu-1gb",
			ImageSlug: "ubuntu-22-04-x64",
			Tags:      []string{"rd-managed"},
			VPCName:   "default",
		},
	}
}

// LoadConfig loads the configuration from a file
func LoadConfig(filename string) (*Config, error) {
	if filename == "" {
		filename = "config.yml"
	}

	// Check if file exists, if not create default
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		cfg := DefaultConfig()
		return cfg, cfg.Save(filename)
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Set defaults for any missing values
	if cfg.Provider == "" {
		cfg.Provider = ProviderAWS
	}

	return &cfg, nil
}

// Save writes the configuration to a file
func (c *Config) Save(filename string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetLatestAMI returns the latest AMI ID for the configured filters
func (c *Config) GetLatestAMI() (string, error) {
	if c.Provider != ProviderAWS {
		return "", fmt.Errorf("getting latest AMI is only supported for AWS")
	}

	// Load AWS SDK configuration
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(c.AWS.Region))
	if err != nil {
		return "", fmt.Errorf("unable to load SDK config: %w", err)
	}

	// Create EC2 client
	client := ec2.NewFromConfig(cfg)

	// Prepare filters for the DescribeImages API call
	filters := []types.Filter{
		{
			Name:   aws.String("name"),
			Values: []string{c.AWS.ImageNameFilter},
		},
		{
			Name:   aws.String("state"),
			Values: []string{"available"},
		},
	}

	// If owner is specified, add it to the request
	var owners []string
	if c.AWS.ImageOwner != "" {
		owners = []string{c.AWS.ImageOwner}
	}

	// Call DescribeImages API
	resp, err := client.DescribeImages(context.TODO(), &ec2.DescribeImagesInput{
		Filters: filters,
		Owners:  owners,
	})
	if err != nil {
		return "", fmt.Errorf("failed to describe images: %w", err)
	}

	if len(resp.Images) == 0 {
		return "", fmt.Errorf("no images found matching the criteria")
	}

	// Find the newest image
	var latestImage types.Image
	var latestTime string

	for _, image := range resp.Images {
		if latestTime == "" || strings.Compare(latestTime, *image.CreationDate) < 0 {
			latestTime = *image.CreationDate
			latestImage = image
		}
	}

	log.Printf("Found latest AMI: %s (%s) created on %s", *latestImage.ImageId, *latestImage.Name, latestTime)
	return *latestImage.ImageId, nil
}

// GetActiveProvider returns the active provider configuration
func (c *Config) GetActiveProvider() interface{} {
	switch c.Provider {
	case ProviderAWS:
		return c.AWS
	case ProviderDigitalOcean:
		return c.DO
	default:
		return nil
	}
}
