package stacks_test

import (
	"testing"

	"github.com/aws/aws-cdk-go/awscdk/v2/assertions"
	"github.com/aws/jsii-runtime-go"
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

// TestApplicationStack_ECSService ECS Serviceの基本テスト
func TestApplicationStack_ECSService(t *testing.T) {
	// Given: テスト用アプリケーション作成
	app := helpers.CreateTestApp(&helpers.TestAppConfig{
		Environment: "dev",
		Region:      "ap-northeast-1",
		Account:     "123456789012",
	})

	// When: ApplicationStackを作成
	stack := stacks.NewApplicationStack(app, "TestApplicationStack", &stacks.ApplicationStackProps{
		Environment:      "dev",
		VpcId:            "vpc-12345",
		DatabaseEndpoint: "aurora-endpoint.cluster-xyz.rds.amazonaws.com",
		RedisEndpoint:    "redis-endpoint.cache.amazonaws.com",
		TestEnvFlag:      true,
	})

	// Then: ECS Serviceが作成されることを確認
	template := assertions.Template_FromStack(stack, nil)
	template.ResourceCountIs(jsii.String("AWS::ECS::Service"), jsii.Number(1))

	// ECS ServiceのTargetGroup紐付け確認
	template.HasResourceProperties(jsii.String("AWS::ECS::Service"), map[string]interface{}{
		"LaunchType":   "FARGATE",
		"DesiredCount": 1,
		"ServiceName":  "service-dev-fargate-service",
	})

	assert.NotNil(t, stack)
}

// TestApplicationStack_TaskDefinition Task Definitionのテスト
func TestApplicationStack_TaskDefinition(t *testing.T) {
	// Given
	app := helpers.CreateTestApp(&helpers.TestAppConfig{
		Environment: "dev",
	})

	// When: ApplicationStackを作成
	stack := stacks.NewApplicationStack(app, "TestApplicationStack", &stacks.ApplicationStackProps{
		Environment:      "dev",
		VpcId:            "vpc-12345",
		DatabaseEndpoint: "aurora-endpoint.cluster-xyz.rds.amazonaws.com",
		RedisEndpoint:    "redis-endpoint.cache.amazonaws.com",
		TestEnvFlag:      true,
	})

	// Then: Task Definitionが作成されることを確認
	template := assertions.Template_FromStack(stack, nil)
	template.ResourceCountIs(jsii.String("AWS::ECS::TaskDefinition"), jsii.Number(1))

	// Task DefinitionのCPU・メモリ設定確認
	template.HasResourceProperties(jsii.String("AWS::ECS::TaskDefinition"), map[string]interface{}{
		"Cpu":                     "512",
		"Memory":                  "1024",
		"NetworkMode":             "awsvpc",
		"RequiresCompatibilities": []interface{}{"FARGATE"},
	})

	assert.NotNil(t, stack)
}

// TestApplicationStack_ContainerDefinitions コンテナ定義のテスト
func TestApplicationStack_ContainerDefinitions(t *testing.T) {
	// Given
	app := helpers.CreateTestApp(&helpers.TestAppConfig{
		Environment: "dev",
	})

	// When: ApplicationStackを作成
	stack := stacks.NewApplicationStack(app, "TestApplicationStack", &stacks.ApplicationStackProps{
		Environment:      "dev",
		VpcId:            "vpc-12345",
		DatabaseEndpoint: "aurora-endpoint.cluster-xyz.rds.amazonaws.com",
		RedisEndpoint:    "redis-endpoint.cache.amazonaws.com",
		TestEnvFlag:      true,
	})

	// Then: コンテナ定義の確認
	template := assertions.Template_FromStack(stack, nil)

	// PHPコンテナの確認
	template.HasResourceProperties(jsii.String("AWS::ECS::TaskDefinition"), map[string]interface{}{
		"ContainerDefinitions": assertions.Match_ArrayWith(&[]interface{}{
			map[string]interface{}{
				"Name":  "php-app",
				"Image": assertions.Match_StringLikeRegexp(jsii.String(".*\\.dkr\\.ecr\\..*\\.amazonaws\\.com/service-dev:latest")),
				"Environment": assertions.Match_ArrayWith(&[]interface{}{
					map[string]interface{}{
						"Name":  "APP_ENV",
						"Value": "development",
					},
					map[string]interface{}{
						"Name":  "DB_CONNECTION",
						"Value": "mysql",
					},
				}),
			},
		}),
	})

	// Nginxコンテナの確認
	template.HasResourceProperties(jsii.String("AWS::ECS::TaskDefinition"), map[string]interface{}{
		"ContainerDefinitions": assertions.Match_ArrayWith(&[]interface{}{
			map[string]interface{}{
				"Name":  "nginx-web",
				"Image": "nginx:1.24-alpine",
				"PortMappings": assertions.Match_ArrayWith(&[]interface{}{
					map[string]interface{}{
						"ContainerPort": 80,
						"Protocol":      "tcp",
					},
				}),
			},
		}),
	})

	assert.NotNil(t, stack)
}

// TestApplicationStack_EnvironmentSpecificSettings 環境別設定のテスト
func TestApplicationStack_EnvironmentSpecificSettings(t *testing.T) {
	testCases := []struct {
		name            string
		environment     string
		expectedCPU     string
		expectedMemory  string
		expectedDesired int
	}{
		{
			name:            "Development Environment",
			environment:     "dev",
			expectedCPU:     "256",
			expectedMemory:  "512",
			expectedDesired: 1,
		},
		{
			name:            "Staging Environment",
			environment:     "staging",
			expectedCPU:     "512",
			expectedMemory:  "1024",
			expectedDesired: 2,
		},
		{
			name:            "Production Environment",
			environment:     "prod",
			expectedCPU:     "1024",
			expectedMemory:  "2048",
			expectedDesired: 4,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Given
			app := helpers.CreateTestApp(&helpers.TestAppConfig{
				Environment: tc.environment,
			})

			// When: ApplicationStackを作成
			stack := stacks.NewApplicationStack(app, "TestApplicationStack", &stacks.ApplicationStackProps{
				Environment:      tc.environment,
				VpcId:            "vpc-12345",
				DatabaseEndpoint: "aurora-endpoint.cluster-xyz.rds.amazonaws.com",
				RedisEndpoint:    "redis-endpoint.cache.amazonaws.com",
				TestEnvFlag:      true,
			})

			// Then: 環境別設定の確認
			template := assertions.Template_FromStack(stack, nil)

			// Task Definition設定確認
			template.HasResourceProperties(jsii.String("AWS::ECS::TaskDefinition"), map[string]interface{}{
				"Cpu":    tc.expectedCPU,
				"Memory": tc.expectedMemory,
			})

			// ECS Service設定確認
			template.HasResourceProperties(jsii.String("AWS::ECS::Service"), map[string]interface{}{
				"DesiredCount": tc.expectedDesired,
			})

			assert.NotNil(t, stack)
		})
	}
}

// TestApplicationStack_ServiceDiscovery Service Discoveryのテスト
func TestApplicationStack_ServiceDiscovery(t *testing.T) {
	// Given
	app := helpers.CreateTestApp(&helpers.TestAppConfig{
		Environment: "prod", // 本番環境でService Discovery有効
	})

	// When: ApplicationStackを作成
	stack := stacks.NewApplicationStack(app, "TestApplicationStack", &stacks.ApplicationStackProps{
		Environment:      "prod",
		VpcId:            "vpc-12345",
		DatabaseEndpoint: "aurora-endpoint.cluster-xyz.rds.amazonaws.com",
		RedisEndpoint:    "redis-endpoint.cache.amazonaws.com",
		TestEnvFlag:      true,
	})

	// Then: Service Discoveryの確認（本番環境のみ）
	template := assertions.Template_FromStack(stack, nil)
	template.ResourceCountIs(jsii.String("AWS::ServiceDiscovery::Service"), jsii.Number(1))

	assert.NotNil(t, stack)
}
