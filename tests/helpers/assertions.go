package helpers

import (
	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/assertions"
	"github.com/aws/jsii-runtime-go"
)

// VPCAssertions VPCリソースのアサーション
type VPCAssertions struct {
	template assertions.Template
}

// NewVPCAssertions VPCアサーションインスタンスを作成
func NewVPCAssertions(stack awscdk.Stack) *VPCAssertions {
	return &VPCAssertions{
		template: assertions.Template_FromStack(stack, nil),
	}
}

// HasSubnetCount 指定された数のサブネットが存在することを確認
func (v *VPCAssertions) HasSubnetCount(count int) *VPCAssertions {
	v.template.ResourceCountIs(jsii.String("AWS::EC2::Subnet"), jsii.Number(count))
	return v
}

// HasInternetGateway インターネットゲートウェイが存在することを確認
func (v *VPCAssertions) HasInternetGateway() *VPCAssertions {
	v.template.ResourceCountIs(jsii.String("AWS::EC2::InternetGateway"), jsii.Number(1))
	return v
}

// ECSAssertions ECSリソースのアサーション
type ECSAssertions struct {
	template assertions.Template
}

// NewECSAssertions ECSアサーションインスタンスの作成
func NewECSAssertions(stack awscdk.Stack) *ECSAssertions {
	return &ECSAssertions{
		template: assertions.Template_FromStack(stack, nil),
	}
}

// HasCluster ECSクラスターが存在することを確認
func (e *ECSAssertions) HasCluster() *ECSAssertions {
	e.template.ResourceCountIs(jsii.String("AWS::ECS::Cluster"), jsii.Number(1))
	return e
}

// HasServiceWithDesiredCount 指定されたDesired Countのサービスが存在することを確認
func (e *ECSAssertions) HasServiceWithDesiredCount(count int) *ECSAssertions {
	e.template.HasResourceProperties(jsii.String("AWS::ECS::Service"), map[string]interface{}{
		"DesiredCount": count,
	})
	return e
}

// RDSAssertions RDSリソースのアサーション
type RDSAssertions struct {
	template assertions.Template
}

// NewRDSAssertions RDSアサーションインスタンスの作成
func NewRDSAssertions(stack awscdk.Stack) *RDSAssertions {
	return &RDSAssertions{
		template: assertions.Template_FromStack(stack, nil),
	}
}

// HasAuroraCluster Auroraクラスターが存在することを確認
func (r *RDSAssertions) HasAuroraCluster() *RDSAssertions {
	r.template.ResourceCountIs(jsii.String("AWS::RDS::DBCluster"), jsii.Number(1))
	return r
}

// HasEngineVersion 指定されたエンジンバージョンが設定されていることを確認
func (r *RDSAssertions) HasEngineVersion(version string) *RDSAssertions {
	r.template.HasResourceProperties(jsii.String("AWS::RDS::DBCluster"), map[string]interface{}{
		"EngineVersion": version,
	})
	return r
}
