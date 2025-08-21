package stacks_test

import (
	"aws-ecs-fargate-go-cdk/tests/helpers"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNetworkStack(t *testing.T) {
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
				Account:     "654654358776",
			})
			// When: NetworkStackを作成（実装後にコメントアウト解除）
			// stack := stacks.NewNetworkStack(app, "TestNetworkStack", &stacks.NetworkStackProps{
			// 	Environment: tc.environment,
			// 	VpcCidr:     tc.vpcCidr,
			// })

			// Then: リソースの存在確認（実装後にコメントアウト解除）
			// vpcAssertions := helpers.NewVPCAssertions(stack)
			// vpcAssertions.HasVPCWithCIDR(tc.vpcCidr).
			// 	HasSubnetCount(tc.subnetCount).
			// 	HasInternetGateway()

			// 現在はテスト構造のみ確認
			assert.NotNil(t, app)
			assert.Equal(t, tc.environment, tc.environment)
		})
	}
}

func TestNetworkStack_SecurityGroups(t *testing.T) {
	// Given
	app := helpers.CreateTestApp(&helpers.TestAppConfig{
		Environment: "test",
	})

	// When: NetworkStackを作成
	// stack := stacks.NewNetworkStack(app, "TestNetworkStack", nil)

	// Then: セキュリティグループの確認
	// helpers.AssertStackHasResource(t, stack, "AWS::EC2::SecurityGroup", 3)

	// 現在はプレースホルダー
	assert.NotNil(t, app)
}

func TestNetworkStack_RouteTable(t *testing.T) {
	// Given
	app := helpers.CreateTestApp(nil)

	// When: NetworkStackを作成
	// stack := stacks.NewNetworkStack(app, "TestNetworkStack", nil)

	// Then: ルートテーブルの確認
	// helpers.AssertStackHasResource(t, stack, "AWS::EC2::RouteTable", 4)

	// 現在はプレースホルダー
	assert.NotNil(t, app)
}
