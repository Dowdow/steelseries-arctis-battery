// Package hid provides minimal cross-platform access to HID devices.
//
// Each platform file implements the same API:
//
//	Enumerate() ([]DeviceInfo, error)
//	Open(path string) (*Device, error)
//	(*Device) Write(data []byte) error
//	(*Device) ReadTimeout(buf []byte, timeout time.Duration) (int, error)
//	(*Device) Close() error
package hid

import "errors"

// DeviceInfo describes an enumerated HID device interface.
type DeviceInfo struct {
	Path      string
	VendorID  uint16
	ProductID uint16
	// UsagePage of the interface's top-level collection; 0 means unknown.
	UsagePage uint16
	// Report lengths are only known on some platforms; 0 means unknown.
	InputReportLength  uint16
	OutputReportLength uint16
}

// ErrReadTimeout is returned by ReadTimeout when the device did not answer in time.
var ErrReadTimeout = errors.New("hid: read timeout")
