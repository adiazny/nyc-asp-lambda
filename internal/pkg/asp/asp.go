package asp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/adiazny/nyc-asp-lambda/internal/pkg/calendar"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/sirupsen/logrus"
)

const (
	getCalendarEndpoint = "api/GetCalendar"

	cacheControlHeaderKey = "Cache-Control"
	noCacheValue          = "no-cache"

	apimSubscriptionKey = "Ocp-Apim-Subscription-Key"
)

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type SNSClient interface {
	Publish(ctx context.Context, params *sns.PublishInput, optFns ...func(*sns.Options)) (*sns.PublishOutput, error)
}

type Config struct {
	APIKey      string
	BaseAPIHost string
	SNSTopicARN string
}

type Client struct {
	Log    *logrus.Entry
	Config Config
	HTTP   HTTPClient
	SNS    SNSClient
}

type LambdaResponse struct {
	Message  string
	ASPItems []calendar.Item
}

// GetASPItems performs HTTP request to obtain calendar items.
func (client *Client) GetASPItems() ([]calendar.Item, error) {
	//fromDate := time.Now()
	fromDate := time.Now().Add(-time.Hour * (24 * 2))
	toDate := time.Now()

	apiEndpoint := fmt.Sprintf("%s/%s?fromDate=%s&toDate=%s",
		client.Config.BaseAPIHost,
		getCalendarEndpoint,
		fromDate.Format(time.RFC3339),
		toDate.Format(time.RFC3339),
	)

	req, err := http.NewRequest(http.MethodGet, apiEndpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating http request %w", err)
	}

	req.Header.Add(cacheControlHeaderKey, noCacheValue)
	req.Header.Add(apimSubscriptionKey, client.Config.APIKey)

	resp, err := client.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error performing http request %w", err)
	}

	if resp.Body != nil {
		defer resp.Body.Close()
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error unexpected response code %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body %w", err)
	}

	calendarResponse := &calendar.Response{}

	err = json.Unmarshal(body, calendarResponse)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling http request body %w", err)
	}

	items := filterItems(calendarResponse, func(item calendar.Item) bool {
		return item.Type == "Alternate Side Parking" && item.Status == "SUSPENDED"
		//return item.Type == "Alternate Side Parking"

	})

	return items, nil
}

// PublishSNS publishes a calendar item to a AWS SNS topic.
func (client *Client) PublishSNS(ctx context.Context, aspItems []calendar.Item) (LambdaResponse, error) {
	if len(aspItems) == 0 {
		client.Log.Infof("no suspended ASP for %v", time.Now().Format(time.RFC3339))

		return LambdaResponse{Message: "No ASP to publish"}, nil
	}

	location, _ := time.LoadLocation("America/New_York")
	formattedTime := time.Now().In(location).Format("Monday, Jan 02 2006")

	topicMsg := fmt.Sprintf("Date: %v\nASP: %s\nDetails: %s", formattedTime, aspItems[0].Status, aspItems[0].Details)

	input := &sns.PublishInput{
		Message:  &topicMsg,
		TopicArn: &client.Config.SNSTopicARN,
	}

	publishOutput, err := client.SNS.Publish(ctx, input)
	if err != nil {
		client.Log.WithError(err).Error()
		return LambdaResponse{}, fmt.Errorf("error pusblishing to AWS SNS topic %s: %w", client.Config.SNSTopicARN, err)
	}

	client.Log.WithField("snsMessageId", publishOutput.MessageId).Info("successfuly published to sns topic")

	return LambdaResponse{Message: "ASP published to SNS", ASPItems: aspItems}, nil
}

func filterItems(res *calendar.Response, matchFunc func(item calendar.Item) bool) []calendar.Item {
	items := make([]calendar.Item, 0)

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
