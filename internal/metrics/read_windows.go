package metrics

import (
	"errors"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
	"localops/internal/localization"
	"unsafe"
)

// DISK_PERFORMANCE, returned for a volume by IOCTL_DISK_PERFORMANCE.
type diskPerformance struct {
	BytesRead, BytesWritten, ReadTime, WriteTime, IdleTime int64
	ReadCount, WriteCount, QueueDepth, SplitCount          uint32
	QueryTime                                              int64
	StorageDeviceNumber                                    uint32
	StorageManagerName                                     [8]uint16
}

func readDisks() ([]ioCounter, error) {
	buffer := make([]uint16, 256)
	n, err := windows.GetLogicalDriveStrings(uint32(len(buffer)), &buffer[0])
	if err != nil {
		return nil, err
	}
	result := []ioCounter{}
	for _, letter := range buffer[:n] {
		if letter < 'A' || letter > 'Z' {
			continue
		}
		drive := string(rune(letter)) + ":"
		if windows.GetDriveType(windows.StringToUTF16Ptr(drive+`\`)) != windows.DRIVE_FIXED {
			continue
		}
		handle, err := windows.CreateFile(windows.StringToUTF16Ptr(`\\.\`+drive), 0, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE, nil, windows.OPEN_EXISTING, 0, 0)
		if err != nil {
			return nil, err
		}
		var value diskPerformance
		var size uint32
		err = windows.DeviceIoControl(handle, 0x70020, nil, 0, (*byte)(unsafe.Pointer(&value)), uint32(unsafe.Sizeof(value)), &size, nil)
		windows.CloseHandle(handle)
		if errors.Is(err, windows.ERROR_INVALID_FUNCTION) || errors.Is(err, windows.ERROR_NOT_SUPPORTED) {
			continue
		}
		if err != nil {
			return nil, err
		}
		result = append(result, ioCounter{id: drive, read: uint64(value.BytesRead), write: uint64(value.BytesWritten)})
	}
	if len(result) == 0 {
		return nil, noCounters("Disk I/O")
	}
	return sortedCounters(result), nil
}

var getPowerStatus = windows.NewLazySystemDLL("kernel32.dll").NewProc("GetSystemPowerStatus")

type powerStatus struct {
	AC, Flags, Percent, Saver byte
	Seconds, FullSeconds      uint32
}

func readBattery() Battery {
	var status powerStatus
	ok, _, err := getPowerStatus.Call(uintptr(unsafe.Pointer(&status)))
	if ok == 0 {
		return Battery{Error: err.Error(), Minutes: -1}
	}
	return batteryFromStatus(status)
}

func batteryFromStatus(s powerStatus) Battery {
	b := Battery{Minutes: -1, ExternalPower: s.AC == 1}
	if s.Flags == 255 {
		b.Error = localization.T("Battery status unavailable")
		return b
	}
	if s.Flags&128 != 0 {
		return b
	}
	b.Present = true
	b.Charging = s.Flags&8 != 0
	if s.Percent > 100 {
		b.Error = localization.T("Battery charge unavailable")
		return b
	}
	b.Percent = float64(s.Percent)
	if s.Seconds != 0xffffffff {
		b.Minutes = int(s.Seconds / 60)
	}
	return b
}

func machineModel() string {
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, `HARDWARE\DESCRIPTION\System\BIOS`, registry.QUERY_VALUE)
	if err != nil {
		return ""
	}
	defer key.Close()
	model, _, err := key.GetStringValue("SystemProductName")
	if err != nil {
		return ""
	}
	return model
}
