package stacks_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"aws-ecs-fargate-go-cdk/internal/stacks"
	"aws-ecs-fargate-go-cdk/tests/helpers"
)

func TestApplicationStack_BasicCreation(t *testing.T) {
	// Given: テスト用アプリケーション作成
	app := helpers.CreateTestApp(&helpers.TestAppConfig{
		Environment: "dev",
		Region:      "ap-northeast-1",
		Account:     "123456789012",
	})

	// When: ApplicationStackを作成（まだ存在しないのでコンパイルエラー）
	stack := stacks.NewApplicationStack(app, "TestApplicationStack", &stacks.ApplicationStackProps{
		Environment:      "dev",
		VpcId:            "vpc-12345", // Mock VPC ID
		DatabaseEndpoint: "mock-aurora-endpoint.cluster-xyz.ap-northeast-1.rds.amazonaws.com",
		RedisEndpoint:    "mock-redis-endpoint.cache.amazonaws.com",
		TestEnvFlag:      true,
	})

	// Then: 基本的なECSクラスターが作成されることを確認
	ecsAssertions := helpers.NewECSAssertions(stack)
	ecsAssertions.HasCluster()

	assert.NotNil(t, stack)
}

func TestApplicationStack_LoadBalancer(t *testing.T) {
	// Given
	app := helpers.CreateTestApp(&helpers.TestAppConfig{
		Environment: "dev",
	})

	// When: ApplicationStackを作成
	stack := stacks.NewApplicationStack(app, "TestApplicationStack", &stacks.ApplicationStackProps{
		Environment: "dev",
		VpcId:       "vpc-12345",
		TestEnvFlag: true,
	})

	// Then: ALBの確認
	helpers.AssertStackHasResource(t, stack, "AWS::ElasticLoadBalancingV2::LoadBalancer", 1)
	helpers.AssertStackHasResource(t, stack, "AWS::ElasticLoadBalancingV2::TargetGroup", 1)

	assert.NotNil(t, stack)
}

func TestApplicationStack_ECR(t *testing.T) {
	// Given
	app := helpers.CreateTestApp(&helpers.TestAppConfig{
		Environment: "dev",
	})

	// When: ApplicationStackを作成
	stack := stacks.NewApplicationStack(app, "TestApplicationStack", &stacks.ApplicationStackProps{
		Environment: "dev",
		VpcId:       "vpc-12345",
		TestEnvFlag: true,
	})

	// Then: ECRリポジトリの確認
	helpers.AssertStackHasResource(t, stack, "AWS::ECR::Repository", 1)

	assert.NotNil(t, stack)
}
