package asp_test

import (
	"context"
	"errors"
	"net/http"
	"os"
	"reflect"
	"testing"

	"github.com/adiazny/nyc-asp-lambda/internal/pkg/asp"
	"github.com/adiazny/nyc-asp-lambda/internal/pkg/calendar"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/sirupsen/logrus"
)

func TestClient_GetASPItems(t *testing.T) {
	type fields struct {
		Log    *logrus.Entry
		Config asp.Config
		HTTP   asp.HTTPClient
		SNS    *sns.Client
	}
	tests := []struct {
		name    string
		fields  fields
		want    []calendar.Item
		wantErr bool
	}{
		{
			name: "success suspended day",
			fields: fields{
				HTTP: newMockHTTPClient(func(req *http.Request) (*http.Response, error) {
					data := mustLoadJSONFile(t, "testdata/valid-suspended-day-response.json")

					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       data,
					}, nil
				}),
			},
			want: []calendar.Item{{
				Details: "Alternate side parking is suspended for Veterans Day. Meters are in effect.",
				Status:  "SUSPENDED",
				Type:    "Alternate Side Parking",
			}},
			wantErr: false,
		},
		{
			name: "success no suspended day",
			fields: fields{
				HTTP: newMockHTTPClient(func(req *http.Request) (*http.Response, error) {
					data := mustLoadJSONFile(t, "testdata/valid-no-suspended-day-response.json")

					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       data,
					}, nil
				}),
			},
			want:    []calendar.Item{},
			wantErr: false,
		},
		{
			name: "non 200 response status",
			fields: fields{
				HTTP: newMockHTTPClient(func(req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: http.StatusNotFound,
						Body:       nil,
					}, nil
				}),
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			client := &asp.Client{
				Log:    tt.fields.Log,
				Config: tt.fields.Config,
				HTTP:   tt.fields.HTTP,
				SNS:    tt.fields.SNS,
			}

			got, err := client.GetASPItems()

			if (err != nil) != tt.wantErr {
				t.Errorf("Client.GetASPItems() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Client.GetASPItems() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClient_PublishSNS(t *testing.T) {
	snsMsgId := "12345"

	type fields struct {
		Log    *logrus.Entry
		Config asp.Config
		HTTP   asp.HTTPClient
		SNS    asp.SNSClient
	}

	type args struct {
		ctx      context.Context
		aspItems []calendar.Item
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    asp.LambdaResponse
		wantErr bool
	}{
		{
			name: "success publish",
			fields: fields{
				Log: logrus.NewEntry(logrus.New()),
				SNS: newMockSNSClient(func(ctx context.Context, params *sns.PublishInput, optFns ...func(*sns.Options)) (*sns.PublishOutput, error) {
					return &sns.PublishOutput{MessageId: &snsMsgId}, nil
				}),
			},
			args: args{
				ctx: context.Background(),
				aspItems: []calendar.Item{{
					Details: "Alternate side parking is suspended for Veterans Day. Meters are in effect.",
					Status:  "SUSPENDED",
					Type:    "Alternate Side Parking",
				}},
			},
			want: asp.LambdaResponse{Message: "ASP published to SNS",
				ASPItems: []calendar.Item{
					{Details: "Alternate side parking is suspended for Veterans Day. Meters are in effect.",
						Status: "SUSPENDED",
						Type:   "Alternate Side Parking",
					},
				}},
			wantErr: false,
		},
		{
			name: "success no publish",
			fields: fields{
				Log: logrus.NewEntry(logrus.New()),
				SNS: newMockSNSClient(func(ctx context.Context, params *sns.PublishInput, optFns ...func(*sns.Options)) (*sns.PublishOutput, error) {
					return nil, nil
				}),
			},
			args: args{
				ctx:      context.Background(),
				aspItems: []calendar.Item{},
			},
			want:    asp.LambdaResponse{Message: "No ASP to publish"},
			wantErr: false,
		},
		{
			name: "error",
			fields: fields{
				Log: logrus.NewEntry(logrus.New()),
				SNS: newMockSNSClient(func(ctx context.Context, params *sns.PublishInput, optFns ...func(*sns.Options)) (*sns.PublishOutput, error) {
					return nil, errors.New("mock error")
				}),
			},
			args: args{
				ctx:      context.Background(),
				aspItems: []calendar.Item{{}},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			client := &asp.Client{
				Log:    tt.fields.Log,
				Config: tt.fields.Config,
				HTTP:   tt.fields.HTTP,
				SNS:    tt.fields.SNS,
			}

			got, err := client.PublishSNS(tt.args.ctx, tt.args.aspItems)
			if (err != nil) != tt.wantErr {
				t.Errorf("Client.PublishSNS() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Client.PublishSNS() = %v, want %v", got, tt.want)
			}

		})
	}
}

func newMockHTTPClient(doFunc func(req *http.Request) (*http.Response, error)) *mockHTTPCleint {
	return &mockHTTPCleint{
		DoFunc: doFunc,
	}
}

type mockHTTPCleint struct {
	DoFunc func(req *http.Request) (*http.Response, error)
}

func (mhc *mockHTTPCleint) Do(req *http.Request) (*http.Response, error) {
	if req == nil {
		return nil, errors.New("error request is nil")
	}

	return mhc.DoFunc(req)
}

func newMockSNSClient(publishFunc func(ctx context.Context, params *sns.PublishInput, optFns ...func(*sns.Options)) (*sns.PublishOutput, error)) *mockSNSClient {
	return &mockSNSClient{
		publishFunc: publishFunc,
	}
}

type mockSNSClient struct {
	publishFunc func(ctx context.Context, params *sns.PublishInput, optFns ...func(*sns.Options)) (*sns.PublishOutput, error)
}

func (msc *mockSNSClient) Publish(ctx context.Context, params *sns.PublishInput, optFns ...func(*sns.Options)) (*sns.PublishOutput, error) {
	return msc.publishFunc(ctx, params, optFns...)
}

func mustLoadJSONFile(t *testing.T, filePath string) *os.File {
	data, err := os.Open(filePath)
	if err != nil {
		t.Fatal(err)
	}

	return data
}
