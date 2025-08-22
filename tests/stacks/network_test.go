package stacks_test

import (
	"aws-ecs-fargate-go-cdk/tests/helpers"
	"testing"

	"github.com/aws/aws-cdk-go/awscdk/v2/assertions"
	"github.com/aws/jsii-runtime-go"
	"github.com/stretchr/testify/assert"

	"aws-ecs-fargate-go-cdk/internal/stacks"
)

func TestNetworkStack_BasicCreation(t *testing.T) {
	// Given: テスト用アプリケーション
	app := helpers.CreateTestApp(&helpers.TestAppConfig{
		Environment: "dev",
		Region:      "ap-northeast-1",
		Account:     "123456789012",
	})

	// When: NetworkStackを作成（この時点では存在しないのでコンパイルエラー）
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

func TestNetworkStack_EnvironmentConfigurations(t *testing.T) {
	// テーブル駆動テスト用のテストケース
	testCases := []struct {
		name        string
		environment string
		vpcCidr     string
		subnetCount int
	}{
		{
			name:        "Development Environment",
			environment: "dev",
			vpcCidr:     "10.0.0.0/16",
			subnetCount: 4, // Public x2, Private x2

		},
		{
			name:        "Staging Environment",
			environment: "staging",
			vpcCidr:     "10.1.0.0/16",
			subnetCount: 4, // Public x2, Private x2
		},
		{
			name:        "Production Environment",
			environment: "prod",
			vpcCidr:     "10.2.0.0/16",
			subnetCount: 4, // Public x2, Private x2
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
			// When: NetworkStackを作成（実装後にコメントアウト解除）
			stack := stacks.NewNetworkStack(app, "TestNetworkStack", &stacks.NetworkStackProps{
				Environment: tc.environment,
				VpcCidr:     tc.vpcCidr,
			})

			// Then: 期待されるリソースが作成されることを確認
			template := assertions.Template_FromStack(stack, nil)

			// VPC確認
			template.ResourceCountIs(jsii.String("AWS::EC2::VPC"), jsii.Number(1))
			template.HasResourceProperties(jsii.String("AWS::EC2::VPC"), map[string]interface{}{
				"CidrBlock": tc.vpcCidr,
			})

			// Subnet数確認（この時点では失敗する可能性がある = Red Phase）
			template.ResourceCountIs(jsii.String("AWS::EC2::Subnet"), jsii.Number(tc.subnetCount))

			// Internet Gateway確認
			template.ResourceCountIs(jsii.String("AWS::EC2::InternetGateway"), jsii.Number(1))

			vpcAssersions := helpers.NewVPCAssertions(stack)
			vpcAssersions.HasSubnetCount(tc.subnetCount).HasInternetGateway()

			assert.NotNil(t, stack)
		})
	}
}

// 🔴 Red Phase: セキュリティグループテスト（まだ実装されていない機能）
func TestNetworkStack_SecurityGroups(t *testing.T) {
	// Given
	app := helpers.CreateTestApp(&helpers.TestAppConfig{
		Environment: "test",
	})

	// When: NetworkStackを作成
	stack := stacks.NewNetworkStack(app, "TestNetworkStack", &stacks.NetworkStackProps{
		Environment: "test",
		VpcCidr:     "10.0.0.0/16",
	})

	// Then: セキュリティグループの確認
	template := assertions.Template_FromStack(stack, nil)

	// 期待: ALB用、ECS用、RDS用のセキュリティグループ
	template.ResourceCountIs(jsii.String("AWS::EC2::SecurityGroup"), jsii.Number(3))

	// 具体的なセキュリティグループルールの確認
	template.HasResourceProperties(jsii.String("AWS::EC2::SecurityGroup"), map[string]interface{}{
		"GroupDescription": "Security group for ALB",
		"SecurityGroupIngress": []interface{}{
			map[string]interface{}{
				"IpProtocol": "tcp",
				"FromPort":   80,
				"ToPort":     80,
				"CidrIp":     "0.0.0.0/0",
			},
			map[string]interface{}{
				"IpProtocol": "tcp",
				"FromPort":   443,
				"ToPort":     443,
				"CidrIp":     "0.0.0.0/0",
			},
		},
	})

	assert.NotNil(t, app)
}

func TestNetworkStack_RouteTables(t *testing.T) {
	// Given
	app := helpers.CreateTestApp(nil)

	// When: NetworkStackを作成
	stack := stacks.NewNetworkStack(app, "TestNetworkStack", &stacks.NetworkStackProps{
		Environment: "test",
		VpcCidr:     "10.0.0.0/16",
	})

	// Then: ルートテーブルの確認
	template := assertions.Template_FromStack(stack, nil)

	// 期待: Public用1つ、Private用2つ（AZ別） = 合計3つ + デフォルト1つ = 4つ
	template.ResourceCountIs(jsii.String("AWS::EC2::RouteTable"), jsii.Number(4))

	// NAT Gatewayの確認（各AZに1つずつ）
	template.ResourceCountIs(jsii.String("AWS::EC2::NatGateway"), jsii.Number(2))

	assert.NotNil(t, stack)
}

func TestNetworkStack_CrossStackExports(t *testing.T) {
	// Given
	app := helpers.CreateTestApp(&helpers.TestAppConfig{
		Environment: "production",
	})

	// When: NetworkStackを作成
	stack := stacks.NewNetworkStack(app, "TestNetworkStack", &stacks.NetworkStackProps{
		Environment: "production",
		VpcCidr:     "10.2.0.0/16",
	})

	// Then: Cross-stack出力の確認
	template := assertions.Template_FromStack(stack, nil)

	// VpcId出力の確認
	template.HasOutput(jsii.String("VpcId"), map[string]interface{}{
		"Description": "VPC ID for Service",
		"Export": map[string]interface{}{
			"Name": "Service-production-VpcId",
		},
	})

	assert.NotNil(t, stack)
}
