package helpers

import (
	"encoding/json"
	"strings"
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
	TestEnvFlag bool
}

// CreateTestApp テスト用のCDKアプリケーションを作成
func CreateTestApp(config *TestAppConfig) awscdk.App {
	app := awscdk.NewApp(nil)

	if config != nil {
		// 環境設定をコンテキストに追加
		app.Node().SetContext(jsii.String("environment"), jsii.String(config.Environment))
		if config.Region != "" {
			app.Node().SetContext(jsii.String("region"), jsii.String(config.Region))
		}
		if config.Account != "" {
			app.Node().SetContext(jsii.String("account"), jsii.String(config.Account))
		}
	}

	return app
}

// CreateTestAppForUnitTest ユニットテスト用のアプリケーション作成（環境のみ指定）
func CreateTestAppForUnitTest(environment string) awscdk.App {
	return CreateTestApp(&TestAppConfig{
		Environment: environment,
		Region:      "ap-northeast-1",
		Account:     "123456789012",
		TestEnvFlag: true,
	})
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

// AssertStackHasOutput スタックに特定の出力が存在することを確認
func AssertStackHasOutput(t *testing.T, stack awscdk.Stack, outputKey string, expectedExportName string) {
	template := assertions.Template_FromStack(stack, nil)
	template.HasOutput(jsii.String(outputKey), map[string]interface{}{
		"Export": map[string]interface{}{
			"Name": expectedExportName,
		},
	})
}

// AssertCrossStackReference Cross-stack参照が正しく設定されていることを確認
func AssertCrossStackReference(t *testing.T, stack awscdk.Stack, importValueName string) {
	template := assertions.Template_FromStack(stack, nil)
	templateJSON := template.ToJSON()

	// CloudFormationテンプレートをJSON文字列に変換
	templateBytes, err := json.Marshal(templateJSON)
	if err != nil {
		t.Errorf("Failed to marshal template: %v", err)
		return
	}

	templateString := string(templateBytes)

	// Fn::ImportValueの存在を文字列検索で確認（簡単で確実な方法）
	expectedImportValue := `"Fn::ImportValue":"` + importValueName + `"`
	if !strings.Contains(templateString, expectedImportValue) {
		// 別の形式もチェック
		expectedImportValueAlt := `{"Fn::ImportValue":"` + importValueName + `"}`
		if !strings.Contains(templateString, expectedImportValueAlt) {
			t.Errorf("Cross-stack reference to '%s' not found in stack template", importValueName)
		}
	}
}

// containsImportValue 再帰的にFn::ImportValueを検索（詳細版）
func containsImportValue(obj interface{}, targetValue string) bool {
	switch v := obj.(type) {
	case map[string]interface{}:
		// Fn::ImportValueを直接チェック
		if fnImport, ok := v["Fn::ImportValue"]; ok {
			if importVal, ok := fnImport.(string); ok && importVal == targetValue {
				return true
			}
		}
		// 再帰的に子要素を検索
		for _, value := range v {
			if containsImportValue(value, targetValue) {
				return true
			}
		}
	case []interface{}:
		for _, item := range v {
			if containsImportValue(item, targetValue) {
				return true
			}
		}
	case string:
		// 文字列の場合は直接比較
		return v == targetValue
	}
	return false
}

// GetStackOutputs スタックの出力値を取得（実用版）
func GetStackOutputs(stack awscdk.Stack) map[string]interface{} {
	template := assertions.Template_FromStack(stack, nil)
	templateJSON := template.ToJSON()

	outputs := make(map[string]interface{})
	if outputsSection, ok := (*templateJSON)["Outputs"].(map[string]interface{}); ok {
		for key, value := range outputsSection {
			outputs[key] = value
		}
	}

	return outputs
}

// ValidateStackTemplateSize CloudFormationテンプレートサイズの妥当性確認
func ValidateStackTemplateSize(t *testing.T, stack awscdk.Stack, maxSizeKB int) {
	template := assertions.Template_FromStack(stack, nil)
	templateJSON := template.ToJSON()

	// テンプレートのJSON文字列サイズを測定
	templateBytes, err := json.Marshal(templateJSON)
	if err != nil {
		t.Errorf("Failed to marshal template: %v", err)
		return
	}

	templateSize := len(templateBytes)
	maxSizeBytes := maxSizeKB * 1024

	if templateSize > maxSizeBytes {
		t.Errorf("CloudFormation template size %d bytes exceeds limit %d KB (%d bytes)",
			templateSize, maxSizeKB, maxSizeBytes)
	} else {
		t.Logf("Template size: %d bytes (limit: %d KB)", templateSize, maxSizeKB)
	}
}

// AssertEnvironmentSpecificTags 環境固有のタグが設定されていることを確認
func AssertEnvironmentSpecificTags(t *testing.T, stack awscdk.Stack, environment string, resourceType string) {
	template := assertions.Template_FromStack(stack, nil)

	expectedTags := map[string]interface{}{
		"Tags": assertions.Match_ArrayWith(&[]interface{}{
			map[string]interface{}{
				"Key":   "Environment",
				"Value": getEnvironmentName(environment),
			},
			map[string]interface{}{
				"Key":   "Project",
				"Value": "PracticeService",
			},
		}),
	}

	template.HasResourceProperties(jsii.String(resourceType), expectedTags)
}

// getEnvironmentName 環境名の正規化
func getEnvironmentName(env string) string {
	switch env {
	case "dev":
		return "development"
	case "prod":
		return "production"
	default:
		return env
	}
}

// AssertSecurityGroupRules セキュリティグループルールの確認
func AssertSecurityGroupRules(t *testing.T, stack awscdk.Stack, expectedIngressRules int) {
	template := assertions.Template_FromStack(stack, nil)
	templateJSON := template.ToJSON()

	resources := (*templateJSON)["Resources"].(map[string]interface{})
	totalIngressRules := 0
	sgCount := 0

	for resourceId, resource := range resources {
		resourceData := resource.(map[string]interface{})
		if resourceData["Type"] == "AWS::EC2::SecurityGroup" {
			sgCount++
			if properties, ok := resourceData["Properties"].(map[string]interface{}); ok {
				if ingress, ok := properties["SecurityGroupIngress"].([]interface{}); ok {
					rulesCount := len(ingress)
					totalIngressRules += rulesCount
					t.Logf("SecurityGroup %s has %d ingress rules", resourceId, rulesCount)
				}
			}
		}
	}

	t.Logf("Total SecurityGroups: %d, Total Ingress Rules: %d", sgCount, totalIngressRules)

	if totalIngressRules < expectedIngressRules {
		t.Errorf("Expected at least %d ingress rules, found %d", expectedIngressRules, totalIngressRules)
	}
}

// AssertResourceTags 特定リソースのタグを確認
func AssertResourceTags(t *testing.T, stack awscdk.Stack, resourceType string, expectedTags map[string]string) {
	template := assertions.Template_FromStack(stack, nil)

	// expectedTagsをassertions形式に変換
	tagMatchers := make([]interface{}, 0, len(expectedTags))
	for key, value := range expectedTags {
		tagMatchers = append(tagMatchers, map[string]interface{}{
			"Key":   key,
			"Value": value,
		})
	}

	template.HasResourceProperties(jsii.String(resourceType), map[string]interface{}{
		"Tags": assertions.Match_ArrayWith(&tagMatchers),
	})
}

// GetResourceProperties 特定リソースのプロパティを取得
func GetResourceProperties(t *testing.T, stack awscdk.Stack, resourceType string) map[string]interface{} {
	template := assertions.Template_FromStack(stack, nil)
	templateJSON := template.ToJSON()

	resources := (*templateJSON)["Resources"].(map[string]interface{})

	for _, resource := range resources {
		resourceData := resource.(map[string]interface{})
		if resourceData["Type"] == resourceType {
			if properties, ok := resourceData["Properties"].(map[string]interface{}); ok {
				return properties
			}
		}
	}

	t.Errorf("Resource type %s not found in stack", resourceType)
	return nil
}

// AssertSubnetConfiguration サブネット設定の確認
func AssertSubnetConfiguration(t *testing.T, stack awscdk.Stack, expectedPublicSubnets int, expectedPrivateSubnets int) {
	template := assertions.Template_FromStack(stack, nil)

	// 実際のサブネット数確認
	template.ResourceCountIs(jsii.String("AWS::EC2::Subnet"), jsii.Number(expectedPublicSubnets+expectedPrivateSubnets))

	templateJSON := template.ToJSON()
	resources := (*templateJSON)["Resources"].(map[string]interface{})

	publicCount := 0
	privateCount := 0

	for _, resource := range resources {
		resourceData := resource.(map[string]interface{})
		if resourceData["Type"] == "AWS::EC2::Subnet" {
			if properties, ok := resourceData["Properties"].(map[string]interface{}); ok {
				// MapPublicIpOnLaunchでPublic/Privateを判定
				if mapPublic, ok := properties["MapPublicIpOnLaunch"].(bool); ok && mapPublic {
					publicCount++
				} else {
					privateCount++
				}
			}
		}
	}

	t.Logf("Found %d public subnets, %d private subnets", publicCount, privateCount)

	if publicCount != expectedPublicSubnets {
		t.Errorf("Expected %d public subnets, found %d", expectedPublicSubnets, publicCount)
	}
	if privateCount != expectedPrivateSubnets {
		t.Errorf("Expected %d private subnets, found %d", expectedPrivateSubnets, privateCount)
	}
}

// ValidateStackNaming スタック名の命名規則確認
func ValidateStackNaming(t *testing.T, stack awscdk.Stack, expectedPrefix string, environment string) {
	stackName := *stack.StackName()
	expectedName := expectedPrefix + "Stack"

	if !strings.Contains(stackName, expectedName) {
		t.Errorf("Stack name '%s' does not contain expected pattern '%s'", stackName, expectedName)
	}

	t.Logf("Stack name validation passed: %s", stackName)
}
