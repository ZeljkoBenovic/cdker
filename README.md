# CDKER (ci-di-key-er)
`cdker` is an AWS CDK framework for GO used to quickly deploy resources on AWS using CDK and AWS CloudFormation.    

## Usage
```go

import (
    "github.com/ZeljkoBenovic/cdker/modules/instance"
    "github.com/ZeljkoBenovic/cdker/stack"
    
    "github.com/aws/aws-cdk-go/awscdk/v2/awsec2"
    "github.com/aws/jsii-runtime-go"
)


app := stack.New().SetStack("xxxxxx", stack.WithCredentials("xxxxxxxxxxx", "xxxxxxx"))
	app.DeployResources(
		instance.New(app.GetStack(), func(o *instance.Options) {
			o.InstanceNamePrefix = "Web Server"
			o.SSHKeySpecs = &instance.SSHKeySpecs{
				Name: "dev-key",
			}
			o.InstanceSpec = []instance.InstanceSpec{
                Class:          awsec2.InstanceClass_M5,
                Size:           awsec2.InstanceSize_XLARGE2,
                SubnetType:     awsec2.SubnetType_PUBLIC,
                AMI:            instance.Ubuntu20,
                VPC:            instance.VPCDefault,
                AssociatePubIP: true,
                SecurityGroupSpecs: []instance.SecurityGroupSpec{
                    {
                        Name:     "http",
                        PeerSpec: awsec2.Peer_AnyIpv4(),
                        PortSpec: awsec2.Port_Tcp(jsii.Number[float64](80)),
                    },
                    {
                        Name:     "ssh",
                        PeerSpec: awsec2.Peer_AnyIpv4(),
                        PortSpec: awsec2.Port_Tcp(jsii.Number[float64](22)),
                    },
                },
            }
		}),
	)

```