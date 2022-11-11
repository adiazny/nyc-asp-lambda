package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	cfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/caarlos0/env"
	"github.com/sirupsen/logrus"
	"go.uber.org/automaxprocs/maxprocs"

	"github.com/adiazny/nyc-asp-lambda/internal/pkg/asp"
	calendar "github.com/adiazny/nyc-asp-lambda/internal/pkg/calendar"
)

const timeout = 10

type environmentVariables struct {
	OCPApimSubscriptionKey string `env:"OCP_APIM_SUBSCRIPTION_KEY,required"`
	BaseAPIHost            string `env:"BASE_API_HOST,required"`
	TopicARN               string `env:"TOPIC_ARN,required"`
}

func setup() (envVars *environmentVariables, err error) {
	_, err = maxprocs.Set()
	if err != nil {
		return nil, fmt.Errorf("error setting GOMAXPROCS %w", err)
	}

	envVars = &environmentVariables{}

	err = env.Parse(envVars)
	if err != nil {
		return nil, fmt.Errorf("error parsing environmenet varilables %w", err)
	}

	return envVars, nil
}

func HandleRequest(ctx context.Context) ([]calendar.Item, error) {
	logger := logrus.New()
	logger.SetOutput(os.Stdout)
	logger.SetFormatter(&logrus.JSONFormatter{})

	log := logrus.NewEntry(logger)
	log.WithField("component", "nyc-asp").Info("starting up")

	defer log.WithField("component", "nyc-asp").Info("shutting down")

	envVars, err := setup()
	if err != nil {
		log.WithError(err).Error()
		os.Exit(1)
	}

	awsConfig, err := cfg.LoadDefaultConfig(ctx)
	if err != nil {
		log.WithError(err).Error()
		os.Exit(1)
	}

	aspClient := &asp.Client{
		Log: log,
		Config: asp.Config{
			APIKey:      envVars.OCPApimSubscriptionKey,
			BaseAPIHost: envVars.BaseAPIHost,
			SNSTopicARN: envVars.TopicARN,
		},
		HTTP: &http.Client{
			Timeout: time.Duration(time.Second * timeout),
		},
		SNS: sns.NewFromConfig(awsConfig),
	}

	aspItems, err := aspClient.GetASPItems()
	if err != nil {
		return nil, err
	}

	err = aspClient.PublishSNS(ctx, aspItems)
	if err != nil {
		return nil, err
	}

	return aspItems, nil
}

func main() {
	lambda.Start(HandleRequest)
}
