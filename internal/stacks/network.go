package stacks

import (
	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsec2"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

// NetworkStackProps NetworkStackのプロパティ
type NetworkStackProps struct {
	awscdk.StackProps
	Environment string
	VpcCidr     string
}

// NewNetworkStack NetworkStackを作成
func NewNetworkStack(scope constructs.Construct, id string, props *NetworkStackProps) awscdk.Stack {
	var sprops awscdk.StackProps
	if props != nil {
		sprops = props.StackProps
	}
	stack := awscdk.NewStack(scope, &id, &sprops)

	// 最小限のVPC作成（Green Phaseでテストを通すため）
	vpcCidr := "10.0.0.0/16" // デフォルト値
	if props != nil && props.VpcCidr != "" {
		vpcCidr = props.VpcCidr
	}

	awsec2.NewVpc(stack, jsii.String("ServiceVPC"), &awsec2.VpcProps{
		IpAddresses: awsec2.IpAddresses_Cidr(jsii.String(vpcCidr)),
		MaxAzs:      jsii.Number(2),
		VpcName:     jsii.String("Service-" + props.Environment + "-VPC"),
	})

	return stack
}
