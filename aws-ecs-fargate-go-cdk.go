// Package main provides AWS ECS Fargate infrastructure using CDK
//
// DISCLAIMER: This code is for educational purposes only.
// Not suitable for production use without proper review and customization.
//
// Copyright (c) 2025 kyotalab
// Licensed under MIT License

package main

import (
	"aws-ecs-fargate-go-cdk/internal/stacks"
	"fmt"
	"os"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/jsii-runtime-go"
)

func main() {
	defer jsii.Close()

	app := awscdk.NewApp(nil)

	// コンテキストから環境を取得
	environment := "dev" // デフォルト値
	if envContext := app.Node().TryGetContext(jsii.String("environment")); envContext != nil {
		if envStr, ok := envContext.(string); ok {
			environment = envStr
		}
	}

	fmt.Printf("🚀 Building infrastructure for environment: %s\n", environment)

	// 1. NetworkStackを作成
	networkStack := stacks.NewNetworkStack(app, "NetworkStack", &stacks.NetworkStackProps{
		StackProps: awscdk.StackProps{
			Env: env(),
		},
		Environment: environment,
		// VpcCidrは環境設定から自動取得される
	})

	// 2. StorageStackを作成（NetworkStackに依存）
	storageStack := stacks.NewStorageStack(app, "StorageStack", &stacks.StorageStackProps{
		StackProps: awscdk.StackProps{
			Env: env(),
		},
		Environment: environment,
		VpcId:       "vpc-from-network-stack", // Cross-stack参照で自動解決
		TestEnvFlag: false,                    // 実際のデプロイ環境
	})

	// 3. 将来のApplicationStackをここに追加
	// applicationStack := stacks.NewApplicationStack(app, "ApplicationStack", &stacks.ApplicationStackProps{
	// 	StackProps: awscdk.StackProps{
	// 		Env: env(),
	// 	},
	// 	Environment: environment,
	// 	VpcId: networkStack.VpcId(),
	// 	DatabaseEndpoint: storageStack.DatabaseEndpoint(),
	// })

	// Stack間の依存関係を設定
	storageStack.AddDependency(networkStack, nil)
	// applicationStack.AddDependency(storageStack, nil)

	fmt.Printf("✅ NetworkStack created for environment: %s\n", environment)
	fmt.Printf("✅ StorageStack created for environment: %s\n", environment)

	app.Synth(nil)
}

// getEnvironmentConfig 環境設定を動的に取得（ハードコーディングなし版）
func getEnvironmentConfig() *awscdk.Environment {
	// 環境変数から取得
	account := os.Getenv("CDK_DEFAULT_ACCOUNT")
	region := os.Getenv("CDK_DEFAULT_REGION")

	// 環境変数が設定されていない場合のエラーハンドリング
	if account == "" {
		fmt.Fprintf(os.Stderr, "❌ Error: CDK_DEFAULT_ACCOUNT environment variable is not set\n")
		fmt.Fprintf(os.Stderr, "💡 Please set: export CDK_DEFAULT_ACCOUNT=your-account-id\n")
		os.Exit(1)
	}

	if region == "" {
		fmt.Fprintf(os.Stderr, "❌ Error: CDK_DEFAULT_REGION environment variable is not set\n")
		fmt.Fprintf(os.Stderr, "💡 Please set: export CDK_DEFAULT_REGION=ap-northeast-1\n")
		os.Exit(1)
	}

	fmt.Printf("🔧 Using AWS Account: %s, Region: %s\n", account, region)

	return &awscdk.Environment{
		Account: jsii.String(account),
		Region:  jsii.String(region),
	}
}

// env determines the AWS environment (account+region) in which our stack is to
// be deployed. For more information see: https://docs.aws.amazon.com/cdk/latest/guide/environments.html
func env() *awscdk.Environment {
	// VPC Lookup使用時は、明示的なアカウント・リージョン設定が必要
	return getEnvironmentConfig()
}
