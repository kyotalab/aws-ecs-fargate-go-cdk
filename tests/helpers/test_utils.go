package helpers

import (
	"testing"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/assertions"
	"github.com/aws/jsii-runtime-go"
)

// TestAppConfig テストアプリケーションの設定
type TestAppConfig struct {
	Environment       string
	Region            string
	Account           string
	IsTestEnvironment bool
}

// CreateTestApp テスト用のCDKアプリケーションを作成
func CreateTestApp(config *TestAppConfig) awscdk.App {
	app := awscdk.NewApp(nil)

	if config != nil {
		// 環境設定をコンテキストに追加
		app.Node().SetContext(jsii.String("environment"), jsii.String(config.Environment))
		app.Node().SetContext(jsii.String("region"), jsii.String(config.Region))
		app.Node().SetContext(jsii.String("account"), jsii.String(config.Account))
	}

	return app
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
