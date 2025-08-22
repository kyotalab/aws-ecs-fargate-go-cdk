package stacks

import (
	"aws-ecs-fargate-go-cdk/internal/config"

	"github.com/aws/aws-cdk-go/awscdk/v2"
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

	// タグ追加
	addApplicationStackTags(stack, envConfig)
	awscdk.Tags_Of(cluster).Add(jsii.String("Component"), jsii.String("Application"), nil)

	return stack
}

// addApplicationStackTags ApplicationStack全体にタグを追加
func addApplicationStackTags(stack awscdk.Stack, envConfig *config.EnvironmentConfig) {
	for key, value := range envConfig.Tags {
		awscdk.Tags_Of(stack).Add(jsii.String(key), jsii.String(value), nil)
	}
	awscdk.Tags_Of(stack).Add(jsii.String("StackType"), jsii.String("Application"), nil)
	awscdk.Tags_Of(stack).Add(jsii.String("ManagedBy"), jsii.String("CDK"), nil)
}
