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

	// セキュリティグループの作成
	// 1. ALB用セキュリティグループ
	albSecurityGroup := awsec2.NewSecurityGroup(stack, jsii.String("ALBSecurityGroup"), &awsec2.SecurityGroupProps{
		Vpc:               vpc,
		Description:       jsii.String("Security group for ALB"),
		SecurityGroupName: jsii.String("Service-" + props.Environment + "-ALB-SG"),
		AllowAllOutbound:  jsii.Bool(true),
	})

	// ALBのIngressルール
	albSecurityGroup.AddIngressRule(
		awsec2.Peer_AnyIpv4(),
		awsec2.Port_Tcp(jsii.Number(80)),
		jsii.String("Allow HTTP traffic"),
		jsii.Bool(false),
	)

	albSecurityGroup.AddIngressRule(
		awsec2.Peer_AnyIpv4(),
		awsec2.Port_Tcp(jsii.Number(443)),
		jsii.String("Allow HTTPS traffic"),
		jsii.Bool(false),
	)

	// 2. ECS用セキュリティグループ
	ecsSecurityGroup := awsec2.NewSecurityGroup(stack, jsii.String("ECSSecurityGroup"), &awsec2.SecurityGroupProps{
		Vpc:               vpc,
		Description:       jsii.String("Security group for ECS tasks"),
		SecurityGroupName: jsii.String("Service-" + props.Environment + "-ECS-SG"),
		AllowAllOutbound:  jsii.Bool(true),
	})

	// ECSのIngressルール（ALBからのアクセスのみ許可）
	ecsSecurityGroup.AddIngressRule(
		awsec2.Peer_SecurityGroupId(albSecurityGroup.SecurityGroupId(), nil),
		awsec2.Port_Tcp(jsii.Number(80)),
		jsii.String("Allow traffic from ALB"),
		jsii.Bool(false),
	)

	// 3. RDS用セキュリティグループ
	rdsSecurityGroup := awsec2.NewSecurityGroup(stack, jsii.String("RDSSecurityGroup"), &awsec2.SecurityGroupProps{
		Vpc:               vpc,
		Description:       jsii.String("Security group for RDS database"),
		SecurityGroupName: jsii.String("Service-" + props.Environment + "-RDS-SG"),
		AllowAllOutbound:  jsii.Bool(false), // RDSは外部通信不要
	})

	// RDSのIngressルール（ECSからのアクセスのみ許可）
	rdsSecurityGroup.AddIngressRule(
		awsec2.Peer_SecurityGroupId(ecsSecurityGroup.SecurityGroupId(), nil),
		awsec2.Port_Tcp(jsii.Number(3306)),
		jsii.String("Allow MySQL traffic from ECS"),
		jsii.Bool(false),
	)

	// Redis用ポートも追加
	rdsSecurityGroup.AddIngressRule(
		awsec2.Peer_SecurityGroupId(ecsSecurityGroup.SecurityGroupId(), nil),
		awsec2.Port_Tcp(jsii.Number(6379)),
		jsii.String("Allow Redis traffic from ECS"),
		jsii.Bool(false),
	)

	// VPCの出力（後のstackで参照できるように）
	awscdk.NewCfnOutput(stack, jsii.String("VpcId"), &awscdk.CfnOutputProps{
		Value:       vpc.VpcId(),
		Description: jsii.String("VPC ID for Service"),
		ExportName:  jsii.String("Service-" + props.Environment + "-VpcId"),
	})

	return stack
}
