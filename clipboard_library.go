package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"rockshinops/internal/localization"
)

type clipboardSnippet struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Text string `json:"text"`
}

func readClipboardJSON(path string, value any) error {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	return json.Unmarshal(data, value)
}

func (s clipboardStore) readPins() (map[string]bool, error) {
	pins := make(map[string]bool)
	if err := readClipboardJSON(filepath.Join(s.dir, "pins.json"), &pins); err != nil {
		return nil, err
	}
	if pins == nil {
		return nil, errors.New("clipboard pins must be an object")
	}
	return pins, nil
}

func (s clipboardStore) setPinned(id string, pinned bool) error {
	return s.locked(func() error {
		path, _, err := s.entryPath(id)
		if err != nil {
			return err
		}
		pins, err := s.readPins()
		if err != nil {
			return err
		}
		name := filepath.Base(path)
		if pins[name] == pinned {
			return nil
		}
		if pinned {
			pins[name] = true
		} else {
			delete(pins, name)
		}
		return writeLocalJSON(filepath.Join(s.dir, "pins.json"), pins)
	})
}

func (s clipboardStore) readSnippets() ([]clipboardSnippet, error) {
	var snippets []clipboardSnippet
	err := readClipboardJSON(filepath.Join(s.dir, "snippets.json"), &snippets)
	return snippets, err
}

func (s clipboardStore) saveSnippet(id, name string) error {
	name = strings.TrimSpace(name)
	if name == "" || utf8.RuneCountInString(name) > 80 || strings.ContainsRune(name, 0) {
		return errors.New(localization.T("片段名称须为 1 到 80 个字符。"))
	}
	return s.locked(func() error {
		entry, err := s.entry(id)
		if err != nil {
			return err
		}
		if entry.format != "txt" {
			return errors.New(localization.T("只有文本可另存为片段。"))
		}
		text := string(entry.data)
		digest := sha256.Sum256([]byte(name + "\x00" + text))
		snippetID := "snippet:" + hex.EncodeToString(digest[:])
		snippets, err := s.readSnippets()
		if err != nil {
			return err
		}
		for _, snippet := range snippets {
			if snippet.ID == snippetID {
				return nil
			}
		}
		if len(snippets) >= s.config.MaxSnippets {
			return errors.New(localization.T("已达到片段数量上限。"))
		}
		snippets = append([]clipboardSnippet{{ID: snippetID, Name: name, Text: text}}, snippets...)
		return writeLocalJSON(filepath.Join(s.dir, "snippets.json"), snippets)
	})
}

func (s clipboardStore) searchSnippets(query string, result *LauncherSearch) error {
	snippets, err := s.readSnippets()
	if err != nil {
		return err
	}
	for _, snippet := range snippets {
		if !strings.Contains(strings.ToLower(snippet.Name), query) && !strings.Contains(strings.ToLower(snippet.Text), query) {
			continue
		}
		result.ClipboardSnippets = append(result.ClipboardSnippets, LauncherResult{
			ID: snippet.ID, Kind: "snippet", Title: snippet.Name, Subtitle: clipboardPreview(snippet.Text),
		})
	}
	return nil
}

func (s clipboardStore) copyEntry(id string) (clipboardContent, error) {
	var entry clipboardContent
	err := s.locked(func() error {
		if !strings.HasPrefix(id, "snippet:") {
			var err error
			entry, err = s.entry(id)
			return err
		}
		snippets, err := s.readSnippets()
		if err != nil {
			return err
		}
		for _, snippet := range snippets {
			if snippet.ID == id {
				entry = clipboardContent{format: "txt", data: []byte(snippet.Text)}
				return nil
			}
		}
		return fs.ErrNotExist
	})
	return entry, err
}
