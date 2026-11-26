package main

import (
	"crypto/rand"
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

var errClipboardHistoryFull = errors.New("pinned clipboard items leave no room for this capture")

type clipboardFile struct {
	name, format string
	bytes        int64
	at           time.Time
	order        time.Time
	pinned       bool
}

// Each item is an immutable file. Its modification time is the displayed
// copy time. Recency overrides preserve copy order across clock rollback.
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
	order := map[string]time.Time{}
	if err := readClipboardJSON(filepath.Join(s.dir, ".recency.json"), &order); err != nil {
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
		files = append(files, clipboardFile{name: name, format: format, bytes: info.Size(), at: info.ModTime(), order: order[name], pinned: pins[name]})
		total += info.Size()
	}
	slices.SortFunc(files, compareClipboardFiles)
	return files, total, nil
}

func (f clipboardFile) sortTime() time.Time {
	if f.order.IsZero() {
		return f.at
	}
	return f.order
}

func compareClipboardFiles(a, b clipboardFile) int {
	if a.pinned != b.pinned {
		if a.pinned {
			return -1
		}
		return 1
	}
	if order := b.sortTime().Compare(a.sortTime()); order != 0 {
		return order
	}
	return strings.Compare(a.name, b.name)
}

func (s clipboardStore) recordOrder(files []clipboardFile, name string, at time.Time) (time.Time, error) {
	order := at
	overrides := make(map[string]time.Time)
	for _, file := range files {
		if !order.After(file.sortTime()) {
			order = file.sortTime().Add(time.Nanosecond)
		}
		if file.name != name && !file.order.IsZero() {
			overrides[file.name] = file.order
		}
	}
	if order.Equal(at) {
		order = time.Time{}
	} else {
		overrides[name] = order
	}
	return order, writeLocalJSON(filepath.Join(s.dir, ".recency.json"), overrides)
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

func (s clipboardStore) clearGeneration() (string, error) {
	data, err := os.ReadFile(filepath.Join(s.dir, ".cleared"))
	if errors.Is(err, fs.ErrNotExist) {
		return "", nil
	}
	if err == nil && len(data) == 0 {
		// Older running versions write an empty marker and update its timestamp.
		marker, err := os.Stat(filepath.Join(s.dir, ".cleared"))
		if err != nil {
			return "", err
		}
		return marker.ModTime().String(), nil
	}
	return string(data), err
}

func (s clipboardStore) record(content clipboardContent, at time.Time, generation string) (bool, error) {
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
		current, err := s.clearGeneration()
		if err != nil {
			return err
		}
		if current != generation {
			return nil // Clear History invalidates captures already in progress in any instance.
		}
		files, total, err := s.files()
		if err != nil {
			return err
		}
		path := filepath.Join(s.dir, name)
		for _, file := range files {
			if file.name == name {
				if _, err := s.recordOrder(files, name, at); err != nil {
					return err
				}
				if err := os.Chtimes(path, at, at); err != nil {
					return err
				}
				recorded = true
				return nil
			}
		}
		pinned := clipboardPinnedCount(files)
		pinnedBytes := int64(0)
		for _, file := range files[:pinned] {
			pinnedBytes += file.bytes
		}
		if pinned >= s.config.MaxEntries || pinnedBytes > s.config.MaxHistoryBytes-size {
			return errClipboardHistoryFull
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
		order, err := s.recordOrder(files, name, at)
		if err != nil {
			return err
		}
		files = append(files, clipboardFile{name: name, format: content.format, bytes: size, at: at, order: order})
		slices.SortFunc(files, compareClipboardFiles)
		keep, _ := clipboardRetention(files, total+size, s.config.MaxEntries, s.config.MaxHistoryBytes)
		if err := os.Rename(file.Name(), path); err != nil {
			return err
		}
		recorded = true
		return s.remove(files[keep:])
	})
	return recorded, err
}

func (s clipboardStore) clear() error {
	return s.locked(func() error {
		files, _, err := s.files()
		if err != nil {
			return err
		}
		if err := writeLocalJSON(filepath.Join(s.dir, ".cleared"), rand.Text()); err != nil {
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
