package instance

import (
	"fmt"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsec2"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
	"github.com/xyproto/randomstring"
)

type Instances interface {
	GetInstances() []awsec2.Instance
	Deploy()
}

type instances struct {
	stack    constructs.Construct
	ec2s     []awsec2.Instance
	vpc      awsec2.IVpc
	secGroup awsec2.SecurityGroup
	init     awsec2.InitServiceRestartHandle

	opts *Options
}

func New(stack constructs.Construct, opts ...func(*Options)) Instances {
	o := &Options{
		InstanceNamePrefix: "ec2-instance",
		InstanceSpec: []InstanceSpec{
			{
				Class:      InstanceClass_T3,
				Size:       InstanceSize_SMALL,
				SubnetType: SubnetType_PUBLIC,
				AMI:        Ubuntu20,
				// default vpc
				VPC:            nil,
				AssociatePubIP: false,
				StorageSpecs: []StorageSpec{
					{
						Size:                10,
						Name:                "/dev/sdf",
						DeleteOnTermination: true,
					},
				},
				SecurityGroupSpecs: []SecurityGroupSpec{
					{
						Name:          "ec2-instance-secrutity-group",
						PeerSpec:      PeerAnyIpv4(),
						PortSpec:      PortTcp(22),
						AllowFromSelf: true,
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
			},
		},
	}

	for _, f := range opts {
		f(o)
	}

	return &instances{
		stack: stack,
		opts:  o,
		init:  awsec2.NewInitServiceRestartHandle(),
	}
}

func (i *instances) Deploy() {
	if i.opts.SSHKeySpecs != nil {
		i.importSSHKey(i.opts.SSHKeySpecs.Name, i.opts.SSHKeySpecs.PublicKey)
	}

	i.deployEC2instances()
	i.displayOutput()
}

func (i *instances) GetInstances() []awsec2.Instance {
	return i.ec2s
}

func (i *instances) SetInstanceName(name string) {
	i.opts.InstanceNamePrefix = name
}

func (i *instances) SetInstancesSpec(spec []InstanceSpec) {
	i.opts.InstanceSpec = spec
}

func (i *InstanceSpec) Clone(num int) []InstanceSpec {
	var (
		clones         = make([]InstanceSpec, 0)
		numberOfClones = make([]struct{}, num)
	)

	for range numberOfClones {
		clones = append(clones, *i)
	}

	return clones
}

func (i *instances) importSSHKey(keyName string, publicKey *string) {
	keyProps := &awsec2.CfnKeyPairProps{
		KeyName:           jsii.String(keyName),
		PublicKeyMaterial: publicKey,
	}

	if publicKey != nil {
		awsec2.NewCfnKeyPair(i.stack, keyProps.KeyName, keyProps)
	}
}

func (i *instances) getSecurityGroup(vpc awsec2.IVpc) awsec2.SecurityGroup {
	if i.secGroup != nil {
		return i.secGroup
	}

	i.secGroup = awsec2.NewSecurityGroup(i.stack, jsii.String(fmt.Sprintf("%s-%s", i.opts.InstanceNamePrefix, randomstring.CookieFriendlyString(5))), &awsec2.SecurityGroupProps{
		Vpc:                  vpc,
		AllowAllIpv6Outbound: jsii.Bool(true),
		AllowAllOutbound:     jsii.Bool(true),
	})

	for _, spec := range i.opts.InstanceSpec {
		for ind, ss := range spec.SecurityGroupSpecs {
			i.secGroup.AddIngressRule(ss.PeerSpec, ss.PortSpec, jsii.String(fmt.Sprintf("%s-%d", ss.Name, ind)), nil)
		}

		if spec.SecurityGroupSpecs[0].AllowFromSelf {
			i.secGroup.AddIngressRule(i.secGroup, awsec2.Port_AllTraffic(), jsii.String("allow from self"), nil)
		}
	}

	return i.secGroup
}

func (i *instances) getVPC(vpc *VPCSpec) awsec2.IVpc {
	if i.vpc != nil {
		return i.vpc
	}

	// when no VPC specified, deploy on default
	switch vpc {
	case nil:
		i.vpc = awsec2.Vpc_FromLookup(i.stack, jsii.String(i.opts.InstanceNamePrefix+"-vpc"), &awsec2.VpcLookupOptions{
			IsDefault: jsii.Bool(true),
		})
	default:
		i.vpc = awsec2.Vpc_FromLookup(i.stack, jsii.String(i.opts.InstanceNamePrefix+"-vpc"), &awsec2.VpcLookupOptions{
			IsDefault: &vpc.IsDefault,
			Region:    &vpc.Region,
			VpcId:     &vpc.ID,
			VpcName:   &vpc.Name,
		})
	}

	return i.vpc
}

func (i *instances) getAMI(ami AMIType) awsec2.IMachineImage {
	// when no AMI specified return Ubuntu20.04 as default
	switch ami {
	default:
		return awsec2.MachineImage_Lookup(&awsec2.LookupMachineImageProps{
			Name: jsii.String("ubuntu/images/hvm-ssd/ubuntu-focal-20.04-amd64-server-*"),
			Owners: &[]*string{
				jsii.String("099720109477"),
			},
			Filters: &map[string]*[]*string{
				"virtualization-type": {
					jsii.String("hvm"),
				},
			},
		})
	}
}

func (i *instances) displayOutput() {
	for ind, ec2 := range i.ec2s {
		awscdk.NewCfnOutput(i.stack,
			jsii.String(fmt.Sprintf("%s-%d-public-ip", i.opts.InstanceNamePrefix, ind)),
			&awscdk.CfnOutputProps{
				Value:       ec2.InstancePublicIp(),
				Description: jsii.String("instance public ip address"),
				ExportName:  jsii.String(fmt.Sprintf("PublicIP%d", ind)),
			})
		awscdk.NewCfnOutput(i.stack,
			jsii.String(fmt.Sprintf("%s-%d-private-ip", i.opts.InstanceNamePrefix, ind)),
			&awscdk.CfnOutputProps{
				Value:       ec2.InstancePrivateIp(),
				Description: jsii.String("instance private ip address"),
				ExportName:  jsii.String(fmt.Sprintf("PrivateIP%d", ind)),
			})
	}
}

func (i *instances) attachVolume(spec InstanceSpec, instanceId *string) {
	if spec.StorageSpecs == nil || len(spec.StorageSpecs) == 0 {
		return
	}

	for ind, s := range spec.StorageSpecs {
		volume := awsec2.NewVolume(i.stack, jsii.String(fmt.Sprintf("%s-%d", i.opts.InstanceNamePrefix, ind)), &awsec2.VolumeProps{
			Size:       awscdk.Size_Gibibytes(jsii.Number[float64](s.Size)),
			VolumeType: awsec2.EbsDeviceVolumeType_GP2,
		})

		awsec2.NewCfnVolumeAttachment(i.stack, jsii.String(fmt.Sprintf("%s-%d", i.opts.InstanceNamePrefix, ind)), &awsec2.CfnVolumeAttachmentProps{
			InstanceId: instanceId,
			VolumeId:   volume.VolumeId(),
			Device:     jsii.String(s.Name),
		})
	}
}

func (i *instances) getBlockStorage(spec InstanceSpec) *[]*awsec2.BlockDevice {
	var blockStore []*awsec2.BlockDevice

	for _, ss := range spec.StorageSpecs {
		blockStore = append(blockStore, &awsec2.BlockDevice{
			DeviceName: jsii.String(ss.Name),
			Volume: awsec2.BlockDeviceVolume_Ebs(jsii.Number[float64](ss.Size), &awsec2.EbsDeviceOptions{
				DeleteOnTermination: jsii.Bool(ss.DeleteOnTermination),
				VolumeType:          awsec2.EbsDeviceVolumeType(ss.VolumeType),
				Encrypted:           jsii.Bool(ss.Encrypted),
			}),
		})
	}

	return &blockStore
}

func (i *instances) getUserData(spec InstanceSpec) awsec2.UserData {
	if spec.BashUserData == nil || len(spec.BashUserData) == 0 || spec.BashUserData[0] == "" {
		return nil
	}

	mud := awsec2.NewMultipartUserData(&awsec2.MultipartUserDataOptions{})

	mud.AddUserDataPart(
		// default #!/bin/bash
		awsec2.MultipartUserData_ForLinux(&awsec2.LinuxUserDataOptions{}),
		awsec2.MultipartBody_SHELL_SCRIPT(),
		jsii.Bool(true),
	)

	for _, cmd := range spec.BashUserData {
		mud.AddCommands(&cmd)
	}

	return mud
}

func (i *instances) deployEC2instances() {

	//TODO: add cloud init support
	for ind, spec := range i.opts.InstanceSpec {
		ec2 := awsec2.NewInstance(i.stack, jsii.String(fmt.Sprintf("%s-%d", i.opts.InstanceNamePrefix, ind)), &awsec2.InstanceProps{
			InstanceType:                    awsec2.InstanceType_Of(awsec2.InstanceClass(spec.Class), awsec2.InstanceSize(spec.Size)),
			MachineImage:                    i.getAMI(spec.AMI),
			Vpc:                             i.getVPC(spec.VPC),
			AssociatePublicIpAddress:        &spec.AssociatePubIP,
			KeyName:                         jsii.String(i.opts.SSHKeySpecs.Name),
			InstanceName:                    jsii.String(fmt.Sprintf("%s-%d", i.opts.InstanceNamePrefix, ind)),
			PropagateTagsToVolumeOnCreation: jsii.Bool(true),
			SecurityGroup:                   i.getSecurityGroup(i.getVPC(spec.VPC)),
			SsmSessionPermissions:           jsii.Bool(true),
			VpcSubnets:                      &awsec2.SubnetSelection{SubnetType: awsec2.SubnetType(spec.SubnetType)},
			BlockDevices:                    i.getBlockStorage(spec),
			UserData:                        i.getUserData(spec),
			UserDataCausesReplacement:       &spec.UserDataCausesReplacement,
		})

		i.ec2s = append(i.ec2s, ec2)
	}
}
