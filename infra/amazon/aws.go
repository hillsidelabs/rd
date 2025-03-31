package amazon

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// AWSConfig holds configuration for AWS services
type AWSConfig struct {
	Region               string
	CreateVPCIfNotExists bool
	VPCCIDR              string
	SubnetCIDR           string
	DefaultAZ            string
}

// NewAWSConfig creates a new AWS configuration with default values
func NewAWSConfig() *AWSConfig {
	return &AWSConfig{
		Region:               "us-west-2",
		CreateVPCIfNotExists: true,
		VPCCIDR:              "10.0.0.0/16",
		SubnetCIDR:           "10.0.1.0/24",
		DefaultAZ:            "a", // Will be combined with region (e.g., us-west-2a)
	}
}

// GetAvailabilityZone returns the full availability zone (region + az suffix)
func (c *AWSConfig) GetAvailabilityZone() string {
	return c.Region + c.DefaultAZ
}

// NewService returns a new EC2 client
func (c *AWSConfig) NewService() (*ec2.Client, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(c.Region))
	if err != nil {
		return nil, fmt.Errorf("unable to load SDK config: %v", err)
	}
	return ec2.NewFromConfig(cfg), nil
}

// VM represents an EC2 instance
type VM struct {
	Config       *AWSConfig
	Name         string
	ImageID      string
	InstanceType string
	Tags         []string
	VPC          string
}

// NewVM creates a new VM instance with the given configuration
func NewVM(name, imageID, instanceType string, tags []string) *VM {
	return &VM{
		Config:       NewAWSConfig(),
		Name:         name,
		ImageID:      imageID,
		InstanceType: instanceType,
		Tags:         tags,
		VPC:          name, // By default, use the VM name as the VPC name
	}
}

// CreateVPC creates a new VPC with the name defined by the VM
// struct. It will also create a public subnet that VMs will use.
func (vm *VM) CreateVPC() error {
	svc, err := vm.Config.NewService()
	if err != nil {
		return fmt.Errorf("failed to create AWS service: %v", err)
	}

	// Create VPC
	vpcInput := &ec2.CreateVpcInput{
		CidrBlock: aws.String(vm.Config.VPCCIDR),
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeVpc,
				Tags: []types.Tag{
					{
						Key:   aws.String("Name"),
						Value: aws.String(vm.Name),
					},
				},
			},
		},
	}

	vpcOutput, err := svc.CreateVpc(context.TODO(), vpcInput)
	if err != nil {
		return fmt.Errorf("failed to create VPC: %v", err)
	}

	// Enable DNS hostnames for the VPC
	_, err = svc.ModifyVpcAttribute(context.TODO(), &ec2.ModifyVpcAttributeInput{
		VpcId:              vpcOutput.Vpc.VpcId,
		EnableDnsHostnames: &types.AttributeBooleanValue{Value: aws.Bool(true)},
	})
	if err != nil {
		return fmt.Errorf("failed to enable DNS hostnames: %v", err)
	}

	// Create public subnet
	subnetInput := &ec2.CreateSubnetInput{
		VpcId:            vpcOutput.Vpc.VpcId,
		CidrBlock:        aws.String(vm.Config.SubnetCIDR),
		AvailabilityZone: aws.String(vm.Config.GetAvailabilityZone()),
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeSubnet,
				Tags: []types.Tag{
					{
						Key:   aws.String("Name"),
						Value: aws.String("public_" + vm.Name),
					},
				},
			},
		},
	}

	subnetOutput, err := svc.CreateSubnet(context.TODO(), subnetInput)
	if err != nil {
		return fmt.Errorf("failed to create subnet: %v", err)
	}

	// Create Internet Gateway
	igwInput := &ec2.CreateInternetGatewayInput{
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeInternetGateway,
				Tags: []types.Tag{
					{
						Key:   aws.String("Name"),
						Value: aws.String(vm.Name + "-igw"),
					},
				},
			},
		},
	}

	igwOutput, err := svc.CreateInternetGateway(context.TODO(), igwInput)
	if err != nil {
		return fmt.Errorf("failed to create internet gateway: %v", err)
	}

	// Attach Internet Gateway to VPC
	_, err = svc.AttachInternetGateway(context.TODO(), &ec2.AttachInternetGatewayInput{
		InternetGatewayId: igwOutput.InternetGateway.InternetGatewayId,
		VpcId:             vpcOutput.Vpc.VpcId,
	})
	if err != nil {
		return fmt.Errorf("failed to attach internet gateway: %v", err)
	}

	// Create route table
	rtInput := &ec2.CreateRouteTableInput{
		VpcId: vpcOutput.Vpc.VpcId,
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeRouteTable,
				Tags: []types.Tag{
					{
						Key:   aws.String("Name"),
						Value: aws.String(vm.Name + "-rt"),
					},
				},
			},
		},
	}

	rtOutput, err := svc.CreateRouteTable(context.TODO(), rtInput)
	if err != nil {
		return fmt.Errorf("failed to create route table: %v", err)
	}

	// Create route to Internet Gateway
	_, err = svc.CreateRoute(context.TODO(), &ec2.CreateRouteInput{
		RouteTableId:         rtOutput.RouteTable.RouteTableId,
		DestinationCidrBlock: aws.String("0.0.0.0/0"),
		GatewayId:            igwOutput.InternetGateway.InternetGatewayId,
	})
	if err != nil {
		return fmt.Errorf("failed to create route: %v", err)
	}

	// Associate route table with subnet
	_, err = svc.AssociateRouteTable(context.TODO(), &ec2.AssociateRouteTableInput{
		RouteTableId: rtOutput.RouteTable.RouteTableId,
		SubnetId:     subnetOutput.Subnet.SubnetId,
	})
	if err != nil {
		return fmt.Errorf("failed to associate route table: %v", err)
	}

	// Store VPC ID in the VM struct
	vm.VPC = *vpcOutput.Vpc.VpcId

	return nil
}

// FindSubnetFromVPC finds a subnet ID from a VPC with the given name.
// It looks for a subnet with "public" as a prefix in the name.
// If the VPC doesn't exist and CreateVPCIfNotExists is true, it will create the VPC.
func (vm *VM) FindSubnetFromVPC() (string, error) {
	svc, err := vm.Config.NewService()
	if err != nil {
		return "", fmt.Errorf("failed to create AWS service: %v", err)
	}

	// First, find the VPC by name
	vpcOutput, err := svc.DescribeVpcs(context.TODO(), &ec2.DescribeVpcsInput{
		Filters: []types.Filter{
			{
				Name: aws.String("tag:Name"),
				Values: []string{
					vm.VPC,
				},
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to describe VPCs: %v", err)
	}

	// If VPC not found and CreateVPCIfNotExists is true, create it
	if len(vpcOutput.Vpcs) == 0 {
		if vm.Config.CreateVPCIfNotExists {
			if err := vm.CreateVPC(); err != nil {
				return "", fmt.Errorf("failed to create VPC: %v", err)
			}
			// Fetch the newly created VPC
			vpcOutput, err = svc.DescribeVpcs(context.TODO(), &ec2.DescribeVpcsInput{
				Filters: []types.Filter{
					{
						Name: aws.String("tag:Name"),
						Values: []string{
							vm.VPC,
						},
					},
				},
			})
			if err != nil {
				return "", fmt.Errorf("failed to describe newly created VPC: %v", err)
			}
		} else {
			return "", fmt.Errorf("no VPC found with name: %s", vm.VPC)
		}
	}

	vpcID := *vpcOutput.Vpcs[0].VpcId

	// Now find the public subnet in this VPC
	subnetOutput, err := svc.DescribeSubnets(context.TODO(), &ec2.DescribeSubnetsInput{
		Filters: []types.Filter{
			{
				Name: aws.String("vpc-id"),
				Values: []string{
					vpcID,
				},
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to describe subnets: %v", err)
	}

	// Look for a subnet with "public" prefix in its name tag
	for _, subnet := range subnetOutput.Subnets {
		for _, tag := range subnet.Tags {
			if *tag.Key == "Name" && strings.HasPrefix(strings.ToLower(*tag.Value), "public") {
				return *subnet.SubnetId, nil
			}
		}
	}

	return "", fmt.Errorf("no public subnet found in VPC %s", vm.VPC)
}

// Create creates a new VM
func (vm *VM) Create() error {
	svc, err := vm.Config.NewService()
	if err != nil {
		return fmt.Errorf("failed to create AWS service: %v", err)
	}

	vmTags := make([]types.Tag, len(vm.Tags))
	for i, tag := range vm.Tags {
		vmTags[i] = types.Tag{
			Key:   aws.String(tag),
			Value: aws.String(tag),
		}

	}

	input := &ec2.RunInstancesInput{
		ImageId:      aws.String(vm.ImageID),
		InstanceType: types.InstanceType(vm.InstanceType),
		MinCount:     aws.Int32(1),
		MaxCount:     aws.Int32(1),
	}

	if len(vmTags) > 0 {
		input.TagSpecifications = []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeInstance,
				Tags:         vmTags,
			},
		}
	}

	if vm.VPC != "" {
		subnetID, err := vm.FindSubnetFromVPC()
		if err != nil {
			return fmt.Errorf("failed to find subnet: %v", err)
		}

		input.NetworkInterfaces = []types.InstanceNetworkInterfaceSpecification{
			{
				AssociatePublicIpAddress: aws.Bool(true),
				DeviceIndex:              aws.Int32(0),
				SubnetId:                 aws.String(subnetID),
			},
		}
	}

	resp, err := svc.RunInstances(context.TODO(), input)
	if err != nil {
		return fmt.Errorf("failed to create instance, %v", err)
	}

	fmt.Printf("Created instance %s\n", *resp.Instances[0].InstanceId)
	return nil
}

// Provider returns the infrastructure provider type
func (vm *VM) Provider() string {
	return "aws"
}

// Metadata returns the AWS-specific metadata
func (vm *VM) Metadata() any {
	return map[string]any{
		"region":   vm.Config.Region,
		"image_id": vm.ImageID,
		"type":     vm.InstanceType,
		"vpc":      vm.VPC,
	}
}
