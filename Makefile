.PHONY: tidy build zip

SHELL := /bin/bash

tidy:
	go mod tidy

build:
	GOOS="linux" go build cmd/main.go

zip:
	zip -jrm nyc-asp-lambda.zip main