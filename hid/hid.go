package hid

import (
	"fmt"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

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
	hid                        = windows.NewLazySystemDLL("hid.dll")
	procHidD_GetAttributes     = hid.NewProc("HidD_GetAttributes")
	procHidD_GetPreparsedData  = hid.NewProc("HidD_GetPreparsedData")
	procHidD_FreePreparsedData = hid.NewProc("HidD_FreePreparsedData")
	procHidP_GetCaps           = hid.NewProc("HidP_GetCaps")
)

func openDevice(path string) (windows.Handle, error) {
	return windows.CreateFile(
		windows.StringToUTF16Ptr(path),
		windows.GENERIC_READ|windows.GENERIC_WRITE,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE,
		nil,
		windows.OPEN_EXISTING,
		0,
		0,
	)
}

func IsDeviceSupported(path string, vendorId uint16, productId uint16, inputBufferLength uint16, outputBufferLength uint16) (bool, error) {
	handle, err := openDevice(path)
	if err != nil {
		return false, fmt.Errorf("error while opening the device: %v", err)
	}
	defer windows.CloseHandle(handle)

	var attrs hiddAttributes
	attrs.size = uint32(unsafe.Sizeof(attrs))

	ret, _, _ := procHidD_GetAttributes.Call(
		uintptr(handle),
		uintptr(unsafe.Pointer(&attrs)),
	)

	if ret == 0 {
		return false, nil
	}

	if attrs.vendorID != vendorId || attrs.productID != productId {
		return false, nil
	}

	// Get Device Capabilities
	var preparsedData uintptr
	ret, _, _ = procHidD_GetPreparsedData.Call(
		uintptr(handle),
		uintptr(unsafe.Pointer(&preparsedData)),
	)

	if ret == 0 {
		return false, fmt.Errorf("HidD_GetPreparsedData failed: %v", err)
	}
	defer procHidD_FreePreparsedData.Call(preparsedData)

	var caps hidpCaps
	ret, _, _ = procHidP_GetCaps.Call(
		preparsedData,
		uintptr(unsafe.Pointer(&caps)),
	)
	if ret == 0 {
		return false, fmt.Errorf("HidP_GetCaps failed: %v", err)
	}

	if caps.inputReportByteLength != inputBufferLength || caps.outputReportByteLength != outputBufferLength {
		return false, nil
	}

	return true, nil
}

func GetBatteryLevel(path string, inputBufferLength uint16, outputBufferLength uint16, command []byte, batteryBufferIndex int) (int, error) {
	handle, err := openDevice(path)
	if err != nil {
		return 0, fmt.Errorf("error while opening the device: %v", err)
	}
	defer windows.CloseHandle(handle)

	// Create a buffer of the appropriate size
	outputBuffer := make([]byte, outputBufferLength)
	// Copy the command into the buffer
	copy(outputBuffer, command)

	// Send command
	var bytesWritten uint32
	err = windows.WriteFile(handle, outputBuffer, &bytesWritten, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to send command (WriteFile): %v", err)
	}

	// Wait a bit to let the device process the command
	time.Sleep(20 * time.Millisecond)

	// Read the answer
	inputBuffer := make([]byte, inputBufferLength)
	var bytesRead uint32
	err = windows.ReadFile(handle, inputBuffer, &bytesRead, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to read response (ReadFile): %v", err)
	}

	return int(inputBuffer[batteryBufferIndex]), nil
}
