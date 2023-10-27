package stack

import (
	"os"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/jsii-runtime-go"
)

type Stack interface {
	DeployResources(handlers ...DeployStack)
	SetStack(name string, stackOpts ...func(stackProps *awscdk.StackProps)) Stack
	GetStack() awscdk.Stack
}

// DeployStack all resource modules must implement this interface
type DeployStack interface {
	Deploy()
}

type stack struct {
	app   awscdk.App
	stack awscdk.Stack
}

func New() Stack {
	s := &stack{}

	s.app = awscdk.NewApp(nil)

	return s
}

func (s *stack) SetStack(name string, stackOpts ...func(stackProps *awscdk.StackProps)) Stack {
	props := &awscdk.StackProps{
		Env: &awscdk.Environment{
			Account: jsii.String(os.Getenv("CDK_DEFAULT_ACCOUNT")),
			Region:  jsii.String(os.Getenv("CDK_DEFAULT_REGION")),
		},
	}

	for _, f := range stackOpts {
		f(props)
	}

	s.stack = awscdk.NewStack(s.app, &name, props)

	return s
}

func (s *stack) GetStack() awscdk.Stack {
	return s.stack
}

func (s *stack) DeployResources(resources ...DeployStack) {
	defer jsii.Close()

	for _, resource := range resources {
		resource.Deploy()
	}
	
	s.app.Synth(nil)
}

func WithCredentials(accountID, region string) func(props *awscdk.StackProps) {
	return func(props *awscdk.StackProps) {
		props.Env = &awscdk.Environment{
			Account: &accountID,
			Region:  &region,
		}
	}
}
