package main

import (
	"context"
	"log"
	"path"
	"path/filepath"
	"time"

	"github.com/egoist/mygo"
	"rockshinops/internal/fileindex"
)

// indexFiles builds the file index, follows changes with hints-only
// watches, and rebuilds it every l.rebuild as a safety net for changes a
// watch missed (or, where watching is unavailable, as the only update).
// Searches use the previous index until a rebuild finishes.
func (l *launcher) indexFiles(ctx context.Context) {
	for {
		ix := fileindex.New(l.home, l.rules)
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
