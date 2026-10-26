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

	"localops/internal/localization"
)

type ScanTarget struct {
	Name string `json:"name"`
	Path string `json:"path"`
}
type SpaceItem struct {
	Name  string `json:"name"`
	Path  string `json:"path"`
	Bytes int64  `json:"bytes"`
	Files int    `json:"files"`
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
	result := ScanResult{Groups: []ScanGroup{}, StartedAt: time.Now().UTC(), Complete: true}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	entries := 0
	for _, target := range targets {
		group := ScanGroup{Name: target.Name, Path: target.Path, Items: []SpaceItem{}}
		items := map[string]*SpaceItem{}
		err := filepath.WalkDir(target.Path, func(path string, entry fs.DirEntry, walkErr error) error {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			entries++
			if entries > maxEntries {
				return fmt.Errorf(localization.T("扫描已达到 %d 项上限"), maxEntries)
			}
			if walkErr != nil {
				group.Skipped++
				if path == target.Path {
					return walkErr
				}
				return nil
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
		group.Items = group.Items[:min(maxItems, len(group.Items))]
		result.Groups = append(result.Groups, group)
		if ctx.Err() != nil || entries > maxEntries {
			result.Warning = localization.T("扫描提前停止，当前仅为已读部分。可缩小 localops.json 中的扫描范围后重试。")
			break
		}
	}
	result.FinishedAt = time.Now().UTC()
	return result
}
