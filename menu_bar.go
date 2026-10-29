package main

import (
	"fmt"
	"log"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/egoist/mygo"
	"localops/internal/localization"
	"localops/internal/menubar"
)

type MenuBarConfiguration struct {
	Settings menubar.Settings `json:"settings"`
	State    menubar.State    `json:"state"`
}
type menuBarService struct {
	mu           sync.Mutex
	settings     menubar.Settings
	path         string
	startupError string
}

var menuBar menuBarService

func menuBarStrings() map[string]string {
	keys := []string{"打开 RockshinOps", "菜单栏设置…", "应用设置…", "刷新图标", "退出 RockshinOps", "请先关闭当前图标菜单，再操作其他图标。", "系统未确认移动事件，请检查辅助功能权限后重试。", "打开展开栏", "请先启用始终隐藏分区。", "图标位于可用菜单栏之外，请在空间更大的显示器上按住 Command 拖动。", "移动菜单栏图标前请松开鼠标。", "macOS 未将图标移入目标分区，请在菜单栏中按住 Command 拖动。", "展开或收起隐藏图标", "分区边界", "按住 Command 拖动图标到分区边界左侧以隐藏", "搜索菜单栏图标", "显示全部分区", "请先退出 Ice，避免两个菜单栏管理器同时运行。", "请先启用菜单栏管理", "展开栏需要屏幕录制权限以显示真实菜单栏图标。", "此分区没有图标", "需要辅助功能权限以操作菜单栏图标。", "图标已经退出，请刷新后重试。", "此应用未提供可操作的菜单栏辅助功能，请在菜单栏中手动打开。", "无法打开此图标，请展开分区后从系统菜单栏点击。"}
	out := map[string]string{}
	for _, key := range keys {
		out[key] = localization.T(key)
	}
	return out
}
func applyMenuBar(s menubar.Settings) error {
	_, err := menubar.Native("apply", map[string]any{"settings": s, "strings": menuBarStrings()})
	return err
}
func menuBarHotkeys(s menubar.Settings) map[string]string {
	return map[string]string{"toggle": s.ToggleHotkey, "search": s.SearchHotkey, "panel": s.PanelHotkey, "menus": s.MenusHotkey}
}
func registerMenuBarHotkeys(s menubar.Settings) error {
	if !s.Enabled {
		return nil
	}
	var registered []string
	for action, key := range menuBarHotkeys(s) {
		if key == "" {
			continue
		}
		action := action
		if err := mygo.GlobalShortcut.Register(key, func() {
			go func() {
				if _, err := menubar.Native(action, nil); err != nil {
					log.Printf("menu bar shortcut: %v", err)
				}
			}()
		}); err != nil {
			for _, claimed := range registered {
				mygo.GlobalShortcut.Unregister(claimed)
			}
			return fmt.Errorf("%s: %w", key, err)
		}
		registered = append(registered, key)
	}
	return nil
}
func unregisterMenuBarHotkeys(s menubar.Settings) {
	if s.Enabled {
		for _, key := range menuBarHotkeys(s) {
			if key != "" {
				mygo.GlobalShortcut.Unregister(key)
			}
		}
	}
}
func startMenuBar(defaults menubar.Settings, dir string) {
	menuBar.path = filepath.Join(dir, "menu-bar.json")
	saved, err := menubar.Load(menuBar.path, defaults)
	if err != nil {
		menuBar.startupError = err.Error()
		menuBar.settings = defaults
		return
	}
	menuBar.settings = saved
	if runtime.GOOS != "darwin" {
		return
	}
	if err = applyMenuBar(saved); err == nil {
		err = registerMenuBarHotkeys(saved)
	}
	if err != nil {
		menuBar.startupError = err.Error()
		_, _ = menubar.Native("stop", nil)
	}
}
func (s *System) MenuBarConfiguration() (MenuBarConfiguration, error) {
	menuBar.mu.Lock()
	defer menuBar.mu.Unlock()
	state, err := menubar.Native("state", nil)
	if menuBar.startupError != "" {
		state.Error = menuBar.startupError
	}
	return MenuBarConfiguration{Settings: menuBar.settings, State: state}, err
}
func (s *System) SaveMenuBarSettings(next menubar.Settings) (MenuBarConfiguration, error) {
	if err := next.Validate(); err != nil {
		return MenuBarConfiguration{}, err
	}
	menuBar.mu.Lock()
	defer menuBar.mu.Unlock()
	old := menuBar.settings
	if runtime.GOOS != "darwin" {
		return MenuBarConfiguration{}, fmt.Errorf("menu bar management is only available on macOS")
	}
	if err := applyMenuBar(next); err != nil {
		return MenuBarConfiguration{}, err
	}
	unregisterMenuBarHotkeys(old)
	err := registerMenuBarHotkeys(next)
	if err == nil {
		err = menubar.Save(menuBar.path, next)
		if err != nil {
			unregisterMenuBarHotkeys(next)
		}
	}
	if err != nil {
		if rollback := applyMenuBar(old); rollback != nil {
			err = fmt.Errorf("%w; restore runtime: %v", err, rollback)
		}
		if rollback := registerMenuBarHotkeys(old); rollback != nil {
			err = fmt.Errorf("%w; restore shortcuts: %v", err, rollback)
		}
		return MenuBarConfiguration{}, err
	}
	menuBar.settings = next
	menuBar.startupError = ""
	state, err := menubar.Native("state", nil)
	return MenuBarConfiguration{Settings: next, State: state}, err
}
func (s *System) MenuBarAction(action string) (menubar.State, error) {
	switch action {
	case "state", "refresh", "toggle", "expand", "collapse", "all", "panel", "search", "menus":
		return menubar.Native(action, nil)
	}
	return menubar.State{}, fmt.Errorf("unknown menu bar action: %s", action)
}
func (s *System) MenuBarPermission(permission string) (menubar.State, error) {
	if permission != "accessibility" && permission != "screenRecording" {
		return menubar.State{}, fmt.Errorf("unknown permission")
	}
	return menubar.Native("permission", permission)
}
func (s *System) ActivateMenuBarItem(id uint32) (menubar.State, error) {
	return menubar.Native("activate", id)
}
func (s *System) MoveMenuBarItem(id uint32, pid int, section string, before uint32) (menubar.State, error) {
	if section != "visible" && section != "hidden" && section != "alwaysHidden" {
		return menubar.State{}, fmt.Errorf("invalid menu bar section")
	}
	return menubar.Native("move", map[string]any{"id": id, "pid": pid, "section": section, "before": before})
}
