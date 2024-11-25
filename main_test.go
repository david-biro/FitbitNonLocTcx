package main

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/url"
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
	cases := []struct {
		length      int
		expectError bool
	}{
		{42, true},   // Shorter than minimum
		{43, false},  // Minimum valid length
		{128, false}, // Maximum valid length
		{129, true},  // Longer than maximum
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("Test case with length=%d", c.length), func(t *testing.T) {
			verifier, err := generateCodeVerifier(c.length)

			if c.expectError {
				if err == nil {
					t.Errorf("Expected error for length %d, got none", c.length)
				}
			} else {
				if err != nil {
					t.Errorf("Did not expect error for length %d, got: %v", c.length, err)
				}
				if len(verifier) != c.length {
					t.Errorf("Expected verifier of length %d, got %d", c.length, len(verifier))
				}
			}
		})
	}
}

func TestGetAuthURL(t *testing.T) {
	testClientID := "test-client-id"
	testRedirectURL := "https://test.com/redirect"
	testCodeChallenge := "testCodeChallenge"

	testCases := []struct {
		testName       string
		oauthCfg       *oauth2.Config
		codeChallenge  string
		expectedResult string
	}{
		{
			testName: "SUCCESS - Valid OAuth Config and Code Challenge",
			oauthCfg: &oauth2.Config{
				ClientID:    testClientID,
				RedirectURL: testRedirectURL,
				Scopes:      []string{"activity", "heartrate", "profile"},
			},
			codeChallenge: testCodeChallenge,
			expectedResult: "https://www.fitbit.com/oauth2/authorize?response_type=token" +
				"&client_id=test-client-id" +
				"&redirect_uri=https%3A%2F%2Ftest.com%2Fredirect" +
				"&scope=activity+heartrate+profile" +
				"&code_challenge=testCodeChallenge" +
				"&code_challenge_method=S256" +
				"&state=",
		},
		{
			testName:      "FAILURE - Empty Code Challenge",
			oauthCfg:      &oauth2.Config{ClientID: testClientID, RedirectURL: testRedirectURL, Scopes: []string{"activity"}},
			codeChallenge: "",
			expectedResult: "https://www.fitbit.com/oauth2/authorize?response_type=token" +
				"&client_id=test-client-id" +
				"&redirect_uri=https%3A%2F%2Ftest.com%2Fredirect" +
				"&scope=activity" +
				"&code_challenge=" +
				"&code_challenge_method=S256" +
				"&state=",
		},
		{
			testName: "FAILURE - Empty ClientID",
			oauthCfg: &oauth2.Config{
				ClientID:    "",
				RedirectURL: testRedirectURL,
				Scopes:      []string{"profile"},
			},
			codeChallenge: testCodeChallenge,
			expectedResult: "https://www.fitbit.com/oauth2/authorize?response_type=token" +
				"&client_id=" +
				"&redirect_uri=https%3A%2F%2Ftest.com%2Fredirect" +
				"&scope=profile" +
				"&code_challenge=testCodeChallenge" +
				"&code_challenge_method=S256" +
				"&state=",
		},
		{
			testName: "FAILURE - Empty Redirect URL",
			oauthCfg: &oauth2.Config{
				ClientID:    testClientID,
				RedirectURL: "",
				Scopes:      []string{"heartrate"},
			},
			codeChallenge: testCodeChallenge,
			expectedResult: "https://www.fitbit.com/oauth2/authorize?response_type=token" +
				"&client_id=test-client-id" +
				"&redirect_uri=" +
				"&scope=heartrate" +
				"&code_challenge=testCodeChallenge" +
				"&code_challenge_method=S256" +
				"&state=",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			actualResult := getAuthURL(tc.codeChallenge, tc.oauthCfg)

			parsedExpectedURL, _ := url.Parse(tc.expectedResult)
			parsedActualURL, _ := url.Parse(actualResult)

			expectedQueryParams := parsedExpectedURL.Query()
			actualQueryParams := parsedActualURL.Query()

			for key, expectedValue := range expectedQueryParams {
				if key != "state" {
					actualValue := actualQueryParams[key]
					assert.True(t, reflect.DeepEqual(expectedValue, actualValue), "Mismatch in query param: "+key)
				}
			}
		})
	}
}
