// Package menubar manages macOS status items independently of metric sampling.
package menubar

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
)

var actionHandler func(string)

// SetActionHandler installs the app actions before native menu-bar management starts.
func SetActionHandler(handler func(string)) { actionHandler = handler }

type Appearance struct {
	Tint   string `json:"tint"`
	Color  string `json:"color"`
	Shadow bool   `json:"shadow"`
	Border bool   `json:"border"`
	Shape  string `json:"shape"`
}
type Settings struct {
	Enabled                bool       `json:"enabled"`
	AlwaysHidden           bool       `json:"alwaysHidden"`
	UsePanel               bool       `json:"usePanel"`
	ShowOnHover            bool       `json:"showOnHover"`
	HoverDelay             float64    `json:"hoverDelay"`
	RehideDelay            float64    `json:"rehideDelay"`
	RevealCheckInterval    float64    `json:"revealCheckInterval"`
	TemporaryRevealSeconds float64    `json:"temporaryRevealSeconds"`
	ShowAllOnDrag          bool       `json:"showAllOnDrag"`
	HideAppMenus           bool       `json:"hideAppMenus"`
	SecondaryMenu          bool       `json:"secondaryMenu"`
	Divider                string     `json:"divider"`
	Dynamic                bool       `json:"dynamic"`
	Light                  Appearance `json:"light"`
	Dark                   Appearance `json:"dark"`
	ToggleHotkey           string     `json:"toggleHotkey"`
	SearchHotkey           string     `json:"searchHotkey"`
	PanelHotkey            string     `json:"panelHotkey"`
	MenusHotkey            string     `json:"menusHotkey"`
}
type Item struct {
	ID      uint32  `json:"id"`
	PID     int     `json:"pid"`
	Name    string  `json:"name"`
	Owner   string  `json:"owner"`
	Section string  `json:"section"`
	Image   string  `json:"image"`
	X       float64 `json:"x"`
	Width   float64 `json:"width"`
	Movable bool    `json:"movable"`
}
type State struct {
	Capturing       bool   `json:"capturing"`
	Supported       bool   `json:"supported"`
	Accessibility   bool   `json:"accessibility"`
	ScreenRecording bool   `json:"screenRecording"`
	Expanded        bool   `json:"expanded"`
	PanelVisible    bool   `json:"panelVisible"`
	IceRunning      bool   `json:"iceRunning"`
	Items           []Item `json:"items"`
	Error           string `json:"error"`
}

func (s Settings) Validate() error {
	if math.IsNaN(s.TemporaryRevealSeconds) || math.IsInf(s.TemporaryRevealSeconds, 0) || s.TemporaryRevealSeconds < 5 || s.TemporaryRevealSeconds > 60 {
		return fmt.Errorf("temporary menu bar reveal duration must be 5–60 seconds")
	}
	if math.IsNaN(s.RevealCheckInterval) || math.IsInf(s.RevealCheckInterval, 0) || s.RevealCheckInterval <= 0 || s.RevealCheckInterval > 5 {
		return fmt.Errorf("menu bar reveal check interval must be greater than 0 and at most 5 seconds")
	}
	if math.IsNaN(s.HoverDelay) || math.IsInf(s.HoverDelay, 0) || s.HoverDelay < 0 || s.HoverDelay > 5 || math.IsNaN(s.RehideDelay) || math.IsInf(s.RehideDelay, 0) || s.RehideDelay < 0 || s.RehideDelay > 300 {
		return fmt.Errorf("menu bar delays must be 0–5 and 0–300 seconds")
	}
	if s.Divider != "none" && s.Divider != "line" && s.Divider != "dot" {
		return fmt.Errorf("invalid divider style")
	}
	for _, a := range []Appearance{s.Light, s.Dark} {
		if a.Tint != "none" && a.Tint != "solid" && a.Tint != "gradient" {
			return fmt.Errorf("invalid menu bar tint")
		}
		if a.Shape != "none" && a.Shape != "rounded" && a.Shape != "split" {
			return fmt.Errorf("invalid menu bar shape")
		}
		if len(a.Color) != 7 || a.Color[0] != '#' || strings.IndexFunc(a.Color[1:], func(r rune) bool { return !strings.ContainsRune("0123456789abcdefABCDEF", r) }) >= 0 {
			return fmt.Errorf("invalid menu bar color")
		}
	}
	keys := map[string]bool{}
	for _, k := range []string{s.ToggleHotkey, s.SearchHotkey, s.PanelHotkey, s.MenusHotkey} {
		if k != "" {
			if keys[k] {
				return fmt.Errorf("duplicate menu bar shortcut: %s", k)
			}
			keys[k] = true
		}
	}
	return nil
}
func Load(path string, defaults Settings) (Settings, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return defaults, defaults.Validate()
	}
	if err != nil {
		return Settings{}, err
	}
	if err = json.Unmarshal(data, &defaults); err != nil {
		return Settings{}, err
	}
	return defaults, defaults.Validate()
}
func Save(path string, s Settings) error {
	if err := s.Validate(); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	if err = os.WriteFile(path+".tmp", append(data, '\n'), 0600); err != nil {
		return err
	}
	return os.Rename(path+".tmp", path)
}
