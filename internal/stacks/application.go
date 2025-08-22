package stacks

import (
	"aws-ecs-fargate-go-cdk/internal/config"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsec2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsecr"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsecs"
	"github.com/aws/aws-cdk-go/awscdk/v2/awselasticloadbalancingv2"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

// ApplicationStackProps ApplicationStackのプロパティ
type ApplicationStackProps struct {
	awscdk.StackProps
	Environment      string
	VpcId            string
	DatabaseEndpoint string
	RedisEndpoint    string
	TestEngFlag      bool
}

// VPCReferenceProps インターフェースの実装
func (p *ApplicationStackProps) GetEnvironment() string {
	return p.Environment
}

func (p *ApplicationStackProps) GetVpcId() string {
	return p.VpcId
}

func (p *ApplicationStackProps) IsTestEnvironment() bool {
	return p.TestEngFlag
}

// ApplicationStack ApplicationStackの構造体
type ApplicationStack struct {
	awscdk.Stack
	ECSCluster    awsecs.Cluster
	ECSService    awsecs.FargateService
	LoadBalancer  awselasticloadbalancingv2.ApplicationLoadBalancer
	ECRRepository awsecr.Repository
}

// NewApplicationStack ApplicationStackを作成（最小実装）
func NewApplicationStack(scope constructs.Construct, id string, props *ApplicationStackProps) awscdk.Stack {
	var sprops awscdk.StackProps
	if props != nil {
		sprops = props.StackProps
	}
	stack := awscdk.NewStack(scope, &id, &sprops)

	// プロパティバリデーション
	if props == nil || props.Environment == "" {
		panic("ApplicationStackProps with Environment is required")
	}

	// 環境設定を取得
	envConfig, err := config.GetEnvironmentConfig(props.Environment)
	if err != nil {
		panic("Invalid environment: " + props.Environment)
	}

	// VPCの参照を取得（ジェネリクス関数使用）
	vpc := GetVPCReference(stack, props)

	// 最小限のECS Clusterを作成（テストを通すため）
	cluster := awsecs.NewCluster(stack, jsii.String("ServiceCluster"), &awsecs.ClusterProps{
		Vpc:         vpc,
		ClusterName: jsii.String("service-" + props.Environment + "-cluster"),

		// コンテナインサイト有効化（本番環境のみ）
		ContainerInsights: jsii.Bool(envConfig.Name == "production"),
	})

	// Application Load Balancer作成
	alb := createApplicationLoadBalancer(stack, vpc, props.Environment)

	// Target Group作成
	targetGroup := createTargetGroup(stack, vpc, props.Environment)

	// ALBにTarget Groupを関連付け
	addALBListener(alb, targetGroup)

	// タグ追加
	addApplicationStackTags(stack, envConfig)
	awscdk.Tags_Of(cluster).Add(jsii.String("Component"), jsii.String("Application"), nil)
	awscdk.Tags_Of(alb).Add(jsii.String("Component"), jsii.String("LoadBalancer"), nil)

	return stack
}

// createApplicationLoadBalancer Application Load Balancerを作成
func createApplicationLoadBalancer(stack awscdk.Stack, vpc awsec2.IVpc, environment string) awselasticloadbalancingv2.ApplicationLoadBalancer {
	return awselasticloadbalancingv2.NewApplicationLoadBalancer(stack, jsii.String("ServiceALB"), &awselasticloadbalancingv2.ApplicationLoadBalancerProps{
		Vpc:              vpc,
		InternetFacing:   jsii.Bool(true), // インターネット向け
		LoadBalancerName: jsii.String("service-" + environment + "-alb"),

		// パブリックサブネットに配置
		VpcSubnets: &awsec2.SubnetSelection{
			SubnetType: awsec2.SubnetType_PUBLIC,
		},

		// セキュリティグループ（Cross-stack参照）
		SecurityGroup: getALBSecurityGroup(stack, environment),
	})
}

// createTargetGroup Target Groupを作成
func createTargetGroup(stack awscdk.Stack, vpc awsec2.IVpc, environment string) awselasticloadbalancingv2.ApplicationTargetGroup {
	return awselasticloadbalancingv2.NewApplicationTargetGroup(stack, jsii.String("ServiceTargetGroup"), &awselasticloadbalancingv2.ApplicationTargetGroupProps{
		Vpc:             vpc,
		Port:            jsii.Number(80),
		Protocol:        awselasticloadbalancingv2.ApplicationProtocol_HTTP,
		TargetType:      awselasticloadbalancingv2.TargetType_IP, // Fargate用
		TargetGroupName: jsii.String("service-" + environment + "-tg"),
		HealthCheck: &awselasticloadbalancingv2.HealthCheck{
			// ヘルスチェック設定
			Path:                    jsii.String("/health"),
			Interval:                awscdk.Duration_Seconds(jsii.Number(30)),
			HealthyThresholdCount:   jsii.Number(2),
			UnhealthyThresholdCount: jsii.Number(5),
		},
	})
}

// addALBListener ALBにListenerを追加
func addALBListener(alb awselasticloadbalancingv2.ApplicationLoadBalancer, targetGroup awselasticloadbalancingv2.ApplicationTargetGroup) {
	alb.AddListener(jsii.String("HTTPListener"), &awselasticloadbalancingv2.BaseApplicationListenerProps{
		Port:     jsii.Number(80),
		Protocol: awselasticloadbalancingv2.ApplicationProtocol_HTTP,
		DefaultTargetGroups: &[]awselasticloadbalancingv2.IApplicationTargetGroup{
			targetGroup,
		},
	})
}

// getALBSecurityGroup ALB用セキュリティグループを取得（Cross-stack参照）
func getALBSecurityGroup(stack awscdk.Stack, environment string) awsec2.ISecurityGroup {
	envName := environment
	if envName == "prod" {
		envName = "production"
	}

	// NetworkStackからセキュリティグループIDをインポート
	sgId := awscdk.Fn_ImportValue(jsii.String("Service-" + envName + "-ALB-SG-Id"))

	return awsec2.SecurityGroup_FromSecurityGroupId(
		stack,
		jsii.String("ImportedALBSecurityGroup"),
		sgId,
		nil,
	)
}

// addApplicationStackTags ApplicationStack全体にタグを追加
func addApplicationStackTags(stack awscdk.Stack, envConfig *config.EnvironmentConfig) {
	for key, value := range envConfig.Tags {
		awscdk.Tags_Of(stack).Add(jsii.String(key), jsii.String(value), nil)
	}
	awscdk.Tags_Of(stack).Add(jsii.String("StackType"), jsii.String("Application"), nil)
	awscdk.Tags_Of(stack).Add(jsii.String("ManagedBy"), jsii.String("CDK"), nil)
}
