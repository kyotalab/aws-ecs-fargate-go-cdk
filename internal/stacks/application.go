package stacks

import (
	"aws-ecs-fargate-go-cdk/internal/config"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsapplicationautoscaling"
	"github.com/aws/aws-cdk-go/awscdk/v2/awscloudwatch"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsec2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsecr"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsecs"
	"github.com/aws/aws-cdk-go/awscdk/v2/awselasticloadbalancingv2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsiam"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslogs"
	"github.com/aws/aws-cdk-go/awscdk/v2/awssecretsmanager"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsservicediscovery"
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
	TestEnvFlag      bool
}

// VPCReferenceProps インターフェースの実装
func (p *ApplicationStackProps) GetEnvironment() string {
	return p.Environment
}

func (p *ApplicationStackProps) GetVpcId() string {
	return p.VpcId
}

func (p *ApplicationStackProps) IsTestEnvironment() bool {
	return p.TestEnvFlag
}

// ApplicationStack ApplicationStackの構造体
type ApplicationStack struct {
	awscdk.Stack
	ECSCluster       awsecs.Cluster
	ECSService       awsecs.FargateService
	TaskDefinition   awsecs.FargateTaskDefinition
	LoadBalancer     awselasticloadbalancingv2.ApplicationLoadBalancer
	TargetGroup      awselasticloadbalancingv2.ApplicationTargetGroup
	ECRRepository    awsecr.Repository
	ServiceDiscovery awsservicediscovery.Service
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

	ecsConfig := config.GetECSConfig(props.Environment)

	// 環境設定を取得
	envConfig, err := config.GetEnvironmentConfig(props.Environment)
	if err != nil {
		panic("Invalid environment: " + props.Environment)
	}

	// VPCの参照を取得（ジェネリクス関数使用）
	vpc := GetVPCReference(stack, props)

	cluster := awsecs.NewCluster(stack, jsii.String("ServiceCluster"), &awsecs.ClusterProps{
		Vpc:         vpc,
		ClusterName: jsii.String("service-" + props.Environment + "-cluster"),

		// コンテナインサイト有効化（本番環境のみ）

		ContainerInsightsV2: func() awsecs.ContainerInsights {
			if envConfig.Name == "production" {
				return awsecs.ContainerInsights_ENHANCED // 本番環境では拡張モニタリング
			}
			return awsecs.ContainerInsights_DISABLED // 開発・ステージング環境では無効
		}(),
	})

	// ECR Repository作成
	ecrRepository := createECRRepository(stack, props.Environment, envConfig)

	// Application Load Balancer作成
	alb := createApplicationLoadBalancer(stack, vpc, props.Environment)

	// // Target Group作成
	// targetGroup := createTargetGroup(stack, vpc, props.Environment)

	// // ALBにTarget Groupを関連付け
	// addALBListener(alb, targetGroup)

	// 🆕 Task Definition作成
	taskDefinition := createTaskDefinition(stack, ecsConfig, props)

	// 🆕 Container Definitions作成
	createContainerDefinitions(stack, taskDefinition, ecsConfig, ecrRepository, props)

	// 🆕 ECS Service作成
	ecsService, targetGroup := createECSServiceWithALB(stack, cluster, taskDefinition, alb, ecsConfig, vpc, props.Environment)

	// 🆕 Service Discovery作成（本番環境のみ）
	// var serviceDiscovery awsservicediscovery.Service
	if ecsConfig.EnableServiceDiscovery {
		serviceDiscovery := createServiceDiscovery(stack, cluster, ecsService, props.Environment)

		// Service Discovery ARN出力
		awscdk.NewCfnOutput(stack, jsii.String("ServiceDiscoveryARN"), &awscdk.CfnOutputProps{
			Value:       serviceDiscovery.ServiceArn(),
			Description: jsii.String("Service Discovery ARN"),
			ExportName:  jsii.String("Service-" + props.Environment + "-SD-ARN"),
		})

		// 内部DNS名出力
		awscdk.NewCfnOutput(stack, jsii.String("InternalServiceDNS"), &awscdk.CfnOutputProps{
			Value:       jsii.String("api.service.local"),
			Description: jsii.String("Internal DNS name for service communication"),
			ExportName:  jsii.String("Service-" + props.Environment + "-Internal-DNS"),
		})

	}

	// 🆕 Auto Scaling設定
	setupAutoScaling(ecsService, targetGroup, ecsConfig, props.Environment)

	// Cross-stack出力の作成
	createApplicationStackOutputs(stack, ecrRepository, alb, targetGroup, props.Environment)

	// タグ追加
	addApplicationStackTags(stack, envConfig)
	awscdk.Tags_Of(cluster).Add(jsii.String("Component"), jsii.String("Application"), nil)
	awscdk.Tags_Of(alb).Add(jsii.String("Component"), jsii.String("LoadBalancer"), nil)
	awscdk.Tags_Of(ecrRepository).Add(jsii.String("Component"), jsii.String("ContainerRegistry"), nil)

	return stack
}

// createECRRepository ECR Repositoryを作成
func createECRRepository(stack awscdk.Stack, environment string, envConfig *config.EnvironmentConfig) awsecr.Repository {
	return awsecr.NewRepository(stack, jsii.String("ServiceECRRepository"), &awsecr.RepositoryProps{
		RepositoryName: jsii.String("service-" + environment),

		// イメージスキャンを有効化
		ImageScanOnPush: jsii.Bool(true),

		// ライフサイクルポリシー（環境別設定）
		LifecycleRules: getECRLifecycleRules(envConfig.Name),

		// 削除保護（本番環境のみ）
		RemovalPolicy: func() awscdk.RemovalPolicy {
			if envConfig.Name == "production" {
				return awscdk.RemovalPolicy_RETAIN
			}
			return awscdk.RemovalPolicy_DESTROY
		}(),

		// イメージタグの可変性（本番環境では不変にすることを推奨）
		ImageTagMutability: func() awsecr.TagMutability {
			if envConfig.Name == "production" {
				return awsecr.TagMutability_IMMUTABLE
			}
			return awsecr.TagMutability_MUTABLE
		}(),
	})
}

// getECRLifecycleRules 環境別のECRライフサイクルルールを取得
func getECRLifecycleRules(environment string) *[]*awsecr.LifecycleRule {
	switch environment {
	case "development":
		// 開発環境：最新5つのイメージのみ保持
		return &[]*awsecr.LifecycleRule{
			{
				Description:   jsii.String("Keep only 5 latest images"),
				MaxImageCount: jsii.Number(5),
				RulePriority:  jsii.Number(1),
				TagStatus:     awsecr.TagStatus_ANY,
			},
		}
	case "staging":
		// ステージング環境：最新10つのイメージを保持
		return &[]*awsecr.LifecycleRule{
			{
				Description:   jsii.String("Keep only 10 latest images"),
				MaxImageCount: jsii.Number(10),
				RulePriority:  jsii.Number(1),
				TagStatus:     awsecr.TagStatus_ANY,
			},
		}
	case "production":
		// 本番環境：タグ付きイメージは30日、未タグは1日
		return &[]*awsecr.LifecycleRule{
			{
				Description:  jsii.String("Keep tagged images for 30 days"),
				MaxImageAge:  awscdk.Duration_Days(jsii.Number(30)),
				RulePriority: jsii.Number(1),
				TagStatus:    awsecr.TagStatus_TAGGED,
				// TagPrefixListを追加してエラーを解決
				TagPrefixList: &[]*string{
					jsii.String("v"),      // v1.0.0, v2.0.0 等のバージョンタグ
					jsii.String("prod"),   // prod-xxx 等の本番タグ
					jsii.String("stable"), // stable-xxx 等の安定版タグ
				},
			},
			{
				Description:  jsii.String("Keep untagged images for 1 day"),
				MaxImageAge:  awscdk.Duration_Days(jsii.Number(1)),
				RulePriority: jsii.Number(2),
				TagStatus:    awsecr.TagStatus_UNTAGGED,
			},
		}
	default:
		// デフォルト：開発環境と同じ
		return &[]*awsecr.LifecycleRule{
			{
				Description:   jsii.String("Keep only 5 latest images"),
				MaxImageCount: jsii.Number(5),
				RulePriority:  jsii.Number(1),
				TagStatus:     awsecr.TagStatus_ANY,
			},
		}
	}
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
// func createTargetGroup(stack awscdk.Stack, vpc awsec2.IVpc, environment string) awselasticloadbalancingv2.ApplicationTargetGroup {
// 	return awselasticloadbalancingv2.NewApplicationTargetGroup(stack, jsii.String("ServiceTargetGroup"), &awselasticloadbalancingv2.ApplicationTargetGroupProps{
// 		Vpc:             vpc,
// 		Port:            jsii.Number(80),
// 		Protocol:        awselasticloadbalancingv2.ApplicationProtocol_HTTP,
// 		TargetType:      awselasticloadbalancingv2.TargetType_IP, // Fargate用
// 		TargetGroupName: jsii.String("service-" + environment + "-tg"),
// 		HealthCheck: &awselasticloadbalancingv2.HealthCheck{
// 			// ヘルスチェック設定
// 			Path:                    jsii.String("/health"),
// 			Interval:                awscdk.Duration_Seconds(jsii.Number(30)),
// 			HealthyThresholdCount:   jsii.Number(2),
// 			UnhealthyThresholdCount: jsii.Number(5),
// 		},
// 	})
// }

// addALBListener ALBにListenerを追加
// func addALBListener(alb awselasticloadbalancingv2.ApplicationLoadBalancer, targetGroup awselasticloadbalancingv2.ApplicationTargetGroup) {
// 	alb.AddListener(jsii.String("HTTPListener"), &awselasticloadbalancingv2.BaseApplicationListenerProps{
// 		Port:     jsii.Number(80),
// 		Protocol: awselasticloadbalancingv2.ApplicationProtocol_HTTP,
// 		DefaultTargetGroups: &[]awselasticloadbalancingv2.IApplicationTargetGroup{
// 			targetGroup,
// 		},
// 	})
// }

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

// createTaskDefinition Task Definitionを作成
func createTaskDefinition(stack awscdk.Stack, ecsConfig *config.ECSConfig, props *ApplicationStackProps) awsecs.FargateTaskDefinition {
	// IAM Execution Role作成
	executionRole := awsiam.NewRole(stack, jsii.String("ECSExecutionRole"), &awsiam.RoleProps{
		AssumedBy: awsiam.NewServicePrincipal(jsii.String("ecs-tasks.amazonaws.com"), nil),
		ManagedPolicies: &[]awsiam.IManagedPolicy{
			awsiam.ManagedPolicy_FromAwsManagedPolicyName(jsii.String("service-role/AmazonECSTaskExecutionRolePolicy")),
		},
		Description: jsii.String("ECS Task Execution Role for " + props.Environment),
	})

	// IAM Task Role作成
	taskRole := awsiam.NewRole(stack, jsii.String("ECSTaskRole"), &awsiam.RoleProps{
		AssumedBy:   awsiam.NewServicePrincipal(jsii.String("ecs-tasks.amazonaws.com"), nil),
		Description: jsii.String("ECS Task Role for " + props.Environment),
	})

	// Task Role に必要な権限を追加
	taskRole.AddToPolicy(awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
		Effect: awsiam.Effect_ALLOW,
		Actions: &[]*string{
			jsii.String("secretsmanager:GetSecretValue"),
			jsii.String("ssm:GetParameter"),
			jsii.String("ssm:GetParameters"),
		},
		Resources: &[]*string{jsii.String("*")},
	}))

	return awsecs.NewFargateTaskDefinition(stack, jsii.String("ServiceTaskDefinition"), &awsecs.FargateTaskDefinitionProps{
		Family:         jsii.String("service-" + props.Environment + "-task"),
		Cpu:            jsii.Number(ecsConfig.CPU),
		MemoryLimitMiB: jsii.Number(ecsConfig.Memory),
		ExecutionRole:  executionRole,
		TaskRole:       taskRole,
	})
}

// createSecretsConfiguration シークレット設定を作成（修正版）
func createSecretsConfiguration(stack awscdk.Stack, props *ApplicationStackProps) map[string]awsecs.Secret {
	secrets := make(map[string]awsecs.Secret)

	if !props.TestEnvFlag {
		// 実環境用のSecrets Manager設定
		dbSecret := awssecretsmanager.NewSecret(stack, jsii.String("DatabaseSecret"), &awssecretsmanager.SecretProps{
			SecretName:  jsii.String("service-" + props.Environment + "-db-credentials"),
			Description: jsii.String("Database credentials for " + props.Environment),
			GenerateSecretString: &awssecretsmanager.SecretStringGenerator{
				SecretStringTemplate: jsii.String(`{"username":"admin"}`),
				GenerateStringKey:    jsii.String("password"),
				ExcludeCharacters:    jsii.String(`"@/\`),
			},
		})

		secrets["DB_PASSWORD"] = awsecs.Secret_FromSecretsManager(dbSecret, jsii.String("password"))
		secrets["DB_USERNAME"] = awsecs.Secret_FromSecretsManager(dbSecret, jsii.String("username"))
	}

	return secrets
}

// createContainerDefinitions Container Definitionsを作成
func createContainerDefinitions(
	stack awscdk.Stack,
	taskDefinition awsecs.FargateTaskDefinition,
	ecsConfig *config.ECSConfig,
	ecrRepository awsecr.Repository,
	props *ApplicationStackProps,
) {
	// CloudWatch Log Group作成
	logGroup := awslogs.NewLogGroup(stack, jsii.String("ServiceLogGroup"), &awslogs.LogGroupProps{
		LogGroupName: jsii.String("/ecs/service-" + props.Environment),
		Retention: func() awslogs.RetentionDays {
			switch props.Environment {
			case "prod":
				return awslogs.RetentionDays_ONE_MONTH
			case "staging":
				return awslogs.RetentionDays_ONE_WEEK
			default:
				return awslogs.RetentionDays_THREE_DAYS
			}
		}(),
		RemovalPolicy: awscdk.RemovalPolicy_DESTROY,
	})

	// 環境変数設定
	environment := createEnvironmentVariables(props)

	// Secrets設定（機密情報用）
	secrets := createSecretsConfiguration(stack, props)

	// Nginxコンテナ（サイドカー）
	nginxContainer := taskDefinition.AddContainer(jsii.String("nginx-web"), &awsecs.ContainerDefinitionOptions{
		ContainerName:        jsii.String("nginx-web"),
		Image:                awsecs.ContainerImage_FromRegistry(jsii.String("nginx:1.24-alpine"), nil),
		MemoryReservationMiB: jsii.Number(ecsConfig.Memory * 40 / 100), // 40%をNginxに割り当て
		Essential:            jsii.Bool(true),
		Logging: awsecs.LogDrivers_AwsLogs(&awsecs.AwsLogDriverProps{
			LogGroup:     logGroup,
			StreamPrefix: jsii.String("nginx"),
		}),
		PortMappings: &[]*awsecs.PortMapping{
			{
				ContainerPort: jsii.Number(80),
				Protocol:      awsecs.Protocol_TCP,
			},
		},
	})

	// PHPアプリケーションコンテナ
	phpContainer := taskDefinition.AddContainer(jsii.String("php-app"), &awsecs.ContainerDefinitionOptions{
		ContainerName:        jsii.String("php-app"),
		Image:                awsecs.ContainerImage_FromEcrRepository(ecrRepository, jsii.String("latest")),
		Environment:          &environment,
		Secrets:              &secrets,
		MemoryReservationMiB: jsii.Number(ecsConfig.Memory * 60 / 100), // 60%をPHPに割り当て
		Essential:            jsii.Bool(true),
		Logging: awsecs.LogDrivers_AwsLogs(&awsecs.AwsLogDriverProps{
			LogGroup:     logGroup,
			StreamPrefix: jsii.String("php"),
		}),
		// ヘルスチェック設定
		HealthCheck: &awsecs.HealthCheck{
			Command: &[]*string{
				jsii.String("CMD-SHELL"),
				jsii.String("php -v || exit 1"), // 簡単なPHPヘルスチェック
			},
			Interval:    awscdk.Duration_Seconds(jsii.Number(30)),
			Timeout:     awscdk.Duration_Seconds(jsii.Number(5)),
			Retries:     jsii.Number(3),
			StartPeriod: awscdk.Duration_Seconds(jsii.Number(60)),
		},
	})

	// コンテナ間の依存関係設定
	nginxContainer.AddContainerDependencies(&awsecs.ContainerDependency{
		Container: phpContainer,
		Condition: awsecs.ContainerDependencyCondition_HEALTHY,
	})
}

// createEnvironmentVariables 環境変数設定を作成
func createEnvironmentVariables(props *ApplicationStackProps) map[string]*string {
	environment := make(map[string]*string)

	// 基本環境変数
	environment["APP_ENV"] = func() *string {
		switch props.Environment {
		case "dev":
			return jsii.String("development")
		case "prod":
			return jsii.String("production")
		default:
			return jsii.String(props.Environment)
		}
	}()

	environment["DB_CONNECTION"] = jsii.String("mysql")
	environment["CACHE_DRIVER"] = jsii.String("redis")
	environment["AWS_DEFAULT_REGION"] = jsii.String("ap-northeast-1")

	// データベース・キャッシュエンドポイント（非機密情報）
	if !props.TestEnvFlag {
		// 実環境ではCross-stack参照
		environment["DB_HOST"] = jsii.String(props.DatabaseEndpoint)
		environment["REDIS_HOST"] = jsii.String(props.RedisEndpoint)
	} else {
		// テスト環境では固定値
		environment["DB_HOST"] = jsii.String("mock-aurora-endpoint.cluster-xyz.rds.amazonaws.com")
		environment["REDIS_HOST"] = jsii.String("mock-redis-endpoint.cache.amazonaws.com")
	}

	return environment
}

// createECSServiceWithALB ECS ServiceとALBを統合して作成（推奨版）
func createECSServiceWithALB(
	stack awscdk.Stack,
	cluster awsecs.Cluster,
	taskDefinition awsecs.FargateTaskDefinition,
	alb awselasticloadbalancingv2.ApplicationLoadBalancer,
	ecsConfig *config.ECSConfig,
	vpc awsec2.IVpc,
	environment string,
) (awsecs.FargateService, awselasticloadbalancingv2.ApplicationTargetGroup) {

	// 1. 最初にECS Serviceを作成
	service := awsecs.NewFargateService(stack, jsii.String("ServiceFargateService"), &awsecs.FargateServiceProps{
		Cluster:        cluster,
		TaskDefinition: taskDefinition,
		ServiceName:    jsii.String("service-" + environment + "-fargate-service"),
		DesiredCount:   jsii.Number(ecsConfig.DesiredCount),

		// ネットワーク設定
		VpcSubnets: &awsec2.SubnetSelection{
			SubnetType: awsec2.SubnetType_PRIVATE_WITH_EGRESS,
		},
		AssignPublicIp: jsii.Bool(false),

		// セキュリティグループ設定
		SecurityGroups: &[]awsec2.ISecurityGroup{
			getECSSecurityGroup(stack, environment),
		},

		// デプロイ設定
		MaxHealthyPercent: jsii.Number(200),
		MinHealthyPercent: jsii.Number(50),

		// ヘルスチェック猶予期間（ALB統合時は必須）
		HealthCheckGracePeriod: awscdk.Duration_Seconds(jsii.Number(300)),

		// プラットフォームバージョン
		PlatformVersion: awsecs.FargatePlatformVersion_LATEST,

		// Fargate Capacity Provider設定
		CapacityProviderStrategies: createCapacityProviderStrategies(ecsConfig),

		// 運用設定
		EnableExecuteCommand: jsii.Bool(environment != "prod"), // 本番環境以外でECS Exec有効
	})

	// 2. Target Groupを作成（ECS Service用に最適化）
	targetGroup := awselasticloadbalancingv2.NewApplicationTargetGroup(stack, jsii.String("ServiceTargetGroup"), &awselasticloadbalancingv2.ApplicationTargetGroupProps{
		Vpc:             vpc,
		Port:            jsii.Number(80),
		Protocol:        awselasticloadbalancingv2.ApplicationProtocol_HTTP,
		TargetType:      awselasticloadbalancingv2.TargetType_IP, // Fargate必須
		TargetGroupName: jsii.String("service-" + environment + "-tg"),

		// ヘルスチェック設定（重要）
		HealthCheck: &awselasticloadbalancingv2.HealthCheck{
			Path:     jsii.String("/health"),
			Port:     jsii.String("80"),
			Protocol: awselasticloadbalancingv2.Protocol_HTTP,

			// タイムアウト設定
			Interval:                awscdk.Duration_Seconds(jsii.Number(30)),
			Timeout:                 awscdk.Duration_Seconds(jsii.Number(5)),
			HealthyThresholdCount:   jsii.Number(2),
			UnhealthyThresholdCount: jsii.Number(3),

			// ヘルスチェックマッチャー
			HealthyHttpCodes: jsii.String("200,301,302"),
		},
	})

	// 3. ALBにListenerを追加（Target Group統合）
	alb.AddListener(jsii.String("HTTPListener"), &awselasticloadbalancingv2.BaseApplicationListenerProps{
		Port:     jsii.Number(80),
		Protocol: awselasticloadbalancingv2.ApplicationProtocol_HTTP,
		DefaultTargetGroups: &[]awselasticloadbalancingv2.IApplicationTargetGroup{
			targetGroup,
		},
	})

	// 4. ECS ServiceをTarget Groupに関連付け
	service.AttachToApplicationTargetGroup(targetGroup)

	return service, targetGroup
}

// createCapacityProviderStrategies Capacity Provider戦略を作成
func createCapacityProviderStrategies(ecsConfig *config.ECSConfig) *[]*awsecs.CapacityProviderStrategy {
	if ecsConfig.EnableFargateSpot {
		// Fargate Spot優先（開発・ステージング環境）
		return &[]*awsecs.CapacityProviderStrategy{
			{
				CapacityProvider: jsii.String("FARGATE"),
				Weight:           jsii.Number(1),
				Base:             jsii.Number(1), // 最低1つは通常のFargateを確保
			},
			{
				CapacityProvider: jsii.String("FARGATE_SPOT"),
				Weight:           jsii.Number(4), // 残りの80%はSpot
			},
		}
	} else {
		// 通常のFargateのみ（本番環境）
		return &[]*awsecs.CapacityProviderStrategy{
			{
				CapacityProvider: jsii.String("FARGATE"),
				Weight:           jsii.Number(1),
			},
		}
	}
}

// getECSSecurityGroup ECS用セキュリティグループを取得（Cross-stack参照）
func getECSSecurityGroup(stack awscdk.Stack, environment string) awsec2.ISecurityGroup {
	envName := environment
	if envName == "prod" {
		envName = "production"
	}

	// NetworkStackからセキュリティグループIDをインポート
	sgId := awscdk.Fn_ImportValue(jsii.String("Service-" + envName + "-ECS-SG-Id"))

	return awsec2.SecurityGroup_FromSecurityGroupId(
		stack,
		jsii.String("ImportedECSSecurityGroup"),
		sgId,
		nil,
	)
}

// createServiceDiscovery Service Discoveryを作成（本番環境用）
func createServiceDiscovery(
	stack awscdk.Stack,
	cluster awsecs.Cluster,
	ecsService awsecs.FargateService,
	environment string,
) awsservicediscovery.Service {

	// Cloud Map Namespaceを作成
	namespace := awsservicediscovery.NewPrivateDnsNamespace(stack, jsii.String("ServiceNamespace"), &awsservicediscovery.PrivateDnsNamespaceProps{
		Name:        jsii.String("service.local"),
		Vpc:         cluster.Vpc(),
		Description: jsii.String("Service discovery namespace for " + environment),
	})

	// Service Discoveryサービスを作成
	discoveryService := namespace.CreateService(jsii.String("ServiceDiscovery"), &awsservicediscovery.DnsServiceProps{
		Name:          jsii.String("api"),
		Description:   jsii.String("Service discovery for API service"),
		DnsRecordType: awsservicediscovery.DnsRecordType_A,
		DnsTtl:        awscdk.Duration_Seconds(jsii.Number(60)),
		CustomHealthCheck: &awsservicediscovery.HealthCheckCustomConfig{
			FailureThreshold: jsii.Number(3),
		},
	})

	// ECS ServiceにService Discoveryを関連付け
	ecsService.AssociateCloudMapService(&awsecs.AssociateCloudMapServiceOptions{
		Service: discoveryService,
	})

	return discoveryService
}

// setupAutoScaling Auto Scaling設定
func setupAutoScaling(
	ecsService awsecs.FargateService,
	targetGroup awselasticloadbalancingv2.ApplicationTargetGroup,
	ecsConfig *config.ECSConfig,
	environment string,
) {
	// タスク数の AutoScaling 対象を作成
	scalingTarget := ecsService.AutoScaleTaskCount(&awsapplicationautoscaling.EnableScalingProps{
		MinCapacity: jsii.Number(ecsConfig.MinCapacity),
		MaxCapacity: jsii.Number(ecsConfig.MaxCapacity),
	})

	// CPU ベース
	scalingTarget.ScaleOnCpuUtilization(jsii.String("CpuScaling"), &awsecs.CpuUtilizationScalingProps{
		TargetUtilizationPercent: func() *float64 {
			if environment == "prod" {
				return jsii.Number(70)
			}
			return jsii.Number(80)
		}(),
		ScaleInCooldown:  awscdk.Duration_Seconds(jsii.Number(300)),
		ScaleOutCooldown: awscdk.Duration_Seconds(jsii.Number(300)),
	})

	// メモリ ベース
	scalingTarget.ScaleOnMemoryUtilization(jsii.String("MemoryScaling"), &awsecs.MemoryUtilizationScalingProps{
		TargetUtilizationPercent: func() *float64 {
			if environment == "prod" {
				return jsii.Number(80)
			}
			return jsii.Number(90)
		}(),
		ScaleInCooldown:  awscdk.Duration_Seconds(jsii.Number(300)),
		ScaleOutCooldown: awscdk.Duration_Seconds(jsii.Number(300)),
	})

	// 本番のみ：ALB RequestCount ベースの Step Scaling
	if environment == "prod" {
		metric := targetGroup.MetricRequestCount(&awscloudwatch.MetricOptions{
			Period:    awscdk.Duration_Minutes(jsii.Number(1)),
			Statistic: jsii.String("Sum"),
			// Namespace/Dimensions は TG がよしなに設定
		})

		scalingTarget.ScaleOnMetric(jsii.String("RequestCountScaling"),
			&awsapplicationautoscaling.BasicStepScalingPolicyProps{
				Metric: metric,
				ScalingSteps: &[]*awsapplicationautoscaling.ScalingInterval{
					{Upper: jsii.Number(100), Change: jsii.Number(-1)}, // リクエスト少→-1
					{Lower: jsii.Number(200), Change: jsii.Number(+1)}, // 200超→+1
					{Lower: jsii.Number(400), Change: jsii.Number(+2)}, // 400超→+2
				},
				AdjustmentType:        awsapplicationautoscaling.AdjustmentType_CHANGE_IN_CAPACITY,
				Cooldown:              awscdk.Duration_Minutes(jsii.Number(2)),
				MetricAggregationType: awsapplicationautoscaling.MetricAggregationType_AVERAGE,
			},
		)
	}
}

// addApplicationStackTags ApplicationStack全体にタグを追加
func addApplicationStackTags(stack awscdk.Stack, envConfig *config.EnvironmentConfig) {
	for key, value := range envConfig.Tags {
		awscdk.Tags_Of(stack).Add(jsii.String(key), jsii.String(value), nil)
	}
	awscdk.Tags_Of(stack).Add(jsii.String("StackType"), jsii.String("Application"), nil)
	awscdk.Tags_Of(stack).Add(jsii.String("ManagedBy"), jsii.String("CDK"), nil)
}

// createApplicationStackOutputs Cross-stack出力を作成
func createApplicationStackOutputs(
	stack awscdk.Stack,
	ecrRepository awsecr.Repository,
	alb awselasticloadbalancingv2.ApplicationLoadBalancer,
	targetGroup awselasticloadbalancingv2.ApplicationTargetGroup,
	environment string,
) {
	// ECRリポジトリURI出力
	awscdk.NewCfnOutput(stack, jsii.String("ECRRepositoryURI"), &awscdk.CfnOutputProps{
		Value:       ecrRepository.RepositoryUri(),
		Description: jsii.String("ECR Repository URI for container images"),
		ExportName:  jsii.String("Service-" + environment + "-ECR-URI"),
	})

	// ALB DNS名出力
	awscdk.NewCfnOutput(stack, jsii.String("LoadBalancerDNS"), &awscdk.CfnOutputProps{
		Value:       alb.LoadBalancerDnsName(),
		Description: jsii.String("Application Load Balancer DNS name"),
		ExportName:  jsii.String("Service-" + environment + "-ALB-DNS"),
	})

	// ALB ARN出力
	awscdk.NewCfnOutput(stack, jsii.String("LoadBalancerArn"), &awscdk.CfnOutputProps{
		Value:       alb.LoadBalancerArn(),
		Description: jsii.String("Application Load Balancer ARN"),
		ExportName:  jsii.String("Service-" + environment + "-ALB-ARN"),
	})

	// Target Group ARN出力
	awscdk.NewCfnOutput(stack, jsii.String("TargetGroupArn"), &awscdk.CfnOutputProps{
		Value:       targetGroup.TargetGroupArn(),
		Description: jsii.String("Target Group ARN for ECS service"),
		ExportName:  jsii.String("Service-" + environment + "-TG-ARN"),
	})

	// 🆕 ECS関連の出力追加
	awscdk.NewCfnOutput(stack, jsii.String("ECSClusterName"), &awscdk.CfnOutputProps{
		Value:       jsii.String("service-" + environment + "-cluster"),
		Description: jsii.String("ECS Cluster name"),
		ExportName:  jsii.String("Service-" + environment + "-Cluster-Name"),
	})

	awscdk.NewCfnOutput(stack, jsii.String("ECSServiceName"), &awscdk.CfnOutputProps{
		Value:       jsii.String("service-" + environment + "-fargate-service"),
		Description: jsii.String("ECS Service name"),
		ExportName:  jsii.String("Service-" + environment + "-Service-Name"),
	})

	// Application URL出力
	awscdk.NewCfnOutput(stack, jsii.String("ApplicationURL"), &awscdk.CfnOutputProps{
		Value: awscdk.Fn_Join(jsii.String(""), &[]*string{
			jsii.String("http://"),
			alb.LoadBalancerDnsName(),
		}),
		Description: jsii.String("Application URL"),
		ExportName:  jsii.String("Service-" + environment + "-App-URL"),
	})
}
