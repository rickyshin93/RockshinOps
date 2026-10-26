// Package localization shares the app's English and Chinese catalogs with the frontend.
package localization

import (
	"embed"
	"encoding/json"
	"strings"
	"sync/atomic"
)

//go:embed en.json zh.json
var catalogs embed.FS
var english, chinese map[string]string
var language atomic.Pointer[string]

func init() {
	for name, target := range map[string]*map[string]string{"en": &english, "zh": &chinese} {
		data, err := catalogs.ReadFile(name + ".json")
		if err != nil {
			panic(err)
		}
		if err = json.Unmarshal(data, target); err != nil {
			panic(err)
		}
	}
	SetLanguage("en")
}

func SetLanguage(tag string) {
	lang := "en"
	if strings.HasPrefix(strings.ToLower(tag), "zh") {
		lang = "zh"
	}
	language.Store(&lang)
}

func T(key string) string {
	catalog := english
	if *language.Load() == "zh" {
		catalog = chinese
	}
	if value, ok := catalog[key]; ok {
		return value
	}
	return key
}
