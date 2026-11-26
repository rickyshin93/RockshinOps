package main

import (
	"errors"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"rockshinops/internal/localization"
)

var ClipboardHistoryChanged = mygo.NewEvent[bool]("clipboardHistoryChanged")

const clipboardPreviewScheme = "clipboard-preview"

type clipboardHistoryConfig struct {
	PollMilliseconds int   `json:"pollMilliseconds"`
	MaxEntries       int   `json:"maxEntries"`
	MaxSnippets      int   `json:"maxSnippets"`
	MaxItemBytes     int   `json:"maxItemBytes"`
	MaxImageBytes    int   `json:"maxImageBytes"`
	MaxHistoryBytes  int64 `json:"maxHistoryBytes"`
}

type clipboardChange struct {
	count          int64
	content        clipboardContent
	accessRequired bool
	tooLarge       bool
}

// Recording and the native pasteboard belong to the main thread. The store
// performs file IO on RPC and capture workers, independently of the UI.
type clipboardHistory struct {
	store          clipboardStore
	stop           chan struct{}
	change         int64
	accessRequired bool
	warning        string
}

func (h *clipboardHistory) setRecording(enabled bool) {
	if enabled == (h.stop != nil) {
		return
	}
	h.accessRequired, h.warning = false, ""
	if !enabled {
		close(h.stop)
		h.stop = nil
		return
	}
	h.change = clipboardChangeCount()
	stop := make(chan struct{})
	h.stop = stop
	go func() {
		ticker := time.NewTicker(time.Duration(h.store.config.PollMilliseconds) * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				h.capture(stop)
			}
		}
	}()
}

func (h *clipboardHistory) capture(stop chan struct{}) {
	generation, err := h.store.clearGeneration()
	var sample clipboardChange
	var sampledAt time.Time
	mygo.RunOnMain(func() {
		if h.stop != stop {
			return
		}
		sampledAt = time.Now()
		config := h.store.config
		sample = readClipboardChange(h.change, config.MaxItemBytes, config.MaxImageBytes)
		accessChanged := h.accessRequired != sample.accessRequired
		h.change, h.accessRequired = sample.count, sample.accessRequired
		if sample.tooLarge && h.warning != "too-large" {
			h.warning = "too-large"
			accessChanged = true
		}
		if accessChanged {
			notifyClipboardHistoryChanged()
		}
	})
	if len(sample.content.data) == 0 {
		return
	}
	var recorded bool
	if err == nil {
		recorded, err = h.store.record(sample.content, sampledAt, generation)
	}
	if !recorded && err == nil {
		return
	}
	warning := ""
	if errors.Is(err, errClipboardHistoryFull) {
		warning = "pinned-full"
	} else if err != nil {
		log.Printf("save clipboard history: %v", err)
		warning = "write"
	}
	mygo.RunOnMain(func() {
		if h.stop == stop {
			h.warning = warning
		}
		notifyClipboardHistoryChanged()
	})
}

func notifyClipboardHistoryChanged() {
	if err := ClipboardHistoryChanged.Broadcast(true); err != nil {
		log.Printf("publish clipboard history change: %v", err)
	}
}

func (s clipboardStore) search(query string, limit int, includeSnippets bool, result *LauncherSearch) error {
	query = strings.ToLower(strings.TrimSpace(query))
	return s.locked(func() error {
		files, total, err := s.files()
		if err != nil {
			return err
		}
		keep, total := clipboardRetention(files, total, s.config.MaxEntries, s.config.MaxHistoryBytes)
		if err := s.remove(files[keep:]); err != nil {
			return err
		}
		result.ClipboardCount, result.ClipboardBytes = keep, total
		result.ClipboardPinnedCount = clipboardPinnedCount(files[:keep])
		for _, file := range files[:keep] {
			if limit > 0 && len(result.Items) == limit {
				break
			}
			item := LauncherResult{ID: "clipboard:" + file.name, Kind: "clipboard", Subtitle: file.at.Local().Format("01-02 15:04"), ClipboardPinned: file.pinned}
			if file.format == "txt" {
				text, err := os.ReadFile(filepath.Join(s.dir, file.name))
				if err != nil {
					return err
				}
				if query != "" && !strings.Contains(strings.ToLower(string(text)), query) {
					continue
				}
				item.Title = clipboardPreview(string(text))
			} else {
				if query != "" && !strings.Contains("image screenshot 图片 截图 "+file.format, query) {
					continue
				}
				item.Title, item.ClipboardImage = localization.T("图片"), true
				item.Subtitle = strings.ToUpper(file.format) + " · " + item.Subtitle
			}
			result.Items = append(result.Items, item)
		}
		if includeSnippets {
			return s.searchSnippets(query, result)
		}
		return nil
	})
}

func clipboardPreview(text string) string {
	text = strings.TrimSpace(text)
	// Limit what crosses the bridge. Copy still uses the full original text.
	const maxRunes = 160
	count := 0
	for i := range text {
		if count == maxRunes {
			return text[:i] + "…"
		}
		count++
	}
	return text
}

func (s clipboardStore) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path, format, err := s.entryPath(r.URL.Query().Get("id"))
	if err != nil || format == "txt" {
		http.NotFound(w, r)
		return
	}
	pixels := 112
	if r.URL.Path == "/preview" {
		pixels = 1440
	}
	data, err := clipboardImagePreview(path, pixels)
	if err != nil {
		http.Error(w, "clipboard image preview unavailable", http.StatusUnprocessableEntity)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-store")
	w.Write(data)
}

func (s *System) LauncherClipboardQuery(query string) (LauncherSearch, error) {
	return s.clipboardQuery(query, launcherResults, false)
}

func (s *System) ClipboardHistory(query string) (LauncherSearch, error) {
	return s.clipboardQuery(query, 0, true)
}

func (s *System) clipboardQuery(query string, limit int, includeSnippets bool) (LauncherSearch, error) {
	result := LauncherSearch{Items: []LauncherResult{}, ClipboardStatus: "on"}
	if s.clipboard == nil {
		return result, errors.ErrUnsupported
	}
	mygo.RunOnMain(func() {
		if s.clipboard.accessRequired {
			result.ClipboardStatus = "permission"
		}
		result.ClipboardWarning = s.clipboard.warning
	})
	result.ClipboardLimit = s.clipboard.store.config.MaxHistoryBytes
	err := s.clipboard.store.search(query, limit, includeSnippets, &result)
	return result, err
}

func (s *System) ClearClipboardHistory() error {
	if s.clipboard == nil {
		return errors.ErrUnsupported
	}
	mygo.RunOnMain(func() {
		if s.clipboard.stop != nil {
			s.clipboard.change = clipboardChangeCount()
		}
		s.clipboard.warning = ""
	})
	err := s.clipboard.store.clear()
	notifyClipboardHistoryChanged()
	return err
}

func (s *System) CopyClipboardHistory(id string) error {
	if s.clipboard == nil {
		return errors.ErrUnsupported
	}
	entry, err := s.clipboard.store.copyEntry(id)
	if err != nil {
		return err
	}
	mygo.RunOnMain(func() { err = writeClipboardContent(entry) })
	return err
}

func (s *System) SetClipboardPinned(id string, pinned bool) error {
	if s.clipboard == nil {
		return errors.ErrUnsupported
	}
	if err := s.clipboard.store.setPinned(id, pinned); err != nil {
		return err
	}
	notifyClipboardHistoryChanged()
	return nil
}

func (s *System) SaveClipboardSnippet(id, name string) error {
	if s.clipboard == nil {
		return errors.ErrUnsupported
	}
	if err := s.clipboard.store.saveSnippet(id, name); err != nil {
		return err
	}
	notifyClipboardHistoryChanged()
	return nil
}
