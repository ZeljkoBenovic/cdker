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
		//VPC:                nil,
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
		// mount EBS volume on boot
		BashUserData: []string{
			"mkdir /home/ubuntu/data",
			"yes | mkfs.ext4 /dev/nvme1n1",
			"mount /dev/nvme1n1 /home/ubuntu/data",
			"uuid=$(sudo blkid /dev/nvme1n1 | sed -n 's/.*UUID=\\\"\\([^\\\"]*\\)\\\".*/\\1/p')",
			"bash -c \"echo 'UUID=${uuid}     /home/ubuntu/data       ext4   defaults' >> /etc/fstab\"",
			"sudo chown -R ubuntu. /home/ubuntu/data",
		},
		// replace VM if user data changes
		UserDataCausesReplacement: true,
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
