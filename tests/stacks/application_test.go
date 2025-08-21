package stacks_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"aws-ecs-fargate-go-cdk/tests/helpers"
)

func TestApplicationStack(t *testing.T) {
	testCases := []struct {
		name         string
		environment  string
		desiredCount int
		cpu          int
		memory       int
	}{
		{
			name:         "Development Environment",
			environment:  "dev",
			desiredCount: 2,
			cpu:          1024, // 1 vCPU
			memory:       2048, // 2 GB
		},
		{
			name:         "Production Environment",
			environment:  "prod",
			desiredCount: 6,
			cpu:          2048, // 2 vCPU
			memory:       4096, // 4 GB
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Given
			app := helpers.CreateTestApp(&helpers.TestAppConfig{
				Environment: tc.environment,
			})

			// When: ApplicationStackを作成（実装後にコメントアウト解除）
			// stack := stacks.NewApplicationStack(app, "TestApplicationStack", &stacks.ApplicationStackProps{
			// 	Environment:  tc.environment,
			// 	VpcId:       "vpc-12345",
			// 	DesiredCount: tc.desiredCount,
			// })

			// Then: ECSリソースの確認（実装後にコメントアウト解除）
			// ecsAssertions := helpers.NewECSAssertions(stack)
			// ecsAssertions.HasCluster().
			// 	HasServiceWithDesiredCount(tc.desiredCount)

			// 現在はプレースホルダー
			assert.NotNil(t, app)
		})
	}
}

func TestApplicationStack_LoadBalancer(t *testing.T) {
	// Given
	app := helpers.CreateTestApp(nil)

	// When: ApplicationStackを作成
	// stack := stacks.NewApplicationStack(app, "TestApplicationStack", nil)

	// Then: ALBの確認
	// helpers.AssertStackHasResource(t, stack, "AWS::ElasticLoadBalancingV2::LoadBalancer", 1)
	// helpers.AssertStackHasResource(t, stack, "AWS::ElasticLoadBalancingV2::TargetGroup", 1)

	assert.NotNil(t, app)
}

func TestApplicationStack_ECR(t *testing.T) {
	// Given
	app := helpers.CreateTestApp(nil)

	// When: ApplicationStackを作成
	// stack := stacks.NewApplicationStack(app, "TestApplicationStack", nil)

	// Then: ECRリポジトリの確認
	// helpers.AssertStackHasResource(t, stack, "AWS::ECR::Repository", 1)

	assert.NotNil(t, app)
}
