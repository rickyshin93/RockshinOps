package main

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/egoist/mygo"
	"rockshinops/internal/localization"
)

// The menu and session are owned by the main thread. An active session with
// a zero deadline is indefinite.
type keepAwake struct {
	menu          *mygo.MenuItem
	extendItem    *mygo.MenuItem
	displayItem   *mygo.MenuItem
	stopItem      *mygo.MenuItem
	separator     *mygo.MenuItem
	presets       []*mygo.MenuItem
	durations     []int
	extendMinutes int
	assertion     uint32
	deadline      time.Duration
	displayAwake  bool
	lastStatus    string
}

func newKeepAwake(minutes []int, extendMinutes int) *keepAwake {
	a := &keepAwake{
		menu: &mygo.MenuItem{Label: "保持唤醒"}, durations: minutes, extendMinutes: extendMinutes,
		separator: mygo.Separator(),
	}
	a.separator.Hidden = true
	a.extendItem = &mygo.MenuItem{Hidden: true, Click: func(*mygo.MenuItem, *mygo.Window) {
		awakeMenuError("无法延长唤醒会话", a.extend())
	}}
	a.displayItem = &mygo.MenuItem{Label: "保持屏幕亮起", Type: mygo.MenuItemCheckbox, Hidden: true, Click: func(*mygo.MenuItem, *mygo.Window) {
		awakeMenuError("无法更改屏幕唤醒状态", a.toggleDisplay())
	}}
	a.stopItem = &mygo.MenuItem{Label: "结束保持唤醒", Hidden: true, Click: func(*mygo.MenuItem, *mygo.Window) {
		awakeMenuError("无法结束保持唤醒", a.stop())
	}}
	a.menu.Submenu = []*mygo.MenuItem{a.extendItem, a.displayItem, a.stopItem, a.separator}
	startItem := func(label string, duration time.Duration) *mygo.MenuItem {
		return &mygo.MenuItem{Label: label, Click: func(*mygo.MenuItem, *mygo.Window) {
			awakeMenuError("无法保持唤醒", a.start(duration))
		}}
	}
	var hours, longHours []*mygo.MenuItem
	for _, minutes := range minutes {
		item := startItem("", time.Duration(minutes)*time.Minute)
		a.presets = append(a.presets, item)
		switch {
		case minutes%60 != 0:
			a.menu.Submenu = append(a.menu.Submenu, item)
		case minutes < 10*60:
			hours = append(hours, item)
		default:
			longHours = append(longHours, item)
		}
	}
	if len(hours) > 0 && len(longHours) > 0 {
		hours = append(hours, mygo.Separator())
	}
	hours = append(hours, longHours...)
	if len(hours) > 0 {
		a.menu.Submenu = append(a.menu.Submenu, &mygo.MenuItem{Label: "唤醒小时菜单", Submenu: hours})
	}
	a.menu.Submenu = append(a.menu.Submenu, startItem("无限期", 0))
	return a
}

func awakeMenuError(message string, err error) {
	if err != nil {
		go mygo.Dialog.Error(localization.T(message), err.Error())
	}
}

func (a *keepAwake) localize() {
	for i, minutes := range a.durations {
		a.presets[i].SetLabel(awakeDurationLabel(minutes))
	}
	a.extendItem.SetLabel(fmt.Sprintf(localization.T("延长 %d 分钟"), a.extendMinutes))
	a.refresh()
}

func awakeDurationLabel(minutes int) string {
	switch {
	case minutes == 60:
		return localization.T("1 小时")
	case minutes%60 == 0:
		return fmt.Sprintf(localization.T("%d 小时"), minutes/60)
	default:
		return fmt.Sprintf(localization.T("%d 分钟"), minutes)
	}
}

func (a *keepAwake) start(duration time.Duration) error {
	var deadline time.Duration
	if duration > 0 {
		deadline = awakeClock() + duration
	}
	return a.replace(deadline, false)
}

func (a *keepAwake) extend() error {
	a.refresh()
	if a.assertion == 0 || a.deadline == 0 {
		return nil
	}
	return a.replace(a.deadline+time.Duration(a.extendMinutes)*time.Minute, a.displayAwake)
}

func (a *keepAwake) toggleDisplay() error {
	a.refresh()
	if a.assertion == 0 {
		return nil
	}
	err := a.replace(a.deadline, !a.displayAwake)
	if err != nil {
		// MyGo toggles the menu check before calling this handler.
		a.displayItem.SetChecked(a.displayAwake)
	}
	return err
}

func (a *keepAwake) replace(deadline time.Duration, displayAwake bool) error {
	var remaining time.Duration
	if deadline != 0 {
		remaining = deadline - awakeClock()
		if remaining <= 0 {
			return a.stop()
		}
	}
	id, err := takeIdleSleepAssertion(remaining, displayAwake)
	if err != nil {
		return err
	}
	if a.assertion != 0 {
		if err := releaseIdleSleepAssertion(a.assertion); err != nil {
			return errors.Join(err, releaseIdleSleepAssertion(id))
		}
	}
	a.assertion, a.deadline, a.displayAwake = id, deadline, displayAwake
	a.syncActions()
	a.refresh()
	return nil
}

func (a *keepAwake) stop() error {
	if a.assertion == 0 {
		return nil
	}
	if err := releaseIdleSleepAssertion(a.assertion); err != nil {
		return err
	}
	a.clearSession()
	return nil
}

func (a *keepAwake) clearSession() {
	a.assertion, a.deadline, a.displayAwake = 0, 0, false
	a.syncActions()
	a.setStatus("")
}

func (a *keepAwake) syncActions() {
	active := a.assertion != 0
	a.extendItem.SetVisible(active && a.deadline != 0)
	a.displayItem.SetVisible(active)
	a.displayItem.SetChecked(a.displayAwake)
	a.stopItem.SetVisible(active)
	a.separator.SetVisible(active)
}

func (a *keepAwake) refresh() {
	a.refreshAt(time.Now(), awakeClock())
}

func (a *keepAwake) refreshAt(now time.Time, elapsed time.Duration) {
	if a.assertion == 0 {
		return
	}
	if a.deadline == 0 {
		a.setStatus(localization.T("保持唤醒中 · 无限期"))
		return
	}
	remaining := a.deadline - elapsed
	if remaining <= 0 {
		if err := a.stop(); err != nil {
			// The native timeout has already turned off the assertion.
			log.Printf("release expired idle sleep assertion: %v", err)
			a.clearSession()
		}
		return
	}
	a.setStatus(awakeCountdownLabel(now, now.Add(remaining)))
}

func awakeCountdownLabel(now, deadline time.Time) string {
	totalSeconds := int((deadline.Sub(now) + time.Second - 1) / time.Second)
	hours, minutes, seconds := totalSeconds/3600, totalSeconds/60%60, totalSeconds%60
	countdown := fmt.Sprintf("%ds", seconds)
	if minutes > 0 || hours > 0 {
		countdown = fmt.Sprintf("%dmin %s", minutes, countdown)
	}
	if hours > 0 {
		countdown = fmt.Sprintf("%dh %s", hours, countdown)
	}
	now, deadline = now.Local(), deadline.Local()
	end := deadline.Format("15:04")
	if now.Year() != deadline.Year() || now.YearDay() != deadline.YearDay() {
		end = deadline.Format(localization.T("唤醒结束日期格式"))
	}
	return fmt.Sprintf(localization.T("剩余 %s · %s 结束"), countdown, end)
}

func (a *keepAwake) setStatus(status string) {
	if a.lastStatus != status {
		a.menu.SetSubtitle(status)
		a.lastStatus = status
	}
}
