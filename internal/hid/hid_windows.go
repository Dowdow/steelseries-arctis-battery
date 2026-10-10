//go:build windows

package hid

import (
	"fmt"
	"time"
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

// Struct for storing HID attributes
type hiddAttributes struct {
	size          uint32
	vendorID      uint16
	productID     uint16
	versionNumber uint16
}

// Struct for storing HID capacities
type hidpCaps struct {
	usage                     uint16
	usagePage                 uint16
	inputReportByteLength     uint16
	outputReportByteLength    uint16
	featureReportByteLength   uint16
	reserved                  [17]uint16
	numberLinkCollectionNodes uint16
	numberInputButtonCaps     uint16
	numberInputValueCaps      uint16
	numberInputDataIndices    uint16
	numberOutputButtonCaps    uint16
	numberOutputValueCaps     uint16
	numberOutputDataIndices   uint16
	numberFeatureButtonCaps   uint16
	numberFeatureValueCaps    uint16
	numberFeatureDataIndices  uint16
}

// Load the necessary DLLs
var (
	setupapi                             = windows.NewLazySystemDLL("setupapi.dll")
	procSetupDiGetClassDevsW             = setupapi.NewProc("SetupDiGetClassDevsW")
	procSetupDiEnumDeviceInterfaces      = setupapi.NewProc("SetupDiEnumDeviceInterfaces")
	procSetupDiGetDeviceInterfaceDetailW = setupapi.NewProc("SetupDiGetDeviceInterfaceDetailW")
	procSetupDiDestroyDeviceInfoList     = setupapi.NewProc("SetupDiDestroyDeviceInfoList")

	hidDLL                     = windows.NewLazySystemDLL("hid.dll")
	procHidD_GetAttributes     = hidDLL.NewProc("HidD_GetAttributes")
	procHidD_GetPreparsedData  = hidDLL.NewProc("HidD_GetPreparsedData")
	procHidD_FreePreparsedData = hidDLL.NewProc("HidD_FreePreparsedData")
	procHidP_GetCaps           = hidDLL.NewProc("HidP_GetCaps")
)

// GUID for HID devices
var guidDevinterfaceHID = windows.GUID{
	Data1: 0x4d1e55b2,
	Data2: 0xf16f,
	Data3: 0x11cf,
	Data4: [8]byte{0x88, 0xcb, 0x00, 0x11, 0x11, 0x00, 0x00, 0x30},
}

// Enumerate lists the HID device interfaces present on the system.
// Devices that cannot be opened (exclusive access, permissions) are skipped.
func Enumerate() ([]DeviceInfo, error) {
	paths, err := scanDevicePaths()
	if err != nil {
		return nil, err
	}

	var devices []DeviceInfo
	for _, path := range paths {
		info, err := queryDeviceInfo(path)
		if err != nil {
			continue
		}
		devices = append(devices, info)
	}

	return devices, nil
}

func scanDevicePaths() ([]string, error) {
	var paths []string

	// Get HID Device List
	deviceInfoSet, _, err := procSetupDiGetClassDevsW.Call(
		uintptr(unsafe.Pointer(&guidDevinterfaceHID)),
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
			uintptr(unsafe.Pointer(&guidDevinterfaceHID)),
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

func queryDeviceInfo(path string) (DeviceInfo, error) {
	device, err := Open(path)
	if err != nil {
		return DeviceInfo{}, err
	}
	defer device.Close()

	var attrs hiddAttributes
	attrs.size = uint32(unsafe.Sizeof(attrs))

	ret, _, callErr := procHidD_GetAttributes.Call(
		uintptr(device.handle),
		uintptr(unsafe.Pointer(&attrs)),
	)
	if ret == 0 {
		return DeviceInfo{}, fmt.Errorf("HidD_GetAttributes failed: %v", callErr)
	}

	var preparsedData uintptr
	ret, _, callErr = procHidD_GetPreparsedData.Call(
		uintptr(device.handle),
		uintptr(unsafe.Pointer(&preparsedData)),
	)
	if ret == 0 {
		return DeviceInfo{}, fmt.Errorf("HidD_GetPreparsedData failed: %v", callErr)
	}
	defer procHidD_FreePreparsedData.Call(preparsedData)

	var caps hidpCaps
	ret, _, callErr = procHidP_GetCaps.Call(
		preparsedData,
		uintptr(unsafe.Pointer(&caps)),
	)
	if ret == 0 {
		return DeviceInfo{}, fmt.Errorf("HidP_GetCaps failed: %v", callErr)
	}

	return DeviceInfo{
		Path:               path,
		VendorID:           attrs.vendorID,
		ProductID:          attrs.productID,
		UsagePage:          caps.usagePage,
		InputReportLength:  caps.inputReportByteLength,
		OutputReportLength: caps.outputReportByteLength,
	}, nil
}

// Device is an open handle to a HID device, using overlapped I/O so reads can time out.
type Device struct {
	handle     windows.Handle
	readEvent  windows.Handle
	writeEvent windows.Handle
}

func Open(path string) (*Device, error) {
	pathPtr, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return nil, err
	}

	handle, err := windows.CreateFile(
		pathPtr,
		windows.GENERIC_READ|windows.GENERIC_WRITE,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE,
		nil,
		windows.OPEN_EXISTING,
		windows.FILE_FLAG_OVERLAPPED,
		0,
	)
	if err != nil {
		return nil, fmt.Errorf("error while opening the device: %w", err)
	}

	readEvent, err := windows.CreateEvent(nil, 1, 0, nil)
	if err != nil {
		windows.CloseHandle(handle)
		return nil, err
	}

	writeEvent, err := windows.CreateEvent(nil, 1, 0, nil)
	if err != nil {
		windows.CloseHandle(readEvent)
		windows.CloseHandle(handle)
		return nil, err
	}

	return &Device{handle: handle, readEvent: readEvent, writeEvent: writeEvent}, nil
}

func (d *Device) Close() error {
	windows.CloseHandle(d.readEvent)
	windows.CloseHandle(d.writeEvent)
	return windows.CloseHandle(d.handle)
}

func (d *Device) Write(data []byte) error {
	overlapped := windows.Overlapped{HEvent: d.writeEvent}
	var written uint32

	err := windows.WriteFile(d.handle, data, &written, &overlapped)
	if err == windows.ERROR_IO_PENDING {
		err = windows.GetOverlappedResult(d.handle, &overlapped, &written, true)
	}
	if err != nil {
		return fmt.Errorf("failed to send command (WriteFile): %w", err)
	}

	return nil
}

func (d *Device) ReadTimeout(buf []byte, timeout time.Duration) (int, error) {
	overlapped := windows.Overlapped{HEvent: d.readEvent}
	var read uint32

	err := windows.ReadFile(d.handle, buf, &read, &overlapped)
	if err == windows.ERROR_IO_PENDING {
		status, waitErr := windows.WaitForSingleObject(d.readEvent, uint32(timeout.Milliseconds()))
		if waitErr != nil || status != windows.WAIT_OBJECT_0 {
			// Timed out or wait failed: cancel the pending read and reap it before returning
			windows.CancelIoEx(d.handle, &overlapped)
			windows.GetOverlappedResult(d.handle, &overlapped, &read, true)
			return 0, ErrReadTimeout
		}
		err = windows.GetOverlappedResult(d.handle, &overlapped, &read, false)
	}
	if err != nil {
		return 0, fmt.Errorf("failed to read response (ReadFile): %w", err)
	}

	return int(read), nil
}
