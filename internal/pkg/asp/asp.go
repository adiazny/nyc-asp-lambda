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

type Config struct {
	APIKey      string
	BaseAPIHost string
	SNSTopicARN string
}

type Client struct {
	Log    *logrus.Entry
	Config Config
	HTTP   http.Client
	SNS    *sns.Client
}

func (client *Client) GetASPItems() ([]calendar.Item, error) {
	fromDate := time.Now()
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

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error status code is not 200 OK, got %d", resp.StatusCode)
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
		//return item.Type == "Alternate Side Parking" && item.Status == "SUSPENDED"
		return item.Type == "Alternate Side Parking"

	})

	return items, nil
}

func (client *Client) PublishSNS(ctx context.Context, aspItems []calendar.Item) error {
	location, _ := time.LoadLocation("America/New_York")
	formattedTime := time.Now().In(location).Format("Monday, Jan 02 2006")

	topicMsg := fmt.Sprintf("Date: %v\nASP: %s\nDetails: %s", formattedTime, aspItems[0].Status, aspItems[0].Details)

	input := &sns.PublishInput{
		Message:  &topicMsg,
		TopicArn: &client.Config.SNSTopicARN,
	}

	_, err := client.SNS.Publish(ctx, input)
	if err != nil {
		client.Log.WithError(err).Error()
		return fmt.Errorf("error pusblishing to AWS SNS topic %s: %w", client.Config.SNSTopicARN, err)
	}

	return nil
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
