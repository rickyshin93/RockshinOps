package metrics

import (
	"golang.org/x/sys/windows"
	"net"
	"syscall"
	"unsafe"
)

// MIB_IF_ROW2 layout adapted from gopsutil (BSD-3-Clause; see THIRD_PARTY_NOTICES.md).
type mibIfRow2 struct {
	InterfaceLuid               uint64
	InterfaceIndex              uint32
	InterfaceGuid               windows.GUID
	Alias                       [257]uint16
	Description                 [257]uint16
	PhysicalAddressLength       uint32
	PhysicalAddress             [32]uint8
	PermanentPhysicalAddress    [32]uint8
	Mtu                         uint32
	Type                        uint32
	TunnelType                  uint32
	MediaType                   uint32
	PhysicalMediumType          uint32
	AccessType                  uint32
	DirectionType               uint32
	InterfaceAndOperStatusFlags uint32
	OperStatus                  uint32
	AdminStatus                 uint32
	MediaConnectState           uint32
	NetworkGuid                 windows.GUID
	ConnectionType              uint32
	padding1                    [(8 - unsafe.Alignof(uint64(0))) % 8]byte
	TransmitLinkSpeed           uint64
	ReceiveLinkSpeed            uint64
	InOctets                    uint64
	InUcastPkts                 uint64
	InNUcastPkts                uint64
	InDiscards                  uint64
	InErrors                    uint64
	InUnknownProtos             uint64
	InUcastOctets               uint64
	InMulticastOctets           uint64
	InBroadcastOctets           uint64
	OutOctets                   uint64
	OutUcastPkts                uint64
	OutNUcastPkts               uint64
	OutDiscards                 uint64
	OutErrors                   uint64
	OutUcastOctets              uint64
	OutMulticastOctets          uint64
	OutBroadcastOctets          uint64
	OutQLen                     uint64
}

var getIfEntry2 = windows.NewLazySystemDLL("iphlpapi.dll").NewProc("GetIfEntry2")

func readNetworks() ([]ioCounter, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	result := []ioCounter{}
	for _, i := range interfaces {
		row := mibIfRow2{InterfaceIndex: uint32(i.Index)}
		code, _, _ := getIfEntry2.Call(uintptr(unsafe.Pointer(&row)))
		if code != 0 {
			return nil, syscall.Errno(code)
		}
		result = append(result, ioCounter{id: i.Name, read: row.InOctets, write: row.OutOctets, up: row.OperStatus == 1, included: row.InterfaceAndOperStatusFlags&1 != 0 && row.InterfaceAndOperStatusFlags&2 == 0})
	}
	return sortedCounters(result), nil
}
