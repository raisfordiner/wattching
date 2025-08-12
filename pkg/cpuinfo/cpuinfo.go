package cpuinfo

import (
	"github.com/klauspost/cpuid"
)

// CPUInfo holds information about the CPU.
type CPUInfo struct {
	VendorString string
	BrandString  string
}

// GetCPUInfo gathers and returns information about the CPU.
func GetCPUInfo() CPUInfo {
	return CPUInfo{
		VendorString: cpuid.CPU.VendorString,
		BrandString:  cpuid.CPU.BrandName,
	}
}

