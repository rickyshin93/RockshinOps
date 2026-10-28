package main

import (
	"net/http"
	"net/url"
	"testing"
)

func TestUpdateProxyOnlyAppliesWhenEnabled(t *testing.T) {
	transport := http.DefaultTransport.(*http.Transport)
	original := transport.Proxy
	t.Cleanup(func() { transport.Proxy = original })
	service := &System{}
	calls := 0
	proxy, _ := url.Parse("http://127.0.0.1:7890")
	configureUpdateProxy(service, func(_ *url.URL) (*url.URL, error) { calls++; return proxy, nil })
	req, _ := http.NewRequest("GET", "https://github.com/example", nil)
	req.Header.Set("User-Agent", "MyGo/1 LocalOps/1")
	if _, err := transport.Proxy(req); err != nil || calls != 0 {
		t.Fatalf("disabled: calls=%d err=%v", calls, err)
	}
	service.preferences.UpdateUseSystemProxy = true
	got, err := transport.Proxy(req)
	if err != nil || got.String() != proxy.String() || calls != 1 {
		t.Fatalf("enabled: got=%v calls=%d err=%v", got, calls, err)
	}
	req.Header.Set("User-Agent", "other")
	if _, err := transport.Proxy(req); err != nil || calls != 1 {
		t.Fatalf("other request: calls=%d err=%v", calls, err)
	}
}
