.PHONY: tidy fmt lint test build zip aws-configure source

SHELL := /bin/bash

PROJECT_NAME="nyc-asp-lambda"

TOPIC_ARN=$(shell aws sns list-topics| jq -r ".Topics[0].TopicArn")

tidy:
	go mod tidy

fmt:
	gofmt -w -s -d .

lint:
	golangci-lint run ./...

test:
	go test -cover -race ./...

build:
	@rm -f cmd/${PROJECT_NAME}
	@GOARCH=amd64 GOOS=linux go build -o cmd/${PROJECT_NAME} cmd/${PROJECT_NAME}.go

zip:
	@rm -f cmd/${PROJECT_NAME}.zip
	@zip -jr cmd/${PROJECT_NAME}.zip cmd/${PROJECT_NAME}

#==========================================================
# AWS Setup

# AWS S3 https://docs.aws.amazon.com/cli/latest/userguide/cli-services-s3-commands.html#using-s3-commands-managing-buckets-creating
aws-configure:
	aws configure

aws-setup: iam sns lambda

iam: aws-create-role aws-put-policy

sns: aws-create-sns aws-subscribe-sns

lambda: aws-create-lambda

# AWS Create Role 
# https://awscli.amazonaws.com/v2/documentation/api/latest/reference/iam/create-role.html
aws-create-role:
	@aws iam create-role \
	--role-name ${PROJECT_NAME} \
	--no-cli-pager \
	--assume-role-policy-document file://config/iam/trust-relationship.json

# AWS Put Policy in role
# https://awscli.amazonaws.com/v2/documentation/api/latest/reference/iam/put-role-policy.html
aws-put-policy:
	@aws iam put-role-policy \
	--role-name ${PROJECT_NAME} \
	--policy-name lambda-policy \
	--no-cli-pager \
	--policy-document file://config/iam/lambda-executor.json

# AWS Get Role
aws-get-role:
	aws iam get-role \
	--no-cli-pager \
    --role-name ${PROJECT_NAME} | jq -r ".Role.Arn"

# AWS Create SNS Topic
# https://awscli.amazonaws.com/v2/documentation/api/latest/reference/sns/create-topic.html
aws-create-sns:
	@aws sns create-topic \
	--no-cli-pager \
	--name ${PROJECT_NAME}

aws-topic-arn:
	export TOPIC_ARN=$(shell aws sns list-topics| jq -r ".Topics[0].TopicArn")

# AWS Subscribe SNS
aws-subscribe-sns:
	@aws sns subscribe \
    --topic-arn ${TOPIC_ARN} \
    --protocol email \
	--return-subscription-arn \
	--no-cli-pager \
    --notification-endpoint adiazny@gmail.com

# AWS Lambda 
# https://awscli.amazonaws.com/v2/documentation/api/latest/reference/lambda/create-function.html
aws-create-lambda:
	@aws lambda create-function \
	--function-name ${PROJECT_NAME} \
	--runtime go1.x \
	--role $(shell aws iam get-role --role-name nyc-asp-lambda | jq -r ".Role.Arn") \
	--handler ${PROJECT_NAME} \
	--environment "Variables={ \
		OCP_APIM_SUBSCRIPTION_KEY=${OCP_APIM_SUBSCRIPTION_KEY}, \
		BASE_API_HOST=${BASE_API_HOST}, \
		TOPIC_ARN=$(shell aws sns list-topics| jq -r ".Topics[0].TopicArn")}" \
	--no-cli-pager \
	--zip-file fileb://cmd/${PROJECT_NAME}.zip


# TODO AWS Event Bridge Lambda Trigger
# Expression: 0 7 * * ? *
# https://docs.aws.amazon.com/cli/latest/reference/events/index.html
