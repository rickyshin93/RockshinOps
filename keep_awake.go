package main

import (
	"fmt"
	"log"
	"time"

	"github.com/egoist/mygo"
	"rockshinops/internal/localization"
)

// The menu, deadline and assertion are owned by the main thread.
type keepAwake struct {
	menu       *mygo.MenuItem
	stopItem   *mygo.MenuItem
	durations  []int
	assertion  uint32
	deadline   time.Time
	lastStatus string
}

func newKeepAwake(minutes []int) *keepAwake {
	a := &keepAwake{menu: &mygo.MenuItem{Label: "保持唤醒"}, durations: minutes}
	for _, minutes := range minutes {
		a.menu.Submenu = append(a.menu.Submenu, &mygo.MenuItem{Click: func(*mygo.MenuItem, *mygo.Window) {
			if err := a.start(time.Duration(minutes) * time.Minute); err != nil {
				go mygo.Dialog.Error(localization.T("无法保持唤醒"), err.Error())
			}
		}})
	}
	a.stopItem = &mygo.MenuItem{Label: "结束保持唤醒", Disabled: true, Click: func(*mygo.MenuItem, *mygo.Window) {
		if err := a.stop(); err != nil {
			go mygo.Dialog.Error(localization.T("无法结束保持唤醒"), err.Error())
		}
	}}
	a.menu.Submenu = append(a.menu.Submenu, mygo.Separator(), a.stopItem)
	return a
}

func (a *keepAwake) localize() {
	for i, minutes := range a.durations {
		a.menu.Submenu[i].SetLabel(awakeDurationLabel(minutes))
	}
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
	id, err := takeIdleSleepAssertion(duration)
	if err != nil {
		return err
	}
	if a.assertion != 0 {
		if err := releaseIdleSleepAssertion(a.assertion); err != nil {
			if cleanupErr := releaseIdleSleepAssertion(id); cleanupErr != nil {
				log.Printf("release replacement idle sleep assertion: %v", cleanupErr)
			}
			return err
		}
	}
	a.assertion, a.deadline = id, time.Now().Round(0).Add(duration)
	a.stopItem.SetEnabled(true)
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
	a.assertion = 0
	a.stopItem.SetEnabled(false)
	a.setStatus("")
	return nil
}

func (a *keepAwake) refresh() {
	if a.assertion == 0 {
		return
	}
	remaining := time.Until(a.deadline)
	if remaining <= 0 {
		if err := a.stop(); err != nil {
			// The system timeout has already turned the assertion off.
			log.Printf("release expired idle sleep assertion: %v", err)
			a.assertion = 0
			a.stopItem.SetEnabled(false)
			a.setStatus("")
		}
		return
	}
	seconds := int((remaining + time.Second - 1) / time.Second)
	a.setStatus(fmt.Sprintf(localization.T("剩余 %s"), fmt.Sprintf("%02d:%02d", seconds/60, seconds%60)))
}

func (a *keepAwake) setStatus(status string) {
	if a.lastStatus != status {
		a.menu.SetSubtitle(status)
		a.lastStatus = status
	}
}
