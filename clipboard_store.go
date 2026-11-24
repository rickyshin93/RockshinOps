package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

type clipboardContent struct {
	format string
	data   []byte
}

type clipboardFile struct {
	name, format string
	bytes        int64
	at           time.Time
	pinned       bool
}

// Each history item is an immutable file. Its modification time orders the
// history, so re-copying an existing item only updates its timestamp.
type clipboardStore struct {
	dir    string
	config clipboardHistoryConfig
}

func clipboardFileFormat(name string) string {
	hash, format, ok := strings.Cut(name, ".")
	if !ok || len(hash) != sha256.Size*2 {
		return ""
	}
	if _, err := hex.DecodeString(hash); err != nil {
		return ""
	}
	switch format {
	case "txt", "png", "tiff", "jpeg":
		return format
	default:
		return ""
	}
}

func (s clipboardStore) locked(action func() error) error {
	if err := os.MkdirAll(s.dir, 0700); err != nil {
		return err
	}
	lock, err := os.OpenFile(filepath.Join(s.dir, ".lock"), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return err
	}
	defer lock.Close()
	if err := lockCalculatorFile(lock); err != nil { // serialize the same local store across Dev and installed instances
		return err
	}
	return action()
}

// files runs under the store lock, before any new temporary file is created.
func (s clipboardStore) files() ([]clipboardFile, int64, error) {
	pins, err := s.readPins()
	if err != nil {
		return nil, 0, err
	}
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, 0, err
	}
	files := make([]clipboardFile, 0, len(entries))
	var total int64
	for _, entry := range entries {
		if !entry.Type().IsRegular() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, ".capture-") {
			if err := os.Remove(filepath.Join(s.dir, name)); err != nil {
				return nil, 0, err
			}
			continue
		}
		format := clipboardFileFormat(name)
		if format == "" {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return nil, 0, err
		}
		files = append(files, clipboardFile{name: name, format: format, bytes: info.Size(), at: info.ModTime(), pinned: pins[name]})
		total += info.Size()
	}
	slices.SortFunc(files, compareClipboardFiles)
	return files, total, nil
}

func compareClipboardFiles(a, b clipboardFile) int {
	if a.pinned != b.pinned {
		if a.pinned {
			return -1
		}
		return 1
	}
	if order := b.at.Compare(a.at); order != 0 {
		return order
	}
	return strings.Compare(a.name, b.name)
}

func clipboardPinnedCount(files []clipboardFile) int {
	for i, file := range files {
		if !file.pinned {
			return i
		}
	}
	return len(files)
}

func clipboardRetention(files []clipboardFile, total int64, maxEntries int, maxBytes int64) (int, int64) {
	keep := len(files)
	for keep > 0 && (keep > maxEntries || total > maxBytes) && !files[keep-1].pinned {
		keep--
		total -= files[keep].bytes
	}
	return keep, total
}

func (s clipboardStore) remove(files []clipboardFile) error {
	for _, file := range files {
		if err := os.Remove(filepath.Join(s.dir, file.name)); err != nil {
			return err
		}
	}
	return nil
}

func (s clipboardStore) record(content clipboardContent, at time.Time) (bool, error) {
	if len(content.data) == 0 || (content.format == "txt" && strings.TrimSpace(string(content.data)) == "") {
		return false, nil
	}
	digest := sha256.Sum256(content.data)
	name := hex.EncodeToString(digest[:]) + "." + content.format
	size := int64(len(content.data))
	if clipboardFileFormat(name) == "" || size > s.config.MaxHistoryBytes {
		return false, errors.New("clipboard item exceeds storage limits or has an unsupported format")
	}
	var recorded bool
	err := s.locked(func() error {
		if marker, err := os.Stat(filepath.Join(s.dir, ".cleared")); err == nil {
			if !at.After(marker.ModTime()) {
				return nil // a queued capture predates Clear History, including in another process
			}
		} else if !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		files, total, err := s.files()
		if err != nil {
			return err
		}
		path := filepath.Join(s.dir, name)
		for _, file := range files {
			if file.name == name {
				if !at.After(file.at) {
					return nil
				}
				if err := os.Chtimes(path, at, at); err != nil {
					return err
				}
				recorded = true
				return nil
			}
		}
		file, err := os.CreateTemp(s.dir, ".capture-")
		if err != nil {
			return err
		}
		defer os.Remove(file.Name())
		_, writeErr := file.Write(content.data)
		if writeErr == nil {
			writeErr = file.Sync()
		}
		if err := errors.Join(writeErr, file.Close()); err != nil {
			return err
		}
		if err := os.Chtimes(file.Name(), at, at); err != nil {
			return err
		}
		files = append(files, clipboardFile{name: name, format: content.format, bytes: size, at: at})
		slices.SortFunc(files, compareClipboardFiles)
		keep, _ := clipboardRetention(files, total+size, s.config.MaxEntries, s.config.MaxHistoryBytes)
		for _, file := range files[keep:] {
			if file.name == name {
				return nil // a capture from another process already displaced this stale item
			}
		}
		if err := s.remove(files[keep:]); err != nil {
			return err
		}
		if err := os.Rename(file.Name(), path); err != nil {
			return err
		}
		recorded = true
		return nil
	})
	return recorded, err
}

func (s clipboardStore) clear() error {
	return s.locked(func() error {
		marker := filepath.Join(s.dir, ".cleared")
		if err := os.WriteFile(marker, nil, 0600); err != nil {
			return err
		}
		now := time.Now()
		if err := os.Chtimes(marker, now, now); err != nil {
			return err
		}
		files, _, err := s.files()
		if err != nil {
			return err
		}
		return s.remove(files[clipboardPinnedCount(files):])
	})
}

func (s clipboardStore) entryPath(id string) (string, string, error) {
	name, _ := strings.CutPrefix(id, "clipboard:")
	format := clipboardFileFormat(name)
	if name == id || format == "" {
		return "", "", fs.ErrNotExist
	}
	path := filepath.Join(s.dir, name)
	info, err := os.Lstat(path)
	if err != nil {
		return "", "", err
	}
	if !info.Mode().IsRegular() {
		return "", "", fs.ErrNotExist
	}
	return path, format, nil
}

func (s clipboardStore) entry(id string) (clipboardContent, error) {
	path, format, err := s.entryPath(id)
	if err != nil {
		return clipboardContent{}, err
	}
	data, err := os.ReadFile(path)
	return clipboardContent{format: format, data: data}, err
}
