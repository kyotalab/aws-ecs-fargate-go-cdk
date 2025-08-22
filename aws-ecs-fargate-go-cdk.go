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
		Environment:       environment,
		VpcId:             "vpc-from-network-stack", // Cross-stack参照で自動解決
		IsTestEnvironment: false,                    // 実際のデプロイ環境
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

// env determines the AWS environment (account+region) in which our stack is to
// be deployed. For more information see: https://docs.aws.amazon.com/cdk/latest/guide/environments.html
func env() *awscdk.Environment {
	// If unspecified, this stack will be "environment-agnostic".
	// Account/Region-dependent features and context lookups will not work, but a
	// single synthesized template can be deployed anywhere.
	//---------------------------------------------------------------------------
	return nil

	// Uncomment if you know exactly what account and region you want to deploy
	// the stack to. This is the recommendation for production stacks.
	//---------------------------------------------------------------------------
	// return &awscdk.Environment{
	//  Account: jsii.String("123456789012"),
	//  Region:  jsii.String("us-east-1"),
	// }

	// Uncomment to specialize this stack for the AWS Account and Region that are
	// implied by the current CLI configuration. This is recommended for dev
	// stacks.
	//---------------------------------------------------------------------------
	// return &awscdk.Environment{
	//  Account: jsii.String(os.Getenv("CDK_DEFAULT_ACCOUNT")),
	//  Region:  jsii.String(os.Getenv("CDK_DEFAULT_REGION")),
	// }
}
