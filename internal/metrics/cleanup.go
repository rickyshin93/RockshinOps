package metrics

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"rockshinops/internal/localization"
)

type ScanTarget struct {
	Name string `json:"name"`
	Path string `json:"path"`
}
type SpaceItem struct {
	Name      string `json:"name"`
	Path      string `json:"path"`
	Bytes     int64  `json:"bytes"`
	Files     int    `json:"files"`
	Directory bool   `json:"directory"`
	Scanned   bool   `json:"scanned"`
}
type ScanGroup struct {
	Name    string      `json:"name"`
	Path    string      `json:"path"`
	Bytes   int64       `json:"bytes"`
	Files   int         `json:"files"`
	Skipped int         `json:"skipped"`
	Error   string      `json:"error"`
	Items   []SpaceItem `json:"items"`
}
type ScanResult struct {
	Groups     []ScanGroup `json:"groups"`
	StartedAt  time.Time   `json:"startedAt"`
	FinishedAt time.Time   `json:"finishedAt"`
	Complete   bool        `json:"complete"`
	Warning    string      `json:"warning"`
}

// ScanSpace only measures ordinary file sizes. It does not follow symbolic links,
// delete files, or label application data as safely reclaimable.
func ScanSpace(ctx context.Context, targets []ScanTarget, maxEntries, maxItems int, timeout time.Duration) ScanResult {
	return scanSpace(ctx, targets, maxEntries, maxItems, timeout, nil)
}

// ScanSpaceWithProgress counts directory entries first, then reports entries
// visited while measuring. The filesystem can change between the two passes.
func ScanSpaceWithProgress(ctx context.Context, target ScanTarget, counting func(), ready func(int64), visited func()) ScanResult {
	var total int64
	_ = filepath.WalkDir(target.Path, func(_ string, _ fs.DirEntry, _ error) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		total++
		counting()
		return nil
	})
	ready(total)
	return scanSpace(ctx, []ScanTarget{target}, 0, 0, 0, visited)
}

func scanSpace(ctx context.Context, targets []ScanTarget, maxEntries, maxItems int, timeout time.Duration, visited func()) ScanResult {
	result := ScanResult{Groups: []ScanGroup{}, StartedAt: time.Now().UTC(), Complete: true}
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	entries := 0
	for _, target := range targets {
		group := ScanGroup{Name: target.Name, Path: target.Path, Items: []SpaceItem{}}
		items := map[string]*SpaceItem{}
		// List all immediate children before traversing any one of them. A large
		// child can exhaust the scan budget before WalkDir reaches its siblings.
		if children, err := os.ReadDir(target.Path); err == nil {
			for _, child := range children {
				if child.Type()&os.ModeSymlink == 0 {
					items[child.Name()] = &SpaceItem{Name: child.Name(), Path: filepath.Join(target.Path, child.Name()), Directory: child.IsDir()}
				}
			}
		}
		err := filepath.WalkDir(target.Path, func(path string, entry fs.DirEntry, walkErr error) error {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if visited != nil {
				visited()
			}
			entries++
			if maxEntries > 0 && entries > maxEntries {
				return fmt.Errorf(localization.T("扫描已达到 %d 项上限"), maxEntries)
			}
			if walkErr != nil {
				group.Skipped++
				if path == target.Path {
					return walkErr
				}
				return nil
			}
			if path != target.Path && filepath.Dir(path) == filepath.Clean(target.Path) && entry.Type()&os.ModeSymlink == 0 {
				if item := items[entry.Name()]; item != nil {
					item.Scanned = true
				}
			}
			if entry.Type()&os.ModeSymlink != 0 || entry.IsDir() {
				return nil
			}
			info, err := entry.Info()
			if err != nil {
				group.Skipped++
				return nil
			}
			if !info.Mode().IsRegular() {
				return nil
			}
			relative, err := filepath.Rel(target.Path, path)
			if err != nil {
				return err
			}
			name := strings.SplitN(relative, string(filepath.Separator), 2)[0]
			item := items[name]
			if item == nil {
				item = &SpaceItem{Name: name, Path: filepath.Join(target.Path, name)}
				items[name] = item
			}
			item.Bytes += info.Size()
			item.Files++
			group.Bytes += info.Size()
			group.Files++
			return nil
		})
		if err != nil {
			group.Error = err.Error()
			result.Complete = false
		}
		if group.Skipped > 0 {
			result.Complete = false
		}
		for _, item := range items {
			group.Items = append(group.Items, *item)
		}
		slices.SortFunc(group.Items, func(a, b SpaceItem) int {
			if a.Bytes > b.Bytes {
				return -1
			}
			if a.Bytes < b.Bytes {
				return 1
			}
			return strings.Compare(a.Path, b.Path)
		})
		if maxItems > 0 && len(group.Items) > maxItems {
			group.Items = group.Items[:maxItems]
		}
		result.Groups = append(result.Groups, group)
		if ctx.Err() != nil || (maxEntries > 0 && entries > maxEntries) {
			if timeout == 0 && maxEntries == 0 {
				result.Warning = localization.T("扫描已中止，当前仅为已读部分。")
			} else {
				result.Warning = localization.T("扫描提前停止，当前仅为已读部分。可缩小 rockshinops.json 中的扫描范围后重试。")
			}
			break
		}
	}
	result.FinishedAt = time.Now().UTC()
	return result
}
