package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func BenchmarkInstalledApps(b *testing.B) {
	s := &System{}
	for range b.N {
		if _, err := s.InstalledApps(); err != nil {
			b.Fatal(err)
		}
	}
}

func TestAppSearchReportsCoverageWithoutBlockingCandidates(t *testing.T) {
	home := t.TempDir()
	lib := filepath.Join(home, "Library")
	blocked := filepath.Join(lib, "Protected")
	candidate := filepath.Join(lib, "Caches", "test.example.app")
	for _, dir := range []string{blocked, candidate} {
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Chmod(blocked, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(blocked, 0700) })
	app := filepath.Join(home, "Applications", "Example.app")
	review, err := scanAppRoots(context.Background(), []string{lib, filepath.Join(home, "Absent")}, home, app,
		"test.example.app", []string{"example"}, AppReview{Files: []AppFile{{Path: app, Reason: "应用包"}}})
	if err != nil {
		t.Fatal(err)
	}
	if len(review.ScanIssues) != 1 || review.ScanIssues[0].Path != blocked || review.ScanIssues[0].Kind != "permission" || review.ScanIssues[0].Error == "" {
		t.Fatalf("expected the inaccessible directory, not missing optional roots: %+v", review.ScanIssues)
	}
	if !slices.ContainsFunc(review.Files, func(file AppFile) bool { return file.Path == candidate }) {
		t.Fatalf("accessible candidate was lost: %+v", review.Files)
	}
}

func TestAppSearchMatchesCanonicalUnicodeNamesLiterally(t *testing.T) {
	home := t.TempDir()
	lib := filepath.Join(home, "Library")
	candidate := filepath.Join(lib, "REA Cafe\u0301.cache")
	if err := os.MkdirAll(candidate, 0700); err != nil {
		t.Fatal(err)
	}
	app := filepath.Join(home, "Applications", "REA Café.app")
	review, err := scanAppRoots(context.Background(), []string{lib}, home, app, "test.unicode", appSearchNames("REA Café"),
		AppReview{Files: []AppFile{{Path: app, Reason: "应用包"}}})
	if err != nil {
		t.Fatal(err)
	}
	if len(review.Files) != 2 || review.Files[1].Path != candidate {
		t.Fatalf("canonical-equivalent filename was missed or rewritten: %+v", review.Files)
	}
	for _, name := range []string{"REA [Bracket]", "REA *Star?", "REA O'Reilly", "REA \"Double\"", "REA Tab\tName", "REA Line\nName", "-REA Dash"} {
		if appFileReason(name+".cache", "", appSearchNames(name)) == "" || appFileReason("prefix"+name+"suffix", "", appSearchNames(name)) != "" {
			t.Errorf("filename boundaries were not literal for %q", name)
		}
	}
	if appFileReason("REA XStara.cache", "", appSearchNames("REA *Star?")) != "" || appFileReason("REA BBracket.cache", "", appSearchNames("REA [Bracket]")) != "" {
		t.Fatal("wildcards in app names expanded into unrelated files")
	}
}

func TestAppReviewDoesNotWaitForDiskScan(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	app := filepath.Join(home, "Applications", "ExampleApp.app")
	if err := os.MkdirAll(filepath.Join(app, "Contents"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(app, "Contents", "Info.plist"), []byte(`<?xml version="1.0" encoding="UTF-8"?><plist version="1.0"><dict><key>CFBundleIdentifier</key><string>test.example.app</string></dict></plist>`), 0600); err != nil {
		t.Fatal(err)
	}
	s := &System{}
	quick, err := s.ReviewAppQuick(app)
	if err != nil {
		t.Fatal(err)
	}
	s.spaceMu.Lock()
	defer s.spaceMu.Unlock()
	done := make(chan error, 1)
	go func() { done <- s.CancelAppReview(quick.ReviewID) }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("app review blocked by an unrelated disk scan")
	}
	if _, err := s.ReviewApp(context.Background(), app, quick.ReviewID); err == nil {
		t.Fatal("cancelled search was restarted")
	}
}

func TestReviewAppFindsCandidatesWithoutSelectingThem(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("TMPDIR", home)
	app := filepath.Join(home, "Applications", "ExampleApp.app")
	if err := os.MkdirAll(filepath.Join(app, "Contents"), 0700); err != nil {
		t.Fatal(err)
	}
	plist := `<?xml version="1.0" encoding="UTF-8"?><!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd"><plist version="1.0"><dict><key>CFBundleIdentifier</key><string>test.rockshinops.exampleapp</string><key>CFBundleName</key><string>Different Name</string></dict></plist>`
	if err := os.WriteFile(filepath.Join(app, "Contents", "Info.plist"), []byte(plist), 0600); err != nil {
		t.Fatal(err)
	}
	cache := filepath.Join(home, "Library", "Caches", "test.rockshinops.exampleapp.dev")
	if err := os.MkdirAll(cache, 0700); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(home, "Library", "Caches", "DifferentName")
	if err := os.MkdirAll(alias, 0700); err != nil {
		t.Fatal(err)
	}
	otherBundle := filepath.Join(home, "Library", "Caches", "DifferentName.app")
	if err := os.MkdirAll(filepath.Join(otherBundle, "Contents"), 0700); err != nil {
		t.Fatal(err)
	}
	s := &System{}
	quick, err := s.ReviewAppQuick(app)
	if err != nil || !slices.ContainsFunc(quick.Files, func(file AppFile) bool { return file.Path == cache }) {
		t.Fatalf("quick review missed the common cache: %+v %v", quick, err)
	}
	if !slices.ContainsFunc(quick.Files, func(file AppFile) bool { return file.Path == alias }) {
		t.Fatalf("quick review missed the app bundle name: %+v", quick.Files)
	}
	if slices.ContainsFunc(quick.Files, func(file AppFile) bool { return file.Path == otherBundle }) {
		t.Fatalf("quick review included a different application bundle: %+v", quick.Files)
	}
	review, err := scanAppRoots(context.Background(), []string{filepath.Join(home, "Library")}, home, app, "test.rockshinops.exampleapp", appSearchNames("ExampleApp"), quick)
	if err != nil {
		t.Fatal(err)
	}
	index := slices.IndexFunc(review.Files, func(file AppFile) bool { return file.Path == cache })
	if index < 0 || review.Files[index].Reason != "包含标识符，可能共享" {
		t.Fatalf("missing candidate: %+v", review.Files)
	}
	if _, err := os.Stat(cache); err != nil {
		t.Fatal("review modified candidate", err)
	}
	if _, err := s.ReviewAppQuick(filepath.Join(home, "not-an-app")); err == nil {
		t.Fatal("accepted an unlisted app")
	}
}

func TestUnboundedAppSearchReachesNestedFiles(t *testing.T) {
	home := t.TempDir()
	root := filepath.Join(home, "Library")
	deep := filepath.Join(root, "a", "b", "c", "d", "e")
	if err := os.MkdirAll(deep, 0700); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 200; i++ {
		if err := os.WriteFile(filepath.Join(root, fmt.Sprintf("item-%03d", i)), nil, 0600); err != nil {
			t.Fatal(err)
		}
	}
	match := filepath.Join(deep, "test.example.app.plist")
	if err := os.WriteFile(match, []byte("data"), 0600); err != nil {
		t.Fatal(err)
	}
	review := AppReview{Files: []AppFile{{Path: filepath.Join(home, "Example.app")}}}
	result, err := scanAppRoots(context.Background(), []string{root}, home, review.Files[0].Path, "test.example.app", appSearchNames("Example"), review)
	if err != nil || !slices.ContainsFunc(result.Files, func(f AppFile) bool { return f.Path == match }) {
		t.Fatalf("nested match missing: %+v %v", result, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := scanAppRoots(ctx, []string{root}, home, review.Files[0].Path, "test.example.app", appSearchNames("Example"), review); err == nil {
		t.Fatal("cancelled search did not stop")
	}
}

func TestAppSearchFindsNamesAtAnyDepth(t *testing.T) {
	home := t.TempDir()
	library := filepath.Join(home, "Library")
	project := filepath.Join(home, "projects", "ExampleApp")
	cache := filepath.Join(library, "Application Support", "ExampleApp")
	nestedName := filepath.Join(library, "Caches", "a", "b", "c", "d", "e", "ExampleApp")
	identifier := filepath.Join(home, "other", "test.example.app.data")
	personalName := filepath.Join(home, "other", "a", "b", "c", "ExampleApp.plist")
	veryDeepName := filepath.Join(library, "Caches", strings.Repeat("d/", 64), "ExampleApp.plist")
	userAppSupport := filepath.Join(home, "Applications", "support", "a", "b", "ExampleApp")
	otherAppData := filepath.Join(home, "Applications", "Other.app", "Contents", "ExampleApp")
	unrelated := filepath.Join(home, "projects", "OtherExampleApp")
	for _, dir := range []string{project, cache, nestedName, identifier, personalName, veryDeepName, userAppSupport, otherAppData, unrelated} {
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	review := AppReview{Files: []AppFile{{Path: filepath.Join(home, "Applications", "ExampleApp.app")}}}
	result, err := scanAppRoots(context.Background(), []string{library, home}, home, review.Files[0].Path, "test.example.app", appSearchNames("ExampleApp"), review)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{project, cache, nestedName, identifier, personalName, veryDeepName, userAppSupport} {
		if !slices.ContainsFunc(result.Files, func(f AppFile) bool { return f.Path == expected }) {
			t.Fatalf("missing candidate %s: %+v", expected, result.Files)
		}
	}
	if slices.ContainsFunc(result.Files, func(f AppFile) bool { return f.Path == unrelated }) {
		t.Fatalf("partial name %s was treated as an app residual: %+v", unrelated, result.Files)
	}
	if slices.ContainsFunc(result.Files, func(f AppFile) bool { return f.Path == otherAppData }) {
		t.Fatalf("included data from a different application bundle: %+v", result.Files)
	}
}

func TestAppNameMatchesUseFilenameBoundaries(t *testing.T) {
	for _, tc := range []struct {
		name string
		want bool
	}{
		{"Agent HUD", true},
		{"AgentHUD.plist", true},
		{"Agent HUD_backup", true},
		{"com.vendor.AgentHUD", true},
		{"com.vendor.AgentHUD.plist", true},
		{"OtherAgentHUD", false},
		{"Agent HUD Dev", false},
		{"OtherAgentHUD.plist", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := appFileReason(tc.name, "app.agenthud.mac", appSearchNames("Agent HUD")) != ""
			if got != tc.want {
				t.Fatalf("match = %v, want %v", got, tc.want)
			}
		})
	}
	if got := appFileReason("prefix.app.agenthud.mac.cache", "app.agenthud.mac", nil); got == "" {
		t.Fatal("bundle identifier substring should still match")
	}
	if got := appFileReason("unrelated.plist", "", appSearchNames("ExampleApp")); got != "" {
		t.Fatal("an absent bundle identifier must not match unrelated files")
	}
}

func TestReviewAppWithoutBundleIdentifierUsesNames(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("TMPDIR", home)
	app := filepath.Join(home, "Applications", "ExampleApp.app")
	cache := filepath.Join(home, "Library", "Caches", "ExampleApp")
	unrelated := filepath.Join(home, "Library", "Caches", "unrelated")
	nested := filepath.Join(home, "projects", "one", "two", "three", "ExampleApp.plist")
	for _, dir := range []string{filepath.Join(app, "Contents"), cache, unrelated, filepath.Dir(nested)} {
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(nested, nil, 0600); err != nil {
		t.Fatal(err)
	}
	plist := `<?xml version="1.0"?><plist version="1.0"><dict><key>CFBundleName</key><string>Library</string></dict></plist>`
	if err := os.WriteFile(filepath.Join(app, "Contents", "Info.plist"), []byte(plist), 0600); err != nil {
		t.Fatal(err)
	}
	quick, err := (&System{}).reviewApp(context.Background(), app, true)
	if err != nil {
		t.Fatal(err)
	}
	if quick.BundleID != "" || len(quick.Files) != 2 || !slices.ContainsFunc(quick.Files, func(f AppFile) bool { return f.Path == cache }) {
		t.Fatalf("name-only review returned unexpected candidates: %+v", quick)
	}
	full, err := scanAppRoots(context.Background(), []string{home}, home, app, "", appSearchNames("ExampleApp"), quick)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.ContainsFunc(full.Files, func(f AppFile) bool { return f.Path == nested }) || slices.ContainsFunc(full.Files, func(f AppFile) bool { return f.Path == unrelated }) {
		t.Fatalf("name-only search missed a deep match or included an unrelated item: %+v", full.Files)
	}
}

func TestAppSearchTraversesWideAndDeepTrees(t *testing.T) {
	home := t.TempDir()
	root := filepath.Join(home, "Library")
	const branches = 160
	for i := 0; i < branches; i++ {
		dir := filepath.Join(root, fmt.Sprintf("branch-%03d", i), "a", "b", "c", "d")
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "test.example.app.plist"), nil, 0600); err != nil {
			t.Fatal(err)
		}
	}
	review := AppReview{Files: []AppFile{{Path: filepath.Join(home, "Example.app")}}}
	result, err := scanAppRoots(context.Background(), []string{root}, home, review.Files[0].Path, "test.example.app", nil, review)
	if err != nil || len(result.Files) != branches+1 {
		t.Fatalf("found %d candidates, want %d: %v", len(result.Files), branches+1, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cancellable := &cancelDuringScan{Context: ctx, cancel: cancel}
	done := make(chan error, 1)
	go func() {
		_, err := scanAppRoots(cancellable, []string{root}, home, review.Files[0].Path, "test.example.app", nil, review)
		done <- err
	}()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("interrupted search returned %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("interrupted search did not drain queued directories")
	}
}

type cancelDuringScan struct {
	context.Context
	cancel context.CancelFunc
	checks atomic.Int32
}

func (c *cancelDuringScan) Err() error {
	if c.checks.Add(1) == 100 {
		c.cancel()
	}
	return c.Context.Err()
}

func TestQuickReviewFindsSystemCacheSibling(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	temp := filepath.Join(home, "system", "T")
	cache := filepath.Join(home, "system", "C", "test.example.app")
	app := filepath.Join(home, "Applications", "Example.app")
	for _, dir := range []string{temp, cache, filepath.Join(app, "Contents")} {
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("TMPDIR", temp)
	cache, err := filepath.EvalSymlinks(cache)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(app, "Contents", "Info.plist"), []byte(`<?xml version="1.0" encoding="UTF-8"?><plist version="1.0"><dict><key>CFBundleIdentifier</key><string>test.example.app</string></dict></plist>`), 0600); err != nil {
		t.Fatal(err)
	}
	review, err := (&System{}).ReviewAppQuick(app)
	if err != nil || !slices.ContainsFunc(review.Files, func(f AppFile) bool { return f.Path == cache }) {
		t.Fatalf("quick review missed system cache: %+v %v", review.Files, err)
	}
	review, err = scanAppRoots(context.Background(), []string{filepath.Dir(cache)}, home, app, "test.example.app", appSearchNames("Example"), AppReview{})
	if err != nil || !slices.ContainsFunc(review.Files, func(f AppFile) bool { return f.Path == cache }) {
		t.Fatalf("complete review missed system cache: %+v %v", review.Files, err)
	}
}

func TestAppSearchDoesNotFollowSymlinks(t *testing.T) {
	home := t.TempDir()
	root := filepath.Join(home, "Library")
	outside := filepath.Join(home, "outside")
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(outside, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outside, "test.example.app.plist"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "link")); err != nil {
		t.Fatal(err)
	}
	review := AppReview{Files: []AppFile{{Path: filepath.Join(home, "Example.app")}}}
	result, err := scanAppRoots(context.Background(), []string{root, filepath.Join(root, "link")}, home, review.Files[0].Path, "test.example.app", appSearchNames("Example"), review)
	if err != nil || len(result.Files) != 1 {
		t.Fatalf("followed a symlink: %+v %v", result, err)
	}
}

func BenchmarkSearchAppRoots(b *testing.B) {
	root := b.TempDir()
	roots := make([]string, 4)
	for i := range roots {
		roots[i] = filepath.Join(root, fmt.Sprintf("root%d", i))
		for j := 0; j < 80; j++ {
			dir := filepath.Join(roots[i], fmt.Sprintf("folder%d", j))
			if err := os.MkdirAll(dir, 0700); err != nil {
				b.Fatal(err)
			}
			for k := 0; k < 4; k++ {
				if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("item%d", k)), nil, 0600); err != nil {
					b.Fatal(err)
				}
			}
		}
	}
	for _, parallel := range []bool{false, true} {
		name := "serial"
		if parallel {
			name = "parallel"
		}
		b.Run(name, func(b *testing.B) {
			for range b.N {
				review := AppReview{Files: []AppFile{{Path: filepath.Join(root, "app")}}}
				if parallel {
					if _, err := scanAppRoots(context.Background(), roots, root, "", "test.example.app", appSearchNames("Example"), review); err != nil {
						b.Fatal(err)
					}
				} else {
					for _, item := range roots {
						if _, err := scanAppRoots(context.Background(), []string{item}, root, "", "test.example.app", appSearchNames("Example"), review); err != nil {
							b.Fatal(err)
						}
					}
				}
			}
		})
	}
}
