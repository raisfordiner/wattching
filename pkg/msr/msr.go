package msr

import (
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"syscall"
)

const (
	// MSRs for power consumption
	PKG_STATUS = 0x611
	PP0_STATUS = 0x639
	PP1_STATUS = 0x641
	DRAM_STATUS = 0x619

	// MSR for unit multipliers
	UNIT_MULTIPLIER = 0x606
)

// MSRFile represents an opened MSR file for a specific CPU core.
type MSRFile struct {
	file *os.File
}

// OpenMSR opens the MSR file for a given CPU core.
// It requires root privileges.
func OpenMSR(cpu int) (*MSRFile, error) {
	path := fmt.Sprintf("/dev/cpu/%d/msr", cpu)
	file, err := os.OpenFile(path, os.O_RDONLY, 0)
	if err != nil {
		if os.IsPermission(err) {
			return nil, fmt.Errorf("failed to open MSR file: permission denied. Please run as root")
		}
		return nil, fmt.Errorf("failed to open MSR file %s: %w", path, err)
	}
	return &MSRFile{file: file}, nil
}

// Close closes the underlying MSR file.
func (m *MSRFile) Close() error {
	return m.file.Close()
}

// ReadMSR reads a 64-bit value from a specific MSR offset.
func (m *MSRFile) ReadMSR(offset int64) (uint64, error) {
	buf := make([]byte, 8)
	if _, err := m.file.ReadAt(buf, offset); err != nil {
		return 0, fmt.Errorf("failed to read MSR at offset 0x%x: %w", offset, err)
	}
	return binary.LittleEndian.Uint64(buf), nil
}

// CheckMSR checks if a given MSR is available for reading.
func (m *MSRFile) CheckMSR(offset int64) (bool, error) {
	_, err := m.ReadMSR(offset)
	if err != nil {
		var errno syscall.Errno
		// For non-existent MSRs, the driver returns EIO.
		if errors.As(err, &errno) && errno == syscall.EIO {
			return false, nil // MSR does not exist, but this is not a program error.
		}
		// For other errors (e.g., permissions), return the error.
		return false, err
	}
	return true, nil
}