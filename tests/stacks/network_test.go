package stacks_test

import (
	"aws-ecs-fargate-go-cdk/tests/helpers"
	"testing"

	"github.com/aws/aws-cdk-go/awscdk/v2/assertions"
	"github.com/aws/jsii-runtime-go"
	"github.com/stretchr/testify/assert"

	"aws-ecs-fargate-go-cdk/internal/stacks"
)

// 既存の基本テスト（リファクタリング対応）
func TestNetworkStack_BasicCreation(t *testing.T) {
	// Given: テスト用アプリケーション作成
	app := helpers.CreateTestApp(&helpers.TestAppConfig{
		Environment: "dev",
		Region:      "ap-northeast-1",
		Account:     "123456789012",
	})

	// When: NetworkStackを作成
	stack := stacks.NewNetworkStack(app, "TestNetworkStack", &stacks.NetworkStackProps{
		Environment: "dev",
		VpcCidr:     "10.0.0.0/16",
	})

	// Then: 基本的なVPCが作成されることを確認
	template := assertions.Template_FromStack(stack, nil)
	template.ResourceCountIs(jsii.String("AWS::EC2::VPC"), jsii.Number(1))

	// VPCのCIDRブロックも確認
	template.HasResourceProperties(jsii.String("AWS::EC2::VPC"), map[string]interface{}{
		"CidrBlock": "10.0.0.0/16",
	})

	assert.NotNil(t, stack)
}

// 環境別設定テスト（リファクタリング対応）
func TestNetworkStack_EnvironmentConfigurations(t *testing.T) {
	testCases := []struct {
		name            string
		environment     string
		expectedVpcCidr string
		expectedMaxAzs  int
		subnetCount     int
	}{
		{
			name:            "Development Environment",
			environment:     "dev",
			expectedVpcCidr: "10.0.0.0/16",
			expectedMaxAzs:  2,
			subnetCount:     4, // Public x2, Private x2
		},
		{
			name:            "Staging Environment",
			environment:     "staging",
			expectedVpcCidr: "10.1.0.0/16",
			expectedMaxAzs:  2,
			subnetCount:     4, // Public x2, Private x2
		},
		{
			name:            "Production Environment",
			environment:     "prod",
			expectedVpcCidr: "10.2.0.0/16",
			expectedMaxAzs:  2,
			subnetCount:     6, // Public x3, Private x3, Database x3 (本番環境は3AZ + Database Subnet)
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Given: テスト用アプリケーション作成
			app := helpers.CreateTestApp(&helpers.TestAppConfig{
				Environment: tc.environment,
				Region:      "ap-northeast-1",
				Account:     "123456789012",
			})

			// When: NetworkStackを作成（VpcCidrを指定しない = 環境設定を使用）
			stack := stacks.NewNetworkStack(app, "TestNetworkStack", &stacks.NetworkStackProps{
				Environment: tc.environment,
				// VpcCidrを指定しない場合、環境設定が使用される
			})

			// Then: 期待されるリソースが作成されることを確認
			template := assertions.Template_FromStack(stack, nil)

			// VPC確認（環境設定からのCIDRを確認）
			template.ResourceCountIs(jsii.String("AWS::EC2::VPC"), jsii.Number(1))
			template.HasResourceProperties(jsii.String("AWS::EC2::VPC"), map[string]interface{}{
				"CidrBlock":          tc.expectedVpcCidr,
				"EnableDnsHostnames": true,
				"EnableDnsSupport":   true,
			})

			// Subnet数確認
			template.ResourceCountIs(jsii.String("AWS::EC2::Subnet"), jsii.Number(tc.subnetCount))

			// Internet Gateway確認
			template.ResourceCountIs(jsii.String("AWS::EC2::InternetGateway"), jsii.Number(1))

			// NAT Gateway確認（MaxAzsの数だけ作成される）
			template.ResourceCountIs(jsii.String("AWS::EC2::NatGateway"), jsii.Number(tc.expectedMaxAzs))

			// VPCアサーションヘルパーを使用
			vpcAssertions := helpers.NewVPCAssertions(stack)
			vpcAssertions.HasSubnetCount(tc.subnetCount).
				HasInternetGateway().
				HasNATGateways(tc.expectedMaxAzs)

			assert.NotNil(t, stack)
		})
	}
}

// VpcCidr上書きテスト
func TestNetworkStack_VpcCidrOverride(t *testing.T) {
	// Given
	app := helpers.CreateTestApp(&helpers.TestAppConfig{
		Environment: "dev",
	})

	customCidr := "172.16.0.0/16"

	// When: VpcCidrを明示的に指定
	stack := stacks.NewNetworkStack(app, "TestNetworkStack", &stacks.NetworkStackProps{
		Environment: "dev",
		VpcCidr:     customCidr,
	})

	// Then: 指定したCIDRが使用される
	template := assertions.Template_FromStack(stack, nil)
	template.HasResourceProperties(jsii.String("AWS::EC2::VPC"), map[string]interface{}{
		"CidrBlock": customCidr,
	})

	assert.NotNil(t, stack)
}

// セキュリティグループテスト（実用的なアプローチ）
func TestNetworkStack_SecurityGroups(t *testing.T) {
	// Given
	app := helpers.CreateTestApp(&helpers.TestAppConfig{
		Environment: "dev",
	})

	// When: NetworkStackを作成
	stack := stacks.NewNetworkStack(app, "TestNetworkStack", &stacks.NetworkStackProps{
		Environment: "dev",
		VpcCidr:     "10.0.0.0/16",
	})

	// Then: セキュリティグループの確認
	template := assertions.Template_FromStack(stack, nil)

	// 基本的なセキュリティグループ数の確認
	template.ResourceCountIs(jsii.String("AWS::EC2::SecurityGroup"), jsii.Number(3))

	// 各セキュリティグループの存在確認（基本プロパティのみ）
	template.HasResourceProperties(jsii.String("AWS::EC2::SecurityGroup"), map[string]interface{}{
		"GroupDescription": "Security group for ALB",
		"GroupName":        "Service-dev-ALB-SG",
	})

	template.HasResourceProperties(jsii.String("AWS::EC2::SecurityGroup"), map[string]interface{}{
		"GroupDescription": "Security group for ECS tasks",
		"GroupName":        "Service-dev-ECS-SG",
	})

	template.HasResourceProperties(jsii.String("AWS::EC2::SecurityGroup"), map[string]interface{}{
		"GroupDescription": "Security group for RDS database",
		"GroupName":        "Service-dev-RDS-SG",
	})

	// CloudFormationテンプレートを直接検証する実用的なアプローチ
	templateMap := template.ToJSON()
	resources := (*templateMap)["Resources"].(map[string]interface{})

	// セキュリティグループのIngressルール数を検証
	securityGroupCount := 0
	albIngressRules := 0
	ecsIngressRules := 0
	rdsIngressRules := 0

	for _, resource := range resources {
		resourceData := resource.(map[string]interface{})
		if resourceData["Type"] == "AWS::EC2::SecurityGroup" {
			securityGroupCount++

			if properties, ok := resourceData["Properties"].(map[string]interface{}); ok {
				if groupDesc, ok := properties["GroupDescription"].(string); ok {
					if ingress, ok := properties["SecurityGroupIngress"].([]interface{}); ok {
						switch groupDesc {
						case "Security group for ALB":
							albIngressRules = len(ingress)
						case "Security group for ECS tasks":
							ecsIngressRules = len(ingress)
						case "Security group for RDS database":
							rdsIngressRules = len(ingress)
						}
					}
				}
			}
		}
	}

	// セキュリティグループとルール数の実用的な検証
	assert.Equal(t, 3, securityGroupCount, "Expected 3 security groups")
	assert.Equal(t, 2, albIngressRules, "Expected 2 ingress rules for ALB (HTTP + HTTPS)")
	assert.Equal(t, 2, ecsIngressRules, "Expected 2 ingress rules for ECS (HTTP + Dynamic ports from ALB)")
	assert.Equal(t, 2, rdsIngressRules, "Expected 2 ingress rules for RDS (MySQL + Redis from ECS)")

	assert.NotNil(t, stack)
}

// ルートテーブルテスト（リファクタリング対応）
// ルートテーブルテスト（リファクタリング対応）
func TestNetworkStack_RouteTables(t *testing.T) {
	testCases := []struct {
		name             string
		environment      string
		expectedNATCount int
		expectedRTCount  int
	}{
		{
			name:             "Development Environment",
			environment:      "dev",
			expectedNATCount: 2, // 2AZ
			expectedRTCount:  4, // デフォルト1 + パブリック1 + プライベート2
		},
		{
			name:             "Production Environment",
			environment:      "prod",
			expectedNATCount: 2, // 実際の利用可能AZ数（2AZ）
			expectedRTCount:  6, // デフォルト1 + パブリック1 + プライベート2 + データベース2
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Given
			app := helpers.CreateTestApp(&helpers.TestAppConfig{
				Environment: tc.environment,
			})

			// When: NetworkStackを作成
			stack := stacks.NewNetworkStack(app, "TestNetworkStack", &stacks.NetworkStackProps{
				Environment: tc.environment,
			})

			// Then: ルートテーブルとNAT Gatewayの確認
			template := assertions.Template_FromStack(stack, nil)

			// Route Table数の確認
			template.ResourceCountIs(jsii.String("AWS::EC2::RouteTable"), jsii.Number(tc.expectedRTCount))

			// NAT Gateway数の確認
			template.ResourceCountIs(jsii.String("AWS::EC2::NatGateway"), jsii.Number(tc.expectedNATCount))

			assert.NotNil(t, stack)
		})
	}
}

// Cross-stack出力テスト（リファクタリング対応）
func TestNetworkStack_CrossStackExports(t *testing.T) {
	// Given
	app := helpers.CreateTestApp(&helpers.TestAppConfig{
		Environment: "production",
	})

	// When: NetworkStackを作成
	stack := stacks.NewNetworkStack(app, "TestNetworkStack", &stacks.NetworkStackProps{
		Environment: "prod", // 環境設定で有効な名前を使用
	})

	// Then: Cross-stack出力の確認
	template := assertions.Template_FromStack(stack, nil)

	// VpcId出力の確認
	template.HasOutput(jsii.String("VpcId"), map[string]interface{}{
		"Description": "VPC ID for Service",
		"Export": map[string]interface{}{
			"Name": "Service-prod-VpcId",
		},
	})

	// セキュリティグループ出力の確認
	template.HasOutput(jsii.String("ALBSecurityGroupId"), map[string]interface{}{
		"Description": "ALB Security Group ID",
		"Export": map[string]interface{}{
			"Name": "Service-prod-ALB-SG-Id",
		},
	})

	template.HasOutput(jsii.String("ECSSecurityGroupId"), map[string]interface{}{
		"Description": "ECS Security Group ID",
		"Export": map[string]interface{}{
			"Name": "Service-prod-ECS-SG-Id",
		},
	})

	template.HasOutput(jsii.String("RDSSecurityGroupId"), map[string]interface{}{
		"Description": "RDS Security Group ID",
		"Export": map[string]interface{}{
			"Name": "Service-prod-RDS-SG-Id",
		},
	})

	// サブネット出力の確認
	template.HasOutput(jsii.String("PrivateSubnetIds"), map[string]interface{}{
		"Description": "Private Subnet IDs",
		"Export": map[string]interface{}{
			"Name": "Service-prod-PrivateSubnetIds",
		},
	})

	template.HasOutput(jsii.String("PublicSubnetIds"), map[string]interface{}{
		"Description": "Public Subnet IDs",
		"Export": map[string]interface{}{
			"Name": "Service-prod-PublicSubnetIds",
		},
	})

	assert.NotNil(t, stack)
}

// タグ付けテスト（新規追加）
func TestNetworkStack_ResourceTags(t *testing.T) {
	// Given
	app := helpers.CreateTestApp(&helpers.TestAppConfig{
		Environment: "staging",
	})

	// When: NetworkStackを作成
	stack := stacks.NewNetworkStack(app, "TestNetworkStack", &stacks.NetworkStackProps{
		Environment: "staging",
	})

	// Then: 適切なタグが設定されることを確認
	template := assertions.Template_FromStack(stack, nil)

	// VPCのタグ確認
	template.HasResourceProperties(jsii.String("AWS::EC2::VPC"), map[string]interface{}{
		"Tags": assertions.Match_ArrayWith(&[]interface{}{
			map[string]interface{}{
				"Key":   "Environment",
				"Value": "staging",
			},
			map[string]interface{}{
				"Key":   "Project",
				"Value": "PracticeService",
			},
		}),
	})

	assert.NotNil(t, stack)
}

// エラーハンドリングテスト（新規追加）
func TestNetworkStack_InvalidEnvironment(t *testing.T) {
	// Given
	app := helpers.CreateTestApp(&helpers.TestAppConfig{
		Environment: "invalid",
	})

	// When & Then: 無効な環境名でpanicが発生することを確認
	assert.Panics(t, func() {
		stacks.NewNetworkStack(app, "TestNetworkStack", &stacks.NetworkStackProps{
			Environment: "invalid-env",
		})
	}, "Should panic with invalid environment")
}
