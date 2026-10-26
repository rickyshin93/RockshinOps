package storage

import (
	"golang.org/x/sys/windows"
	"localops/internal/localization"
	"os"
	"strings"
	"unsafe"
)

var getVolumePath = windows.NewLazySystemDLL("kernel32.dll").NewProc("GetVolumePathNameW")

func Read() (Volume, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Volume{}, err
	}
	path, err := windows.UTF16PtrFromString(home)
	if err != nil {
		return Volume{}, err
	}
	var available, total, free uint64
	if err = windows.GetDiskFreeSpaceEx(path, &available, &total, &free); err != nil {
		return Volume{}, err
	}
	mount := make([]uint16, windows.MAX_PATH+1)
	ok, _, err := getVolumePath.Call(uintptr(unsafe.Pointer(path)), uintptr(unsafe.Pointer(&mount[0])), uintptr(len(mount)))
	if ok == 0 {
		return Volume{}, err
	}
	fs := make([]uint16, windows.MAX_PATH+1)
	if err = windows.GetVolumeInformation(&mount[0], nil, 0, nil, nil, nil, &fs[0], uint32(len(fs))); err != nil {
		return Volume{}, err
	}
	return Volume{Name: localization.T("用户数据盘"), Mount: windows.UTF16ToString(mount), Format: strings.ToUpper(windows.UTF16ToString(fs)), Total: total, Available: available}, nil
}
