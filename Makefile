.PHONY: tidy build zip aws-configure

SHELL := /bin/bash

tidy:
	go mod tidy

build:
	GOOS="linux" go build cmd/main.go

zip:
	zip -jrm nyc-asp-lambda.zip main

# AWS S3 https://docs.aws.amazon.com/cli/latest/userguide/cli-services-s3-commands.html#using-s3-commands-managing-buckets-creating

aws-configure:
	aws configure

s3-create:
	aws s3 mb s3://nyc-asp-lambda.io

s3-list:
	aws s3 ls s3://nyc-asp-lambda.io

s3-copy:
	aws s3 cp nyc-asp-lambda.zip s3://nyc-asp-lambda.io

