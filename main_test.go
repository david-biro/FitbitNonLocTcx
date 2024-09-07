package main

import (
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/fitbit"
)

// FaultyReader simulates an io.Reader that always fails
type FaultyReader struct{}

func (f *FaultyReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("simulated read error")
}

func TestReadCredFile(t *testing.T) {
	testClientID := "test-client-id"
	testClientSecret := "test-client-secret"
	testRedirectURL := "https://test.com/redirect"

	testCases := []struct {
		testName            string
		actualJSON          string
		osReaderMock        func(s string) io.Reader
		expectedResult      bool
		expectedErr         error
		expectedOAuthConfig *oauth2.Config
	}{
		{
			testName: "SUCCESS - everything filled up, returned OAuth Config",
			actualJSON: `{
					"clientID": "` + testClientID + `",
					"clientSecret": "` + testClientSecret + `",
					"redirectUrl": "` + testRedirectURL + `"
				}`,
			osReaderMock: func(s string) io.Reader {
				return strings.NewReader(s)
			},
			expectedResult: true,
			expectedErr:    nil,
			expectedOAuthConfig: &oauth2.Config{
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
				RedirectURL:  "https://test.com/redirect",
				Scopes:       []string{"activity", "heartrate", "location", "profile"},
				Endpoint:     fitbit.Endpoint,
			},
		},
		{
			testName:   "FAILURE - read json error",
			actualJSON: "",
			osReaderMock: func(s string) io.Reader {
				return &FaultyReader{}
			},
			expectedResult: false,
			expectedErr:    fmt.Errorf("failed to read file: simulated read error"),
			expectedOAuthConfig: &oauth2.Config{
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
				RedirectURL:  "https://test.com/redirect",
				Scopes:       []string{"activity", "heartrate", "location", "profile"},
				Endpoint:     fitbit.Endpoint,
			},
		},
		{
			testName:   "FAILURE - json unmarshal error",
			actualJSON: "",
			osReaderMock: func(s string) io.Reader {
				return strings.NewReader(s)
			},
			expectedResult: false,
			expectedErr:    fmt.Errorf("failed to unmarshal JSON: unexpected end of JSON input"),
			expectedOAuthConfig: &oauth2.Config{
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
				RedirectURL:  "https://test.com/redirect",
				Scopes:       []string{"activity", "heartrate", "location", "profile"},
				Endpoint:     fitbit.Endpoint,
			},
		},
		{
			testName: "FAILURE - missing client id",
			actualJSON: `{
					"clientID": "",
					"clientSecret": "` + testClientSecret + `",
					"redirectUrl": "` + testRedirectURL + `"
				}`,
			osReaderMock: func(s string) io.Reader {
				return strings.NewReader(s)
			},
			expectedResult: false,
			expectedErr:    fmt.Errorf("ERROR The clientID and redirect URL cannot be empty."),
			expectedOAuthConfig: &oauth2.Config{
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
				RedirectURL:  "https://test.com/redirect",
				Scopes:       []string{"activity", "heartrate", "location", "profile"},
				Endpoint:     fitbit.Endpoint,
			},
		},
		{
			testName: "FAILURE - missing client id",
			actualJSON: `{
					"clientID": "` + testClientID + `",
					"clientSecret": "` + testClientSecret + `",
					"redirectUrl": ""
				}`,
			osReaderMock: func(s string) io.Reader {
				return strings.NewReader(s)
			},
			expectedResult: false,
			expectedErr:    fmt.Errorf("ERROR The clientID and redirect URL cannot be empty."),
			expectedOAuthConfig: &oauth2.Config{
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
				RedirectURL:  "https://test.com/redirect",
				Scopes:       []string{"activity", "heartrate", "location", "profile"},
				Endpoint:     fitbit.Endpoint,
			},
		},
	}

	for _, tc := range testCases {
		reader := tc.osReaderMock(tc.actualJSON)
		oauthCfg, err := readCredFile(reader)
		if tc.expectedResult {
			assert.True(t, reflect.DeepEqual(tc.expectedOAuthConfig, oauthCfg))
			assert.Nil(t, err)
		} else {
			assert.Error(t, err)
			assert.EqualError(t, err, tc.expectedErr.Error())
		}
	}
}
