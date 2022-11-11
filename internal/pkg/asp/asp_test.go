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
		// TODO: Add test cases.
		{
			name: "success suspended day",
			fields: fields{
				HTTP: newMockClient(func(req *http.Request) (*http.Response, error) {
					data := mustLoadJsonFile(t, "testdata/valid-suspended-day-response.json")

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
				HTTP: newMockClient(func(req *http.Request) (*http.Response, error) {
					data := mustLoadJsonFile(t, "testdata/valid-no-suspended-day-response.json")

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
				HTTP: newMockClient(func(req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: http.StatusNotFound,
						Body:       nil,
					}, nil
				}),
			},
			want:    nil,
			wantErr: true,
		},
		// {
		// 	name: "error performing http Do",
		// },
		// {
		// 	name: "error reading response body",
		// },
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
	type fields struct {
		Log    *logrus.Entry
		Config asp.Config
		HTTP   *http.Client
		SNS    *sns.Client
	}
	type args struct {
		ctx      context.Context
		aspItems []calendar.Item
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &asp.Client{
				Log:    tt.fields.Log,
				Config: tt.fields.Config,
				HTTP:   tt.fields.HTTP,
				SNS:    tt.fields.SNS,
			}
			if err := client.PublishSNS(tt.args.ctx, tt.args.aspItems); (err != nil) != tt.wantErr {
				t.Errorf("Client.PublishSNS() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

type mockCleint struct {
	DoFunc func(req *http.Request) (*http.Response, error)
}

func (mc *mockCleint) Do(req *http.Request) (*http.Response, error) {
	if req == nil {
		return nil, errors.New("error request is nil")
	}
	return mc.DoFunc(req)
}

func newMockClient(doFunc func(req *http.Request) (*http.Response, error)) *mockCleint {
	return &mockCleint{
		DoFunc: doFunc,
	}
}

func mustLoadJsonFile(t *testing.T, filePath string) *os.File {
	data, err := os.Open(filePath)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
