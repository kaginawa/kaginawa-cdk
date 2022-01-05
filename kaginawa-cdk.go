package main

import (
	"log"
	"os"
	"strconv"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awscertificatemanager"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsdynamodb"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsec2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsecs"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsecspatterns"
	"github.com/aws/aws-cdk-go/awscdk/v2/awselasticloadbalancingv2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsiam"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
	"github.com/joho/godotenv"
)

type KaginawaCdkStackProps struct {
	awscdk.StackProps
}

func NewKaginawaCdkStack(scope constructs.Construct, id string, props *KaginawaCdkStackProps) awscdk.Stack {
	var sprops awscdk.StackProps
	if props != nil {
		sprops = props.StackProps
	}
	stack := awscdk.NewStack(scope, &id, &sprops)

	// ACM
	cert := awscertificatemanager.NewCertificate(stack, jsii.String("KaginawaCertificate"), &awscertificatemanager.CertificateProps{
		DomainName: jsii.String(os.Getenv("DOMAIN_NAME")),
		Validation: awscertificatemanager.CertificateValidation_FromDns(nil),
	})

	// VPC
	vpc := awsec2.NewVpc(stack, jsii.String("KaginawaVPC"), &awsec2.VpcProps{
		MaxAzs:      jsii.Number(2),
		NatGateways: jsii.Number(0),
		SubnetConfiguration: &[]*awsec2.SubnetConfiguration{{
			Name:       jsii.String("KaginawaVPCSubnet"),
			CidrMask:   jsii.Number(24),
			SubnetType: awsec2.SubnetType_PUBLIC,
		}},
	})

	// EC2 Instances
	nServers := 1
	if n, err := strconv.Atoi(os.Getenv("NUM_OF_SSH_SERVERS")); err == nil && n > 0 {
		nServers = n
	}
	for idx := 1; idx <= nServers; idx++ {
		i := strconv.Itoa(idx)
		awsec2.NewCfnEIPAssociation(stack, jsii.String("KaginawaEIPAssoc"+i), &awsec2.CfnEIPAssociationProps{
			Eip: awsec2.NewCfnEIP(stack, jsii.String("KaginawaEIP"+i), nil).Ref(),
			InstanceId: awsec2.NewInstance(stack, jsii.String("KaginawaSSHInstance"+i), &awsec2.InstanceProps{
				InstanceName: jsii.String("kssh" + i),
				// t4g.micro
				InstanceType: awsec2.InstanceType_Of(awsec2.InstanceClass_BURSTABLE4_GRAVITON, awsec2.InstanceSize_MICRO),
				// Debian 10, SSD Volume Type, arm64
				MachineImage: awsec2.MachineImage_GenericLinux(&map[string]*string{
					"ap-northeast-1": jsii.String("ami-0ed400c2ea06a311c"),
				}, nil),
				Role: awsiam.NewRole(stack, jsii.String("KaginawaSSHInstanceSSM"), &awsiam.RoleProps{
					AssumedBy: awsiam.NewServicePrincipal(jsii.String("ec2.amazonaws.com"), nil),
					ManagedPolicies: &[]awsiam.IManagedPolicy{
						awsiam.ManagedPolicy_FromAwsManagedPolicyName(jsii.String("AmazonSSMManagedInstanceCore")),
					},
				}),
				Vpc: vpc,
			}).InstanceId(),
		})
	}

	// DynamoDB auto-scaling properties
	tableScaling := awsdynamodb.EnableScalingProps{MinCapacity: jsii.Number(1), MaxCapacity: jsii.Number(100)}
	tableScalingTarget := awsdynamodb.UtilizationScalingProps{TargetUtilizationPercent: jsii.Number(80)}

	// DynamoDB KaginawaKeys table
	keysTable := awsdynamodb.NewTable(stack, jsii.String("KaginawaKeys"), &awsdynamodb.TableProps{
		TableName:           jsii.String("KaginawaKeys"),
		PartitionKey:        &awsdynamodb.Attribute{Name: jsii.String("Key"), Type: awsdynamodb.AttributeType_STRING},
		ReadCapacity:        jsii.Number(1),
		WriteCapacity:       jsii.Number(1),
		RemovalPolicy:       awscdk.RemovalPolicy_DESTROY,
		PointInTimeRecovery: jsii.Bool(true),
	})
	keysTable.AutoScaleReadCapacity(&tableScaling).ScaleOnUtilization(&tableScalingTarget)
	keysTable.AutoScaleWriteCapacity(&tableScaling).ScaleOnUtilization(&tableScalingTarget)

	// DynamoDB KaginawaServers table
	serversTable := awsdynamodb.NewTable(stack, jsii.String("KaginawaServers"), &awsdynamodb.TableProps{
		TableName:           jsii.String("KaginawaServers"),
		PartitionKey:        &awsdynamodb.Attribute{Name: jsii.String("Host"), Type: awsdynamodb.AttributeType_STRING},
		ReadCapacity:        jsii.Number(1),
		WriteCapacity:       jsii.Number(1),
		RemovalPolicy:       awscdk.RemovalPolicy_DESTROY,
		PointInTimeRecovery: jsii.Bool(true),
	})
	serversTable.AutoScaleReadCapacity(&tableScaling).ScaleOnUtilization(&tableScalingTarget)
	serversTable.AutoScaleWriteCapacity(&tableScaling).ScaleOnUtilization(&tableScalingTarget)

	// DynamoDB KaginawaNodes table
	nodesTable := awsdynamodb.NewTable(stack, jsii.String("KaginawaNodes"), &awsdynamodb.TableProps{
		TableName:     jsii.String("KaginawaNodes"),
		PartitionKey:  &awsdynamodb.Attribute{Name: jsii.String("ID"), Type: awsdynamodb.AttributeType_STRING},
		ReadCapacity:  jsii.Number(1),
		WriteCapacity: jsii.Number(1),
		RemovalPolicy: awscdk.RemovalPolicy_DESTROY,
	})
	nodesTable.AutoScaleReadCapacity(&tableScaling).ScaleOnUtilization(&tableScalingTarget)
	nodesTable.AutoScaleWriteCapacity(&tableScaling).ScaleOnUtilization(&tableScalingTarget)
	customIDIndex := awsdynamodb.GlobalSecondaryIndexProps{
		IndexName:      jsii.String("CustomID-index"),
		PartitionKey:   &awsdynamodb.Attribute{Name: jsii.String("CustomID"), Type: awsdynamodb.AttributeType_STRING},
		ProjectionType: awsdynamodb.ProjectionType_ALL,
		ReadCapacity:   jsii.Number(1),
		WriteCapacity:  jsii.Number(1),
	}
	nodesTable.AddGlobalSecondaryIndex(&customIDIndex)
	nodesTable.AutoScaleGlobalSecondaryIndexReadCapacity(customIDIndex.IndexName, &tableScaling).ScaleOnUtilization(&tableScalingTarget)
	nodesTable.AutoScaleGlobalSecondaryIndexWriteCapacity(customIDIndex.IndexName, &tableScaling).ScaleOnUtilization(&tableScalingTarget)

	// DynamoDB kaginawaLogs table
	logsTable := awsdynamodb.NewTable(stack, jsii.String("KaginawaLogs"), &awsdynamodb.TableProps{
		TableName:           jsii.String("KaginawaLogs"),
		PartitionKey:        &awsdynamodb.Attribute{Name: jsii.String("ID"), Type: awsdynamodb.AttributeType_STRING},
		SortKey:             &awsdynamodb.Attribute{Name: jsii.String("ServerTime"), Type: awsdynamodb.AttributeType_NUMBER},
		TimeToLiveAttribute: jsii.String("TTL"),
		ReadCapacity:        jsii.Number(1),
		WriteCapacity:       jsii.Number(1),
		RemovalPolicy:       awscdk.RemovalPolicy_DESTROY,
	})
	logsTable.AutoScaleReadCapacity(&tableScaling).ScaleOnUtilization(&tableScalingTarget)
	logsTable.AutoScaleWriteCapacity(&tableScaling).ScaleOnUtilization(&tableScalingTarget)

	// DynamoDB kaginawaSessions table
	sessionsTable := awsdynamodb.NewTable(stack, jsii.String("KaginawaSessions"), &awsdynamodb.TableProps{
		TableName:           jsii.String("KaginawaSessions"),
		PartitionKey:        &awsdynamodb.Attribute{Name: jsii.String("ID"), Type: awsdynamodb.AttributeType_STRING},
		TimeToLiveAttribute: jsii.String("TTL"),
		ReadCapacity:        jsii.Number(1),
		WriteCapacity:       jsii.Number(1),
		RemovalPolicy:       awscdk.RemovalPolicy_DESTROY,
	})
	sessionsTable.AutoScaleReadCapacity(&tableScaling).ScaleOnUtilization(&tableScalingTarget)
	sessionsTable.AutoScaleWriteCapacity(&tableScaling).ScaleOnUtilization(&tableScalingTarget)

	// ECS cluster
	cluster := awsecs.NewCluster(stack, jsii.String("KaginawaServerStack"), &awsecs.ClusterProps{
		Vpc:         vpc,
		ClusterName: jsii.String("KaginawaServer"),
	})

	// ECS KaginawaServer service
	service := awsecspatterns.NewApplicationLoadBalancedFargateService(stack, jsii.String("KaginawaServer"), &awsecspatterns.ApplicationLoadBalancedFargateServiceProps{
		ServiceName:    jsii.String("KaginawaServer"),
		Cluster:        cluster,
		AssignPublicIp: jsii.Bool(true),
		Cpu:            jsii.Number(256), // 0.25 vCPU
		MemoryLimitMiB: jsii.Number(512), // 0.5 GB
		ListenerPort:   jsii.Number(8080),
		TaskImageOptions: &awsecspatterns.ApplicationLoadBalancedTaskImageOptions{
			Image:         awsecs.ContainerImage_FromRegistry(jsii.String("kaginawa/kaginawa-server"), nil),
			ContainerPort: jsii.Number(8080),
			Environment: &map[string]*string{
				"DYNAMO_KEYS":              keysTable.TableName(),
				"DYNAMO_SERVERS":           serversTable.TableName(),
				"DYNAMO_NODES":             nodesTable.TableName(),
				"DYNAMO_LOGS":              logsTable.TableName(),
				"DYNAMO_SESSIONS":          sessionsTable.TableName(),
				"DYNAMO_CUSTOM_IDS":        customIDIndex.IndexName,
				"OAUTH_TYPE":               jsii.String(os.Getenv("OAUTH_TYPE")),
				"OAUTH_DOMAIN":             jsii.String(os.Getenv("OAUTH_DOMAIN")),
				"OAUTH_CLIENT_ID":          jsii.String(os.Getenv("OAUTH_CLIENT_ID")),
				"OAUTH_CLIENT_SECRET":      jsii.String(os.Getenv("OAUTH_CLIENT_SECRET")),
				"DYNAMO_LOGS_TTL_DAYS":     jsii.String("90"),
				"DYNAMO_SESSIONS_TTL_DAYS": jsii.String("180"),
				"SELF_URL":                 jsii.String("https://" + os.Getenv("DOMAIN_NAME")),
			},
		},
		PublicLoadBalancer: jsii.Bool(true),
	})
	service.TargetGroup().EnableCookieStickiness(awscdk.Duration_Minutes(jsii.Number(3)), jsii.String("kaginawa-alb"))
	service.LoadBalancer().AddListener(jsii.String("KaginawaServerALB443"), &awselasticloadbalancingv2.BaseApplicationListenerProps{
		Port:                jsii.Number(443),
		DefaultTargetGroups: &[]awselasticloadbalancingv2.IApplicationTargetGroup{service.TargetGroup()},
		Certificates:        &[]awselasticloadbalancingv2.IListenerCertificate{cert},
	})
	service.TaskDefinition().AddToTaskRolePolicy(awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
		Resources: &[]*string{
			keysTable.TableArn(),
			serversTable.TableArn(),
			nodesTable.TableArn(),
			logsTable.TableArn(),
			sessionsTable.TableArn(),
		},
		Actions: &[]*string{jsii.String("dynamodb:*")},
		Effect:  awsiam.Effect_ALLOW,
	}))

	return stack
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("please create .env first")
	}

	app := awscdk.NewApp(nil)

	NewKaginawaCdkStack(app, "KaginawaCdkStack", &KaginawaCdkStackProps{
		awscdk.StackProps{
			Env: env(),
		},
	})

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
