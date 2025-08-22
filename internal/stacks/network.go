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

	// プロパティのバリデーション
	if props == nil {
		panic("NetworkStackProps is required")
	}
	if props.VpcCidr == "" {
		props.VpcCidr = "10.0.0.0/16" // デフォルト値
	}
	if props.Environment == "" {
		props.Environment = "dev" // デフォルト値
	}

	vpc := awsec2.NewVpc(stack, jsii.String("ServiceVPC"), &awsec2.VpcProps{
		IpAddresses: awsec2.IpAddresses_Cidr(jsii.String(props.VpcCidr)),
		MaxAzs:      jsii.Number(2),
		VpcName:     jsii.String("Service-" + props.Environment + "-VPC"),

		// Subnetの基本構成も追加
		SubnetConfiguration: &[]*awsec2.SubnetConfiguration{
			{
				Name:       jsii.String("Public"),
				SubnetType: awsec2.SubnetType_PUBLIC,
				CidrMask:   jsii.Number(24),
			},
			{
				Name:       jsii.String("Private"),
				SubnetType: awsec2.SubnetType_PRIVATE_WITH_EGRESS,
				CidrMask:   jsii.Number(24),
			},
		},
	})

	// VPCの出力（後のstackで参照できるように）
	awscdk.NewCfnOutput(stack, jsii.String("VpcId"), &awscdk.CfnOutputProps{
		Value:       vpc.VpcId(),
		Description: jsii.String("VPC ID for Service"),
		ExportName:  jsii.String("Service-" + props.Environment + "-VpcId"),
	})

	return stack
}
