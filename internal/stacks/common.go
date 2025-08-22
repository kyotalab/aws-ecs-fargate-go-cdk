package stacks

import (
	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsec2"
	"github.com/aws/jsii-runtime-go"
)

// VPCReferenceProps VPC参照に必要な共通プロパティのインターフェース
type VPCReferenceProps interface {
	GetEnvironment() string
	GetVpcId() string
	IsTestEnvironment() bool
}

// getVPCReference Cross-stack参照でVPCを取得（テスト環境対応）
func GetVPCReference[T VPCReferenceProps](stack awscdk.Stack, props T) awsec2.IVpc {
	// テスト環境の場合は、モックVPCを作成
	if props.IsTestEnvironment() {
		return createMockVPC(stack, props.GetEnvironment())
	}

	// 実際のVPC IDが提供された場合（実環境でのテスト）
	if props.GetVpcId() != "" && props.GetVpcId() != "vpc-from-network-stack" && props.GetVpcId() != "vpc-12345" {
		return awsec2.Vpc_FromLookup(stack, jsii.String("ExistingVPC"), &awsec2.VpcLookupOptions{
			VpcId: jsii.String(props.GetVpcId()),
		})
	}

	// 実環境デプロイ：VPC Lookupでタグベース検索（推奨アプローチ）
	// この方法により、ルートテーブルIDも自動的に取得され、警告が解決されます
	// return awsec2.Vpc_FromLookup(stack, jsii.String("NetworkStackVPC"), &awsec2.VpcLookupOptions{
	// 	Tags: &map[string]*string{
	// 		"Environment": jsii.String("production"), // 実際のタグ値
	// 		"Component":   jsii.String("Network"),    // 実際のタグ値
	// 		"ManagedBy":   jsii.String("CDK"),        // 実際のタグ値
	// 	},
	// })

	// Cross-stack参照版（NetworkStackと同時synthが可能）
	envName := props.GetEnvironment()
	if envName == "prod" {
		envName = "production"
	}

	return createVPCFromCrossStackReference(stack, envName)
}

// createMockVPC テスト環境用のモックVPCを作成（環境別ID対応）
func createMockVPC(stack awscdk.Stack, environment string) awsec2.IVpc {
	return awsec2.Vpc_FromVpcAttributes(stack, jsii.String("TestVPC"), &awsec2.VpcAttributes{
		VpcId: jsii.String("vpc-test-" + environment + "-12345"),
		AvailabilityZones: &[]*string{
			jsii.String("ap-northeast-1a"),
			jsii.String("ap-northeast-1c"),
		},
		PrivateSubnetIds: &[]*string{
			jsii.String("subnet-test-private-1-" + environment),
			jsii.String("subnet-test-private-2-" + environment),
		},
		PublicSubnetIds: &[]*string{
			jsii.String("subnet-test-public-1-" + environment),
			jsii.String("subnet-test-public-2-" + environment),
		},
		// Isolatedサブネットも追加
		IsolatedSubnetIds: &[]*string{
			jsii.String("subnet-test-isolated-1-" + environment),
			jsii.String("subnet-test-isolated-2-" + environment),
		},
	})
}

// createVPCFromCrossStackReference Cross-stack参照でVPCを構築
func createVPCFromCrossStackReference(stack awscdk.Stack, envName string) awsec2.IVpc {
	vpcId := awscdk.Fn_ImportValue(jsii.String("Service-" + envName + "-VpcId"))

	// サブネットIDを個別にインポート
	privateSubnetId1 := awscdk.Fn_Select(jsii.Number(0),
		awscdk.Fn_Split(jsii.String(","),
			awscdk.Fn_ImportValue(jsii.String("Service-"+envName+"-PrivateSubnetIds")), nil))
	privateSubnetId2 := awscdk.Fn_Select(jsii.Number(1),
		awscdk.Fn_Split(jsii.String(","),
			awscdk.Fn_ImportValue(jsii.String("Service-"+envName+"-PrivateSubnetIds")), nil))

	publicSubnetId1 := awscdk.Fn_Select(jsii.Number(0),
		awscdk.Fn_Split(jsii.String(","),
			awscdk.Fn_ImportValue(jsii.String("Service-"+envName+"-PublicSubnetIds")), nil))
	publicSubnetId2 := awscdk.Fn_Select(jsii.Number(1),
		awscdk.Fn_Split(jsii.String(","),
			awscdk.Fn_ImportValue(jsii.String("Service-"+envName+"-PublicSubnetIds")), nil))

	return awsec2.Vpc_FromVpcAttributes(stack, jsii.String("ImportedVPC"), &awsec2.VpcAttributes{
		VpcId: vpcId,
		AvailabilityZones: &[]*string{
			jsii.String("ap-northeast-1a"),
			jsii.String("ap-northeast-1c"),
		},

		// 個別のサブネットIDを指定
		PrivateSubnetIds: &[]*string{
			privateSubnetId1,
			privateSubnetId2,
		},
		PublicSubnetIds: &[]*string{
			publicSubnetId1,
			publicSubnetId2,
		},

		// ルートテーブルIDも追加（警告軽減のため）
		PrivateSubnetRouteTableIds: &[]*string{
			awscdk.Fn_Select(jsii.Number(0),
				awscdk.Fn_Split(jsii.String(","),
					awscdk.Fn_ImportValue(jsii.String("Service-"+envName+"-PrivateRouteTableIds")), nil)),
			awscdk.Fn_Select(jsii.Number(1),
				awscdk.Fn_Split(jsii.String(","),
					awscdk.Fn_ImportValue(jsii.String("Service-"+envName+"-PrivateRouteTableIds")), nil)),
		},
		PublicSubnetRouteTableIds: &[]*string{
			awscdk.Fn_Select(jsii.Number(0),
				awscdk.Fn_Split(jsii.String(","),
					awscdk.Fn_ImportValue(jsii.String("Service-"+envName+"-PublicRouteTableIds")), nil)),
			awscdk.Fn_Select(jsii.Number(1),
				awscdk.Fn_Split(jsii.String(","),
					awscdk.Fn_ImportValue(jsii.String("Service-"+envName+"-PublicRouteTableIds")), nil)),
		},
	})
}
