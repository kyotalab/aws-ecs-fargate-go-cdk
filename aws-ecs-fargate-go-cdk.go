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
	"github.com/aws/aws-cdk-go/awscdk/v2/awssqs"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

type AwsEcsFargateGoCdkStackProps struct {
	awscdk.StackProps
}

func NewAwsEcsFargateGoCdkStack(scope constructs.Construct, id string, props *AwsEcsFargateGoCdkStackProps) awscdk.Stack {
	var sprops awscdk.StackProps
	if props != nil {
		sprops = props.StackProps
	}
	stack := awscdk.NewStack(scope, &id, &sprops)

	// The code that defines your stack goes here

	// example resource
	queue := awssqs.NewQueue(stack, jsii.String("AwsEcsFargateGoCdkQueue"), &awssqs.QueueProps{
		VisibilityTimeout: awscdk.Duration_Seconds(jsii.Number(300)),
	})

	fmt.Printf("%v\n", queue)

	return stack
}

func main() {
	defer jsii.Close()

	app := awscdk.NewApp(nil)

	// NewAwsEcsFargateGoCdkStack(app, "AwsEcsFargateGoCdkStack", &AwsEcsFargateGoCdkStackProps{
	// 	awscdk.StackProps{
	// 		Env: env(),
	// 	},
	// })

	stacks.NewNetworkStack(app, "NetworkStack", &stacks.NetworkStackProps{})

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
