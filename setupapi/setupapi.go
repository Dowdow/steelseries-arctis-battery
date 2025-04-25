package setupapi

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

// SetupAPI constants
const (
	digcfPresent         = 0x00000002
	digcfDeviceInterface = 0x00000010
)

type spDeviceInterfaceData struct {
	cbSize             uint32
	interfaceClassGuid windows.GUID
	flags              uint32
	reserved           uintptr
}

type spDeviceInterfaceDetailDataW struct {
	cbSize     uint32
	devicePath [1]uint16
}

// Load the necessary DLLs
var (
	setupapi                             = windows.NewLazySystemDLL("setupapi.dll")
	procSetupDiGetClassDevsW             = setupapi.NewProc("SetupDiGetClassDevsW")
	procSetupDiEnumDeviceInterfaces      = setupapi.NewProc("SetupDiEnumDeviceInterfaces")
	procSetupDiGetDeviceInterfaceDetailW = setupapi.NewProc("SetupDiGetDeviceInterfaceDetailW")
	procSetupDiDestroyDeviceInfoList     = setupapi.NewProc("SetupDiDestroyDeviceInfoList")
)

func ScanHIDDevices() ([]string, error) {
	var paths []string

	// GUIDs for HID devices
	GUID_DEVINTERFACE_HID := windows.GUID{
		Data1: 0x4d1e55b2,
		Data2: 0xf16f,
		Data3: 0x11cf,
		Data4: [8]byte{0x88, 0xcb, 0x00, 0x11, 0x11, 0x00, 0x00, 0x30},
	}

	// Get HID Device List
	deviceInfoSet, _, err := procSetupDiGetClassDevsW.Call(
		uintptr(unsafe.Pointer(&GUID_DEVINTERFACE_HID)),
		0,
		0,
		uintptr(digcfPresent|digcfDeviceInterface),
	)

	if deviceInfoSet == 0 {
		return nil, fmt.Errorf("SetupDiGetClassDevs failed: %v", err)
	}
	defer procSetupDiDestroyDeviceInfoList.Call(deviceInfoSet)

	// Browse each interface
	var deviceInterfaceData spDeviceInterfaceData
	deviceInterfaceData.cbSize = uint32(unsafe.Sizeof(deviceInterfaceData))

	for memberIndex := uint32(0); ; memberIndex++ {
		// List device interfaces
		ret, _, _ := procSetupDiEnumDeviceInterfaces.Call(
			deviceInfoSet,
			0,
			uintptr(unsafe.Pointer(&GUID_DEVINTERFACE_HID)),
			uintptr(memberIndex),
			uintptr(unsafe.Pointer(&deviceInterfaceData)),
		)

		if ret == 0 {
			break // No more interfaces
		}

		// Get the required size for the details
		var deviceInterfaceDetailDataSize uint32
		procSetupDiGetDeviceInterfaceDetailW.Call(
			deviceInfoSet,
			uintptr(unsafe.Pointer(&deviceInterfaceData)),
			0,
			0,
			uintptr(unsafe.Pointer(&deviceInterfaceDetailDataSize)),
			0,
		)

		// Allocate memory for details
		buf := make([]byte, deviceInterfaceDetailDataSize)
		detailData := (*spDeviceInterfaceDetailDataW)(unsafe.Pointer(&buf[0]))

		// On 64-bit Windows, cbSize should be 8, on 32-bit Windows, cbSize should be 6
		if unsafe.Sizeof(uintptr(0)) == 8 {
			detailData.cbSize = 8
		} else {
			detailData.cbSize = 6
		}

		// Get interface details
		ret, _, _ = procSetupDiGetDeviceInterfaceDetailW.Call(
			deviceInfoSet,
			uintptr(unsafe.Pointer(&deviceInterfaceData)),
			uintptr(unsafe.Pointer(detailData)),
			uintptr(deviceInterfaceDetailDataSize),
			0,
			0,
		)

		if ret == 0 {
			continue
		}

		// Convert device path to string
		path := windows.UTF16PtrToString((*uint16)(unsafe.Pointer(&detailData.devicePath[0])))
		paths = append(paths, path)
	}

	return paths, nil
}
