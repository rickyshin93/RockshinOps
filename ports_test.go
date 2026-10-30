package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestFavoritePortsPersistWithoutChangingSettings(t *testing.T) {
	dir := t.TempDir()
	service := &System{preferencesPath: filepath.Join(dir, "preferences.json")}
	initial, err := service.FavoritePorts()
	if err != nil || len(initial) != 0 || initial == nil {
		t.Fatalf("first run: %v %v", initial, err)
	}
	settings := []byte(`{"language":"zh","menuBarItems":["disk"]}`)
	if err := os.WriteFile(service.preferencesPath, settings, 0600); err != nil {
		t.Fatal(err)
	}
	want := []int{3000, 5432}
	if _, err := service.SaveFavoritePorts([]int{5432, 3000}); err != nil {
		t.Fatal(err)
	}
	restarted := &System{preferencesPath: service.preferencesPath}
	got, err := restarted.FavoritePorts()
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("restart: %v %v", got, err)
	}
	for _, invalid := range [][]int{{0}, {-1}, {65536}, {3000, 3000}} {
		if _, err := service.SaveFavoritePorts(invalid); err == nil {
			t.Fatalf("accepted %v", invalid)
		}
	}
	got, err = service.FavoritePorts()
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("invalid save changed favorites: %v %v", got, err)
	}
	if _, err := service.SaveFavoritePorts([]int{}); err != nil {
		t.Fatal(err)
	}
	got, err = restarted.FavoritePorts()
	if err != nil || len(got) != 0 || got == nil {
		t.Fatalf("empty favorites: %v %v", got, err)
	}
	after, err := os.ReadFile(service.preferencesPath)
	if err != nil || string(after) != string(settings) {
		t.Fatalf("unrelated settings changed: %s %v", after, err)
	}
}

func TestFavoritePortsReportsCorruptFileAndSaveFailure(t *testing.T) {
	dir := t.TempDir()
	service := &System{preferencesPath: filepath.Join(dir, "preferences.json")}
	for _, data := range []string{"null", "{}", "[1,1]", "[65536]", "invalid"} {
		if err := os.WriteFile(filepath.Join(dir, "ports.json"), []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := service.FavoritePorts(); err == nil {
			t.Fatalf("corrupt file silently accepted: %s", data)
		}
	}
	service.preferencesPath = filepath.Join(dir, "ports.json", "preferences.json")
	if _, err := service.SaveFavoritePorts([]int{3000}); err == nil {
		t.Fatal("failed write reported success")
	}
}
