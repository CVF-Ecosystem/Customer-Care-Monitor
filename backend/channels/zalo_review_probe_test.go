package channels

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

// Independent R024 probe: numeric strings must not be admitted as numeric API codes.
func TestZaloReviewRejectsQuotedErrorBeforeRefresh(t *testing.T) {
	calls, persisted := 0, false
	a := NewZaloOAAdapter(ZaloOACredentials{AccessToken: "old", RefreshToken: "old-refresh"})
	a.client = &http.Client{Transport: refreshTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		body := `{"error":"-216"}`
		if r.URL.Host == "oauth.zaloapp.com" {
			body = `{"access_token":"new","refresh_token":"new-refresh"}`
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header), Request: r}, nil
	})}
	a.SetTokenRefreshCallback(func(string, string) error { persisted = true; return nil })
	_, err := a.FetchRecentConversations(context.Background(), time.Time{}, 0)
	if err == nil || calls != 1 || persisted {
		t.Fatalf("malformed API code caused refresh: error=%v requests=%d persisted=%v", err, calls, persisted)
	}
}

func TestZaloReviewRejectsQuotedRefreshError(t *testing.T) {
	persisted := false
	a := NewZaloOAAdapter(ZaloOACredentials{AccessToken: "old", RefreshToken: "old-refresh"})
	a.client = &http.Client{Transport: refreshTransport(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"error":"0","access_token":"new","refresh_token":"new-refresh"}`)), Header: make(http.Header), Request: r}, nil
	})}
	a.SetTokenRefreshCallback(func(string, string) error { persisted = true; return nil })
	err := a.refreshToken(context.Background())
	if err == nil || persisted || a.creds.AccessToken != "old" {
		t.Fatalf("malformed refresh code admitted: error=%v persisted=%v", err, persisted)
	}
}

func TestZaloReviewSharedHelpersRequireNumericErrorCodes(t *testing.T) {
	for _, body := range []string{`{"error":"0","data":[]}`, `{"error":0,"data":[]}`} {
		for _, api := range []string{"messages", "health"} {
			t.Run(api+body, func(t *testing.T) {
				a := NewZaloOAAdapter(ZaloOACredentials{})
				a.client = &http.Client{Transport: refreshTransport(func(r *http.Request) (*http.Response, error) {
					return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
				})}
				var err error
				if api == "messages" {
					_, err = a.FetchMessages(context.Background(), "customer", time.Time{})
				} else {
					err = a.HealthCheck(context.Background())
				}
				wantError := strings.Contains(body, `"error":"0"`)
				if (err != nil) != wantError {
					t.Fatalf("error=%v wantError=%v", err, wantError)
				}
			})
		}
	}
}

func TestZaloReviewRefreshSuccessControls(t *testing.T) {
	for _, prefix := range []string{``, `"error":0,`} {
		a := NewZaloOAAdapter(ZaloOACredentials{AccessToken: "old", RefreshToken: "old-refresh"})
		a.client = &http.Client{Transport: refreshTransport(func(r *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{` + prefix + `"access_token":"new","refresh_token":"new-refresh"}`)), Request: r}, nil
		})}
		persisted := false
		a.SetTokenRefreshCallback(func(string, string) error { persisted = true; return nil })
		if err := a.refreshToken(context.Background()); err != nil || !persisted || a.creds.AccessToken != "new" {
			t.Fatalf("valid refresh failed: error=%v persisted=%v", err, persisted)
		}
	}
}
