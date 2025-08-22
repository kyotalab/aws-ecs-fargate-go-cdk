package constructs

import (
	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsec2"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

// SecurityGroupsProps セキュリティグループ作成のプロパティ
type SecurityGroupsProps struct {
	Vpc         awsec2.IVpc
	Environment string
}

// SecurityGroupsResult セキュリティグループの作成結果
type SecurityGroupsResult struct {
	ALBSecurityGroup awsec2.SecurityGroup
	ECSSecurityGroup awsec2.SecurityGroup
	RDSSecurityGroup awsec2.SecurityGroup
}

// CreateSecurityGroups セキュリティグループ群を作成（関数ベース）
func CreateSecurityGroups(scope constructs.Construct, props *SecurityGroupsProps) *SecurityGroupsResult {
	result := &SecurityGroupsResult{}

	// ALB用セキュリティグループ
	result.ALBSecurityGroup = createALBSecurityGroup(scope, props)

	// ECS用セキュリティグループ
	result.ECSSecurityGroup = createECSSecurityGroup(scope, props, result.ALBSecurityGroup)

	// RDS用セキュリティグループ
	result.RDSSecurityGroup = createRDSSecurityGroup(scope, props, result.ECSSecurityGroup)

	return result
}

// createALBSecurityGroup ALB用セキュリティグループを作成
func createALBSecurityGroup(scope constructs.Construct, props *SecurityGroupsProps) awsec2.SecurityGroup {
	albSG := awsec2.NewSecurityGroup(scope, jsii.String("ALBSecurityGroup"), &awsec2.SecurityGroupProps{
		Vpc:               props.Vpc,
		Description:       jsii.String("Security group for ALB"),
		SecurityGroupName: jsii.String("Service-" + props.Environment + "-ALB-SG"),
		AllowAllOutbound:  jsii.Bool(true),
	})

	// HTTP/HTTPS アクセス許可
	addHTTPIngressRules(albSG)

	// タグ追加
	awscdk.Tags_Of(albSG).Add(jsii.String("Name"), jsii.String("Service-"+props.Environment+"-ALB-SG"), nil)
	awscdk.Tags_Of(albSG).Add(jsii.String("Environment"), jsii.String(props.Environment), nil)
	awscdk.Tags_Of(albSG).Add(jsii.String("Component"), jsii.String("LoadBalancer"), nil)

	return albSG
}

// createECSSecurityGroup ECS用セキュリティグループを作成
func createECSSecurityGroup(scope constructs.Construct, props *SecurityGroupsProps, albSG awsec2.SecurityGroup) awsec2.SecurityGroup {
	ecsSG := awsec2.NewSecurityGroup(scope, jsii.String("ECSSecurityGroup"), &awsec2.SecurityGroupProps{
		Vpc:               props.Vpc,
		Description:       jsii.String("Security group for ECS tasks"),
		SecurityGroupName: jsii.String("Service-" + props.Environment + "-ECS-SG"),
		AllowAllOutbound:  jsii.Bool(true),
	})

	// ALBからのアクセス許可
	ecsSG.AddIngressRule(
		awsec2.Peer_SecurityGroupId(albSG.SecurityGroupId(), nil),
		awsec2.Port_Tcp(jsii.Number(80)),
		jsii.String("Allow HTTP traffic from ALB"),
		jsii.Bool(false),
	)

	// 動的ポート範囲のアクセス許可（ECS Dynamic Port Mapping用）
	ecsSG.AddIngressRule(
		awsec2.Peer_SecurityGroupId(albSG.SecurityGroupId(), nil),
		awsec2.Port_TcpRange(jsii.Number(32768), jsii.Number(65535)),
		jsii.String("Allow dynamic port range from ALB"),
		jsii.Bool(false),
	)

	// タグ追加
	awscdk.Tags_Of(ecsSG).Add(jsii.String("Name"), jsii.String("Service-"+props.Environment+"-ECS-SG"), nil)
	awscdk.Tags_Of(ecsSG).Add(jsii.String("Environment"), jsii.String(props.Environment), nil)
	awscdk.Tags_Of(ecsSG).Add(jsii.String("Component"), jsii.String("Application"), nil)

	return ecsSG
}

// createRDSSecurityGroup RDS用セキュリティグループを作成
func createRDSSecurityGroup(scope constructs.Construct, props *SecurityGroupsProps, ecsSG awsec2.SecurityGroup) awsec2.SecurityGroup {
	rdsSG := awsec2.NewSecurityGroup(scope, jsii.String("RDSSecurityGroup"), &awsec2.SecurityGroupProps{
		Vpc:               props.Vpc,
		Description:       jsii.String("Security group for RDS database"),
		SecurityGroupName: jsii.String("Service-" + props.Environment + "-RDS-SG"),
		AllowAllOutbound:  jsii.Bool(false), // データベースは外部通信不要
	})

	// MySQL アクセス許可
	rdsSG.AddIngressRule(
		awsec2.Peer_SecurityGroupId(ecsSG.SecurityGroupId(), nil),
		awsec2.Port_Tcp(jsii.Number(3306)),
		jsii.String("Allow MySQL traffic from ECS"),
		jsii.Bool(false),
	)

	// Redis アクセス許可
	rdsSG.AddIngressRule(
		awsec2.Peer_SecurityGroupId(ecsSG.SecurityGroupId(), nil),
		awsec2.Port_Tcp(jsii.Number(6379)),
		jsii.String("Allow Redis traffic from ECS"),
		jsii.Bool(false),
	)

	// タグ追加
	awscdk.Tags_Of(rdsSG).Add(jsii.String("Name"), jsii.String("Service-"+props.Environment+"-RDS-SG"), nil)
	awscdk.Tags_Of(rdsSG).Add(jsii.String("Environment"), jsii.String(props.Environment), nil)
	awscdk.Tags_Of(rdsSG).Add(jsii.String("Component"), jsii.String("Database"), nil)

	return rdsSG
}

// addHTTPIngressRules HTTP/HTTPSのIngressルールを追加
func addHTTPIngressRules(securityGroup awsec2.SecurityGroup) {
	// HTTP (80)
	securityGroup.AddIngressRule(
		awsec2.Peer_AnyIpv4(),
		awsec2.Port_Tcp(jsii.Number(80)),
		jsii.String("Allow HTTP traffic from internet"),
		jsii.Bool(false),
	)

	// HTTPS (443)
	securityGroup.AddIngressRule(
		awsec2.Peer_AnyIpv4(),
		awsec2.Port_Tcp(jsii.Number(443)),
		jsii.String("Allow HTTPS traffic from internet"),
		jsii.Bool(false),
	)
}
