package stacks

import (
	"aws-ecs-fargate-go-cdk/internal/config"
	networkConstruct "aws-ecs-fargate-go-cdk/internal/constructs"

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

// NetworkStack NetworkStackの構造体
type NetworkStack struct {
	awscdk.Stack
	Vpc            awsec2.Vpc
	SecurityGroups *networkConstruct.SecurityGroupsResult
	VpcIdOutput    awscdk.CfnOutput
}

// NewNetworkStack NetworkStackを作成
func NewNetworkStack(scope constructs.Construct, id string, props *NetworkStackProps) awscdk.Stack {
	var sprops awscdk.StackProps
	if props != nil {
		sprops = props.StackProps
	}
	stack := awscdk.NewStack(scope, &id, &sprops)

	// プロパティとバリデーション
	if props == nil {
		panic("NetworkStackProps is required")
	}
	if props.Environment == "" {
		props.Environment = "dev"
	}

	// 環境設定を取得
	envConfig, err := config.GetEnvironmentConfig(props.Environment)
	if err != nil {
		panic("Invalid environment: " + props.Environment)
	}

	networkConfig := config.GetNetworkConfig(props.Environment)

	// VpcCidrの設定（プロパティで指定されていない場合は環境設定を使用）
	if props.VpcCidr == "" {
		props.VpcCidr = envConfig.VpcCidr
	}

	// VPC作成
	vpc := awsec2.NewVpc(stack, jsii.String("ServiceVPC"), &awsec2.VpcProps{
		IpAddresses:        awsec2.IpAddresses_Cidr(jsii.String(props.VpcCidr)),
		MaxAzs:             jsii.Number(envConfig.MaxAzs),
		VpcName:            jsii.String("Service-" + props.Environment + "-VPC"),
		EnableDnsHostnames: jsii.Bool(networkConfig.EnableDNSHostnames),
		EnableDnsSupport:   jsii.Bool(networkConfig.EnableDNSSupport),

		// Subnetの設定
		SubnetConfiguration: createSubnetConfiguration(envConfig, networkConfig),

		// NAT Gateway設定
		NatGateways: func() *float64 {
			if envConfig.EnableNATGateway {
				return jsii.Number(envConfig.MaxAzs)
			}
			return jsii.Number(0)
		}(),
	})

	// VPCにタグを追加
	addVPCTags(vpc, envConfig)

	// セキュリティグループの作成
	securityGroups := networkConstruct.CreateSecurityGroups(stack, &networkConstruct.SecurityGroupsProps{
		Vpc:         vpc,
		Environment: props.Environment,
	})

	// Cross-stack出力の作成
	createStackOutputs(stack, vpc, securityGroups, props.Environment)

	return stack
}

// createSubnetConfiguration サブネット設定を作成
func createSubnetConfiguration(envConfig *config.EnvironmentConfig, networkConfig *config.NetworkConfig) *[]*awsec2.SubnetConfiguration {
	subnets := []*awsec2.SubnetConfiguration{
		{
			Name:       jsii.String("Public"),
			SubnetType: awsec2.SubnetType_PUBLIC,
			CidrMask:   jsii.Number(networkConfig.SubnetCidrMask),
		},
		{
			Name:       jsii.String("Private"),
			SubnetType: awsec2.SubnetType_PRIVATE_WITH_EGRESS,
			CidrMask:   jsii.Number(networkConfig.SubnetCidrMask),
		},
	}

	// 本番環境では分離されたデータベースサブネットを追加
	if envConfig.Name == "production" {
		subnets = append(subnets, &awsec2.SubnetConfiguration{
			Name:       jsii.String("Database"),
			SubnetType: awsec2.SubnetType_PRIVATE_ISOLATED,
			CidrMask:   jsii.Number(networkConfig.SubnetCidrMask),
		})
	}

	return &subnets
}

// addVPCTags VPCにタグを追加
func addVPCTags(vpc awsec2.Vpc, envConfig *config.EnvironmentConfig) {
	for key, value := range envConfig.Tags {
		awscdk.Tags_Of(vpc).Add(jsii.String(key), jsii.String(value), nil)
	}

	// 追加のネットワーク固有タグ
	awscdk.Tags_Of(vpc).Add(jsii.String("Component"), jsii.String("Network"), nil)
	awscdk.Tags_Of(vpc).Add(jsii.String("ManagedBy"), jsii.String("CDK"), nil)
}

// createStackOutputs Cross-stack出力を作成
func createStackOutputs(stack awscdk.Stack, vpc awsec2.Vpc, securityGroups *networkConstruct.SecurityGroupsResult, environment string) {
	// VPC ID出力
	awscdk.NewCfnOutput(stack, jsii.String("VpcId"), &awscdk.CfnOutputProps{
		Value:       vpc.VpcId(),
		Description: jsii.String("VPC ID for Service"),
		ExportName:  jsii.String("Service-" + environment + "-VpcId"),
	})

	// セキュリティグループID出力
	awscdk.NewCfnOutput(stack, jsii.String("ALBSecurityGroupId"), &awscdk.CfnOutputProps{
		Value:       securityGroups.ALBSecurityGroup.SecurityGroupId(),
		Description: jsii.String("ALB Security Group ID"),
		ExportName:  jsii.String("Service-" + environment + "-ALB-SG-Id"),
	})

	awscdk.NewCfnOutput(stack, jsii.String("ECSSecurityGroupId"), &awscdk.CfnOutputProps{
		Value:       securityGroups.ECSSecurityGroup.SecurityGroupId(),
		Description: jsii.String("ECS Security Group ID"),
		ExportName:  jsii.String("Service-" + environment + "-ECS-SG-Id"),
	})

	awscdk.NewCfnOutput(stack, jsii.String("RDSSecurityGroupId"), &awscdk.CfnOutputProps{
		Value:       securityGroups.RDSSecurityGroup.SecurityGroupId(),
		Description: jsii.String("RDS Security Group ID"),
		ExportName:  jsii.String("Service-" + environment + "-RDS-SG-Id"),
	})

	// サブネット出力（後のStackで使用）
	privateSubnetIds := make([]*string, len(*vpc.PrivateSubnets()))
	for i, subnet := range *vpc.PrivateSubnets() {
		privateSubnetIds[i] = subnet.SubnetId()
	}

	publicSubnetIds := make([]*string, len(*vpc.PublicSubnets()))
	for i, subnet := range *vpc.PublicSubnets() {
		publicSubnetIds[i] = subnet.SubnetId()
	}

	awscdk.NewCfnOutput(stack, jsii.String("PrivateSubnetIds"), &awscdk.CfnOutputProps{
		Value:       awscdk.Fn_Join(jsii.String(","), &privateSubnetIds),
		Description: jsii.String("Private Subnet IDs"),
		ExportName:  jsii.String("Service-" + environment + "-PrivateSubnetIds"),
	})

	awscdk.NewCfnOutput(stack, jsii.String("PublicSubnetIds"), &awscdk.CfnOutputProps{
		Value:       awscdk.Fn_Join(jsii.String(","), &publicSubnetIds),
		Description: jsii.String("Public Subnet IDs"),
		ExportName:  jsii.String("Service-" + environment + "-PublicSubnetIds"),
	})
}
