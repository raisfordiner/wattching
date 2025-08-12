package msr

import (
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"syscall"
)

const (
	PKG_STATUS  = 0x611
	PP0_STATUS  = 0x639
	PP1_STATUS  = 0x641
	DRAM_STATUS = 0x619

	UNIT_MULTIPLIER = 0x606
)

type MSRFile struct {
	file *os.File
}

// Require Root
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

func (m *MSRFile) Close() error {
	return m.file.Close()
}

func (m *MSRFile) ReadMSR(offset int64) (uint64, error) {
	buf := make([]byte, 8)
	if _, err := m.file.ReadAt(buf, offset); err != nil {
		return 0, fmt.Errorf("failed to read MSR at offset 0x%x: %w", offset, err)
	}
	return binary.LittleEndian.Uint64(buf), nil
}

func (m *MSRFile) CheckMSR(offset int64) (bool, error) {
	_, err := m.ReadMSR(offset)
	if err != nil {
		var errno syscall.Errno
		if errors.As(err, &errno) && errno == syscall.EIO {
			return false, nil // MSR does not exist, but this is not a program error.
		}
		return false, err
	}
	return true, nil
}

