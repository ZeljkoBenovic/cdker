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

type Options struct {
	InstanceNamePrefix string
	SSHKeySpecs        *SSHKeySpecs
	InstanceSpec       []InstanceSpec
}

type InstanceSpec struct {
	Class              awsec2.InstanceClass
	Size               awsec2.InstanceSize
	SubnetType         awsec2.SubnetType
	AMI                AMIType
	VPC                VPCType
	AssociatePubIP     bool
	StorageSpecs       []StorageSpec
	SecurityGroupSpecs []SecurityGroupSpec
}

type StorageSpec struct {
	Size                float64
	Name                string
	DeleteOnTermination bool
	VolumeType          awsec2.EbsDeviceVolumeType
	Encrypted           bool
}

type SecurityGroupSpec struct {
	Name          string
	PeerSpec      awsec2.IPeer
	PortSpec      awsec2.Port
	AllowFromSelf bool
}

type SSHKeySpecs struct {
	Name      string
	PublicKey *string
}

type AMIType int
type VPCType int

const Ubuntu20 AMIType = iota
const VPCDefault VPCType = iota

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
				Class:          awsec2.InstanceClass_T3,
				Size:           awsec2.InstanceSize_SMALL,
				SubnetType:     awsec2.SubnetType_PUBLIC,
				AMI:            Ubuntu20,
				VPC:            VPCDefault,
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
						PeerSpec:      awsec2.Peer_AnyIpv4(),
						PortSpec:      awsec2.Port_Tcp(jsii.Number[float64](22)),
						AllowFromSelf: true,
					},
				},
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

func (i *instances) SetInstanceName(name string) {
	i.opts.InstanceNamePrefix = name
}

func (i *instances) SetInstancesSpec(spec []InstanceSpec) {
	i.opts.InstanceSpec = spec
}

func (i *instances) GetInstances() []awsec2.Instance {
	return i.ec2s
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
		Description:          jsii.String(fmt.Sprintf("%s-secgroup", i.opts.InstanceNamePrefix)),
		SecurityGroupName:    jsii.String(fmt.Sprintf("%s-secgroup", i.opts.InstanceNamePrefix)),
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

func (i *instances) getVPC(vpc VPCType) awsec2.IVpc {
	if i.vpc != nil {
		return i.vpc
	}

	switch vpc {
	//TODO: remove enum, check if nil instead
	case VPCDefault:
		i.vpc = awsec2.Vpc_FromLookup(i.stack, jsii.String(i.opts.InstanceNamePrefix+"-vpc"), &awsec2.VpcLookupOptions{
			IsDefault: jsii.Bool(true),
		})
	}

	return i.vpc
}

func (i *instances) getAMI(ami AMIType) awsec2.IMachineImage {
	//TODO: remove enum, check if nil instead
	switch ami {
	case Ubuntu20:
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
	default:
		return nil
	}
}

func (i *instances) displayOutput() {
	for ind, ec2 := range i.ec2s {
		awscdk.NewCfnOutput(i.stack,
			jsii.String(fmt.Sprintf("%s-%d-public-ip", i.opts.InstanceNamePrefix, ind)),
			&awscdk.CfnOutputProps{
				Value:       ec2.InstancePublicIp(),
				Description: jsii.String("instance public ip address"),
				ExportName:  jsii.String(fmt.Sprintf("%s-%d-pub", i.opts.InstanceNamePrefix, ind)),
			})
	}

	for ind, ec2 := range i.ec2s {
		awscdk.NewCfnOutput(i.stack,
			jsii.String(fmt.Sprintf("%s-%d-private-ip", i.opts.InstanceNamePrefix, ind)),
			&awscdk.CfnOutputProps{
				Value:       ec2.InstancePrivateIp(),
				Description: jsii.String("instance private ip address"),
				ExportName:  jsii.String(fmt.Sprintf("%s-%d-priv", i.opts.InstanceNamePrefix, ind)),
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
				VolumeType:          ss.VolumeType,
				Encrypted:           jsii.Bool(ss.Encrypted),
			}),
		})
	}

	return &blockStore
}

func (i *instances) deployEC2instances() {

	//TODO: add cloud init support
	for ind, spec := range i.opts.InstanceSpec {
		ec2 := awsec2.NewInstance(i.stack, jsii.String(fmt.Sprintf("%s-%d", i.opts.InstanceNamePrefix, ind)), &awsec2.InstanceProps{
			InstanceType:                    awsec2.InstanceType_Of(spec.Class, spec.Size),
			MachineImage:                    i.getAMI(spec.AMI),
			Vpc:                             i.getVPC(spec.VPC),
			AssociatePublicIpAddress:        &spec.AssociatePubIP,
			KeyName:                         jsii.String(i.opts.SSHKeySpecs.Name),
			InstanceName:                    jsii.String(fmt.Sprintf("%s-%d", i.opts.InstanceNamePrefix, ind)),
			PropagateTagsToVolumeOnCreation: jsii.Bool(true),
			SecurityGroup:                   i.getSecurityGroup(i.getVPC(spec.VPC)),
			SsmSessionPermissions:           jsii.Bool(true),
			VpcSubnets:                      &awsec2.SubnetSelection{SubnetType: spec.SubnetType},
			BlockDevices:                    i.getBlockStorage(spec),
		})

		// TODO: attach volumes
		//i.attachVolume(spec, ec2.InstanceId())

		i.ec2s = append(i.ec2s, ec2)
	}
}
