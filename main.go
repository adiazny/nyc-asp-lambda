package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	cfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/caarlos0/env"
	"github.com/sirupsen/logrus"
	"go.uber.org/automaxprocs/maxprocs"
)

const (
	getAPICalendarEndpoint = "api/GetCalendar"
)

type environmentVariables struct {
	OCPApimSubscriptionKey string `env:"OCP_APIM_SUBSCRIPTION_KEY,required"`
	BaseAPIHost            string `env:"BASE_API_HOST,required"`
	TopicARN               string `env:"TOPIC_ARN,required"`
}

type config struct {
	apiKey      string
	baseAPIHost string
}

type nycClient struct {
	*logrus.Entry
	config
	http.Client
}

type Response struct {
	Days []Day `json:"days"`
}

type Day struct {
	TodayID string `json:"today_id"`
	Items   []Item `json:"items"`
}

type Item struct {
	Details string `json:"details"`
	Status  string `json:"status"`
	Type    string `json:"type"`
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

func (client *nycClient) getASPItems() ([]Item, error) {
	//fromDate := time.Now().Add(-time.Hour * (24 * 7))
	fromDate := time.Now()
	toDate := time.Now()

	apiEndpoint := fmt.Sprintf("%s/%s?fromDate=%s&toDate=%s",
		client.baseAPIHost,
		getAPICalendarEndpoint,
		fromDate.Format(time.RFC3339),
		toDate.Format(time.RFC3339),
	)

	req, err := http.NewRequest("GET", apiEndpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating http request %w", err)
	}

	req.Header.Add("Cache-Control", "no-cache")
	req.Header.Add("Ocp-Apim-Subscription-Key", client.apiKey)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error performing http request %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error status code is not 200 OK, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body %w", err)
	}

	data := &Response{}

	err = json.Unmarshal(body, data)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling http request body %w", err)
	}

	items := filterItems(data, func(item Item) bool {
		//return item.Type == "Alternate Side Parking" && item.Status == "SUSPENDED"
		return item.Type == "Alternate Side Parking"

	})

	return items, nil
}

func filterItems(res *Response, matchFunc func(item Item) bool) []Item {
	items := make([]Item, 0)

	for _, day := range res.Days {
		for _, item := range day.Items {
			match := matchFunc(item)

			if match {
				items = append(items, item)
			}
		}
	}

	return items
}

func HandleRequest(ctx context.Context) ([]Item, error) {
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

	apiClient := &nycClient{
		log,
		config{
			apiKey:      envVars.OCPApimSubscriptionKey,
			baseAPIHost: envVars.BaseAPIHost,
		},
		*http.DefaultClient,
	}

	aspItems, err := apiClient.getASPItems()
	if err != nil {
		log.WithError(err).Error()
		os.Exit(1)
	}

	// Publish to SNS Topic
	location, _ := time.LoadLocation("America/New_York")
	formattedTime := time.Now().In(location).Format("Monday, Jan 02 2006")

	topicMsg := fmt.Sprintf("Date: %v\nASP: %s\nDetails: %s", formattedTime, aspItems[0].Status, aspItems[0].Details)

	cfg, err := cfg.LoadDefaultConfig(ctx)
	if err != nil {
		log.WithError(err).Error()
		os.Exit(1)
	}

	topicArn := envVars.TopicARN

	client := sns.NewFromConfig(cfg)

	input := &sns.PublishInput{
		Message:  &topicMsg,
		TopicArn: &topicArn,
	}

	_, err = client.Publish(ctx, input)
	if err != nil {
		log.WithError(err).Error()
		os.Exit(1)
	}

	return aspItems, nil
}

func main() {
	lambda.Start(HandleRequest)
}
