package instance

import "github.com/aws/aws-cdk-go/awscdk/v2/awsec2"

type Options struct {
	InstanceNamePrefix string
	SSHKeySpecs        *SSHKeySpecs
	InstanceSpec       []InstanceSpec
}

type InstanceSpec struct {
	Class                     InstanceClass
	Size                      InstanceSize
	SubnetType                SubnetType
	AMI                       AMIType
	VPC                       *VPCSpec
	AssociatePubIP            bool
	StorageSpecs              []StorageSpec
	SecurityGroupSpecs        []SecurityGroupSpec
	BashUserData              []string
	UserDataCausesReplacement bool
}

type StorageSpec struct {
	Size                float64
	Name                string
	DeleteOnTermination bool
	VolumeType          EBSVolumeType
	Encrypted           bool
}

type SecurityGroupSpec struct {
	Name          string
	PeerSpec      SecurityGroupPeer
	PortSpec      SecurityGroupPort
	AllowFromSelf bool
}

type SSHKeySpecs struct {
	Name      string
	PublicKey *string
}

type VPCSpec struct {
	ID        string
	Name      string
	Region    string
	IsDefault bool
}

type AMIType int

type InstanceClass string
type InstanceSize string
type SubnetType string
type EBSVolumeType string

type SecurityGroupPeer interface {
	awsec2.IPeer
}
type SecurityGroupPort interface {
	awsec2.Port
}

const Ubuntu20 AMIType = iota + 1

const (
	InstanceClass_T3 = InstanceClass(awsec2.InstanceClass_T3)
	InstanceClass_M5 = InstanceClass(awsec2.InstanceClass_M5)

	InstanceSize_SMALL   = InstanceSize(awsec2.InstanceSize_SMALL)
	InstanceSize_MEDIUM  = InstanceSize(awsec2.InstanceSize_MEDIUM)
	InstanceSize_LARGE   = InstanceSize(awsec2.InstanceSize_LARGE)
	InstanceSize_xLARGE  = InstanceSize(awsec2.InstanceSize_XLARGE)
	InstanceSize_XLARGE2 = InstanceSize(awsec2.InstanceSize_XLARGE2)

	SubnetType_PUBLIC           = SubnetType(awsec2.SubnetType_PUBLIC)
	SubnetType_PRIVATE          = SubnetType(awsec2.SubnetType_PRIVATE_WITH_EGRESS)
	SubnetType_PRIVATE_ISOLATED = SubnetType(awsec2.SubnetType_PRIVATE_ISOLATED)

	EBSVolumeType_GP2      = EBSVolumeType(awsec2.EbsDeviceVolumeType_GP2)
	EBSVolumeType_GP3      = EBSVolumeType(awsec2.EbsDeviceVolumeType_GP3)
	EBSVolumeType_STANDARD = EBSVolumeType(awsec2.EbsDeviceVolumeType_STANDARD)
	EBSVolumeType_IO1      = EBSVolumeType(awsec2.EbsDeviceVolumeType_IO1)
	EBSVolumeType_IO2      = EBSVolumeType(awsec2.EbsDeviceVolumeType_IO2)
)

func PeerAnyIpv4() SecurityGroupPeer {
	return awsec2.Peer_AnyIpv4()
}

func PortTcp(port int) SecurityGroupPort {
	flPort := float64(port)
	return awsec2.Port_Tcp(&flPort)
}
