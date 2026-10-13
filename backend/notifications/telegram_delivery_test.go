package notifications

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

type r091TelegramTransport func(*http.Request) (*http.Response, error)

func (fn r091TelegramTransport) RoundTrip(req *http.Request) (*http.Response, error) { return fn(req) }

type r091TelegramReadFailure struct{}

func (r091TelegramReadFailure) Read([]byte) (int, error) {
	return 0, errors.New("synthetic response read failure")
}
func (r091TelegramReadFailure) Close() error { return nil }

func TestR091TelegramAcceptedRejectedAndAmbiguousOutcomesAreTypedAndSecretSafe(t *testing.T) {
	token := "SYNTHETIC_R091_" + "BOT_TOKEN"
	cases := []struct {
		name  string
		fn    func(*http.Request) (*http.Response, error)
		want  string
		isNil bool
	}{
		{"confirmed acceptance", func(req *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"ok":true}`)), Request: req}, nil
		}, deliverySent, true},
		{"explicit API rejection", func(req *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 400, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"ok":false,"description":"rejected SYNTHETIC_R091_BOT_TOKEN"}`)), Request: req}, nil
		}, deliveryFailedStatus, false},
		{"transport ambiguity", func(*http.Request) (*http.Response, error) { return nil, errors.New("synthetic transport failure") }, deliveryUnknown, false},
		{"response read ambiguity", func(req *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 200, Header: make(http.Header), Body: r091TelegramReadFailure{}, Request: req}, nil
		}, deliveryUnknown, false},
		{"response decode ambiguity", func(req *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"ok":`)), Request: req}, nil
		}, deliveryUnknown, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			n := NewTelegramNotifier(token, "synthetic-chat")
			n.client = &http.Client{Transport: r091TelegramTransport(tc.fn)}
			err := n.Send(context.Background(), "synthetic subject", "synthetic body")
			if tc.isNil {
				if err != nil {
					t.Fatalf("confirmed Telegram response returned an error: %v", err)
				}
				return
			}
			var typed *deliveryError
			if !errors.As(err, &typed) || typed.status != tc.want {
				t.Fatalf("Telegram outcome=%T %v, want %s", err, err, tc.want)
			}
			if strings.Contains(err.Error(), token) || strings.Contains(err.Error(), "description") {
				t.Fatalf("Telegram error included credential/provider response detail: %v", err)
			}
		})
	}
}

func TestR091TelegramHealthCheckErrorsDoNotExposeToken(t *testing.T) {
	token := "SYNTHETIC_R091_" + "HEALTH_TOKEN"
	n := NewTelegramNotifier(token, "synthetic-chat")
	n.client = &http.Client{Transport: r091TelegramTransport(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("synthetic transport error")
	})}
	if err := n.HealthCheck(context.Background()); err == nil || strings.Contains(err.Error(), token) {
		t.Fatalf("Telegram health error is nil or exposed the token: %v", err)
	}
}

func TestR091R1TelegramUnverifiableAcceptanceEnvelopeIsUnknown(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{name: "missing ok", body: `{"description":"synthetic response"}`},
		{name: "null ok", body: `{"ok":null}`},
		{name: "null envelope", body: `null`},
		{name: "array envelope", body: `[]`},
		{name: "string ok", body: `{"ok":"false"}`},
		{name: "number ok", body: `{"ok":0}`},
		{name: "object ok", body: `{"ok":{}}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			n := NewTelegramNotifier("synthetic-token", "synthetic-chat")
			n.client = &http.Client{Transport: r091TelegramTransport(func(req *http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(tc.body)), Request: req}, nil
			})}
			err := n.Send(context.Background(), "synthetic subject", "synthetic body")
			var typed *deliveryError
			if !errors.As(err, &typed) || typed.status != deliveryUnknown {
				t.Fatalf("unverifiable Telegram acceptance envelope was classified as %T %v; want UNKNOWN", err, err)
			}
			if strings.Contains(err.Error(), "synthetic-token") || strings.Contains(err.Error(), tc.body) {
				t.Fatalf("Telegram ambiguity error exposed request or response details: %v", err)
			}
		})
	}
}
