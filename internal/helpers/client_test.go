package helpers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHTTPPrefixAndTLSWithDeadline(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/custom/v1/auth/token" {
			t.Errorf("wrong URL %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"token":"token"}`))
	}))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	token, err := NewHttpCaller(ctx, true).GetJwtToken(ctx, server.URL+"/custom", "root@localhost", "secret")
	if err != nil || token != "token" {
		t.Fatalf("%q %v", token, err)
	}
	if _, err := NewHttpCaller(ctx, false).GetJwtToken(ctx, server.URL+"/custom", "root@localhost", "secret"); err == nil {
		t.Fatal("trusted untrusted certificate")
	}
	for host, want := range map[string]string{"http://mac:8080": "http://mac:8080/api/v1", "http://mac:8080/api": "http://mac:8080/api/v1", "https://[::1]:8443/custom/": "https://[::1]:8443/custom/v1"} {
		if got := GetHostApiVersionedBaseUrl(host); got != want {
			t.Fatalf("%s != %s", got, want)
		}
	}
}

func TestHTTPStatusRedactsResponseAndHonorsCancellation(t *testing.T) {
	secret := "a-very-sensitive-password"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		_, _ = w.Write([]byte(`{"message":"` + secret + `","code":200}`))
	}))
	defer server.Close()
	response, err := NewHttpCaller(context.Background(), false).GetDataFromClient(context.Background(), server.URL, nil, nil, nil)
	var status *HTTPStatusError
	if !errors.As(err, &status) || status.StatusCode != 401 || response.ApiError.Code != 401 || strings.Contains(err.Error(), secret) {
		t.Fatalf("invalid sanitized status %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = NewHttpCaller(ctx, false).GetDataFromClient(ctx, server.URL, nil, nil, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("lost cancellation: %v", err)
	}
}
