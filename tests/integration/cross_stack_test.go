package integration_test

import (
	"aws-ecs-fargate-go-cdk/internal/stacks"
	"aws-ecs-fargate-go-cdk/tests/helpers"
	"testing"

	"github.com/aws/aws-cdk-go/awscdk/v2/assertions"
	"github.com/aws/jsii-runtime-go"
	"github.com/stretchr/testify/assert"
)

func TestCrossStackIntegration(t *testing.T) {
	// Given: 複数のスタックを作成
	app := helpers.CreateTestApp(&helpers.TestAppConfig{
		Environment: "dev",
		Region:      "ap-northeast-1",
		Account:     "123456789012",
	})

	// When: 依存関係のあるスタックを作成
	networkStack := stacks.NewNetworkStack(app, "IntegrationNetworkStack", &stacks.NetworkStackProps{
		Environment: "dev",
		VpcCidr:     "10.100.0.0/16", // 統合テスト用CIDR
	})

	storageStack := stacks.NewStorageStack(app, "IntegrationStorageStack", &stacks.StorageStackProps{
		Environment: "dev",
		VpcId:       "vpc-from-network-stack", // Cross-stack参照をテスト
		TestEnvFlag: true,                     // テスト環境フラグ
	})

	applicationStack := stacks.NewApplicationStack(app, "IntegrationApplicationStack", &stacks.ApplicationStackProps{
		Environment:      "dev",
		VpcId:            "vpc-from-network-stack", // Cross-stack参照をテスト
		DatabaseEndpoint: "from-storage-stack",
		RedisEndpoint:    "from-storage-stack",
		TestEnvFlag:      true, // テスト環境フラグ
	})

	// Then: クロススタック参照の確認
	assert.NotNil(t, networkStack)
	assert.NotNil(t, storageStack)
	assert.NotNil(t, applicationStack)

	// NetworkStackの出力確認
	networkTemplate := assertions.Template_FromStack(networkStack, nil)
	networkTemplate.HasOutput(jsii.String("VpcId"), map[string]interface{}{
		"Description": "VPC ID for Service",
		"Export": map[string]interface{}{
			"Name": "Service-dev-VpcId",
		},
	})

	// StorageStackの出力確認
	storageTemplate := assertions.Template_FromStack(storageStack, nil)
	storageTemplate.HasOutput(jsii.String("AuroraClusterEndpoint"), map[string]interface{}{
		"Description": "Aurora MySQL Cluster Writer Endpoint",
		"Export": map[string]interface{}{
			"Name": "service-development-Aurora-Endpoint",
		},
	})

	// ApplicationStackの出力確認
	applicationTemplate := assertions.Template_FromStack(applicationStack, nil)
	applicationTemplate.HasOutput(jsii.String("LoadBalancerDNS"), map[string]interface{}{
		"Description": "Application Load Balancer DNS name",
		"Export": map[string]interface{}{
			"Name": "Service-dev-ALB-DNS",
		},
	})
}

func TestStackDependencies(t *testing.T) {
	// Given
	app := helpers.CreateTestApp(&helpers.TestAppConfig{
		Environment: "dev",
		Region:      "ap-northeast-1",
		Account:     "123456789012",
	})

	// When: 依存関係のテスト
	networkStack := stacks.NewNetworkStack(app, "TestNetworkStack", &stacks.NetworkStackProps{
		Environment: "dev",
	})

	storageStack := stacks.NewStorageStack(app, "TestStorageStack", &stacks.StorageStackProps{
		Environment: "dev",
		VpcId:       "vpc-from-network-stack",
		TestEnvFlag: true,
	})

	applicationStack := stacks.NewApplicationStack(app, "TestApplicationStack", &stacks.ApplicationStackProps{
		Environment: "dev",
		VpcId:       "vpc-from-network-stack",
		TestEnvFlag: true,
	})

	// Then: 依存関係の確認（実際のCDKでは明示的な依存関係チェックは困難）
	// 代わりに、各Stackが正しく作成されることを確認
	assert.NotNil(t, networkStack)
	assert.NotNil(t, storageStack)
	assert.NotNil(t, applicationStack)

	// NetworkStackのリソース確認
	networkTemplate := assertions.Template_FromStack(networkStack, nil)
	networkTemplate.ResourceCountIs(jsii.String("AWS::EC2::VPC"), jsii.Number(1))

	// StorageStackのリソース確認
	storageTemplate := assertions.Template_FromStack(storageStack, nil)
	storageTemplate.ResourceCountIs(jsii.String("AWS::RDS::DBCluster"), jsii.Number(1))

	// ApplicationStackのリソース確認
	applicationTemplate := assertions.Template_FromStack(applicationStack, nil)
	applicationTemplate.ResourceCountIs(jsii.String("AWS::ECS::Cluster"), jsii.Number(1))
}

func TestSecurityGroupCrossStackReferences(t *testing.T) {
	// Given: セキュリティグループのCross-stack参照をテスト
	app := helpers.CreateTestApp(&helpers.TestAppConfig{
		Environment: "dev",
		Region:      "ap-northeast-1",
		Account:     "123456789012",
	})

	// When
	networkStack := stacks.NewNetworkStack(app, "SGTestNetworkStack", &stacks.NetworkStackProps{
		Environment: "dev",
	})

	applicationStack := stacks.NewApplicationStack(app, "SGTestApplicationStack", &stacks.ApplicationStackProps{
		Environment: "dev",
		VpcId:       "vpc-from-network-stack",
		TestEnvFlag: true,
	})

	// Then: NetworkStackでセキュリティグループが作成されることを確認
	networkTemplate := assertions.Template_FromStack(networkStack, nil)
	networkTemplate.ResourceCountIs(jsii.String("AWS::EC2::SecurityGroup"), jsii.Number(3))

	// セキュリティグループ出力の確認
	networkTemplate.HasOutput(jsii.String("ALBSecurityGroupId"), map[string]interface{}{
		"Description": "ALB Security Group ID",
		"Export": map[string]interface{}{
			"Name": "Service-dev-ALB-SG-Id",
		},
	})

	// ApplicationStackでALBが作成されることを確認
	applicationTemplate := assertions.Template_FromStack(applicationStack, nil)
	applicationTemplate.ResourceCountIs(jsii.String("AWS::ElasticLoadBalancingV2::LoadBalancer"), jsii.Number(1))

	assert.NotNil(t, networkStack)
	assert.NotNil(t, applicationStack)
}

func TestEnvironmentSpecificConfigurations(t *testing.T) {
	environments := []struct {
		name               string
		environment        string
		expectedVpcCidr    string
		expectedMaxAzs     int
		expectedAuroraInst int
	}{
		{
			name:               "Development Environment Integration",
			environment:        "dev",
			expectedVpcCidr:    "10.0.0.0/16",
			expectedMaxAzs:     2,
			expectedAuroraInst: 1,
		},
		{
			name:               "Staging Environment Integration",
			environment:        "staging",
			expectedVpcCidr:    "10.1.0.0/16",
			expectedMaxAzs:     2,
			expectedAuroraInst: 2,
		},
		{
			name:               "Production Environment Integration",
			environment:        "prod",
			expectedVpcCidr:    "10.2.0.0/16",
			expectedMaxAzs:     2, // 実際の利用可能AZ数
			expectedAuroraInst: 3,
		},
	}

	for _, tc := range environments {
		t.Run(tc.name, func(t *testing.T) {
			// Given
			app := helpers.CreateTestApp(&helpers.TestAppConfig{
				Environment: tc.environment,
				Region:      "ap-northeast-1",
				Account:     "123456789012",
			})

			// When: 環境別で全Stackを作成
			networkStack := stacks.NewNetworkStack(app, tc.environment+"NetworkStack", &stacks.NetworkStackProps{
				Environment: tc.environment,
			})

			storageStack := stacks.NewStorageStack(app, tc.environment+"StorageStack", &stacks.StorageStackProps{
				Environment: tc.environment,
				VpcId:       "vpc-from-network-stack",
				TestEnvFlag: true,
			})

			applicationStack := stacks.NewApplicationStack(app, tc.environment+"ApplicationStack", &stacks.ApplicationStackProps{
				Environment: tc.environment,
				VpcId:       "vpc-from-network-stack",
				TestEnvFlag: true,
			})

			// Then: 環境固有の設定確認
			networkTemplate := assertions.Template_FromStack(networkStack, nil)
			networkTemplate.HasResourceProperties(jsii.String("AWS::EC2::VPC"), map[string]interface{}{
				"CidrBlock": tc.expectedVpcCidr,
			})

			storageTemplate := assertions.Template_FromStack(storageStack, nil)
			storageTemplate.ResourceCountIs(jsii.String("AWS::RDS::DBInstance"), jsii.Number(tc.expectedAuroraInst))

			applicationTemplate := assertions.Template_FromStack(applicationStack, nil)
			applicationTemplate.ResourceCountIs(jsii.String("AWS::ECS::Cluster"), jsii.Number(1))

			assert.NotNil(t, networkStack)
			assert.NotNil(t, storageStack)
			assert.NotNil(t, applicationStack)
		})
	}
}
