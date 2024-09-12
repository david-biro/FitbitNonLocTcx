package main

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
	"testing"
	"time"

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

func TestConvertTimestamp(t *testing.T) {
	testTimestamps := []struct {
		testName       string
		timeStamp      string
		addSecond      time.Duration
		expectedValue  string
		expectedErr    error
		expectedResult bool
	}{
		{
			testName:       "Valid RFC3339 timestamp with no added seconds",
			timeStamp:      "2024-09-07T10:00:00Z",
			addSecond:      0 * time.Second,
			expectedValue:  "2024-09-07T10:00:00Z",
			expectedResult: false,
		},
		{
			testName:       "Valid RFC3339 timestamp with 30 seconds added",
			timeStamp:      "2024-09-07T10:00:00Z",
			addSecond:      30 * time.Second,
			expectedValue:  "2024-09-07T10:00:30Z",
			expectedResult: false,
		},
		{
			testName:       "Valid RFC3339 timestamp with negative duration",
			timeStamp:      "2024-09-07T10:00:00Z",
			addSecond:      -30 * time.Second,
			expectedValue:  "2024-09-07T09:59:30Z",
			expectedResult: false,
		},
		{
			testName:       "Empty RFC3339 timestamp",
			timeStamp:      "",
			addSecond:      0 * time.Second,
			expectedErr:    &time.ParseError{},
			expectedResult: true,
		},
		{
			testName:       "Invalid RFC3339 timestamp",
			timeStamp:      "2006-01-02T15:04:05Z07:00",
			addSecond:      0 * time.Second,
			expectedErr:    &time.ParseError{},
			expectedResult: true,
		},
	}

	for _, tc := range testTimestamps {

		result, err := convertTimestamp(tc.timeStamp, tc.addSecond)

		if tc.expectedResult {
			// If an error is expected, check if the error matches the expected error (time.ParseError)
			assert.Error(t, err)
			assert.IsType(t, tc.expectedErr, err)
		} else {
			assert.NoError(t, err)
			assert.Equal(t, tc.expectedValue, result)
		}
	}
}

func TestGenerateCodeChallenge(t *testing.T) {
	tcTwoVerifier := "testverifier"
	expectedHashTcTwo := sha256.Sum256([]byte(tcTwoVerifier))
	expectedChallenge := base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(expectedHashTcTwo[:])

	testCodeChallenges := []struct {
		testName       string
		codeVerifier   string
		expectedValue  string
		expectedErr    error
		expectedResult bool
	}{
		{
			testName:       "FAILURE - test case 1, empty codeVerifier string",
			codeVerifier:   "",
			expectedValue:  "",
			expectedErr:    fmt.Errorf("error: empty codeVerifier string"),
			expectedResult: false,
		},
		{
			testName:       "SUCCESS - test case 2, use a known codeVerifier",
			codeVerifier:   "testverifier",
			expectedValue:  expectedChallenge,
			expectedErr:    nil,
			expectedResult: true,
		},
	}

	for _, tc := range testCodeChallenges {
		result, err := generateCodeChallenge(tc.codeVerifier)
		if tc.expectedResult {
			assert.True(t, reflect.DeepEqual(tc.expectedValue, result))
			assert.Nil(t, err)
		} else {
			assert.Error(t, err)
			assert.EqualError(t, err, tc.expectedErr.Error())
		}
	}
}

func TestGenerateCodeVerifier(t *testing.T) {
	verifier, err := generateCodeVerifier()
	handleVerifierTestError(t, err)

	_, err = base64.URLEncoding.WithPadding(base64.NoPadding).DecodeString(verifier)
	if err != nil {
		t.Errorf("Code verifier is not a valid base64 URL encoded string: %v", err)
	}
}

// Error handler specific to the test
func handleVerifierTestError(t *testing.T, err error) {

	if err != nil {
		assert.Error(t, err)
		t.Errorf("Unexpected error: %v", err)
	}
}
