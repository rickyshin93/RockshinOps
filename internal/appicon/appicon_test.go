package appicon

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"net/url"
	"runtime"
	"testing"
)

func get(path string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "appicon://localhost/?path="+url.QueryEscape(path), nil))
	return rec
}

func TestRejectsNonAppPaths(t *testing.T) {
	for _, path := range []string{"", "/etc/passwd", "relative/Foo.app", "/Applications/../etc"} {
		if rec := get(path); rec.Code != http.StatusBadRequest {
			t.Errorf("%q: got %d, want 400", path, rec.Code)
		}
	}
}

func TestServesBundleIconAsPNG(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("app icons come from NSWorkspace")
	}
	rec := get("/System/Applications/Calculator.app")
	if rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != "image/png" {
		t.Fatalf("got %d %q", rec.Code, rec.Header().Get("Content-Type"))
	}
	if !bytes.HasPrefix(rec.Body.Bytes(), []byte("\x89PNG")) {
		t.Fatal("body is not a PNG")
	}
}

func TestServesFileTypeIcons(t *testing.T) {
	serve := func(query string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "appicon://localhost/type?"+query, nil))
		return rec
	}
	for _, query := range []string{"ext=..%2Fx", "ext=a.b", "ext=" + url.QueryEscape("超长扩展名")} {
		if rec := serve(query); rec.Code != http.StatusBadRequest {
			t.Errorf("%s: got %d, want 400", query, rec.Code)
		}
	}
	if runtime.GOOS != "darwin" {
		t.Skip("type icons come from NSWorkspace")
	}
	folder, pdf := serve("folder=1"), serve("ext=pdf")
	for _, rec := range []*httptest.ResponseRecorder{folder, pdf, serve("ext="), serve("ext=XLSX")} {
		if rec.Code != http.StatusOK || !bytes.HasPrefix(rec.Body.Bytes(), []byte("\x89PNG")) {
			t.Fatalf("got %d %q", rec.Code, rec.Header().Get("Content-Type"))
		}
	}
	if bytes.Equal(folder.Body.Bytes(), pdf.Body.Bytes()) {
		t.Fatal("folder and PDF icons are the same")
	}
}
