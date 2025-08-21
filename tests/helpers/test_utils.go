package helpers

import (
	"os"
	"testing"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/assertions"
	"github.com/aws/jsii-runtime-go"
)

// TestAppConfig テストアプリケーションの設定
type TestAppConfig struct {
	Environment string
	Region      string
	Account     string
}

// CreateTestApp テスト用のCDKアプリケーションを作成
func CreateTestApp(config *TestAppConfig) awscdk.App {
	app := awscdk.NewApp(nil)

	// 設定が指定されない場合は環境変数から取得
	if config == nil {
		config = &TestAppConfig{
			Environment: getEnvWithDefault("CDK_ENVIRONMENT", "test"),
			Region:      getEnvWithDefault("CDK_DEFAULT_REGION", "ap-northeast-1"),
			Account:     getEnvWithDefault("CDK_DEFAULT_ACCOUNT", "123456789012"), // テスト用デフォルト
		}
	} else {
		// 個別設定が空の場合は環境変数で補完
		if config.Environment == "" {
			config.Environment = getEnvWithDefault("CDK_ENVIRONMENT", "test")
		}
		if config.Region == "" {
			config.Region = getEnvWithDefault("CDK_DEFAULT_REGION", "ap-northeast-1")
		}
		if config.Account == "" {
			config.Account = getEnvWithDefault("CDK_DEFAULT_ACCOUNT", "123456789012")
		}
	}

	// 環境設定をコンテキストに追加
	app.Node().SetContext(jsii.String("environment"), jsii.String(config.Environment))
	app.Node().SetContext(jsii.String("region"), jsii.String(config.Region))
	app.Node().SetContext(jsii.String("account"), jsii.String(config.Account))

	return app
}

// CreateTestAppWithEnv 環境変数のみでテストアプリを作成
func CreateTestAppWithEnv() awscdk.App {
	return CreateTestApp(nil)
}

// CreateTestAppForUnitTest 単体テスト用（アカウント非依存）
func CreateTestAppForUnitTest(environment string) awscdk.App {
	return CreateTestApp(&TestAppConfig{
		Environment: environment,
		Region:      "ap-northeast-1",
		Account:     "123456789012", // 単体テスト用固定値
	})
}

// ヘルパー関数：環境変数取得とデフォルト値設定
func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// AssertStackHasResource スタックに特定のリソースが存在することを確認
func AssertStackHasResource(t *testing.T, stack awscdk.Stack, resourceType string, count int) {
	template := assertions.Template_FromStack(stack, nil)
	template.ResourceCountIs(jsii.String(resourceType), jsii.Number(count))
}

// AssertStackHasResourceWithProperties スタックに特定のプロパティを持つリソースが存在することを確認
func AssertStackHasResourceWithProperties(t *testing.T, stack awscdk.Stack, resourceType string, properties map[string]interface{}) {
	template := assertions.Template_FromStack(stack, nil)
	template.HasResourceProperties(jsii.String(resourceType), properties)
}

// GetStackOutputs スタックの出力値を取得
func GetStackOutputs(stack awscdk.Stack) map[string]awscdk.CfnOutput {
	outputs := make(map[string]awscdk.CfnOutput)
	// 実装はスタック出力の取得ロジック
	return outputs
}
