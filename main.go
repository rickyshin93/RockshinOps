package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/egoist/mygo"
	_ "github.com/egoist/mygo/locales/zh-Hans"
	"github.com/egoist/mygo/plugins/updater"
	"rockshinops/internal/appicon"
	"rockshinops/internal/localization"
	"rockshinops/internal/menubar"
	"rockshinops/internal/metrics"
	"rockshinops/internal/storage"
	"rockshinops/internal/windowchrome"
)

var MetricsChanged = mygo.NewEvent[metrics.Snapshot]("metricsChanged")
var SettingsOpened = mygo.NewEvent[bool]("settingsOpened")
var MenuBarSettingsOpened = mygo.NewEvent[bool]("menuBarSettingsOpened")

func (s *System) CheckForUpdates() { updater.CheckForUpdates() }

type AppInfo struct {
	Machine               metrics.Machine      `json:"machine"`
	Displays              []DisplayInfo        `json:"displays"`
	GoVersion             string               `json:"goVersion"`
	Version               string               `json:"version"`
	RefreshSeconds        int                  `json:"refreshSeconds"`
	DiskRefreshSeconds    int                  `json:"diskRefreshSeconds"`
	ProcessRefreshSeconds int                  `json:"processRefreshSeconds"`
	HistorySeconds        int                  `json:"historySeconds"`
	ScanTargets           []metrics.ScanTarget `json:"scanTargets"`
}

type DisplayInfo struct {
	Name    string  `json:"name"`
	Width   int     `json:"width"`
	Height  int     `json:"height"`
	Scale   float64 `json:"scale"`
	Primary bool    `json:"primary"`
}

type Bootstrap struct {
	Info     AppInfo          `json:"info"`
	Snapshot metrics.Snapshot `json:"snapshot"`
	History  []metrics.Trend  `json:"history"`
}

type System struct {
	startup         *startupSettings
	preferences     Preferences
	preferencesPath string
	mu              sync.Mutex
	refreshMu       sync.Mutex
	monitor         *metrics.Monitor
	config          Config
	info            AppInfo
	update          func(metrics.Snapshot)
	spaceMu         sync.Mutex
	space           *metrics.SpaceTree
	spaceSkipped    int64
	spaceStarted    time.Time
	spaceFinished   time.Time
	spaceProgress   atomic.Pointer[metrics.SpaceProgress]
	appRemovalMu    sync.Mutex
	appRemoval      appRemoval
	appReviewSeq    uint64
	scanCancelMu    sync.Mutex
	scanCancel      context.CancelFunc
}

func (s *System) Initialize() Bootstrap {
	snapshot := s.monitor.Current()
	if snapshot.UpdatedAt.IsZero() {
		snapshot = s.refresh(true)
	}
	info := s.info
	info.GoVersion = runtime.Version()
	info.Displays = []DisplayInfo{}
	for _, display := range mygo.Screen.Displays() {
		info.Displays = append(info.Displays, DisplayInfo{Name: display.Label, Width: display.Bounds.Width, Height: display.Bounds.Height, Scale: display.ScaleFactor, Primary: display.Primary})
	}
	return Bootstrap{Info: info, Snapshot: snapshot, History: s.monitor.History()}
}
func (s *System) Refresh() metrics.Snapshot { return s.refresh(true) }
func (s *System) History() []metrics.Trend  { return s.monitor.History() }
func (s *System) HistorySince(after string) ([]metrics.Trend, error) {
	at, err := time.Parse(time.RFC3339Nano, after)
	if err != nil {
		return nil, fmt.Errorf("invalid history timestamp: %w", err)
	}
	return s.monitor.HistorySince(at), nil
}
func (s *System) Processes() metrics.ProcessList { return s.monitor.Processes() }
func (s *System) Connections(ctx context.Context) (metrics.Connections, error) {
	return metrics.ReadConnections(ctx, time.Duration(s.config.ConnectionTimeoutSeconds)*time.Second, time.Duration(s.config.ConnectionSampleIntervalSeconds)*time.Second)
}

func (s *System) OpenFullDiskAccessSettings() error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("Full Disk Access settings are available on macOS only")
	}
	return mygo.Shell.OpenExternal("x-apple.systempreferences:com.apple.settings.PrivacySecurity.extension?Privacy_AllFiles")
}
func (s *System) HideWindow(ctx context.Context) { mygo.CallerWindow(ctx).Hide() }
func (s *System) refresh(force bool) metrics.Snapshot {
	s.refreshMu.Lock()
	defer s.refreshMu.Unlock()
	snapshot := s.monitor.Refresh(force)
	s.mu.Lock()
	s.update(snapshot)
	s.mu.Unlock()
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
	configDir, err = migrateConfigDirectory(configDir)
	if err != nil {
		log.Fatal(err)
	}
	preferencesPath := filepath.Join(configDir, "preferences.json")
	preferences, err := readPreferences(preferencesPath, cfg.DefaultMenuBarItems, cfg.DefaultLanguage, cfg.DefaultUpdateUseSystemProxy, cfg.DefaultHideDockOnClose)
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
	}), info: AppInfo{Machine: metrics.ReadMachine(time.Duration(cfg.MachineInfoTimeoutSeconds) * time.Second), Version: mygo.App.Version(), RefreshSeconds: cfg.MetricsIntervalSeconds,
		DiskRefreshSeconds: cfg.SampleIntervalSeconds, ProcessRefreshSeconds: cfg.ProcessIntervalSeconds,
		HistorySeconds: cfg.HistorySeconds, ScanTargets: targets}}
	configureUpdateProxy(service, systemUpdateProxy)
	service.startup = newStartupSettings(configDir)
	mygo.Bind(service)
	if err := mygo.Protocol.Handle(appicon.Scheme, appicon.Handler()); err != nil {
		log.Fatal(err)
	}

	mygo.App.SetLanguages("en", "zh")
	mygo.Use(updater.New(updater.Options{Interval: time.Duration(cfg.UpdateIntervalSeconds) * time.Second}))
	mygo.App.WhenReady(func() {
		applyLanguage(preferences.Language)
		localization.SetLanguage(mygo.App.Locale())
		service.startup.initialize(cfg.DefaultOpenAtLogin)
		win := mygo.NewWindow(mygo.WindowOptions{
			Title: "RockshinOps", URL: "/", Width: 1250, Height: 820,
			MinWidth: 900, MinHeight: 600, StateKey: "main", Hidden: true,
			TitleBarStyle:        mygo.TitleBarHiddenInset,
			TrafficLightPosition: &mygo.Point{X: 18, Y: 20},
			Transparent:          true,
			Vibrancy:             mygo.VibrancySidebar,
		})
		windowchrome.InstallTopDrag(win.NativeHandle())
		if !mygo.App.WasOpenedAtLogin() {
			win.Show()
		} else if preferences.HideDockOnClose {
			mygo.App.Dock.Hide()
		}
		openWindow := func() {
			mygo.App.Dock.Show()
			win.Show()
			win.Focus()
			go service.refresh(true)
		}
		openSettings := func(*mygo.MenuItem, *mygo.Window) {
			openWindow()
			if err := SettingsOpened.Broadcast(true); err != nil {
				log.Printf("open settings: %v", err)
			}
		}
		menubar.SetActionHandler(func(action string) {
			mygo.RunOnMain(func() {
				switch action {
				case "open":
					openWindow()
				case "settings":
					openSettings(nil, nil)
				case "menuBarSettings":
					openWindow()
					if err := MenuBarSettingsOpened.Broadcast(true); err != nil {
						log.Printf("open menu bar settings: %v", err)
					}
				case "quit":
					mygo.App.Quit()
				}
			})
		})
		startMenuBar(cfg.MenuBarManager, filepath.Dir(preferencesPath))
		mygo.App.SetMenu(localizedMenu([]*mygo.MenuItem{
			{Role: mygo.RoleAppMenu, Submenu: []*mygo.MenuItem{
				{Role: mygo.RoleAbout},
				{Label: "检查更新…", Click: func(*mygo.MenuItem, *mygo.Window) { updater.CheckForUpdates() }}, mygo.Separator(),
				{Label: "设置…", Accelerator: "CmdOrCtrl+,", Click: openSettings}, mygo.Separator(),
				{Role: mygo.RoleServices}, mygo.Separator(), {Role: mygo.RoleHide}, {Role: mygo.RoleHideOthers}, {Role: mygo.RoleUnhide}, mygo.Separator(), {Role: mygo.RoleQuit},
			}},
			{Role: mygo.RoleEditMenu}, viewMenu(mygo.IsDev()), {Role: mygo.RoleWindowMenu},
		}))
		capacityItem := &mygo.MenuItem{Label: "正在读取磁盘容量…", Disabled: true}
		updatedItem := &mygo.MenuItem{Label: "RockshinOps · 本机监控", Disabled: true}
		activityItem := &mygo.MenuItem{Label: "正在采样…", Disabled: true}
		tray, err := mygo.NewTray(mygo.TrayOptions{
			ID: "rockshinops.storage", Icon: diskIcon(), IconIsTemplate: true,
			Title: "—", ToolTip: localization.T("RockshinOps · 剩余可写空间"),
			Menu: localizedMenu([]*mygo.MenuItem{
				capacityItem, activityItem, updatedItem, mygo.Separator(),
				{Label: "打开 RockshinOps", Click: func(*mygo.MenuItem, *mygo.Window) { openWindow() }},
				{Label: "设置…", Click: openSettings},
				{Label: "检查更新…", Click: func(*mygo.MenuItem, *mygo.Window) { updater.CheckForUpdates() }},
				{Label: "立即刷新", Click: func(*mygo.MenuItem, *mygo.Window) { go service.refresh(true) }},
				mygo.Separator(), {Role: mygo.RoleQuit, Label: "退出 RockshinOps"},
			}),
		})
		if err != nil {
			log.Fatalf("create RockshinOps tray: %v", err)
		}
		lastTitle := "—"
		var lastImage []byte
		service.update = func(s metrics.Snapshot) {
			selectedTitle := menuBarTitle(service.preferences, s)
			selectedItems := append([]string{}, service.preferences.MenuBarItems...)
			mygo.RunOnMain(func() {
				title := "—"
				if s.Disk.CapacityError == "" {
					title = storage.FormatCapacity(s.Disk.Volume.Available)
					capacityItem.SetLabel(localization.T("剩余 ") + title + localization.T(" / 共 ") + storage.FormatCapacity(s.Disk.Volume.Total))
					tray.SetToolTip(localization.T("RockshinOps · 剩余可写空间 ") + title)
				} else {
					capacityItem.SetLabel(localization.T("容量读取失败"))
					tray.SetToolTip("RockshinOps · " + s.Disk.CapacityError)
				}
				// macOS shows icons with values as one template image; elsewhere, or if drawing fails, the labeled text title.
				image := menuBarImage(selectedItems, s)
				if image != nil && !bytes.Equal(image, lastImage) {
					if err := tray.SetIcon(image, true); err != nil {
						log.Printf("set menu bar image: %v", err)
						image = nil
					} else {
						tray.SetTitle("")
						lastImage, lastTitle = image, ""
					}
				}
				if image == nil && (selectedTitle != lastTitle || lastImage != nil) {
					if lastImage != nil {
						if err := tray.SetIcon(diskIcon(), true); err != nil {
							log.Printf("restore menu bar icon: %v", err)
						}
					}
					tray.SetTitle(selectedTitle)
					lastImage, lastTitle = nil, selectedTitle
				}
				tray.SetToolTip("RockshinOps" + func() string {
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
		mygo.App.OnBeforeQuit(func(*mygo.QuitEvent) { quitting = true; cancel(); _, _ = menubar.Native("stop", nil) })
		win.OnClose(func(e *mygo.CloseEvent) {
			if !quitting {
				e.PreventDefault()
				win.Hide()
				if service.Preferences().HideDockOnClose {
					mygo.App.Dock.Hide()
				}
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

func viewMenu(dev bool) *mygo.MenuItem {
	if dev {
		return &mygo.MenuItem{Role: mygo.RoleViewMenu}
	}
	// MyGo's default View menu includes reload shortcuts even in production.
	return &mygo.MenuItem{Role: mygo.RoleViewMenu, Submenu: []*mygo.MenuItem{
		{Role: mygo.RoleResetZoom}, {Role: mygo.RoleZoomIn}, {Role: mygo.RoleZoomOut},
		mygo.Separator(), {Role: mygo.RoleToggleFullScreen},
	}}
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
