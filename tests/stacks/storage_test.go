package stacks_test

import (
	"aws-ecs-fargate-go-cdk/tests/helpers"
	"testing"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/assertions"
	"github.com/aws/jsii-runtime-go"
	"github.com/stretchr/testify/assert"

	"aws-ecs-fargate-go-cdk/internal/stacks" // これがコンパイルエラーになる
)

// TestAppConfig テストアプリケーションの設定
type TestAppConfig struct {
	Environment       string
	Region            string
	Account           string
	IsTestEnvironment bool // テスト環境フラグ追加
}

func CreateTestAppForStorageStack(environment string) awscdk.App {
	return helpers.CreateTestApp(&helpers.TestAppConfig{
		Environment:       environment,
		Region:            "ap-northeast-1",
		Account:           "123456789012",
		IsTestEnvironment: true,
	})
}

func TestStorageStack_BasicCreation(t *testing.T) {
	// Given: テスト用アプリケーション作成
	app := helpers.CreateTestApp(&helpers.TestAppConfig{
		Environment: "dev",
		Region:      "ap-northeast-1",
		Account:     "123456789012",
	})

	// When: StorageStackを作成（テスト環境フラグ追加）
	stack := stacks.NewStorageStack(app, "TestStorageStack", &stacks.StorageStackProps{
		Environment:       "dev",
		VpcId:             "vpc-12345", // Mock VPC ID
		IsTestEnvironment: true,        // テスト環境フラグ
	})

	// Then: 基本的なAuroraクラスターが作成されることを確認
	template := assertions.Template_FromStack(stack, nil)
	template.ResourceCountIs(jsii.String("AWS::RDS::DBCluster"), jsii.Number(1))

	assert.NotNil(t, stack)
}

func TestStorageStack_EnvironmentConfigurations(t *testing.T) {
	testCases := []struct {
		name          string
		environment   string
		instanceCount int
		engineVersion string
		enableBackup  bool
	}{
		{
			name:          "Development Environment",
			environment:   "dev",
			instanceCount: 1,
			engineVersion: "5.7.12",
			enableBackup:  false,
		},
		{
			name:          "Staging Environment",
			environment:   "staging",
			instanceCount: 2,
			engineVersion: "5.7.12",
			enableBackup:  true,
		},
		{
			name:          "Production Environment",
			environment:   "prod",
			instanceCount: 3,
			engineVersion: "5.7.12",
			enableBackup:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Given
			app := helpers.CreateTestApp(&helpers.TestAppConfig{
				Environment: tc.environment,
			})

			// When: StorageStackを作成
			stack := stacks.NewStorageStack(app, "TestStorageStack", &stacks.StorageStackProps{
				Environment:       tc.environment,
				VpcId:             "vpc-12345",
				IsTestEnvironment: true, // テスト環境フラグ
			})

			// Then: Aurora設定の確認
			template := assertions.Template_FromStack(stack, nil)

			// Aurora Cluster確認
			template.ResourceCountIs(jsii.String("AWS::RDS::DBCluster"), jsii.Number(1))
			template.HasResourceProperties(jsii.String("AWS::RDS::DBCluster"), map[string]interface{}{
				"Engine":        "aurora-mysql",
				"EngineVersion": tc.engineVersion,
			})

			// Aurora Instance確認
			template.ResourceCountIs(jsii.String("AWS::RDS::DBInstance"), jsii.Number(tc.instanceCount))

			assert.NotNil(t, stack)
		})
	}
}

func TestStorageStack_ElastiCache(t *testing.T) {
	// Given
	app := helpers.CreateTestApp(&helpers.TestAppConfig{
		Environment: "dev",
	})

	// When: StorageStackを作成
	stack := stacks.NewStorageStack(app, "TestStorageStack", &stacks.StorageStackProps{
		Environment:       "dev",
		VpcId:             "vpc-12345",
		IsTestEnvironment: true, // テスト環境フラグ
	})

	// Then: ElastiCacheクラスターの確認
	template := assertions.Template_FromStack(stack, nil)
	template.ResourceCountIs(jsii.String("AWS::ElastiCache::ReplicationGroup"), jsii.Number(1))

	// Redis設定確認
	template.HasResourceProperties(jsii.String("AWS::ElastiCache::ReplicationGroup"), map[string]interface{}{
		"Engine":                   "redis",
		"CacheNodeType":            "cache.t3.micro",
		"NumCacheClusters":         1,     // Primary
		"AutomaticFailoverEnabled": false, // 1ノードの場合はfalse
		"MultiAZEnabled":           false, // 開発環境ではfalse
	})

	assert.NotNil(t, stack)
}

func TestStorageStack_ElastiCacheConfigurations(t *testing.T) {
	testCases := []struct {
		name                      string
		environment               string
		expectedNodeType          string
		expectedNumNodes          int
		expectedAutomaticFailover bool
		expectedMultiAZ           bool
	}{
		{
			name:                      "Development - Single Node",
			environment:               "dev",
			expectedNodeType:          "cache.t3.micro",
			expectedNumNodes:          1,
			expectedAutomaticFailover: false, // 1ノードの場合は無効
			expectedMultiAZ:           false,
		},
		{
			name:                      "Staging - Multi Node",
			environment:               "staging",
			expectedNodeType:          "cache.t3.small",
			expectedNumNodes:          2,
			expectedAutomaticFailover: true, // 2ノード以上で有効
			expectedMultiAZ:           true,
		},
		{
			name:                      "Production - High Availability",
			environment:               "prod",
			expectedNodeType:          "cache.r6g.large",
			expectedNumNodes:          3,
			expectedAutomaticFailover: true, // 3ノードで高可用性
			expectedMultiAZ:           true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Given
			app := CreateTestAppForStorageStack(tc.environment)

			// When
			stack := stacks.NewStorageStack(app, "TestStorageStack", &stacks.StorageStackProps{
				Environment:       tc.environment,
				VpcId:             "vpc-12345",
				IsTestEnvironment: true,
			})

			// Then
			template := assertions.Template_FromStack(stack, nil)

			// ElastiCache設定確認
			template.HasResourceProperties(jsii.String("AWS::ElastiCache::ReplicationGroup"), map[string]interface{}{
				"Engine":                   "redis",
				"CacheNodeType":            tc.expectedNodeType,
				"NumCacheClusters":         tc.expectedNumNodes,
				"AutomaticFailoverEnabled": tc.expectedAutomaticFailover,
				"MultiAZEnabled":           tc.expectedMultiAZ,
			})

			assert.NotNil(t, stack)
		})
	}
}

func TestStorageStack_S3Buckets(t *testing.T) {
	// Given
	app := helpers.CreateTestApp(&helpers.TestAppConfig{
		Environment: "dev",
	})

	// When: StorageStackを作成
	stack := stacks.NewStorageStack(app, "TestStorageStack", &stacks.StorageStackProps{
		Environment:       "dev",
		VpcId:             "vpc-12345",
		IsTestEnvironment: true, // テスト環境フラグ
	})

	// Then: S3バケットの確認（静的アセット、ログ、バックアップ用）
	template := assertions.Template_FromStack(stack, nil)
	template.ResourceCountIs(jsii.String("AWS::S3::Bucket"), jsii.Number(3))

	// 静的アセット用バケット確認
	template.HasResourceProperties(jsii.String("AWS::S3::Bucket"), map[string]interface{}{
		"BucketName": "service-development-static-assets",
		"PublicAccessBlockConfiguration": map[string]interface{}{
			"BlockPublicAcls":       true,
			"BlockPublicPolicy":     true,
			"IgnorePublicAcls":      true,
			"RestrictPublicBuckets": true,
		},
	})

	assert.NotNil(t, stack)
}

func TestStorageStack_CrossStackExports(t *testing.T) {
	// Given
	app := helpers.CreateTestApp(&helpers.TestAppConfig{
		Environment: "dev",
	})

	// When: StorageStackを作成
	stack := stacks.NewStorageStack(app, "TestStorageStack", &stacks.StorageStackProps{
		Environment: "dev",
		VpcId:       "vpc-12345",
	})

	// Then: Cross-stack出力の確認
	template := assertions.Template_FromStack(stack, nil)

	// Aurora Endpoint出力
	template.HasOutput(jsii.String("AuroraClusterEndpoint"), map[string]interface{}{
		"Description": "Aurora MySQL Cluster Writer Endpoint",
		"Export": map[string]interface{}{
			"Name": "service-development-Aurora-Endpoint",
		},
	})

	// ElastiCache Endpoint出力
	template.HasOutput(jsii.String("ElastiCacheEndpoint"), map[string]interface{}{
		"Description": "ElastiCache Redis Primary Endpoint",
		"Export": map[string]interface{}{
			"Name": "service-development-Redis-Endpoint",
		},
	})

	// S3 Bucket名出力
	template.HasOutput(jsii.String("StaticAssetsBucketName"), map[string]interface{}{
		"Description": "Static Assets S3 Bucket Name",
		"Export": map[string]interface{}{
			"Name": "service-development-Static-Bucket",
		},
	})

	assert.NotNil(t, stack)
}

// TestStorageStack_SecuritySettings セキュリティ設定のテスト
func TestStorageStack_SecuritySettings(t *testing.T) {
	// Given
	app := CreateTestAppForStorageStack("prod")

	// When
	stack := stacks.NewStorageStack(app, "TestStorageStack", &stacks.StorageStackProps{
		Environment:       "prod",
		VpcId:             "vpc-12345",
		IsTestEnvironment: true, // テスト環境フラグ
	})

	// Then
	template := assertions.Template_FromStack(stack, nil)

	// Aurora暗号化設定確認
	template.HasResourceProperties(jsii.String("AWS::RDS::DBCluster"), map[string]interface{}{
		"StorageEncrypted":   true,
		"DeletionProtection": true, // 本番環境では削除保護有効
	})

	// ElastiCache暗号化設定確認
	template.HasResourceProperties(jsii.String("AWS::ElastiCache::ReplicationGroup"), map[string]interface{}{
		"AtRestEncryptionEnabled":  true,
		"TransitEncryptionEnabled": true,
	})

	// S3バケット暗号化設定確認
	template.HasResourceProperties(jsii.String("AWS::S3::Bucket"), map[string]interface{}{
		"BucketEncryption": map[string]interface{}{
			"ServerSideEncryptionConfiguration": assertions.Match_AnyValue(),
		},
		"PublicAccessBlockConfiguration": map[string]interface{}{
			"BlockPublicAcls":       true,
			"BlockPublicPolicy":     true,
			"IgnorePublicAcls":      true,
			"RestrictPublicBuckets": true,
		},
	})

	assert.NotNil(t, stack)
}

// TestStorageStack_InstanceTypes Aurora環境別インスタンスタイプのテスト
func TestStorageStack_InstanceTypes(t *testing.T) {
	testCases := []struct {
		name                 string
		environment          string
		expectedInstanceType string
		expectedInstances    int
	}{
		{
			name:                 "Development - Small Instance",
			environment:          "dev",
			expectedInstanceType: "db.t3.small",
			expectedInstances:    1,
		},
		{
			name:                 "Staging - Medium Instance",
			environment:          "staging",
			expectedInstanceType: "db.t3.medium",
			expectedInstances:    2,
		},
		{
			name:                 "Production - Large Instance",
			environment:          "prod",
			expectedInstanceType: "db.r5.large",
			expectedInstances:    3,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Given
			app := CreateTestAppForStorageStack(tc.environment)

			// When
			stack := stacks.NewStorageStack(app, "TestStorageStack", &stacks.StorageStackProps{
				Environment:       tc.environment,
				VpcId:             "vpc-12345",
				IsTestEnvironment: true, // テスト環境フラグ
			})

			// Then: RDS設定の確認
			template := assertions.Template_FromStack(stack, nil)

			// Aurora Instance数確認
			template.ResourceCountIs(jsii.String("AWS::RDS::DBInstance"), jsii.Number(tc.expectedInstances))
			template.HasResourceProperties(jsii.String("AWS::RDS::DBInstance"), map[string]interface{}{
				"DBInstanceClass": tc.expectedInstanceType,
			})

			assert.NotNil(t, stack)
		})
	}
}

// TestStorageStack_BackupConfiguration バックアップ設定のテスト
func TestStorageStack_BackupConfiguration(t *testing.T) {
	testCases := []struct {
		name                      string
		environment               string
		expectedRetentionDays     int
		expectedSnapshotRetention int
	}{
		{
			name:                      "Development - Minimal Backup",
			environment:               "dev",
			expectedRetentionDays:     1,
			expectedSnapshotRetention: 1,
		},
		{
			name:                      "Staging - Standard Backup",
			environment:               "staging",
			expectedRetentionDays:     7,
			expectedSnapshotRetention: 3,
		},
		{
			name:                      "Production - Extended Backup",
			environment:               "prod",
			expectedRetentionDays:     30,
			expectedSnapshotRetention: 7,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Given
			app := CreateTestAppForStorageStack(tc.environment)

			// When
			stack := stacks.NewStorageStack(app, "TestStorageStack", &stacks.StorageStackProps{
				Environment:       tc.environment,
				VpcId:             "vpc-12345",
				IsTestEnvironment: true, // テスト環境フラグ
			})

			// Then
			template := assertions.Template_FromStack(stack, nil)

			// Aurora バックアップ設定確認
			template.HasResourceProperties(jsii.String("AWS::RDS::DBCluster"), map[string]interface{}{
				"BackupRetentionPeriod": tc.expectedRetentionDays,
				"PreferredBackupWindow": "03:00-04:00",
			})

			// ElastiCache スナップショット設定確認
			template.HasResourceProperties(jsii.String("AWS::ElastiCache::ReplicationGroup"), map[string]interface{}{
				"SnapshotRetentionLimit": tc.expectedSnapshotRetention,
				"SnapshotWindow":         "03:00-05:00",
			})

			assert.NotNil(t, stack)
		})
	}
}

// TestStorageStack_ErrorHandling エラーハンドリングのテスト
func TestStorageStack_ErrorHandling(t *testing.T) {
	// Given
	app := CreateTestAppForStorageStack("test")

	// When & Then: 無効な環境でpanicが発生することを確認
	assert.Panics(t, func() {
		stacks.NewStorageStack(app, "TestStorageStack", &stacks.StorageStackProps{
			Environment:       "invalid-environment",
			VpcId:             "vpc-12345",
			IsTestEnvironment: true,
		})
	}, "Should panic with invalid environment")

	// When & Then: 空のプロパティでpanicが発生することを確認
	assert.Panics(t, func() {
		stacks.NewStorageStack(app, "TestStorageStack", nil)
	}, "Should panic with nil props")
}
