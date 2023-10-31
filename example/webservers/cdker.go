package main

import (
	"github.com/ZeljkoBenovic/cdker/modules/instance"
	"github.com/ZeljkoBenovic/cdker/stack"
)

func main() {
	app := stack.New().SetStack("example-stack", stack.WithCredentials("368322230844", "eu-west-1"))
	app.DeployResources(deploy2WebServers(app))
}

func deploy2WebServers(app stack.Stack) stack.DeployStack {
	// define a single instance
	webServer := instance.InstanceSpec{
		Class:      instance.InstanceClass_T3,
		Size:       instance.InstanceSize_SMALL,
		SubnetType: instance.SubnetType_PUBLIC,
		// Default instance is Ubuntu20.04 and default VPC
		//AMI:                instance.Ubuntu20,
		//VPC:                instance.VPCDefault,
		AssociatePubIP: true,
		StorageSpecs: []instance.StorageSpec{
			{
				Size:                30,
				Name:                "/dev/sdf",
				DeleteOnTermination: true,
				VolumeType:          instance.EBSVolumeType_GP2,
				Encrypted:           false,
			},
		},
		SecurityGroupSpecs: []instance.SecurityGroupSpec{
			{
				Name:          "http",
				PeerSpec:      instance.PeerAnyIpv4(),
				PortSpec:      instance.PortTcp(80),
				AllowFromSelf: false,
			},
			{
				Name:          "https",
				PeerSpec:      instance.PeerAnyIpv4(),
				PortSpec:      instance.PortTcp(443),
				AllowFromSelf: false,
			},
		},
	}

	// clone instance x number of times
	webServers := webServer.Clone(2)

	return instance.New(app.GetStack(), func(o *instance.Options) {
		o.InstanceNamePrefix = "example"
		o.SSHKeySpecs = &instance.SSHKeySpecs{
			// ssh key is already imported, we just reference its name
			Name: "devops-zex",
		}
		o.InstanceSpec = webServers
	})
}
