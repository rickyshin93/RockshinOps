package main

import (
	"net/http"
	"net/url"
	"strings"
)

// MyGo uses the default HTTP client for its update requests.
func configureUpdateProxy(s *System, resolve func(*url.URL) (*url.URL, error)) {
	transport := http.DefaultTransport.(*http.Transport)
	transport.Proxy = func(req *http.Request) (*url.URL, error) {
		s.mu.Lock()
		enabled := s.preferences.UpdateUseSystemProxy
		s.mu.Unlock()
		if enabled && strings.HasPrefix(req.UserAgent(), "MyGo/") {
			return resolve(req.URL)
		}
		return http.ProxyFromEnvironment(req)
	}
}
