package channels

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

type refreshTransport func(*http.Request) (*http.Response, error)

func (f refreshTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestZaloRefreshPersistenceFailureStopsRetry(t *testing.T) {
	adapter := NewZaloOAAdapter(ZaloOACredentials{AppID: "synthetic", AppSecret: "synthetic", AccessToken: "old", RefreshToken: "old-refresh"})
	requests := 0
	adapter.client = &http.Client{Transport: refreshTransport(func(r *http.Request) (*http.Response, error) {
		requests++
		body := `{"error":-216}`
		if strings.Contains(r.URL.Host, "oauth.zaloapp.com") {
			body = `{"error":0,"access_token":"new","refresh_token":"new-refresh"}`
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
	})}
	persistFailure := errors.New("bounded write failed")
	adapter.SetTokenRefreshCallback(func(access, refresh string) error {
		if access != "new" || refresh != "new-refresh" {
			t.Fatalf("unexpected synthetic token pair")
		}
		return persistFailure
	})
	_, err := adapter.doRequest(context.Background(), http.MethodGet, "https://openapi.zalo.me/v3.0/oa/synthetic", nil)
	if !errors.Is(err, persistFailure) || requests != 2 {
		t.Fatalf("refresh must stop after failed persist; err %v, requests %d", err, requests)
	}
	if adapter.creds.AccessToken != "old" || adapter.creds.RefreshToken != "old-refresh" {
		t.Fatal("failed persistence replaced in-memory credentials")
	}
}
