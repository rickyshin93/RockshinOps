package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/egoist/mygo"
	"golang.org/x/text/unicode/norm"
)

func (s *System) InstalledApps() ([]InstalledApp, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	apps := []InstalledApp{}
	for _, root := range []string{"/Applications", filepath.Join(home, "Applications")} {
		if _, err := os.Stat(root); os.IsNotExist(err) {
			continue
		} else if err != nil {
			return nil, err
		}
		err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() && strings.HasSuffix(strings.ToLower(entry.Name()), ".app") {
				apps = append(apps, InstalledApp{Name: strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name())), Path: path})
				return filepath.SkipDir
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	slices.SortFunc(apps, func(a, b InstalledApp) int { return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name)) })
	return apps, nil
}

func (s *System) ReviewApp(ctx context.Context, path string, id uint64) (AppReview, error) {
	s.appRemovalMu.Lock()
	if s.appRemoval.id != id || s.appRemoval.app != path || s.appRemoval.cancelled {
		s.appRemovalMu.Unlock()
		return AppReview{}, fmt.Errorf("app review was cancelled or replaced")
	}
	ctx, cancel := context.WithCancel(ctx)
	s.appRemoval.cancel = cancel
	s.appRemovalMu.Unlock()
	defer cancel()
	review, err := s.reviewApp(ctx, path, false)
	if err != nil {
		return AppReview{}, err
	}
	s.appRemovalMu.Lock()
	defer s.appRemovalMu.Unlock()
	if ctx.Err() != nil || s.appRemoval.id != id || s.appRemoval.cancelled {
		return AppReview{}, fmt.Errorf("app review was cancelled or replaced")
	}
	review.ReviewID = id
	s.rememberAppFiles(review)
	s.appRemoval.cancel = nil
	return review, nil
}

func (s *System) ReviewAppQuick(path string) (AppReview, error) {
	review, err := s.reviewApp(context.Background(), path, true)
	if err != nil {
		return AppReview{}, err
	}
	s.appRemovalMu.Lock()
	defer s.appRemovalMu.Unlock()
	if s.appRemoval.cancel != nil {
		s.appRemoval.cancel()
	}
	s.appReviewSeq++
	review.ReviewID = s.appReviewSeq
	s.appRemoval = appRemoval{app: path, id: review.ReviewID}
	s.rememberAppFiles(review)
	return review, nil
}

func (s *System) rememberAppFiles(review AppReview) {
	s.appRemoval.files = make(map[string]os.FileInfo, len(review.Files))
	for _, file := range review.Files {
		if info, err := os.Lstat(file.Path); err == nil {
			s.appRemoval.files[file.Path] = info
		}
	}
}

func (s *System) CancelAppReview(id uint64) error {
	s.appRemovalMu.Lock()
	defer s.appRemovalMu.Unlock()
	if s.appRemoval.id != id {
		return fmt.Errorf("app review was replaced")
	}
	s.appRemoval.cancelled = true
	if s.appRemoval.cancel != nil {
		s.appRemoval.cancel()
	}
	return nil
}

func appFileReason(name, id string, appNames []string) string {
	name = norm.NFC.String(strings.ToLower(name))
	if id != "" && strings.Contains(name, id) {
		return "包含标识符，可能共享"
	}
	for _, appName := range appNames {
		if name == appName || strings.HasPrefix(name, appName+".") || strings.HasPrefix(name, appName+"_") || strings.HasSuffix(name, "."+appName) || strings.Contains(name, "."+appName+".") {
			return "名称匹配，可能共享"
		}
	}
	return ""
}

func isAppBundle(path string, entry os.DirEntry) bool {
	if !entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".app") {
		return false
	}
	info, err := os.Lstat(filepath.Join(path, "Contents"))
	return err == nil && info.IsDir()
}

func appSearchNames(names ...string) []string {
	var result []string
	for _, name := range names {
		name = norm.NFC.String(strings.ToLower(strings.TrimSpace(name)))
		for _, variant := range []string{name, strings.ReplaceAll(name, " ", "")} {
			if variant != "" && !slices.Contains(result, variant) {
				result = append(result, variant)
			}
		}
	}
	return result
}

func (s *System) reviewApp(ctx context.Context, path string, quick bool) (AppReview, error) {
	apps, err := s.InstalledApps()
	if err != nil {
		return AppReview{}, err
	}
	index := slices.IndexFunc(apps, func(app InstalledApp) bool { return app.Path == path })
	if index < 0 {
		return AppReview{}, fmt.Errorf("app is not in the installed apps list")
	}
	app := apps[index]
	output, err := exec.Command("/usr/bin/plutil", "-convert", "json", "-o", "-", filepath.Join(path, "Contents", "Info.plist")).Output()
	if err != nil {
		return AppReview{}, fmt.Errorf("cannot read app metadata: %w", err)
	}
	var metadata struct {
		BundleID   string `json:"CFBundleIdentifier"`
		BundleName string `json:"CFBundleName"`
	}
	if err := json.Unmarshal(output, &metadata); err != nil {
		return AppReview{}, fmt.Errorf("invalid app metadata: %w", err)
	}
	id := strings.TrimSpace(metadata.BundleID)
	if strings.ContainsAny(id, "/\\") || id == "." || id == ".." {
		return AppReview{}, fmt.Errorf("invalid app bundle identifier")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return AppReview{}, err
	}
	review := AppReview{App: app, BundleID: id, Files: []AppFile{{Path: path, Reason: "应用包"}}, ScanIssues: []AppScanIssue{}}
	bundleName := metadata.BundleName
	if strings.EqualFold(strings.TrimSpace(bundleName), "Library") {
		bundleName = ""
	}
	matchID, matchNames := strings.ToLower(id), appSearchNames(app.Name, bundleName)
	lib := filepath.Join(home, "Library")
	quickRoots := []string{}
	for _, dir := range []string{"Application Support", "Caches", "Preferences", "Containers", "Group Containers", "Logs", "WebKit", "Saved Application State", "HTTPStorages"} {
		quickRoots = append(quickRoots, filepath.Join(lib, dir))
	}
	if temp, err := filepath.EvalSymlinks(os.TempDir()); err == nil && filepath.Base(temp) == "T" {
		quickRoots = append(quickRoots, temp, filepath.Join(filepath.Dir(temp), "C"))
	}
	for _, root := range quickRoots {
		children, err := os.ReadDir(root)
		if err != nil {
			continue // The complete search reports inaccessible locations.
		}
		for _, child := range children {
			if child.Type()&os.ModeSymlink != 0 || isAppBundle(filepath.Join(root, child.Name()), child) {
				continue
			}
			if reason := appFileReason(child.Name(), matchID, matchNames); reason != "" {
				review.Files = append(review.Files, AppFile{Path: filepath.Join(root, child.Name()), Reason: reason})
			}
		}
	}
	if quick {
		return review, nil
	}
	roots := []string{lib, "/Library", "/Applications", "/private/var", home}
	if temp, err := filepath.EvalSymlinks(os.TempDir()); err == nil && temp != "/private/var" && !strings.HasPrefix(temp, "/private/var/") && temp != home && !strings.HasPrefix(temp, home+string(filepath.Separator)) {
		roots = append(roots, temp)
	}
	return scanAppRoots(ctx, roots, home, path, id, matchNames, review)
}

func scanAppRoots(ctx context.Context, roots []string, home, path, id string, appNames []string, review AppReview) (AppReview, error) {
	id = strings.ToLower(id)
	seen := make(map[string]bool, len(review.Files))
	for _, file := range review.Files {
		seen[file.Path] = true
	}
	type searchResult struct {
		files  []AppFile
		issues []AppScanIssue
	}
	lib := filepath.Join(home, "Library")
	results := make(chan searchResult, len(roots))
	for _, root := range roots {
		go func(root string) {
			result := searchResult{}
			info, err := os.Lstat(root)
			if err != nil {
				// Optional search roots may not exist on this Mac.
				if !os.IsNotExist(err) {
					result.issues = append(result.issues, appScanIssue(root, err))
				}
				results <- result
				return
			}
			if !info.IsDir() {
				results <- result
				return
			}
			const workers = 4
			// A full queue falls back to local traversal, so it does not limit the search.
			jobs := make(chan string, 64)
			var pending sync.WaitGroup
			pending.Add(1)
			jobs <- root
			go func() { pending.Wait(); close(jobs) }()
			parts := make(chan searchResult, workers)
			for range workers {
				go func() {
					part := searchResult{}
					for first := range jobs {
						dirs := []string{first}
						for len(dirs) > 0 {
							dir := dirs[len(dirs)-1]
							dirs = dirs[:len(dirs)-1]
							if ctx.Err() != nil {
								pending.Done()
								continue
							}
							file, err := os.Open(dir)
							if err != nil {
								part.issues = append(part.issues, appScanIssue(dir, err))
								pending.Done()
								continue
							}
							for ctx.Err() == nil {
								entries, err := file.ReadDir(256)
								for _, entry := range entries {
									if ctx.Err() != nil {
										break
									}
									candidate := filepath.Join(dir, entry.Name())
									if entry.Type()&os.ModeSymlink != 0 || candidate == path || (root == home && (candidate == lib || candidate == filepath.Join(home, ".Trash"))) || isAppBundle(candidate, entry) {
										continue
									}
									if reason := appFileReason(entry.Name(), id, appNames); reason != "" {
										part.files = append(part.files, AppFile{Path: candidate, Reason: reason})
										if entry.IsDir() {
											continue
										}
									}
									if entry.IsDir() {
										pending.Add(1)
										select {
										case jobs <- candidate:
										default:
											dirs = append(dirs, candidate)
										}
									}
								}
								if err != nil {
									if !errors.Is(err, io.EOF) {
										part.issues = append(part.issues, appScanIssue(dir, err))
									}
									break
								}
							}
							file.Close()
							pending.Done()
						}
					}
					parts <- part
				}()
			}
			for range workers {
				part := <-parts
				result.files = append(result.files, part.files...)
				result.issues = append(result.issues, part.issues...)
			}

			results <- result
		}(root)
	}
	for range roots {
		result := <-results
		review.ScanIssues = append(review.ScanIssues, result.issues...)
		for _, file := range result.files {
			if !seen[file.Path] {
				seen[file.Path] = true
				review.Files = append(review.Files, file)
			}
		}
	}
	if err := ctx.Err(); err != nil {
		return AppReview{}, err
	}
	slices.SortFunc(review.Files[1:], func(a, b AppFile) int { return strings.Compare(a.Path, b.Path) })
	slices.SortFunc(review.ScanIssues, func(a, b AppScanIssue) int { return strings.Compare(a.Path, b.Path) })
	return review, nil
}

func appScanIssue(path string, err error) AppScanIssue {
	kind := "read"
	if os.IsPermission(err) {
		kind = "permission"
	} else if os.IsNotExist(err) {
		kind = "changed"
	}
	var pathErr *os.PathError
	if errors.As(err, &pathErr) {
		err = pathErr.Err
	}
	return AppScanIssue{Path: path, Kind: kind, Error: err.Error()}
}

func (s *System) RemoveAppFiles(paths []string) error {
	s.appRemovalMu.Lock()
	defer s.appRemovalMu.Unlock()
	if len(paths) == 0 {
		return fmt.Errorf("no items selected")
	}
	processes := s.monitor.Processes()
	if processes.Error != "" {
		return fmt.Errorf("cannot verify running processes: %s", processes.Error)
	}
	for _, process := range processes.Items {
		if process.App == s.appRemoval.app {
			return fmt.Errorf("quit the app before removing its files")
		}
	}
	seen := map[string]bool{}
	for _, path := range paths {
		old, ok := s.appRemoval.files[path]
		if !ok || seen[path] {
			return fmt.Errorf("item was not reviewed: %s", path)
		}
		seen[path] = true
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		if !os.SameFile(old, info) {
			return fmt.Errorf("item changed since review: %s", path)
		}
	}
	var failures []error
	for _, path := range paths {
		if err := mygo.Shell.TrashItem(path); err != nil {
			failures = append(failures, fmt.Errorf("%s: %w", path, err))
		} else {
			delete(s.appRemoval.files, path)
		}
	}
	return errors.Join(failures...)
}
