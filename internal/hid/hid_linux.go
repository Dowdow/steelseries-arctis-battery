//go:build linux

package hid

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sys/unix"
)

// Enumerate lists the hidraw device interfaces present on the system.
// Vendor and product IDs are read from sysfs; report lengths are left unknown.
func Enumerate() ([]DeviceInfo, error) {
	entries, err := os.ReadDir("/sys/class/hidraw")
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("error while listing hidraw devices: %w", err)
	}

	var devices []DeviceInfo
	for _, entry := range entries {
		uevent := filepath.Join("/sys/class/hidraw", entry.Name(), "device", "uevent")
		vendorID, productID, err := parseUevent(uevent)
		if err != nil {
			continue
		}

		descriptor := filepath.Join("/sys/class/hidraw", entry.Name(), "device", "report_descriptor")
		usagePage := parseUsagePage(descriptor)

		devices = append(devices, DeviceInfo{
			Path:      filepath.Join("/dev", entry.Name()),
			VendorID:  vendorID,
			ProductID: productID,
			UsagePage: usagePage,
		})
	}

	return devices, nil
}

// parseUevent extracts the vendor and product IDs from a sysfs uevent file,
// which contains a line like "HID_ID=0003:00001038:000012AD" (bus:vendor:product).
func parseUevent(path string) (uint16, uint16, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, 0, err
	}

	for _, line := range strings.Split(string(data), "\n") {
		value, ok := strings.CutPrefix(line, "HID_ID=")
		if !ok {
			continue
		}

		parts := strings.Split(value, ":")
		if len(parts) != 3 {
			break
		}

		vendorID, err := strconv.ParseUint(parts[1], 16, 32)
		if err != nil {
			break
		}
		productID, err := strconv.ParseUint(parts[2], 16, 32)
		if err != nil {
			break
		}

		return uint16(vendorID), uint16(productID), nil
	}

	return 0, 0, fmt.Errorf("no valid HID_ID in %s", path)
}

// parseUsagePage reads a HID report descriptor and returns the first Usage
// Page item, which is the usage page of the interface's top-level collection.
// Returns 0 when it cannot be determined.
func parseUsagePage(path string) uint16 {
	descriptor, err := os.ReadFile(path)
	if err != nil {
		return 0
	}

	for i := 0; i < len(descriptor); {
		item := descriptor[i]

		// Long item: skip tag, size byte and payload
		if item == 0xFE {
			if i+1 >= len(descriptor) {
				return 0
			}
			i += 3 + int(descriptor[i+1])
			continue
		}

		size := int(item & 0x03)
		if size == 3 {
			size = 4
		}

		// Usage Page: global item, tag 0 (prefix 0000 01 nn)
		if item&0xFC == 0x04 && i+size < len(descriptor) {
			switch size {
			case 1:
				return uint16(descriptor[i+1])
			case 2:
				return uint16(descriptor[i+1]) | uint16(descriptor[i+2])<<8
			}
			return 0
		}

		i += 1 + size
	}

	return 0
}

// Device is an open file descriptor to a hidraw device.
type Device struct {
	fd int
}

func Open(path string) (*Device, error) {
	fd, err := unix.Open(path, unix.O_RDWR|unix.O_NONBLOCK|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, fmt.Errorf("error while opening the device: %w", err)
	}
	return &Device{fd: fd}, nil
}

func (d *Device) Close() error {
	return unix.Close(d.fd)
}

func (d *Device) Write(data []byte) error {
	for {
		n, err := unix.Write(d.fd, data)
		if err == unix.EAGAIN {
			// The fd is non-blocking (for ReadTimeout's sake); wait for the
			// device to accept output before retrying.
			fds := []unix.PollFd{{Fd: int32(d.fd), Events: unix.POLLOUT}}
			if _, err := unix.Poll(fds, -1); err != nil && err != unix.EINTR {
				return fmt.Errorf("failed to poll device for write: %w", err)
			}
			continue
		}
		if err != nil {
			return fmt.Errorf("failed to send command: %w", err)
		}
		if n != len(data) {
			return fmt.Errorf("partial write: %d/%d bytes", n, len(data))
		}
		return nil
	}
}

func (d *Device) ReadTimeout(buf []byte, timeout time.Duration) (int, error) {
	deadline := time.Now().Add(timeout)

	for {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return 0, ErrReadTimeout
		}

		fds := []unix.PollFd{{Fd: int32(d.fd), Events: unix.POLLIN}}
		n, err := unix.Poll(fds, int(remaining.Milliseconds())+1)
		if err == unix.EINTR {
			continue
		}
		if err != nil {
			return 0, fmt.Errorf("failed to poll device: %w", err)
		}
		if n == 0 {
			return 0, ErrReadTimeout
		}

		read, err := unix.Read(d.fd, buf)
		if err == unix.EAGAIN {
			continue
		}
		if err != nil {
			return 0, fmt.Errorf("failed to read response: %w", err)
		}
		return read, nil
	}
}
