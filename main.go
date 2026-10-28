package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/egoist/mygo"
	_ "github.com/egoist/mygo/locales/zh-Hans"
	"github.com/egoist/mygo/plugins/updater"
	"localops/internal/appicon"
	"localops/internal/localization"
	"localops/internal/metrics"
	"localops/internal/storage"
	"localops/internal/windowchrome"
)

var MetricsChanged = mygo.NewEvent[metrics.Snapshot]("metricsChanged")
var SettingsOpened = mygo.NewEvent[bool]("settingsOpened")

func (s *System) CheckForUpdates() { updater.CheckForUpdates() }

type AppInfo struct {
	Machine               metrics.Machine      `json:"machine"`
	Version               string               `json:"version"`
	RefreshSeconds        int                  `json:"refreshSeconds"`
	DiskRefreshSeconds    int                  `json:"diskRefreshSeconds"`
	ProcessRefreshSeconds int                  `json:"processRefreshSeconds"`
	HistorySeconds        int                  `json:"historySeconds"`
	ScanTargets           []metrics.ScanTarget `json:"scanTargets"`
}

type Bootstrap struct {
	Info     AppInfo          `json:"info"`
	Snapshot metrics.Snapshot `json:"snapshot"`
	History  []metrics.Trend  `json:"history"`
}

type System struct {
	preferences     Preferences
	preferencesPath string
	mu              sync.Mutex
	monitor         *metrics.Monitor
	config          Config
	info            AppInfo
	update          func(metrics.Snapshot)
}

func (s *System) Initialize() Bootstrap {
	snapshot := s.refresh(true)
	info := s.info
	info.Machine = metrics.ReadMachine()
	return Bootstrap{Info: info, Snapshot: snapshot, History: s.monitor.History()}
}
func (s *System) Refresh() metrics.Snapshot      { return s.refresh(true) }
func (s *System) History() []metrics.Trend       { return s.monitor.History() }
func (s *System) Processes() metrics.ProcessList { return s.monitor.Processes() }
func (s *System) Connections(ctx context.Context) (metrics.Connections, error) {
	return metrics.ReadConnections(ctx, time.Duration(s.config.ConnectionTimeoutSeconds)*time.Second)
}
func (s *System) ScanCleanup(ctx context.Context) metrics.ScanResult {
	return metrics.ScanSpace(ctx, s.info.ScanTargets, s.config.ScanMaxEntries, s.config.ScanMaxItems, time.Duration(s.config.ScanTimeoutSeconds)*time.Second)
}
func (s *System) HideWindow(ctx context.Context) { mygo.CallerWindow(ctx).Hide() }
func (s *System) refresh(force bool) metrics.Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	snapshot := s.monitor.Refresh(force)
	s.update(snapshot)
	if err := MetricsChanged.Broadcast(snapshot); err != nil {
		log.Printf("publish metrics: %v", err)
	}
	return snapshot
}

func main() {
	cfg, err := parseConfig(configJSON)
	if err != nil {
		log.Fatal(err)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}
	configDir, err := os.UserConfigDir()
	if err != nil {
		log.Fatal(err)
	}
	preferencesPath := filepath.Join(configDir, "LocalOps", "preferences.json")
	preferences, err := readPreferences(preferencesPath, cfg.DefaultMenuBarItems, cfg.DefaultLanguage)
	if err != nil {
		log.Fatal(err)
	}
	platformTargets := cfg.ScanTargets[runtime.GOOS]
	targets := make([]metrics.ScanTarget, len(platformTargets))
	for i, target := range platformTargets {
		if suffix, found := strings.CutPrefix(target.Path, "~/"); found {
			target.Path = filepath.Join(home, suffix)
		}
		targets[i] = target
	}
	service := &System{config: cfg, preferences: preferences, preferencesPath: preferencesPath, monitor: metrics.NewMonitor(metrics.Options{
		Interval:        time.Duration(cfg.MetricsIntervalSeconds) * time.Second,
		DiskInterval:    time.Duration(cfg.SampleIntervalSeconds) * time.Second,
		ProcessInterval: time.Duration(cfg.ProcessIntervalSeconds) * time.Second,
		HistorySeconds:  cfg.HistorySeconds,
	}), info: AppInfo{Machine: metrics.ReadMachine(), Version: mygo.App.Version(), RefreshSeconds: cfg.MetricsIntervalSeconds,
		DiskRefreshSeconds: cfg.SampleIntervalSeconds, ProcessRefreshSeconds: cfg.ProcessIntervalSeconds,
		HistorySeconds: cfg.HistorySeconds, ScanTargets: targets}}
	mygo.Bind(service)
	if err := mygo.Protocol.Handle(appicon.Scheme, appicon.Handler()); err != nil {
		log.Fatal(err)
	}

	mygo.App.SetLanguages("en", "zh")
	mygo.Use(updater.New(updater.Options{Interval: time.Duration(cfg.UpdateIntervalSeconds) * time.Second}))
	mygo.App.WhenReady(func() {
		applyLanguage(preferences.Language)
		localization.SetLanguage(mygo.App.Locale())
		win := mygo.NewWindow(mygo.WindowOptions{
			Title: "LocalOps", URL: "/", Width: 1250, Height: 820,
			MinWidth: 900, MinHeight: 600, StateKey: "main", Hidden: true,
			TitleBarStyle:        mygo.TitleBarHiddenInset,
			TrafficLightPosition: &mygo.Point{X: 18, Y: 20},
			Transparent:          true,
			Vibrancy:             mygo.VibrancySidebar,
		})
		windowchrome.InstallTopDrag(win.NativeHandle())
		win.Show()
		openWindow := func() { win.Show(); win.Focus(); go service.refresh(true) }
		openSettings := func(*mygo.MenuItem, *mygo.Window) {
			openWindow()
			if err := SettingsOpened.Broadcast(true); err != nil {
				log.Printf("open settings: %v", err)
			}
		}
		mygo.App.SetMenu(localizedMenu([]*mygo.MenuItem{
			{Role: mygo.RoleAppMenu, Submenu: []*mygo.MenuItem{
				{Role: mygo.RoleAbout},
				{Label: "检查更新…", Click: func(*mygo.MenuItem, *mygo.Window) { updater.CheckForUpdates() }}, mygo.Separator(),
				{Label: "设置…", Accelerator: "CmdOrCtrl+,", Click: openSettings}, mygo.Separator(),
				{Role: mygo.RoleServices}, mygo.Separator(), {Role: mygo.RoleHide}, {Role: mygo.RoleHideOthers}, {Role: mygo.RoleUnhide}, mygo.Separator(), {Role: mygo.RoleQuit},
			}},
			{Role: mygo.RoleEditMenu}, {Role: mygo.RoleViewMenu}, {Role: mygo.RoleWindowMenu},
		}))
		capacityItem := &mygo.MenuItem{Label: "正在读取磁盘容量…", Disabled: true}
		updatedItem := &mygo.MenuItem{Label: "LocalOps · 本机监控", Disabled: true}
		activityItem := &mygo.MenuItem{Label: "正在采样…", Disabled: true}
		tray, err := mygo.NewTray(mygo.TrayOptions{
			ID: "localops.storage", Icon: diskIcon(), IconIsTemplate: true,
			Title: "—", ToolTip: localization.T("LocalOps · 剩余可写空间"),
			Menu: localizedMenu([]*mygo.MenuItem{
				capacityItem, activityItem, updatedItem, mygo.Separator(),
				{Label: "打开 LocalOps", Click: func(*mygo.MenuItem, *mygo.Window) { openWindow() }},
				{Label: "设置…", Click: openSettings},
				{Label: "检查更新…", Click: func(*mygo.MenuItem, *mygo.Window) { updater.CheckForUpdates() }},
				{Label: "立即刷新", Click: func(*mygo.MenuItem, *mygo.Window) { go service.refresh(true) }},
				mygo.Separator(), {Role: mygo.RoleQuit, Label: "退出 LocalOps"},
			}),
		})
		if err != nil {
			log.Fatalf("create LocalOps tray: %v", err)
		}
		lastTitle := "—"
		service.update = func(s metrics.Snapshot) {
			selectedTitle := menuBarTitle(service.preferences, s)
			mygo.RunOnMain(func() {
				title := "—"
				if s.Disk.CapacityError == "" {
					title = storage.FormatCapacity(s.Disk.Volume.Available)
					capacityItem.SetLabel(localization.T("剩余 ") + title + localization.T(" / 共 ") + storage.FormatCapacity(s.Disk.Volume.Total))
					tray.SetToolTip(localization.T("LocalOps · 剩余可写空间 ") + title)
				} else {
					capacityItem.SetLabel(localization.T("容量读取失败"))
					tray.SetToolTip("LocalOps · " + s.Disk.CapacityError)
				}
				if selectedTitle != lastTitle {
					tray.SetTitle(selectedTitle)
					lastTitle = selectedTitle
				}
				tray.SetToolTip("LocalOps" + func() string {
					if selectedTitle == "" {
						return localization.T(" · 本机监控")
					}
					return " · " + selectedTitle
				}())
				if s.CPU.Ready && s.Memory.Error == "" {
					activityItem.SetLabel(fmt.Sprintf(localization.T("CPU %.0f%% · 内存 %s"), s.CPU.Usage, storage.FormatCapacity(s.Memory.Used)))
				} else {
					activityItem.SetLabel(localization.T("CPU / 内存采样中或不可用"))
				}
				updatedItem.SetLabel(localization.T("更新于 ") + s.UpdatedAt.Local().Format("15:04:05"))
			})
		}
		mygo.App.OnLocaleChanged(func(language string) {
			localization.SetLanguage(language)
			go service.refresh(true)
		})
		ctx, cancel := context.WithCancel(context.Background())
		quitting := false
		mygo.App.OnBeforeQuit(func(*mygo.QuitEvent) { quitting = true; cancel() })
		win.OnClose(func(e *mygo.CloseEvent) {
			if !quitting {
				e.PreventDefault()
				win.Hide()
			}
		})
		mygo.App.OnActivate(func(visible bool) {
			if !visible {
				openWindow()
			}
		})
		mygo.Power.OnResume(func() { go service.refresh(true) })
		go func() {
			service.refresh(true)
			ticker := time.NewTicker(time.Duration(cfg.MetricsIntervalSeconds) * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					service.refresh(false)
				}
			}
		}()
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}

func localizedMenu(items []*mygo.MenuItem) *mygo.Menu {
	var bind func([]*mygo.MenuItem)
	bind = func(items []*mygo.MenuItem) {
		for _, item := range items {
			if item.Label != "" {
				key := item.Label
				item.Label = localization.T(key)
				mygo.App.OnLocaleChanged(func(language string) {
					localization.SetLanguage(language)
					item.SetLabel(localization.T(key))
				})
			}
			bind(item.Submenu)
		}
	}
	bind(items)
	return mygo.NewMenu(items)
}
