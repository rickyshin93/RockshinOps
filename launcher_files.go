package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"rockshinops/internal/fileindex"
	"rockshinops/internal/localization"
)

// Called with l.mu held. Wait for the previous publisher before replacing its
// index so a cancelled private scan cannot put results back after disabling it.
func (l *launcher) stopIndexing() {
	if l.stopFiles != nil {
		l.stopFiles()
		<-l.filesDone
		l.stopFiles = nil
	}
	l.shownMu.Lock()
	l.index.Store(nil)
	l.fileScope.Add(1)
	l.shown = nil
	l.shownMu.Unlock()
}

func (l *launcher) startIndexing() {
	rules := l.rules
	if l.weChat {
		rules.Include = append(append([]string{}, rules.Include...), l.weChatPaths...)
	}
	ctx, cancel := context.WithCancel(context.Background())
	l.stopFiles = cancel
	l.filesDone = make(chan struct{})
	l.indexing.Store(true)
	go func() {
		defer close(l.filesDone)
		defer l.indexing.Store(false)
		l.indexFiles(ctx, rules)
	}()
}

func (l *launcher) setWeChat(enabled bool) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("%s", localization.T("微信文件搜索仅支持 macOS。"))
	}
	if !l.prefs.Enabled || !l.prefs.Files {
		return fmt.Errorf("%s", localization.T("请先启用启动器和文件搜索。"))
	}
	if enabled == l.weChat {
		return nil
	}
	if enabled {
		found := false
		for _, pattern := range l.weChatPaths {
			// Glob suppresses filesystem errors. Read the parent explicitly so a
			// denied container or missing installation is reported to the user.
			prefix, _, _ := strings.Cut(pattern, "*")
			if _, err := os.ReadDir(filepath.Join(l.home, filepath.FromSlash(prefix))); err != nil {
				return fmt.Errorf("%s%w", localization.T("无法访问微信文件："), err)
			}
			matches, err := filepath.Glob(filepath.Join(l.home, filepath.FromSlash(pattern)))
			if err != nil {
				return err
			}
			found = found || len(matches) > 0
		}
		if !found {
			return fmt.Errorf("%s", localization.T("未找到可访问的微信文件目录，请确认微信已接收文件并允许访问后重试。"))
		}
	}
	l.stopIndexing()
	l.weChat = enabled
	l.startIndexing()
	return nil
}

// indexFiles builds the file index, follows changes with hints-only
// watches, and rebuilds it every l.rebuild as a safety net for changes a
// watch missed (or, where watching is unavailable, as the only update).
// Searches use the previous index until a rebuild finishes.
func (l *launcher) indexFiles(ctx context.Context, rules fileindex.Rules) {
	for {
		ix := fileindex.New(l.home, rules)
		if l.index.Load() == nil {
			l.index.Store(ix) // the first build is searchable while it runs
		}
		l.indexing.Store(true)
		start := time.Now()
		err := ix.Build(ctx)
		l.indexing.Store(false)
		if ctx.Err() != nil {
			return
		}
		if err != nil {
			log.Printf("file index: %v", err)
		}
		log.Printf("file index: %d entries in %v", ix.Len(), time.Since(start).Round(time.Millisecond))
		l.index.Store(ix)
		// A full scan creates large temporary directory lists and growing arenas.
		// Return that batch's unused pages before the index becomes idle.
		debug.FreeOSMemory()
		watching, stop := context.WithCancel(ctx)
		l.watchFiles(watching, ix)
		select {
		case <-ctx.Done():
			stop()
			return
		case <-time.After(l.rebuild):
			stop()
		}
	}
}

func (l *launcher) watchFiles(ctx context.Context, ix *fileindex.Index) {
	for k, root := range ix.Roots() {
		prefix := "" // the watched folder relative to the home folder, for the skip rules
		if k > 0 {
			rel, _ := filepath.Rel(l.home, root)
			prefix = filepath.ToSlash(rel)
		}
		w, err := mygo.WatchFiles(ctx, root, mygo.FileWatchOptions{
			Recursive: true, HintsOnly: true,
			Ignore: func(p string, isDir bool) bool { return isDir && p != "." && ix.Skipped(path.Join(prefix, p)) },
		})
		if err != nil {
			log.Printf("file index: watch %s: %v", root, err)
			continue
		}
		go func() {
			defer w.Close()
			for {
				events, err := w.Next(ctx)
				if err != nil {
					return
				}
				for _, e := range events {
					if e.Op == mygo.FileHint {
						ix.Refresh(filepath.Join(root, filepath.FromSlash(e.Path)))
					}
				}
			}
		}()
	}
}
