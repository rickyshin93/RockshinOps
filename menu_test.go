package main

import (
	"testing"

	"github.com/egoist/mygo"
)

func TestViewMenuBuildMode(t *testing.T) {
	for _, dev := range []bool{false, true} {
		name := "production"
		if dev {
			name = "development"
		}
		t.Run(name, func(t *testing.T) {
			// Expand through MyGo too: a nil submenu silently restores its defaults.
			menu := mygo.NewMenu([]*mygo.MenuItem{viewMenu(dev)})
			roles := map[mygo.MenuRole]bool{}
			for _, item := range menu.Items()[0].Submenu {
				roles[item.Role] = true
				if !dev && (item.Accelerator == "CmdOrCtrl+R" || item.Accelerator == "Shift+CmdOrCtrl+R") {
					t.Errorf("production registers reload accelerator %q", item.Accelerator)
				}
			}
			for _, role := range []mygo.MenuRole{mygo.RoleReload, mygo.RoleForceReload, mygo.RoleToggleDevTools} {
				if roles[role] != dev {
					t.Errorf("role %s present = %v, want %v", role, roles[role], dev)
				}
			}
			for _, role := range []mygo.MenuRole{mygo.RoleResetZoom, mygo.RoleZoomIn, mygo.RoleZoomOut, mygo.RoleToggleFullScreen} {
				if !roles[role] {
					t.Errorf("missing normal view action %s", role)
				}
			}
		})
	}
}
