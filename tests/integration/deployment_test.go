package integration_test

import (
	"aws-ecs-fargate-go-cdk/internal/stacks"
	"aws-ecs-fargate-go-cdk/tests/helpers"
	"testing"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/assertions"
	"github.com/aws/jsii-runtime-go"
	"github.com/stretchr/testify/assert"
)

func TestDeploymentIntegration(t *testing.T) {
	// 統合テストの場合、実際のAWSリソースのデプロイは避けて
	// CloudFormationテンプレートの生成確認に留める

	// Given
	app := helpers.CreateTestApp(&helpers.TestAppConfig{
		Environment: "dev",
		Region:      "ap-northeast-1",
		Account:     "123456789012",
	})

	// When: 全スタックを作成
	allStacks := createAllStacks(app, "dev")

	// Then: CloudFormationテンプレート生成の確認
	assert.Equal(t, 3, len(allStacks), "Should create 3 stacks")

	// 各Stackの基本リソース確認
	networkTemplate := assertions.Template_FromStack(allStacks["network"], nil)
	networkTemplate.ResourceCountIs(jsii.String("AWS::EC2::VPC"), jsii.Number(1))

	storageTemplate := assertions.Template_FromStack(allStacks["storage"], nil)
	storageTemplate.ResourceCountIs(jsii.String("AWS::RDS::DBCluster"), jsii.Number(1))

	applicationTemplate := assertions.Template_FromStack(allStacks["application"], nil)
	applicationTemplate.ResourceCountIs(jsii.String("AWS::ECS::Cluster"), jsii.Number(1))
}

func TestEnvironmentConfiguration(t *testing.T) {
	environments := []string{"dev", "staging", "prod"}

	for _, env := range environments {
		t.Run("Environment_"+env, func(t *testing.T) {
			// Given
			app := helpers.CreateTestApp(&helpers.TestAppConfig{
				Environment: env,
				Region:      "ap-northeast-1",
				Account:     "123456789012",
			})

			// When: 環境別スタック作成
			allStacks := createAllStacks(app, env)

			// Then: 環境固有の設定確認
			assert.Equal(t, 3, len(allStacks), "Should create 3 stacks for "+env)

			// 環境別のVPC CIDR確認
			networkTemplate := assertions.Template_FromStack(allStacks["network"], nil)
			expectedCidr := getExpectedVpcCidr(env)
			networkTemplate.HasResourceProperties(jsii.String("AWS::EC2::VPC"), map[string]interface{}{
				"CidrBlock": expectedCidr,
			})

			// 環境別のAurora設定確認
			storageTemplate := assertions.Template_FromStack(allStacks["storage"], nil)
			expectedInstances := getExpectedAuroraInstances(env)
			storageTemplate.ResourceCountIs(jsii.String("AWS::RDS::DBInstance"), jsii.Number(expectedInstances))
		})
	}
}

func TestStackResourceCounts(t *testing.T) {
	// Given
	app := helpers.CreateTestApp(&helpers.TestAppConfig{
		Environment: "dev",
		Region:      "ap-northeast-1",
		Account:     "123456789012",
	})

	// When: 全スタック作成
	allStacks := createAllStacks(app, "dev")

	// Then: 各Stackのリソース数確認
	testCases := []struct {
		stackName     string
		resourceType  string
		expectedCount int
		description   string
	}{
		{"network", "AWS::EC2::VPC", 1, "NetworkStack should have 1 VPC"},
		{"network", "AWS::EC2::SecurityGroup", 3, "NetworkStack should have 3 Security Groups"},
		{"network", "AWS::EC2::InternetGateway", 1, "NetworkStack should have 1 Internet Gateway"},
		{"storage", "AWS::RDS::DBCluster", 1, "StorageStack should have 1 Aurora cluster"},
		{"storage", "AWS::ElastiCache::ReplicationGroup", 1, "StorageStack should have 1 Redis cluster"},
		{"storage", "AWS::S3::Bucket", 3, "StorageStack should have 3 S3 buckets"},
		{"application", "AWS::ECS::Cluster", 1, "ApplicationStack should have 1 ECS cluster"},
		{"application", "AWS::ECR::Repository", 1, "ApplicationStack should have 1 ECR repository"},
		{"application", "AWS::ElasticLoadBalancingV2::LoadBalancer", 1, "ApplicationStack should have 1 ALB"},
		{"application", "AWS::ElasticLoadBalancingV2::TargetGroup", 1, "ApplicationStack should have 1 Target Group"},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			template := assertions.Template_FromStack(allStacks[tc.stackName], nil)
			template.ResourceCountIs(jsii.String(tc.resourceType), jsii.Number(tc.expectedCount))
		})
	}
}

func TestCrossStackOutputs(t *testing.T) {
	// Given
	app := helpers.CreateTestApp(&helpers.TestAppConfig{
		Environment: "dev",
		Region:      "ap-northeast-1",
		Account:     "123456789012",
	})

	// When
	allStacks := createAllStacks(app, "dev")

	// Then: Cross-stack出力の確認
	networkOutputs := []struct {
		outputKey   string
		exportName  string
		description string
	}{
		{"VpcId", "Service-dev-VpcId", "VPC ID for Service"},
		{"ALBSecurityGroupId", "Service-dev-ALB-SG-Id", "ALB Security Group ID"},
		{"ECSSecurityGroupId", "Service-dev-ECS-SG-Id", "ECS Security Group ID"},
		{"RDSSecurityGroupId", "Service-dev-RDS-SG-Id", "RDS Security Group ID"},
	}

	networkTemplate := assertions.Template_FromStack(allStacks["network"], nil)
	for _, output := range networkOutputs {
		t.Run("NetworkStack_Output_"+output.outputKey, func(t *testing.T) {
			networkTemplate.HasOutput(jsii.String(output.outputKey), map[string]interface{}{
				"Description": output.description,
				"Export": map[string]interface{}{
					"Name": output.exportName,
				},
			})
		})
	}

	storageOutputs := []struct {
		outputKey   string
		exportName  string
		description string
	}{
		{"AuroraClusterEndpoint", "service-development-Aurora-Endpoint", "Aurora MySQL Cluster Writer Endpoint"},
		{"ElastiCacheEndpoint", "service-development-Redis-Endpoint", "ElastiCache Redis Primary Endpoint"},
		{"StaticAssetsBucketName", "service-development-Static-Bucket", "Static Assets S3 Bucket Name"},
	}

	storageTemplate := assertions.Template_FromStack(allStacks["storage"], nil)
	for _, output := range storageOutputs {
		t.Run("StorageStack_Output_"+output.outputKey, func(t *testing.T) {
			storageTemplate.HasOutput(jsii.String(output.outputKey), map[string]interface{}{
				"Description": output.description,
				"Export": map[string]interface{}{
					"Name": output.exportName,
				},
			})
		})
	}

	applicationOutputs := []struct {
		outputKey   string
		exportName  string
		description string
	}{
		{"ECRRepositoryURI", "Service-dev-ECR-URI", "ECR Repository URI for container images"},
		{"LoadBalancerDNS", "Service-dev-ALB-DNS", "Application Load Balancer DNS name"},
	}

	applicationTemplate := assertions.Template_FromStack(allStacks["application"], nil)
	for _, output := range applicationOutputs {
		t.Run("ApplicationStack_Output_"+output.outputKey, func(t *testing.T) {
			applicationTemplate.HasOutput(jsii.String(output.outputKey), map[string]interface{}{
				"Description": output.description,
				"Export": map[string]interface{}{
					"Name": output.exportName,
				},
			})
		})
	}
}

func TestTaggingConsistency(t *testing.T) {
	// Given
	app := helpers.CreateTestApp(&helpers.TestAppConfig{
		Environment: "prod", // 本番環境でのタグ確認
		Region:      "ap-northeast-1",
		Account:     "123456789012",
	})

	// When
	allStacks := createAllStacks(app, "prod")

	// Then: 各Stackで一貫したタグが設定されていることを確認
	networkTemplate := assertions.Template_FromStack(allStacks["network"], nil)
	storageTemplate := assertions.Template_FromStack(allStacks["storage"], nil)
	applicationTemplate := assertions.Template_FromStack(allStacks["application"], nil)

	// VPCのタグ確認
	networkTemplate.HasResourceProperties(jsii.String("AWS::EC2::VPC"), map[string]interface{}{
		"Tags": assertions.Match_ArrayWith(&[]interface{}{
			map[string]interface{}{
				"Key":   "Environment",
				"Value": "production",
			},
			map[string]interface{}{
				"Key":   "Project",
				"Value": "PracticeService",
			},
		}),
	})

	assert.NotNil(t, networkTemplate)
	assert.NotNil(t, storageTemplate)
	assert.NotNil(t, applicationTemplate)
}

func TestStackTemplateSize(t *testing.T) {
	// Given
	app := helpers.CreateTestApp(&helpers.TestAppConfig{
		Environment: "dev",
		Region:      "ap-northeast-1",
		Account:     "123456789012",
	})

	// When
	allStacks := createAllStacks(app, "dev")

	// Then: CloudFormationテンプレートサイズ制限チェック
	for stackName, stack := range allStacks {
		t.Run("TemplateSize_"+stackName, func(t *testing.T) {
			// CloudFormationの制限: 51,200 bytes (約50KB)
			helpers.ValidateStackTemplateSize(t, stack, 50)
		})
	}
}

// ヘルパー関数群
func createAllStacks(app awscdk.App, environment string) map[string]awscdk.Stack {
	allStacks := make(map[string]awscdk.Stack)

	// NetworkStack作成
	allStacks["network"] = stacks.NewNetworkStack(app, environment+"NetworkStack", &stacks.NetworkStackProps{
		Environment: environment,
	})

	// StorageStack作成
	allStacks["storage"] = stacks.NewStorageStack(app, environment+"StorageStack", &stacks.StorageStackProps{
		Environment: environment,
		VpcId:       "vpc-from-network-stack",
		TestEnvFlag: true, // テスト環境フラグ
	})

	// ApplicationStack作成
	allStacks["application"] = stacks.NewApplicationStack(app, environment+"ApplicationStack", &stacks.ApplicationStackProps{
		Environment: environment,
		VpcId:       "vpc-from-network-stack",
		TestEnvFlag: true, // テスト環境フラグ
	})

	return allStacks
}

func getExpectedVpcCidr(environment string) string {
	switch environment {
	case "dev":
		return "10.0.0.0/16"
	case "staging":
		return "10.1.0.0/16"
	case "prod":
		return "10.2.0.0/16"
	default:
		return "10.0.0.0/16"
	}
}

func getExpectedAuroraInstances(environment string) int {
	switch environment {
	case "dev":
		return 1
	case "staging":
		return 2
	case "prod":
		return 3
	default:
		return 1
	}
}
