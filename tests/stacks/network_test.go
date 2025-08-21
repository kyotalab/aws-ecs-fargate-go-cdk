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

// func TestNetworkStack(t *testing.T) {
// 	// テーブル駆動テスト用のテストケース
// 	testCases := []struct {
// 		name        string
// 		environment string
// 		vpcCidr     string
// 		subnetCount int
// 	}{
// 		{
// 			name:        "Development Environment",
// 			environment: "dev",
// 			vpcCidr:     "10.0.0.0/16",
// 			subnetCount: 4, // Public x2, Private x2

// 		},
// 		{
// 			name:        "Staging Environment",
// 			environment: "staging",
// 			vpcCidr:     "10.1.0.0/16",
// 			subnetCount: 4, // Public x2, Private x2
// 		},
// 		{
// 			name:        "Production Environment",
// 			environment: "prod",
// 			vpcCidr:     "10.2.0.0/16",
// 			subnetCount: 4, // Public x2, Private x2
// 		},
// 	}

// 	for _, tc := range testCases {
// 		t.Run(tc.name, func(t *testing.T) {
// 			// Given: テスト用アプリケーション作成
// 			app := helpers.CreateTestApp(&helpers.TestAppConfig{
// 				Environment: tc.environment,
// 				Region:      "ap-northeast-1",
// 				Account:     "123456789012",
// 			})
// 			// When: NetworkStackを作成（実装後にコメントアウト解除）
// 			// stack := stacks.NewNetworkStack(app, "TestNetworkStack", &stacks.NetworkStackProps{
// 			// 	Environment: tc.environment,
// 			// 	VpcCidr:     tc.vpcCidr,
// 			// })

// 			// Then: リソースの存在確認（実装後にコメントアウト解除）
// 			// vpcAssertions := helpers.NewVPCAssertions(stack)
// 			// vpcAssertions.HasVPCWithCIDR(tc.vpcCidr).
// 			// 	HasSubnetCount(tc.subnetCount).
// 			// 	HasInternetGateway()

// 			// 現在はテスト構造のみ確認
// 			assert.NotNil(t, app)
// 			assert.Equal(t, tc.environment, tc.environment)
// 		})
// 	}
// }

// func TestNetworkStack_SecurityGroups(t *testing.T) {
// 	// Given
// 	app := helpers.CreateTestApp(&helpers.TestAppConfig{
// 		Environment: "test",
// 	})

// 	// When: NetworkStackを作成
// 	// stack := stacks.NewNetworkStack(app, "TestNetworkStack", nil)

// 	// Then: セキュリティグループの確認
// 	// helpers.AssertStackHasResource(t, stack, "AWS::EC2::SecurityGroup", 3)

// 	// 現在はプレースホルダー
// 	assert.NotNil(t, app)
// }

// func TestNetworkStack_RouteTable(t *testing.T) {
// 	// Given
// 	app := helpers.CreateTestApp(nil)

// 	// When: NetworkStackを作成
// 	// stack := stacks.NewNetworkStack(app, "TestNetworkStack", nil)

// 	// Then: ルートテーブルの確認
// 	// helpers.AssertStackHasResource(t, stack, "AWS::EC2::RouteTable", 4)

// 	// 現在はプレースホルダー
// 	assert.NotNil(t, app)
// }
